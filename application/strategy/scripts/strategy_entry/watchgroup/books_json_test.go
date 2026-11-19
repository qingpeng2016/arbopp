package watchgroup

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestBooksJSONForOutcomePair(t *testing.T) {
	yesTok, noTok := "yt", "nt"
	mid := "pred-mid"
	store := book.NewStore()
	ts := time.Now().UTC()
	store.ApplyPolyBook(yesTok,
		[]book.Level{{Price: 0.36, Size: 800}}, []book.Level{{Price: 0.38, Size: 720}}, 0.01, 0, ts)
	store.ApplyPolyBook(noTok,
		[]book.Level{{Price: 0.62, Size: 400}}, []book.Level{{Price: 0.64, Size: 300}}, 0.01, 0, ts)
	store.ApplyPredictBook(mid,
		[]book.Level{{Price: 0.55, Size: 1400}}, []book.Level{{Price: 0.56, Size: 1200}}, ts)

	wg := &Watch{
		Outcomes: []persistententity.MarketOutcome{
			{ID: 1, Platform: appcommon.PlatformPolymarket, YesTokenID: yesTok, NoTokenID: noTok, PlatformMarketOutcomeID: "pm1"},
			{ID: 2, Platform: appcommon.PlatformPredict, PlatformMarketOutcomeID: mid},
		},
		Books: store,
	}
	raw := wg.BooksJSONForOutcomePair(0, 1)
	if !strings.Contains(raw, `"YES_TOKEN"`) || !strings.Contains(raw, `"YES_BOOK"`) {
		t.Fatalf("missing roles: %s", raw)
	}
	var parsed struct {
		Outcomes []struct {
			Books []struct {
				Asks []book.Level `json:"asks"`
			} `json:"books"`
		} `json:"outcomes"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Outcomes) != 2 {
		t.Fatalf("outcomes %d", len(parsed.Outcomes))
	}
	if len(parsed.Outcomes[0].Books) < 2 {
		t.Fatal("poly books")
	}
	if parsed.Outcomes[0].Books[0].Asks[0].Price != 0.38 {
		t.Fatalf("poly yes ask %v", parsed.Outcomes[0].Books[0].Asks[0].Price)
	}
}