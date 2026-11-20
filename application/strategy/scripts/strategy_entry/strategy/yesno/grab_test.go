package yesno

import (
	"testing"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestGrabOneSkipsWhenCannotTrade(t *testing.T) {
	yesTok, noTok := "y", "n"
	kTicker := "k-tick"
	store := book.NewStore()
	store.ApplyPolyBook(yesTok, []book.Level{{Price: 0.4, Size: 800}}, []book.Level{{Price: 0.42, Size: 800}}, 0.01, 1, time.Now().UTC())
	store.ApplyPolyBook(noTok, []book.Level{{Price: 0.58, Size: 800}}, []book.Level{{Price: 0.6, Size: 800}}, 0.01, 1, time.Now().UTC())
	store.ApplyKalshiSnapshot(kTicker, []book.Level{{Price: 0.55, Size: 800}}, []book.Level{{Price: 0.35, Size: 800}}, 1, time.Now().UTC())

	comp := "英超"
	polyName := "Will Arsenal win"
	kalshiName := "Arsenal wins"
	slug := "epl-ars-sun-2026-11-21-ars"
	wg := &watchgroup.Watch{
		MapKey: "test",
		Outcomes: []persistententity.MarketOutcome{
			{
				ID: 1, Platform: appcommon.PlatformPolymarket, Competition: &comp,
				OutcomeName: &polyName, Slug: &slug,
				PlatformMarketOutcomeID: "pm", YesTokenID: yesTok, NoTokenID: noTok,
			},
			{
				ID: 2, Platform: appcommon.PlatformKalshi, Competition: &comp,
				OutcomeName: &kalshiName, PlatformMarketOutcomeID: kTicker,
			},
		},
		Books: store,
	}
	group := NewOutcomeGroup()
	group.Add(appcommon.PlatformPolymarket, 0)
	group.Add(appcommon.PlatformKalshi, 1)

	groupKey := marketmap.OutcomeGroupKey(&wg.Outcomes[0])
	if groupKey == "" {
		t.Fatal("outcome group key")
	}
	r := &Runner{
		w:             wg,
		outcomeGroups: map[string]OutcomeGroup{groupKey: group},
	}
	r.w.Ops.CanTrade = func(string) bool { return false }
	r.grabOne(1)
}