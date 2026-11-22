package sxbet

import (
	"math"
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestLevelsToYesBook_TakerFromMakerFrame(t *testing.T) {
	// outcomeOne 挂单 maker 30.1% → 页面 Not 十进制 ≈ 1.43
	one := []httpentity.SxbetBookLevel{{
		PercentageOdds: "30100000000000000000",
		Size:           "774000000",
	}}
	// outcomeTwo 挂单 maker 68.5% → 页面 Back BB 十进制 ≈ 3.17（taker 31.5%）
	two := []httpentity.SxbetBookLevel{{
		PercentageOdds: "68500000000000000000",
		Size:           "774000000",
	}}
	bids, asks := LevelsToYesBook(one, two)
	if len(asks) != 1 || len(bids) != 1 {
		t.Fatalf("asks=%d bids=%d", len(asks), len(bids))
	}
	wantAsk := 0.315
	if math.Abs(asks[0].Price-wantAsk) > 1e-4 {
		t.Fatalf("ask price got %v want %v (decimal ~%.2f)", asks[0].Price, wantAsk, 1/wantAsk)
	}
	wantBid := 0.301
	if math.Abs(bids[0].Price-wantBid) > 1e-4 {
		t.Fatalf("bid price got %v want %v", bids[0].Price, wantBid)
	}
	notDecimal := 1 / (1 - wantBid)
	if math.Abs(notDecimal-1.4308) > 0.02 {
		t.Fatalf("not decimal ~%.3f", notDecimal)
	}
}