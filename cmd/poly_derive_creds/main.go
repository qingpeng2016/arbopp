// 用 Polymarket 钱包私钥 derive CLOB L2 三件套，打印可写入 account 表的字段。
//
//	# EOA 绑定（老账号 / signature_type=0/1/2）
//	POLY_PRIVATE_KEY='0x...' go run ./cmd/poly_derive_creds
//	go run ./cmd/poly_derive_creds -private_key='0x...'
//
//	# Deposit Wallet 绑定（signature_type=3 / POLY_1271）
//	go run ./cmd/poly_derive_creds -private_key='0x...' -proxy='0xDeposit...' -signature_type=3
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	apphttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http/crypto/prediction/polymarket"
	"go.uber.org/zap/zapcore"
)

func main() {
	// CreateOrDerive 路径会打 InfoZ；未 Init 时 zap 为 nil 会 panic。
	logger.NewLogger("poly_derive_creds", "./log", zapcore.ErrorLevel)

	pkFlag := flag.String("private_key", "", "EOA 私钥（也可用环境变量 POLY_PRIVATE_KEY）")
	addrFlag := flag.String("address", "", "EOA 路径的 POLY_ADDRESS；默认由私钥推导")
	proxyFlag := flag.String("proxy", "", "Deposit Wallet 地址（signature_type=3 必填，即主页交易钱包）")
	sigTypeFlag := flag.Int("signature_type", 0, "0/1/2=EOA 绑定 derive；3=POLY_1271 绑到 -proxy")
	flag.Parse()

	pk := strings.TrimSpace(*pkFlag)
	if pk == "" {
		pk = strings.TrimSpace(os.Getenv("POLY_PRIVATE_KEY"))
	}
	if pk == "" {
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintln(os.Stderr, "  POLY_PRIVATE_KEY=0x... go run ./cmd/poly_derive_creds")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/poly_derive_creds -private_key=0x... [-address=0xEOA]")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/poly_derive_creds -private_key=0x... -proxy=0xDeposit -signature_type=3")
		os.Exit(2)
	}

	eoa, err := addrFromPrivateKey(pk)
	if err != nil {
		fmt.Fprintf(os.Stderr, "private key: %v\n", err)
		os.Exit(1)
	}

	poly := polymarket.NewPolymarket(apphttp.NewHttpClient())
	ctx := context.Background()

	var (
		apiKey, secret, passphrase string
		boundAddress               string
	)

	sigType := *sigTypeFlag
	proxy := strings.TrimSpace(*proxyFlag)
	if proxy != "" {
		fmt.Printf("# 提示: 主页 POLY_PROXY(0x44e999…) 已被 CLOB 拒 maker；有 Deposit Wallet 时必须 signature_type=3\n")
		fmt.Printf("#       本 EOA 的 beacon deposit 多为 CREATE2 派生地址（资金常在那边，不在主页 proxy）\n")
	}
	if sigType == 3 {
		if proxy == "" {
			fmt.Fprintln(os.Stderr, "signature_type=3 requires -proxy=<deposit wallet address>")
			os.Exit(2)
		}
		// 0xa219 这类 EIP-1167 → proxy_implementation 是 POLY_PROXY，强制 3 只会 401。
		if impl, ok := eip1167Impl(proxy); ok && strings.EqualFold(impl, polyProxyImplementation) {
			fmt.Fprintf(os.Stderr,
				"proxy %s 是 POLY_PROXY（impl=%s），不能用 signature_type=3。\n改用:\n  go run ./cmd/poly_derive_creds -private_key=... \n并把 account.signature_type 设为 1。\n",
				proxy, impl)
			os.Exit(2)
		}
		boundAddress = proxy
		apiKey, secret, passphrase, err = poly.CreateOrDeriveAPIKeyPoly1271(ctx, pk, proxy)
	} else {
		signer := strings.TrimSpace(*addrFlag)
		if signer == "" {
			signer = eoa
		}
		boundAddress = signer
		apiKey, secret, passphrase, err = poly.CreateOrDeriveAPIKey(ctx, pk, signer)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "derive failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("# account 表 POLYMARKET 字段（勿提交 git / 勿贴群）")
	fmt.Printf("eoa_address=%s\n", eoa)
	if proxy != "" {
		fmt.Printf("proxy_address=%s\n", proxy)
	} else if !strings.EqualFold(boundAddress, eoa) {
		fmt.Printf("signer_address=%s\n", boundAddress)
	}
	fmt.Printf("api_key=%s\n", apiKey)
	fmt.Printf("api_secret_enc=%s\n", secret)
	fmt.Printf("api_passphrase_enc=%s\n", passphrase)
	if sigType == 3 {
		fmt.Println("signature_type=3")
		fmt.Println("# L2 POLY_ADDRESS / 订单 maker+signer 均为 proxy_address（deposit）")
	} else {
		fmt.Printf("# signature_type 按钱包类型自填（当前 derive 绑定地址=%s）\n", boundAddress)
		fmt.Println("# 另需自行填: private_key_enc=你的私钥, proxy_address=proxy(有则填), status=ACTIVE")
	}
}

func addrFromPrivateKey(privateKeyHex string) (string, error) {
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	priv, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", err
	}
	return crypto.PubkeyToAddress(priv.PublicKey).Hex(), nil
}

// Polymarket production POLY_PROXY implementation（见 py-sdk environments.PRODUCTION）。
const polyProxyImplementation = "0x44e999d5c2F66Ef0861317f9A4805AC2e90aEB4f"

// eip1167Impl 若 address 是 EIP-1167 minimal proxy，返回实现合约地址。
func eip1167Impl(addr string) (string, bool) {
	code, err := ethGetCode(addr)
	if err != nil || len(code) < 45 {
		return "", false
	}
	// runtime: 363d3d373d3d3d363d73 + impl(20) + 5af43d82803e903d91602b57fd5bf3
	if !(code[0] == 0x36 && code[1] == 0x3d && code[9] == 0x73) {
		return "", false
	}
	impl := "0x" + hex.EncodeToString(code[10:30])
	return impl, true
}

func ethGetCode(addr string) ([]byte, error) {
	body := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"eth_getCode","params":["%s","latest"]}`,
		addr,
	)
	resp, err := http.Post("https://polygon-bor-rpc.publicnode.com", "application/json", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	raw := strings.TrimPrefix(strings.TrimSpace(out.Result), "0x")
	if raw == "" || raw == "0" {
		return nil, fmt.Errorf("no code")
	}
	return hex.DecodeString(raw)
}
