package polymarket

import (
	"context"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestPolymarketExchangeDomainMatchesSourceContract(t *testing.T) {
	if eip712DomainVersion != "2" || ctfExchangeAddress != "0xE111180000d2663C0091e4f400237545B87B996B" || negRiskCTFExchangeAddress != "0xe2222d279d744050d28e00520010520000310F59" {
		t.Fatalf("EIP-712 domain mismatch: version=%q ctf=%q negRisk=%q", eip712DomainVersion, ctfExchangeAddress, negRiskCTFExchangeAddress)
	}
}

const (
	testFunder  = "0x1234567890123456789012345678901234567890"
	testTokenID = "71321045679252212594626385532706912750332728571942532289631379312455583992585"
)

func newTestPolymarket() *Polymarket {
	// digest 推导 / 字段校验流程不依赖 client / baseURL，零值结构体即可。
	return &Polymarket{}
}

func limitBuyOrderData() map[string]interface{} {
	return map[string]interface{}{
		"side":       "BUY",
		"price":      "0.5",
		"size":       "100",
		"amount":     "",
		"token_id":   testTokenID,
		"order_type": "GTC",
	}
}

// poly1271TestOrder 构造一个字段类型与 signOrderPoly1271 产出一致的待签 order（maker==signer==funder）。
func poly1271TestOrder() map[string]interface{} {
	tokenID, _ := new(big.Int).SetString(testTokenID, 10)
	return map[string]interface{}{
		"salt":          big.NewInt(123456789),
		"maker":         strings.ToLower(testFunder),
		"signer":        strings.ToLower(testFunder),
		"tokenId":       tokenID,
		"makerAmount":   big.NewInt(2000000),
		"takerAmount":   big.NewInt(1980000),
		"side":          0,
		"signatureType": sigType1271,
		"timestamp":     big.NewInt(1781767975002),
		"metadata":      zeroBytes32,
		"builder":       zeroBytes32,
	}
}

// TestPoly1271TypeStringsMatchOfficial 验证三个 EIP-712 类型字符串与官方 py-clob-client-v2 字节级一致。
// 任何 typo 都会让链上 1271 验签直接失败。
func TestPoly1271TypeStringsMatchOfficial(t *testing.T) {
	wantOrder := "Order(uint256 salt,address maker,address signer,uint256 tokenId," +
		"uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType," +
		"uint256 timestamp,bytes32 metadata,bytes32 builder)"
	wantSolady := "TypedDataSign(Order contents,string name,string version,uint256 chainId," +
		"address verifyingContract,bytes32 salt)" + wantOrder
	wantDomain := "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"

	if orderTypeStringV2 != wantOrder {
		t.Errorf("orderTypeStringV2 mismatch\n  got:  %q\n  want: %q", orderTypeStringV2, wantOrder)
	}
	if soladyTypeStringV2 != wantSolady {
		t.Errorf("soladyTypeStringV2 mismatch\n  got:  %q\n  want: %q", soladyTypeStringV2, wantSolady)
	}
	if domainTypeStringV2 != wantDomain {
		t.Errorf("domainTypeStringV2 mismatch\n  got:  %q\n  want: %q", domainTypeStringV2, wantDomain)
	}
	if depositWalletDomainNameV2 != "DepositWallet" {
		t.Errorf("depositWalletDomainNameV2 = %q, want DepositWallet", depositWalletDomainNameV2)
	}
	if depositWalletDomainVersionV2 != "1" {
		t.Errorf("depositWalletDomainVersionV2 = %q, want 1", depositWalletDomainVersionV2)
	}
}

// TestPoly1271Digest_Derivation 验证 1271 内层待签 digest 的纯推导。
// 逐项独立重算 appDomainSep / tdsHash / digest，与 poly1271Digest 输出做字节级比对。
func TestPoly1271Digest_Derivation(t *testing.T) {
	p := newTestPolymarket()
	for _, negRisk := range []bool{false, true} {
		t.Run(map[bool]string{false: "exchange_v2", true: "neg_risk_exchange_v2"}[negRisk], func(t *testing.T) {
			order := poly1271TestOrder()
			digest, appDomainSep, contentsHash, err := p.poly1271Digest(order, negRisk)
			if err != nil {
				t.Fatalf("poly1271Digest: %v", err)
			}
			if len(digest) != 32 || len(appDomainSep) != 32 || len(contentsHash) != 32 {
				t.Fatalf("hash 长度异常: digest=%d appDomainSep=%d contentsHash=%d",
					len(digest), len(appDomainSep), len(contentsHash))
			}

			verifyingContract := ctfExchangeAddress
			if negRisk {
				verifyingContract = negRiskCTFExchangeAddress
			}
			chainIDBytes := common.LeftPadBytes(big.NewInt(int64(eip712ChainID)).Bytes(), 32)

			// appDomainSep 必须用正确的 V2 合约地址算出
			wantAppDomainSep := crypto.Keccak256(
				crypto.Keccak256([]byte(domainTypeStringV2)),
				crypto.Keccak256([]byte(eip712DomainName)),
				crypto.Keccak256([]byte(eip712DomainVersion)),
				chainIDBytes,
				common.LeftPadBytes(common.HexToAddress(verifyingContract).Bytes(), 32),
			)
			if !bytesEqual(appDomainSep, wantAppDomainSep) {
				t.Errorf("appDomainSep mismatch\n  got:  %x\n  want: %x", appDomainSep, wantAppDomainSep)
			}

			// 用返回的 contentsHash 重算 tdsHash + digest，验证 digest 组装正确（signer == funder）
			signerBytes := common.LeftPadBytes(common.HexToAddress(testFunder).Bytes(), 32)
			tdsHash := crypto.Keccak256(
				crypto.Keccak256([]byte(soladyTypeStringV2)),
				contentsHash,
				crypto.Keccak256([]byte(depositWalletDomainNameV2)),
				crypto.Keccak256([]byte(depositWalletDomainVersionV2)),
				chainIDBytes,
				signerBytes,
				make([]byte, 32),
			)
			wantDigest := crypto.Keccak256([]byte{0x19, 0x01}, appDomainSep, tdsHash)
			if !bytesEqual(digest, wantDigest) {
				t.Errorf("digest mismatch\n  got:  %x\n  want: %x", digest, wantDigest)
			}
		})
	}
}

// TestPoly1271_RejectsMissingProxy POLY_1271 必须在 ProxyAddress 里给 deposit wallet 地址。
func TestPoly1271_RejectsMissingProxy(t *testing.T) {
	p := newTestPolymarket()
	account := entity.PolymarketAccount{
		SignatureType: sigType1271,
		// ProxyAddress 留空
	}
	_, _, err := p.signOrderPoly1271(context.Background(), limitBuyOrderData(), account, false)
	if err == nil {
		t.Fatal("expected error for missing ProxyAddress, got nil")
	}
	if !strings.Contains(err.Error(), "proxy_address") {
		t.Errorf("err = %v, want it to mention proxy_address", err)
	}
}

// TestPoly1271ClobAuthTypeStrings L1 ClobAuth 包装用的类型字符串形状固定。
func TestPoly1271ClobAuthTypeStrings(t *testing.T) {
	wantClob := "ClobAuth(address address,string timestamp,uint256 nonce,string message)"
	wantDomain := "EIP712Domain(string name,string version,uint256 chainId)"
	wantSolady := "TypedDataSign(ClobAuth contents,string name,string version,uint256 chainId," +
		"address verifyingContract,bytes32 salt)" + wantClob
	if clobAuthTypeString != wantClob {
		t.Errorf("clobAuthTypeString mismatch\n  got:  %q\n  want: %q", clobAuthTypeString, wantClob)
	}
	if l1DomainTypeString != wantDomain {
		t.Errorf("l1DomainTypeString mismatch\n  got:  %q\n  want: %q", l1DomainTypeString, wantDomain)
	}
	if soladyTypeStringClobAuth != wantSolady {
		t.Errorf("soladyTypeStringClobAuth mismatch\n  got:  %q\n  want: %q", soladyTypeStringClobAuth, wantSolady)
	}
}

// TestPoly1271ClobAuthDigest_Derivation 验证 L1 ClobAuth ERC-7739 digest 组装。
func TestPoly1271ClobAuthDigest_Derivation(t *testing.T) {
	p := newTestPolymarket()
	ts := "1700000000"
	nonce := int64(0)
	digest, appDomainSep, contentsHash, err := p.poly1271ClobAuthDigest(testFunder, ts, nonce)
	if err != nil {
		t.Fatalf("poly1271ClobAuthDigest: %v", err)
	}
	if len(digest) != 32 || len(appDomainSep) != 32 || len(contentsHash) != 32 {
		t.Fatalf("hash 长度异常: digest=%d appDomainSep=%d contentsHash=%d",
			len(digest), len(appDomainSep), len(contentsHash))
	}

	chainIDBytes := common.LeftPadBytes(big.NewInt(int64(eip712ChainID)).Bytes(), 32)
	wantAppDomainSep := crypto.Keccak256(
		crypto.Keccak256([]byte(l1DomainTypeString)),
		crypto.Keccak256([]byte("ClobAuthDomain")),
		crypto.Keccak256([]byte(clobAuthDomainVersion)),
		chainIDBytes,
	)
	if !bytesEqual(appDomainSep, wantAppDomainSep) {
		t.Errorf("ClobAuth appDomainSep mismatch\n  got:  %x\n  want: %x", appDomainSep, wantAppDomainSep)
	}

	depositBytes := common.LeftPadBytes(common.HexToAddress(testFunder).Bytes(), 32)
	tdsHash := crypto.Keccak256(
		crypto.Keccak256([]byte(soladyTypeStringClobAuth)),
		contentsHash,
		crypto.Keccak256([]byte(depositWalletDomainNameV2)),
		crypto.Keccak256([]byte(depositWalletDomainVersionV2)),
		chainIDBytes,
		depositBytes,
		make([]byte, 32),
	)
	wantDigest := crypto.Keccak256([]byte{0x19, 0x01}, appDomainSep, tdsHash)
	if !bytesEqual(digest, wantDigest) {
		t.Errorf("ClobAuth digest mismatch\n  got:  %x\n  want: %x", digest, wantDigest)
	}
}

// TestBuildPoly1271ClobAuthSignature_PacksTrailer 签名长度 = 65 + 32 + 32 + typeLen + 2。
func TestBuildPoly1271ClobAuthSignature_PacksTrailer(t *testing.T) {
	// anvil #0 公开测试私钥
	const pk = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	p := newTestPolymarket()
	sig, err := p.buildPoly1271ClobAuthSignature(pk, testFunder, "1700000000", 0)
	if err != nil {
		t.Fatalf("buildPoly1271ClobAuthSignature: %v", err)
	}
	raw := common.FromHex(sig)
	wantLen := 65 + 32 + 32 + len(clobAuthTypeString) + 2
	if len(raw) != wantLen {
		t.Fatalf("wrapped sig len=%d, want %d", len(raw), wantLen)
	}
	typeStart := 65 + 32 + 32
	typeEnd := typeStart + len(clobAuthTypeString)
	if string(raw[typeStart:typeEnd]) != clobAuthTypeString {
		t.Errorf("contents type string mismatch in trailer")
	}
}

// bytesEqual 简单 byte-slice 相等比较，避免引入 reflect.DeepEqual 开销。
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
