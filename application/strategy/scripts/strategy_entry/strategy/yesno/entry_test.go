package yesno

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestBuildOutcomeGroupsSkipsNonMoneyline(t *testing.T) {
	comp := "意甲"
	mk := "v1|意甲|sea-sas-gen-2026-11-22"
	win := "Will US Sassuolo Calcio win on 2026-11-22?"
	total := "US Sassuolo Calcio vs. Genoa CFC: O/U 0.5"
	slugWin := "sea-sas-gen-2026-11-22-sas"
	slugTotal := "sea-sas-gen-2026-11-22-total-0pt5"
	wg := &watchgroup.Watch{
		MapKey: mk,
		Outcomes: []persistententity.MarketOutcome{
			{
				ID: 1, Platform: appcommon.PlatformPredict, OutcomeType: "moneyline",
				Competition: &comp, OutcomeName: &win, Slug: &slugWin, MapKey: &mk,
			},
			{
				ID: 2, Platform: appcommon.PlatformPolymarket, OutcomeType: "total",
				Competition: &comp, OutcomeName: &total, Slug: &slugTotal, MapKey: &mk,
			},
		},
	}
	r := Entry(wg)
	for _, g := range r.outcomeGroups {
		if _, has := g.PlatformIndex(appcommon.PlatformPolymarket); has {
			t.Fatal("total market must not register POLYMARKET in yesno OutcomeGroup")
		}
	}
}

func TestBuildOutcomeGroupsUsesMoneylineWhenBothSynced(t *testing.T) {
	comp := "意甲"
	mk := "v1|意甲|sea-sas-gen-2026-11-22"
	win := "Will US Sassuolo Calcio win on 2026-11-22?"
	total := "US Sassuolo vs. Genoa CFC: O/U 0.5"
	slugWin := "sea-sas-gen-2026-11-22-sas"
	slugTotal := "sea-sas-gen-2026-11-22-total-0pt5"
	wg := &watchgroup.Watch{
		MapKey: mk,
		Outcomes: []persistententity.MarketOutcome{
			{
				ID: 2, Platform: appcommon.PlatformPolymarket, OutcomeType: "total",
				Competition: &comp, OutcomeName: &total, Slug: &slugTotal, MapKey: &mk,
			},
			{
				ID: 3, Platform: appcommon.PlatformPolymarket, OutcomeType: "moneyline",
				Competition: &comp, OutcomeName: &win, Slug: &slugWin, MapKey: &mk,
			},
		},
	}
	r := Entry(wg)
	for _, g := range r.outcomeGroups {
		idx, has := g.PlatformIndex(appcommon.PlatformPolymarket)
		if !has {
			continue
		}
		if idx != 1 {
			t.Fatalf("POLYMARKET should index moneyline outcome (1), got %d", idx)
		}
	}
}