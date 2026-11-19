package predict

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
)

func TestOrderHash_DocExample(t *testing.T) {
	msg := map[string]any{
		"salt":          "123456789",
		"maker":         "0x1111111111111111111111111111111111111111",
		"signer":        "0x1111111111111111111111111111111111111111",
		"taker":         zeroAddress,
		"tokenId":       "1000000000000000000",
		"makerAmount":   "4000000000000000000",
		"takerAmount":   "10000000000000000000",
		"expiration":    math.NewHexOrDecimal256(4102444800),
		"nonce":         "0",
		"feeRateBps":    "200",
		"side":          math.NewHexOrDecimal256(0),
		"signatureType": math.NewHexOrDecimal256(0),
	}
	h, err := orderHash(msg, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "0xe6a0edf257e9904e5fd4a1b8fdc3dd092e8789c4fa596ff6275097515ee4fd34"
	if h.Hex() != want {
		t.Fatalf("order hash %s want %s", h.Hex(), want)
	}
}