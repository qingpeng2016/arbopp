package polymarket

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// sigType1271 标识 Polymarket V2 升级后新增的 POLY_1271 签名路径（新账号 / Deposit Wallet）。
// PlaceOrder 用它判断分发：account.SignatureType == sigType1271 → 走本文件 signOrderPoly1271；
// 其他取值（0=EOA, 1=POLY_PROXY, 2=GNOSIS_SAFE）仍走原 signOrder，不经过本文件任何代码。
const sigType1271 = 3

// Solady TypedDataSign + DepositWallet EIP-1271 包装签名相关常量
// 与官方 py-clob-client-v2 (ExchangeOrderBuilderV2) 完全对齐。
const (
	orderTypeStringV2 = "Order(uint256 salt,address maker,address signer,uint256 tokenId," +
		"uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType," +
		"uint256 timestamp,bytes32 metadata,bytes32 builder)"

	soladyTypeStringV2 = "TypedDataSign(Order contents,string name,string version,uint256 chainId," +
		"address verifyingContract,bytes32 salt)" + orderTypeStringV2

	domainTypeStringV2 = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"

	// L1 ClobAuthDomain 没有 verifyingContract，与下单 Exchange domain 不同。
	l1DomainTypeString = "EIP712Domain(string name,string version,uint256 chainId)"

	clobAuthTypeString = "ClobAuth(address address,string timestamp,uint256 nonce,string message)"

	soladyTypeStringClobAuth = "TypedDataSign(ClobAuth contents,string name,string version,uint256 chainId," +
		"address verifyingContract,bytes32 salt)" + clobAuthTypeString

	depositWalletDomainNameV2    = "DepositWallet"
	depositWalletDomainVersionV2 = "1"
)

// signOrderPoly1271 实现 POLY_1271 (signatureType=3) 路径的下单签名。
// 调用方是 PlaceOrder：当 account.SignatureType == sigType1271 时进入这里，
// 其他取值仍走原 p.signOrder，老账号路径不经过本文件任何代码。
//
// 账号字段语义（与老账号字段集合完全一致，不再单独引入 FunderAddress）：
//   - ProxyAddress：资金所在地址。老账号 = Polymarket 代理钱包；新账号 = Deposit Wallet 合约地址。
//   - EOAAddress / PrivateKey：内层 ECDSA 用的 EOA 及其私钥。
//
// 与老路径关键差异：
//  1. maker == signer == ProxyAddress（这里语义是 deposit wallet），而不是老路径里的 signer=EOA。
//  2. 签名不是普通 EIP-712 ECDSA，而是 Solady TypedDataSign + EIP-1271 包装结构：
//     0x | innerECDSA(65) | appDomainSeparator(32) | contentsHash(32) | contentsTypeString | contentsTypeLen(2 BE)
//  3. 内层 ECDSA 用 EOA 私钥签 digest =
//     keccak( 0x1901 || appDomainSep(CTF Exchange V2) || keccak(SoladyTypeHash || contentsHash
//     || depositWalletNameHash || depositWalletVersionHash
//     || chainId || signer || zeroSalt ) )
func (p *Polymarket) signOrderPoly1271(
	ctx context.Context,
	orderData map[string]interface{},
	account entity.PolymarketAccount,
	negRisk bool,
) (map[string]interface{}, string, error) {
	funder := strings.TrimSpace(account.ProxyAddress)
	if funder == "" {
		return nil, "", fmt.Errorf("POLY_1271 账号必须在 proxy_address 配置 Deposit Wallet 地址")
	}
	// 1) 金额 / 方向 / tokenID 计算（与 signOrder 一致，保持手续费 0）
	sideStr, sideNum, makerAmount, takerAmount, tokenID, err := p.computeOrderAmountsV2(orderData)
	if err != nil {
		return nil, "", err
	}
	feeRateBps := "0"

	// 2) 关键差异：maker == signer == funder（Deposit Wallet 地址）。
	// 与 rs-clob-client-v2 (Polymarket 官方 Rust SDK) 的 OrderBuilder Poly1271 分支一致。
	makerLower := strings.ToLower(funder)
	signerLower := strings.ToLower(funder)

	// 3) 其它字段
	salt, err := p.generateSalt()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate salt: %w", err)
	}
	timestamp := time.Now().UnixMilli()
	builder := zeroBytes32
	if v, ok := orderData["builder"].(string); ok && strings.TrimSpace(v) != "" {
		builder = strings.TrimSpace(v)
	}
	expiration := int64(0)
	if orderTypeStr, ok := orderData["order_type"].(string); ok {
		if OrderConditionType(strings.ToUpper(orderTypeStr)) == OrderConditionTypeGTD {
			expiration = time.Now().Add(24 * time.Hour).Unix()
		}
	}

	// 4) 用于签名的 Order 结构（11 字段，与 V2 EIP-712 Order struct 对齐）
	orderForSigning := map[string]interface{}{
		"salt":          salt,
		"maker":         makerLower,
		"signer":        signerLower,
		"tokenId":       tokenID,
		"makerAmount":   makerAmount,
		"takerAmount":   takerAmount,
		"side":          sideNum,
		"signatureType": sigType1271,
		"timestamp":     new(big.Int).SetInt64(timestamp),
		"metadata":      zeroBytes32,
		"builder":       builder,
	}

	// 5) Solady TypedDataSign + EIP-1271 包装签名
	signature, err := p.buildPoly1271Signature(ctx, orderForSigning, negRisk, account.PrivateKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build POLY_1271 signature: %w", err)
	}

	// 6) 输出 wire body：和老 signOrder 返回完全相同的形状
	saltAbs := new(big.Int).Abs(salt)
	maxInt64 := big.NewInt(9223372036854775807)
	if saltAbs.Cmp(maxInt64) > 0 {
		saltAbs.Mod(saltAbs, maxInt64)
	}
	saltInt64 := saltAbs.Int64()
	if saltInt64 == 0 {
		saltInt64 = 1
	}
	return map[string]interface{}{
		"salt":          saltInt64,
		"maker":         makerLower,
		"signer":        signerLower,
		"tokenId":       tokenID.String(),
		"makerAmount":   makerAmount.String(),
		"takerAmount":   takerAmount.String(),
		"expiration":    strconv.FormatInt(expiration, 10),
		"side":          sideStr,
		"signatureType": sigType1271,
		"timestamp":     strconv.FormatInt(timestamp, 10),
		"metadata":      zeroBytes32,
		"builder":       builder,
		"signature":     signature,
	}, feeRateBps, nil
}

