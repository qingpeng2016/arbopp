package onextwo

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

type pendingDetect struct {
	outcomeID uint
	platform  string
	start     time.Time
}

// Runner 足球 1X2 三路全覆盖（YES 包 / NO 包）。
type Runner struct {
	w             *watchgroup.Watch
	outcomeGroups map[string]yesno.OutcomeGroup
	recent        watchgroup.RecentHits
	orderMu       sync.Mutex
	pending       atomic.Pointer[pendingDetect]
}

// Entry 绑定 watch，预建 OutcomeGroup。
func Entry(w *watchgroup.Watch) *Runner {
	return &Runner{
		w:             w,
		recent:        make(watchgroup.RecentHits),
		outcomeGroups: buildOutcomeGroups(w),
	}
}

func buildOutcomeGroups(w *watchgroup.Watch) map[string]yesno.OutcomeGroup {
	groups := map[string]yesno.OutcomeGroup{}
	n := w.OutcomeCount()
	for i := 0; i < n; i++ {
		oc := w.OutcomeAt(i)
		if oc == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(oc.OutcomeType), "moneyline") {
			continue
		}
		key := w.OutcomeGroupKeyAt(i)
		if key == "" {
			continue
		}
		group := groups[key]
		group.AddMarketOutcome(oc, i, w.OutcomeAt)
		groups[key] = group
	}
	return groups
}

func (r *Runner) OnBookUpdate(outcomeID uint, platform string, start time.Time) {
	idx, ok := r.w.OutcomeIndex(outcomeID)
	if !ok {
		return
	}
	oc := r.w.OutcomeAt(idx)
	if oc == nil || !strings.EqualFold(strings.TrimSpace(oc.OutcomeType), "moneyline") {
		return
	}
	if r.w.Ops.BacktestMode {
		return
	}
	r.grabOne(outcomeID, platform, start)
}

func (r *Runner) limits() Limits {
	var lim Limits
	if r.w.Ops.OneX2Limits != nil {
		lim.YesTripleMax, lim.NoTripleMax = r.w.Ops.OneX2Limits()
	}
	if r.w.Ops.StrategyParams != nil {
		_, lim.MinAmount, lim.MaxAmount = r.w.Ops.StrategyParams()
	}
	if r.w.Ops.FillShareMismatchPct != nil {
		lim.FillShareMismatchPct = r.w.Ops.FillShareMismatchPct()
	}
	return lim
}

func (r *Runner) snapTripleLegs(pick TriplePick) [3]watchgroup.LegPersistSnap {
	var snaps [3]watchgroup.LegPersistSnap
	for i, leg := range pick.Legs {
		snaps[i] = leg.PersistQuote(pick.BuyYes, pick.Costs[i], pick.LegSizes[i])
	}
	return snaps
}
