package polymarket

import (
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestPolymarketEventLeagueScope(t *testing.T) {
	ev := &httpentity.PolymarketEvent{Slug: "ucl-gal-ast-2026-11-24"}
	if !polymarketEventInLeagueScope(ev, "ucl") {
		t.Fatal("ucl event slug")
	}
	m := &httpentity.Market{Slug: "ucl-gal-ast-2026-11-24-gal", Question: "Will Galatasaray SK win on 2026-11-24?", GameStartTime: "2026-11-24T18:45:00Z"}
	if !polymarketMarketInLeagueScope(m, "ucl") {
		t.Fatal("ucl market slug")
	}
}