// buildPoly1271Signature 构造 Solady TypedDataSign + EIP-1271 包装签名。
// 算法与 py-clob-client-v2 ExchangeOrderBuilderV2._build_poly_1271_order_signature 严格对齐。
func (p *Polymarket) buildPoly1271Signature(
	ctx context.Context,
	order map[string]interface{},
	negRisk bool,
	privateKey string,
) (string, error) {
	// (1)-(4) 纯计算：推导待签 digest 及包装所需的 appDomainSep / contentsHash。
	digest, appDomainSep, contentsHash, err := p.poly1271Digest(order, negRisk)
	if err != nil {
		return "", err
	}

	// (5) 用账号私钥对 digest 做本地 ECDSA。crypto.Sign 的 v 是 0/1，下面补成 27/28。
	innerSig, err := signLocal(privateKey, digest)
	if err != nil {
		return "", fmt.Errorf("inner ECDSA sign: %w", err)
	}
	// go-ethereum 返回 v ∈ {0,1}，EIP-1271 / eth_account 标准用 27/28
	if len(innerSig) != 65 {
		return "", fmt.Errorf("unexpected inner sig length: %d", len(innerSig))
	}
	if innerSig[64] < 27 {
		innerSig[64] += 27
	}

	// (6) 拼装最终签名：
	//   innerSig(65) || appDomainSep(32) || contentsHash(32) || contentsTypeString || contentsTypeLen(2 BE)
	contentsTypeBytes := []byte(orderTypeStringV2)
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(len(contentsTypeBytes)))

	final := make([]byte, 0, len(innerSig)+32+32+len(contentsTypeBytes)+2)
	final = append(final, innerSig...)
	final = append(final, appDomainSep...)
	final = append(final, contentsHash...)
	final = append(final, contentsTypeBytes...)
	final = append(final, lenBytes...)

	return hexutil.Encode(final), nil
}

