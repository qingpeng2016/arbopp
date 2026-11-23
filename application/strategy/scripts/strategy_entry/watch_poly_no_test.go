package strategyentry

import (
	"context"
	"testing"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestShouldSyncPolyComplementFromYes(t *testing.T) {
	yesTok, noTok := "y1", "n1"
	w := newWatch(context.Background(), nil, group{
		MapKey: "test",
		Outcomes: []persistententity.MarketOutcome{
			{ID: 1, Platform: appcommon.PlatformPolymarket, YesTokenID: yesTok, NoTokenID: noTok},
		},
	})
	if w.shouldSyncPolyComplementFromYes(yesTok) {
		t.Fatal("dual token WS: should not complement from YES")
	}
	if w.shouldSyncPolyComplementFromYes(noTok) {
		t.Fatal("NO token update is not YES complement source")
	}
	w2 := newWatch(context.Background(), nil, group{
		MapKey: "test",
		Outcomes: []persistententity.MarketOutcome{
			{ID: 2, Platform: appcommon.PlatformPolymarket, YesTokenID: "solo", NoTokenID: ""},
		},
	})
	if !w2.shouldSyncPolyComplementFromYes("solo") {
		t.Fatal("missing NO token: complement allowed")
	}
}
func TestPolyArchivePreservesEachTokenTime(t *testing.T) {
	w := newWatch(t.Context(), nil, group{
		MapKey: "test",
		Outcomes: []persistententity.MarketOutcome{
			{ID: 1, Platform: appcommon.PlatformPolymarket, YesTokenID: "y", NoTokenID: "n"},
		},
	})
	now := time.Now()
	w.Books.ApplyPolyBook("n", nil, []book.Level{{Price: .5, Size: 12}}, .01, 0, now.Add(-time.Second))
	w.Books.ApplyPolyBook("y", nil, []book.Level{{Price: .4, Size: 11}}, .01, 0, now)
	rows := w.orderBookRowsFromEvent(event{Kind: "poly_all_book", ID: "y", TS: now})
	if len(rows) != 2 || rows[0].SnapshotAtMs != now.UnixMilli() ||
		rows[1].SnapshotAtMs != now.Add(-time.Second).UnixMilli() {
		t.Fatalf("expected each row to carry its own quote timestamp: %+v", rows)
	}
}

func TestSxbetArchiveIncludesActualNoSide(t *testing.T) {
	w := newWatch(t.Context(), nil, group{
		MapKey: "test",
		Outcomes: []persistententity.MarketOutcome{
			{ID: 1, Platform: appcommon.PlatformSxbet, PlatformMarketOutcomeID: "hash", YesTokenID: "hash"},
		},
	})
	now := time.Now()
	w.Books.ApplySxbetBook("hash", []book.Level{{Price: .7, Size: 10}}, nil, now)
	w.Books.ApplySxbetNotAsksBook("hash", []book.Level{{Price: .35, Size: 20}}, now)
	rows := w.orderBookRowsFromEvent(event{Kind: "sxbet_all_book", ID: "hash", TS: now})
	if len(rows) != 2 || rows[1].TokenID != "hash:no" || rows[1].OrderBookAsks != `[["0.35","20"]]` ||
		rows[1].SnapshotAtMs != now.UnixMilli() {
		t.Fatalf("expected SX NO archive to use the actual selected side: %+v", rows)
	}
}
