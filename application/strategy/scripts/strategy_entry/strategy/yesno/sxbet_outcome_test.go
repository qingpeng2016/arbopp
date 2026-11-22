package yesno

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestSxbetBinaryArbRankPrefers1X2OverMoneyline(t *testing.T) {
	mlName := "Juventus vs Venezia [Moneyline]"
	mlCfg := `{"yes_label":"Juventus","no_label":"Venezia"}`
	x2Name := "Juventus vs Not Juventus [1X2]"
	x2Cfg := `{"yes_label":"Juventus","no_label":"Not Juventus"}`
	ml := &persistententity.MarketOutcome{
		Platform: appcommon.PlatformSxbet, OutcomeName: &mlName, Config: &mlCfg,
	}
	x2 := &persistententity.MarketOutcome{
		Platform: appcommon.PlatformSxbet, OutcomeName: &x2Name, Config: &x2Cfg,
	}
	if got := sxbetBinaryArbRank(ml); got >= sxbetBinaryArbRank(x2) {
		t.Fatalf("moneyline rank %d should be < 1x2 rank %d", got, sxbetBinaryArbRank(x2))
	}
}

func TestOutcomeGroupSxbetPicks1X2WhenBothRegistered(t *testing.T) {
	mlName := "Juventus vs Venezia [Moneyline]"
	x2Name := "Juventus vs Not Juventus [1X2]"
	ml := &persistententity.MarketOutcome{Platform: appcommon.PlatformSxbet, OutcomeName: &mlName}
	x2 := &persistententity.MarketOutcome{Platform: appcommon.PlatformSxbet, OutcomeName: &x2Name}
	outcomes := []*persistententity.MarketOutcome{ml, x2}
	prev := func(i int) *persistententity.MarketOutcome { return outcomes[i] }
	var g OutcomeGroup
	g.AddMarketOutcome(ml, 0, prev)
	g.AddMarketOutcome(x2, 1, prev)
	idx, ok := g.PlatformIndex(appcommon.PlatformSxbet)
	if !ok || idx != 1 {
		t.Fatalf("want sx index 1 (1X2), got %d ok=%v", idx, ok)
	}
}