// poly1271Digest 推导 POLY_1271 内层 ECDSA 待签的 32 字节 digest，并一并返回签名包装所需的
// appDomainSep / contentsHash。纯计算，便于单测验证 digest。
// 算法与 py-clob-client-v2 ExchangeOrderBuilderV2._build_poly_1271_order_signature 严格对齐。
func (p *Polymarket) poly1271Digest(order map[string]interface{}, negRisk bool) (digest, appDomainSep, contentsHash []byte, err error) {
	// domain 各字段统一取自 orderEIP712Domain，与 signOrderWithEIP712 共用同一份定义，
	// 避免两处各写一份导致 version/合约地址不一致（历史上就是因为 version 写成 1 导致下单被拒）。
	// domainTypeStringV2 的字段顺序必须与 orderEIP712DomainTypes 一致。
	domain := orderEIP712Domain(negRisk)
	domainName, _ := domain["name"].(string)
	domainVersion, _ := domain["version"].(string)
	domainChainID, _ := domain["chainId"].(int)
	domainVerifyingContract, _ := domain["verifyingContract"].(string)

	// (1) appDomainSeparator = keccak(DOMAIN_TYPE_HASH || nameHash || versionHash || chainId(32) || verifyingContract(32))
	domainTypeHash := crypto.Keccak256([]byte(domainTypeStringV2))
	ctfNameHash := crypto.Keccak256([]byte(domainName))
	ctfVersionHash := crypto.Keccak256([]byte(domainVersion))
	chainIDBytes := common.LeftPadBytes(big.NewInt(int64(domainChainID)).Bytes(), 32)
	verifyingContractBytes := common.LeftPadBytes(common.HexToAddress(domainVerifyingContract).Bytes(), 32)

	appDomainSep = crypto.Keccak256(
		concatBytes(domainTypeHash, ctfNameHash, ctfVersionHash, chainIDBytes, verifyingContractBytes),
	)

	// (2) contentsHash = keccak(ORDER_TYPE_HASH || abi.encode(order fields...))
	orderTypeHash := crypto.Keccak256([]byte(orderTypeStringV2))
	encodedOrder := make([]byte, 0, 32*12)
	encodedOrder = append(encodedOrder, orderTypeHash...)
	orderFields := []struct {
		name, kind string
	}{
		{"salt", "uint256"},
		{"maker", "address"},
		{"signer", "address"},
		{"tokenId", "uint256"},
		{"makerAmount", "uint256"},
		{"takerAmount", "uint256"},
		{"side", "uint8"},
		{"signatureType", "uint8"},
		{"timestamp", "uint256"},
		{"metadata", "bytes32"},
		{"builder", "bytes32"},
	}
	for _, f := range orderFields {
		enc, encErr := p.encodeValue(f.kind, order[f.name])
		if encErr != nil {
			return nil, nil, nil, fmt.Errorf("encode order field %s: %w", f.name, encErr)
		}
		encodedOrder = append(encodedOrder, enc...)
	}
	contentsHash = crypto.Keccak256(encodedOrder)

	// (3) typedDataSignStructHash = keccak(SOLADY_TYPE_HASH || contentsHash || depositNameHash || depositVersionHash
	//                                       || chainId(32) || signer(32) || zeroSalt(32))
	soladyTypeHash := crypto.Keccak256([]byte(soladyTypeStringV2))
	depositNameHash := crypto.Keccak256([]byte(depositWalletDomainNameV2))
	depositVersionHash := crypto.Keccak256([]byte(depositWalletDomainVersionV2))
	signerStr, ok := order["signer"].(string)
	if !ok {
		return nil, nil, nil, fmt.Errorf("order signer must be string")
	}
	signerBytes := common.LeftPadBytes(common.HexToAddress(signerStr).Bytes(), 32)
	zeroSalt := make([]byte, 32)

	tdsHash := crypto.Keccak256(
		concatBytes(
			soladyTypeHash, contentsHash, depositNameHash, depositVersionHash,
			chainIDBytes, signerBytes, zeroSalt,
		),
	)

	// (4) digest = keccak( 0x1901 || appDomainSep || tdsHash )
	digest = crypto.Keccak256(
		concatBytes([]byte{0x19, 0x01}, appDomainSep, tdsHash),
	)
	return digest, appDomainSep, contentsHash, nil
}

