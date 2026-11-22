package strategyentry

import (
	"context"
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
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