package kalshi

import (
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestKalshiEventScopeMatchesOnly(t *testing.T) {
	ev := httpentity.KalshiEvent{
		Category:        "Sports",
		ProductMetadata: &httpentity.KalshiProductMetadata{CompetitionScope: "Game"},
	}
	if !kalshiEventIsMatchMarket(ev) {
		t.Fatal("game scope should sync")
	}
	ev.ProductMetadata.CompetitionScope = "Regulation Time Moneyline"
	if !kalshiEventIsMatchMarket(ev) {
		t.Fatal("regulation time moneyline is still a match")
	}
	for _, scope := range []string{"Future", "Awards", "Host", "Spread", "Group Winner", "BTTS"} {
		ev.ProductMetadata.CompetitionScope = scope
		if kalshiEventIsMatchMarket(ev) {
			t.Fatalf("%s scope should stay out", scope)
		}
	}
}
