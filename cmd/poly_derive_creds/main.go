// 用 Polymarket 钱包私钥 derive CLOB L2 三件套，打印可写入 account 表的字段。
//
//	POLY_PRIVATE_KEY='0x...' go run ./cmd/poly_derive_creds
//	go run ./cmd/poly_derive_creds -private_key='0x...'
//	go run ./cmd/poly_derive_creds -private_key='0x...' -address='0xProxyOrEOA'
package main

import (
	"context"
	"flag"
	"fmt"
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

	pkFlag := flag.String("private_key", "", "钱包私钥（也可用环境变量 POLY_PRIVATE_KEY）")
	addrFlag := flag.String("address", "", "signer 地址；默认由私钥推导 EOA")
	flag.Parse()

	pk := strings.TrimSpace(*pkFlag)
	if pk == "" {
		pk = strings.TrimSpace(os.Getenv("POLY_PRIVATE_KEY"))
	}
	if pk == "" {
		fmt.Fprintln(os.Stderr, "usage: POLY_PRIVATE_KEY=0x... go run ./cmd/poly_derive_creds")
		fmt.Fprintln(os.Stderr, "   or: go run ./cmd/poly_derive_creds -private_key=0x... [-address=0x...]")
		os.Exit(2)
	}

	eoa, err := addrFromPrivateKey(pk)
	if err != nil {
		fmt.Fprintf(os.Stderr, "private key: %v\n", err)
		os.Exit(1)
	}
	signer := strings.TrimSpace(*addrFlag)
	if signer == "" {
		signer = eoa
	}

	poly := polymarket.NewPolymarket(apphttp.NewHttpClient())
	apiKey, secret, passphrase, err := poly.CreateOrDeriveAPIKey(context.Background(), pk, signer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "derive failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("# account 表 POLYMARKET 字段（勿提交 git / 勿贴群）")
	fmt.Printf("eoa_address=%s\n", eoa)
	if !strings.EqualFold(signer, eoa) {
		fmt.Printf("signer_address=%s\n", signer)
	}
	fmt.Printf("api_key=%s\n", apiKey)
	fmt.Printf("api_secret_enc=%s\n", secret)
	fmt.Printf("api_passphrase_enc=%s\n", passphrase)
	fmt.Println("# 另需自行填: private_key_enc=你的私钥, proxy_address=proxy(有则填), signature_type, status=ACTIVE")
}

func addrFromPrivateKey(privateKeyHex string) (string, error) {
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	priv, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", err
	}
	return crypto.PubkeyToAddress(priv.PublicKey).Hex(), nil
}
