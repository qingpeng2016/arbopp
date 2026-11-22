package watchgroup

import (
	"encoding/json"
	"testing"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestLegSnapshotTopAskMatchesJSON(t *testing.T) {
	store := book.NewStore()
	yesTok := "yes-tok"
	store.ApplyPolyBook(yesTok,
		nil, []book.Level{{Price: 0.21, Size: 10}, {Price: 0.22, Size: 5}},
		0.01, 1, time.Now().UTC())
	wg := &Watch{
		Books: store,
		Outcomes: []persistententity.MarketOutcome{
			{Platform: appcommon.PlatformPolymarket, PlatformMarketOutcomeID: "pm-1", YesTokenID: yesTok, NoTokenID: "no-tok"},
		},
	}
	snap := wg.LegSnapshot(0, true)
	if !snap.OK || snap.TopAsk != 0.21 {
		t.Fatalf("top ask=%v ok=%v", snap.TopAsk, snap.OK)
	}
	var body struct {
		Books []struct {
			Asks []book.Level `json:"asks"`
		} `json:"books"`
	}
	if err := json.Unmarshal([]byte(snap.OrderBooks), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Books) == 0 || len(body.Books[0].Asks) == 0 || body.Books[0].Asks[0].Price != snap.TopAsk {
		t.Fatalf("JSON first ask does not match leg_best_ask source: snap=%v json=%v", snap.TopAsk, body.Books[0].Asks[0].Price)
	}
}