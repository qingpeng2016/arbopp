package orderbook

import (
	"testing"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestSplitOutcomesByPlatformIncludesPredict(t *testing.T) {
	kalshi, poly, pred := splitOutcomesByPlatform([]persistententity.MarketOutcome{
		{Platform: "PREDICT", PlatformMarketOutcomeID: "4211675"},
		{Platform: "PREDICT", PlatformMarketOutcomeID: "4211677"},
		{Platform: "POLYMARKET", PlatformMarketOutcomeID: "x", YesTokenID: "tok"},
		{Platform: "KALSHI", PlatformMarketOutcomeID: "T"},
	})
	if len(kalshi) != 1 || len(poly) != 1 || len(pred) != 2 {
		t.Fatalf("k=%d p=%d d=%d", len(kalshi), len(poly), len(pred))
	}
}
