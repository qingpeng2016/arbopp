package synthetic

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

func TestFakeBookPricesOnTick(t *testing.T) {
	bids, asks := fakeBookAtSecond(500, string(cryptoPredictionEnum.PlatformPolymarket), 1)
	if strings.Contains(bids, "00000000000") || strings.Contains(asks, "00000000000") {
		t.Fatalf("float noise in json bids=%s asks=%s", bids, asks)
	}
	var bl, al [][2]float64
	if err := json.Unmarshal([]byte(bids), &bl); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(asks), &al); err != nil {
		t.Fatal(err)
	}
	for _, lv := range append(bl, al...) {
		ticked := roundTick(lv[0], defaultTick)
		if math.Abs(ticked-lv[0]) > 1e-9 {
			t.Fatalf("price %v not on 0.01 tick (rounded %v)", lv[0], ticked)
		}
	}
}