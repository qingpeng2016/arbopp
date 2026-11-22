package strategyentry

import (
	"context"
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestOutcomeIDOfSxbet(t *testing.T) {
	hash := "0xabc"
	w := newWatch(context.Background(), nil, group{
		MapKey: "test",
		Outcomes: []persistententity.MarketOutcome{
			{ID: 42, Platform: appcommon.PlatformSxbet, PlatformMarketOutcomeID: hash},
		},
	})
	id, ok := w.outcomeIDOf(appcommon.PlatformSxbet, hash)
	if !ok || id != 42 {
		t.Fatalf("sxbet outcomeIDOf: got id=%d ok=%v", id, ok)
	}
}

func TestOutcomeIDsForGrabPolyNoToken(t *testing.T) {
	yesTok, noTok := "y1", "n1"
	w := newWatch(context.Background(), nil, group{
		MapKey: "test",
		Outcomes: []persistententity.MarketOutcome{
			{ID: 7, Platform: appcommon.PlatformPolymarket, YesTokenID: yesTok, NoTokenID: noTok, PlatformMarketOutcomeID: "pm"},
		},
	})
	ids := w.outcomeIDsForGrab(event{
		Kind:     "poly_all_book",
		Platform: appcommon.PlatformPolymarket,
		ID:       noTok,
	})
	if len(ids) != 1 || ids[0] != 7 {
		t.Fatalf("poly NO all_book: ids=%v", ids)
	}
}