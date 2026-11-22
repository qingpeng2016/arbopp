package onextwo

import (
	"strings"
	"sync"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

// Runner 足球 1X2 三路全覆盖（YES 包 / NO 包）。
type Runner struct {
	w             *watchgroup.Watch
	outcomeGroups map[string]yesno.OutcomeGroup
	orderMu       sync.Mutex
}

// Entry 绑定 watch，预建 OutcomeGroup。
func Entry(w *watchgroup.Watch) *Runner {
	return &Runner{
		w:             w,
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

func (r *Runner) OnBookUpdate(outcomeID uint) {
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
	r.grabOne(outcomeID)
}

func (r *Runner) limits() Limits {
	lim := Limits{YesTripleMax: 0.95, NoTripleMax: 1.90}
	if r.w.Ops.OneX2Limits != nil {
		lim.YesTripleMax, lim.NoTripleMax = r.w.Ops.OneX2Limits()
	}
	if r.w.Ops.StrategyParams != nil {
		_, lim.OrderAmount, lim.LevelFactor = r.w.Ops.StrategyParams()
	}
	if lim.YesTripleMax <= 0 {
		lim.YesTripleMax = 0.95
	}
	if lim.NoTripleMax <= 0 {
		lim.NoTripleMax = 1.90
	}
	return lim
}

func (r *Runner) snapTripleLegs(pick TriplePick) [3]watchgroup.LegPersistSnap {
	var snaps [3]watchgroup.LegPersistSnap
	if r.w == nil {
		return snaps
	}
	buy := pick.BuyYes
	for i := range pick.Legs {
		snaps[i] = r.w.LegSnapshot(pick.Legs[i].Index, buy)
	}
	return snaps
}

func (r *Runner) execSide(outcomeIndex int) (yesno.ExecSide, bool) {
	oc := r.w.OutcomeAt(outcomeIndex)
	store := r.w.Store()
	if oc == nil || store == nil {
		return yesno.ExecSide{}, false
	}
	fee := 0.0
	if r.w.Ops.TradeFee != nil {
		fee = r.w.Ops.TradeFee(oc.Platform, deref(oc.Category), deref(oc.SubCategory))
	}
	bo := yesno.OutcomeFields(oc.Platform, oc.PlatformMarketOutcomeID, oc.YesTokenID, oc.NoTokenID)
	side, ok := yesno.ExecSideAt(store, outcomeIndex, bo, fee)
	if !ok {
		return yesno.ExecSide{}, false
	}
	side.OutcomeID = oc.ID
	return side, true
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