// computeOrderAmountsV2 复刻 signOrder 中 size/usdcAmount/maker-taker amount/side/tokenID 的计算逻辑。
// 该函数不修改 orderData。与老路径计算结果完全一致（仅 maker/signer 地址在 1271 路径里自行处理）。
func (p *Polymarket) computeOrderAmountsV2(orderData map[string]interface{}) (
	sideStr string, sideNum int, makerAmount, takerAmount, tokenID *big.Int, err error,
) {
	sideStr = strings.ToUpper(orderData["side"].(string))

	priceStr, _ := orderData["price"].(string)
	sizeStr, _ := orderData["size"].(string)
	amountStr, _ := orderData["amount"].(string)
	isMarket := priceStr == ""

	var size, usdcAmountDecimal *big.Int
	if isMarket {
		if sideStr == "BUY" {
			if amountStr == "" {
				return "", 0, nil, nil, nil, fmt.Errorf("市价买单必须提供 amount")
			}
			usdcAmountDecimal, err = p.parseDecimalToBigInt(amountStr, usdcDecimals)
			if err != nil {
				return "", 0, nil, nil, nil, fmt.Errorf("parse market amount: %w", err)
			}
			usdcAmountDecimal = new(big.Int).Mul(new(big.Int).Div(usdcAmountDecimal, big.NewInt(10000)), big.NewInt(10000))
			if usdcAmountDecimal.Sign() <= 0 {
				return "", 0, nil, nil, nil, fmt.Errorf("市价买单金额取整后无效或为负")
			}
			maxPrice := new(big.Int).SetInt64(990000)
			if worstStr, ok := orderData["market_worst_price"].(string); ok && worstStr != "" {
				if parsed, perr := p.parseDecimalToBigInt(worstStr, usdcDecimals); perr == nil && parsed.Sign() > 0 && parsed.Cmp(big.NewInt(1000000)) < 0 {
					if parsed.Cmp(big.NewInt(990000)) > 0 || parsed.Cmp(big.NewInt(10000)) < 0 {
						maxPrice = parsed
					}
				}
			}
			multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(conditionalTokenDecimals), nil)
			size = new(big.Int).Div(new(big.Int).Mul(usdcAmountDecimal, multiplier), maxPrice)
			if size.Sign() <= 0 {
				return "", 0, nil, nil, nil, fmt.Errorf("市价买单推导份额无效或为负")
			}
			size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(10000)), big.NewInt(10000))
			// 隐含价 = makerAmount/takerAmount = usdcAmount/size，必须严格 < 1，否则 CLOB 报 invalid price。
			// 特殊情况：worst_price 为 3 位小数(tick 0.001)，且 0.01 网格下 size 被压到 <= usdcAmount
			// (隐含价被抬到 >= 1.0)。此时改用 0.001 网格(该 tick 下实测可下单)按 worst_price 动态反算 size，
			// 使隐含价向上贴近最近一档(>= worst_price)：既能穿越卖一成交、又尽量不超付。
			// 仅对真正 3 位小数价位生效(maxPrice 是 0.001 的倍数但不是 0.01 的倍数)；0.01 价位维持原网格不动。
			// 必须 floor(向下取整)：size 取小→价取高→保证 ≥ 卖一；round 偏大会跌破卖一导致 FAK 撮合不到。
			if size.Cmp(usdcAmountDecimal) <= 0 &&
				new(big.Int).Rem(maxPrice, big.NewInt(1000)).Sign() == 0 &&
				new(big.Int).Rem(maxPrice, big.NewInt(10000)).Sign() != 0 {
				size = new(big.Int).Div(new(big.Int).Mul(usdcAmountDecimal, multiplier), maxPrice)
				size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(1000)), big.NewInt(1000))
				// 极小额动态兜底：若 0.001 网格仍塌陷，补到严格大于 usdcAmount 的最小 0.001 档(非固定增量)，保证隐含价 < 1
				if size.Cmp(usdcAmountDecimal) <= 0 {
					size = new(big.Int).Mul(new(big.Int).Add(new(big.Int).Div(usdcAmountDecimal, big.NewInt(1000)), big.NewInt(1)), big.NewInt(1000))
				}
			}
		} else {
			if sizeStr == "" {
				return "", 0, nil, nil, nil, fmt.Errorf("市价卖单必须提供 size")
			}
			size, err = p.parseDecimalToBigInt(sizeStr, conditionalTokenDecimals)
			if err != nil {
				return "", 0, nil, nil, nil, fmt.Errorf("parse market size: %w", err)
			}
			size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(10000)), big.NewInt(10000))
			if size.Sign() <= 0 {
				return "", 0, nil, nil, nil, fmt.Errorf("市价卖单份额取整后无效或为负")
			}
			minPrice := new(big.Int).SetInt64(10000)
			if worstStr, ok := orderData["market_worst_price"].(string); ok && worstStr != "" {
				if parsed, perr := p.parseDecimalToBigInt(worstStr, usdcDecimals); perr == nil && parsed.Sign() > 0 && parsed.Cmp(big.NewInt(1000000)) < 0 {
					if parsed.Cmp(big.NewInt(990000)) > 0 || parsed.Cmp(big.NewInt(10000)) < 0 {
						minPrice = parsed
					}
				}
			}
			divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(conditionalTokenDecimals), nil)
			usdcAmountDecimal = new(big.Int).Div(new(big.Int).Mul(size, minPrice), divisor)
			if usdcAmountDecimal.Sign() <= 0 {
				return "", 0, nil, nil, nil, fmt.Errorf("市价卖单推导金额无效或为负")
			}
			usdcAmountDecimal = new(big.Int).Mul(new(big.Int).Div(usdcAmountDecimal, big.NewInt(10000)), big.NewInt(10000))
		}
	} else {
		size, err = p.parseDecimalToBigInt(sizeStr, conditionalTokenDecimals)
		if err != nil {
			return "", 0, nil, nil, nil, fmt.Errorf("parse limit size: %w", err)
		}
		size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(10000)), big.NewInt(10000))
		if size.Sign() <= 0 {
			return "", 0, nil, nil, nil, fmt.Errorf("限价单 size 无效或为负")
		}
		priceDecimal, perr := p.parseDecimalToBigInt(priceStr, usdcDecimals)
		if perr != nil {
			return "", 0, nil, nil, nil, fmt.Errorf("parse limit price: %w", perr)
		}
		if priceDecimal.Sign() <= 0 {
			return "", 0, nil, nil, nil, fmt.Errorf("限价单 price 无效或为负")
		}
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(conditionalTokenDecimals), nil)
		usdcAmountDecimal = new(big.Int).Div(new(big.Int).Mul(priceDecimal, size), divisor)
		usdcAmountDecimal = new(big.Int).Mul(new(big.Int).Div(usdcAmountDecimal, big.NewInt(10000)), big.NewInt(10000))
		if usdcAmountDecimal.Sign() <= 0 {
			return "", 0, nil, nil, nil, fmt.Errorf("限价单金额取整后无效或为负")
		}
	}

	sideNum = 0
	if sideStr == "SELL" {
		sideNum = 1
	}
	if sideNum == 0 {
		makerAmount = new(big.Int).Set(usdcAmountDecimal)
		takerAmount = new(big.Int).Set(size)
	} else {
		makerAmount = new(big.Int).Set(size)
		takerAmount = new(big.Int).Set(usdcAmountDecimal)
	}

	tokenIDStr, _ := orderData["token_id"].(string)
	var ok bool
	tokenID, ok = new(big.Int).SetString(tokenIDStr, 10)
	if !ok {
		return "", 0, nil, nil, nil, fmt.Errorf("invalid token_id")
	}
	return sideStr, sideNum, makerAmount, takerAmount, tokenID, nil
}

