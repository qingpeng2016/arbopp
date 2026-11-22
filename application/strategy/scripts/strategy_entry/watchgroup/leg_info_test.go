package watchgroup

import (
	"strings"
	"testing"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestYesLegInfoJSONKeyOrder(t *testing.T) {
	market := "Lakers vs Celtics"
	outcome := "Lakers ML"
	w := &Watch{
		Outcomes: []persistententity.MarketOutcome{{
			ID: 42, PlatformMarketID: "fx-1", MarketName: &market, OutcomeName: &outcome,
			MarketURL: "https://poly.example/event/lal-bcn", YesTokenID: "yt", NoTokenID: "nt",
		}},
	}
	raw := w.YesLegInfoJSON(0, true)
	wantPrefix := "{\n  \"yes_market_id\": \"fx-1\",\n  \"yes_market_name\": \"Lakers vs Celtics\",\n  \"market_url\": \"https://poly.example/event/lal-bcn\",\n  \"yes_outcome_id\": 42"
	if !strings.HasPrefix(raw, wantPrefix) {
		t.Fatalf("unexpected leg2 JSON prefix:\n%s", raw)
	}
	if !strings.Contains(raw, "\"yes_token_id\": \"yt\"") {
		t.Fatalf("missing token_id:\n%s", raw)
	}
}