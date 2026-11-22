package sxbet

import (
	"math"
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestLevelsToYesBook_TakerFromMakerFrame(t *testing.T) {
	one := []httpentity.SxbetBookLevel{{
		PercentageOdds: "35200000000000000000", // maker 0.352 → back BB @ decimal 2.84
		Size:           "638000000",
	}}
	two := []httpentity.SxbetBookLevel{{
		PercentageOdds: "33300000000000000000", // maker 0.333 → Not @ 1/(1-0.333)≈1.50
		Size:           "1201000000",
	}}
	bids, asks := LevelsToYesBook(one, two)
	if len(asks) != 1 || len(bids) != 1 {
		t.Fatalf("asks=%d bids=%d", len(asks), len(bids))
	}
	wantAsk := 0.352
	if math.Abs(asks[0].Price-wantAsk) > 1e-4 {
		t.Fatalf("ask price got %v want %v (decimal ~%.2f)", asks[0].Price, wantAsk, 1/wantAsk)
	}
	wantBid := 0.333
	if math.Abs(bids[0].Price-wantBid) > 1e-4 {
		t.Fatalf("bid price got %v want %v", bids[0].Price, wantBid)
	}
	notDecimal := 1 / (1 - wantBid)
	if math.Abs(notDecimal-1.49925) > 0.02 {
		t.Fatalf("not decimal ~%.3f", notDecimal)
	}
}