// signLocal 用账号私钥对 32 字节 digest 做 secp256k1 签名，返回 65 字节 r||s||v，v 为 0 或 1。
func signLocal(privateKey string, digest []byte) ([]byte, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(privateKey), "0x")
	if raw == "" {
		return nil, fmt.Errorf("account private_key is required to sign the order")
	}
	if len(digest) != 32 {
		return nil, fmt.Errorf("order digest length %d != 32", len(digest))
	}
	key, err := crypto.HexToECDSA(raw)
	if err != nil {
		return nil, fmt.Errorf("parse account private key: %w", err)
	}
	return crypto.Sign(digest, key)
}

// concatBytes 把多个字节切片拼成一段，避免反复用 append 难读。
func concatBytes(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]byte, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// buildPoly1271ClobAuthSignature 用 ERC-7739 TypedDataSign 包装 ClobAuth，供 L1 create/derive
// 把 API Key 绑到 deposit wallet（POLY_ADDRESS=deposit），而不是 EOA。
//
// 形状与下单 POLY_1271 一致，只是 contents=ClobAuth、appDomain=ClobAuthDomain（无 verifyingContract）：
//
//	innerECDSA(65) || appDomainSep(32) || contentsHash(32) || ClobAuthTypeString || len(2 BE)
func (p *Polymarket) buildPoly1271ClobAuthSignature(
	privateKeyHex, depositWallet, timestamp string, nonce int64,
) (string, error) {
	depositWallet = strings.TrimSpace(depositWallet)
	if depositWallet == "" {
		return "", fmt.Errorf("POLY_1271 L1 auth requires deposit wallet address")
	}
	digest, appDomainSep, contentsHash, err := p.poly1271ClobAuthDigest(depositWallet, timestamp, nonce)
	if err != nil {
		return "", err
	}
	innerSig, err := signLocal(privateKeyHex, digest)
	if err != nil {
		return "", fmt.Errorf("ClobAuth inner ECDSA sign: %w", err)
	}
	if len(innerSig) != 65 {
		return "", fmt.Errorf("unexpected ClobAuth inner sig length: %d", len(innerSig))
	}
	if innerSig[64] < 27 {
		innerSig[64] += 27
	}

	contentsTypeBytes := []byte(clobAuthTypeString)
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(len(contentsTypeBytes)))

	final := make([]byte, 0, len(innerSig)+32+32+len(contentsTypeBytes)+2)
	final = append(final, innerSig...)
	final = append(final, appDomainSep...)
	final = append(final, contentsHash...)
	final = append(final, contentsTypeBytes...)
	final = append(final, lenBytes...)
	return hexutil.Encode(final), nil
}

// poly1271ClobAuthDigest 推导 L1 ClobAuth 的 ERC-7739 内层 digest。
// ClobAuth.address / TypedDataSign.verifyingContract / POLY_ADDRESS 都是 deposit wallet。
func (p *Polymarket) poly1271ClobAuthDigest(
	depositWallet, timestamp string, nonce int64,
) (digest, appDomainSep, contentsHash []byte, err error) {
	chainIDBytes := common.LeftPadBytes(big.NewInt(int64(eip712ChainID)).Bytes(), 32)
	depositBytes := common.LeftPadBytes(common.HexToAddress(depositWallet).Bytes(), 32)

	// (1) appDomainSep = ClobAuthDomain separator（无 verifyingContract）
	appDomainSep = crypto.Keccak256(concatBytes(
		crypto.Keccak256([]byte(l1DomainTypeString)),
		crypto.Keccak256([]byte("ClobAuthDomain")),
		crypto.Keccak256([]byte(clobAuthDomainVersion)),
		chainIDBytes,
	))

	// (2) contentsHash = keccak(ClobAuth typeHash || encode fields)，address=deposit
	clobAuthTypeHash := crypto.Keccak256([]byte(clobAuthTypeString))
	encoded := make([]byte, 0, 32*5)
	encoded = append(encoded, clobAuthTypeHash...)
	for _, f := range []struct {
		kind string
		val  interface{}
	}{
		{"address", strings.ToLower(depositWallet)},
		{"string", timestamp},
		{"uint256", nonce},
		{"string", clobAuthMessage},
	} {
		enc, encErr := p.encodeValue(f.kind, f.val)
		if encErr != nil {
			return nil, nil, nil, fmt.Errorf("encode ClobAuth field %s: %w", f.kind, encErr)
		}
		encoded = append(encoded, enc...)
	}
	contentsHash = crypto.Keccak256(encoded)

	// (3) TypedDataSign struct hash：DepositWallet domain，verifyingContract=deposit
	tdsHash := crypto.Keccak256(concatBytes(
		crypto.Keccak256([]byte(soladyTypeStringClobAuth)),
		contentsHash,
		crypto.Keccak256([]byte(depositWalletDomainNameV2)),
		crypto.Keccak256([]byte(depositWalletDomainVersionV2)),
		chainIDBytes,
		depositBytes,
		make([]byte, 32),
	))

	// (4) digest = keccak(0x1901 || appDomainSep || tdsHash)
	digest = crypto.Keccak256(concatBytes([]byte{0x19, 0x01}, appDomainSep, tdsHash))
	return digest, appDomainSep, contentsHash, nil
}
