package yesno

import (
	"math"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// Pick 跨平台两两套利候选。
type Pick struct {
	YesSide, NoSide     Side
	YesBuyYes, NoBuyYes bool
	Qty, Per            float64
}

// FilledPick 带平台、盘口证据和净兑付口径的候选。
type FilledPick struct {
	Yes, No             ExecSide
	YesBuyYes, NoBuyYes bool
	Costs, Sizes        [2]float64 // Sizes：PayoutQuote 深度（净兑付份数）
	TopSizes            [2]float64 // 各腿卖一原始挂单量（arb_qty 取 min）
	Qty, Per            float64
	At                  time.Time
	view                book.Snapshot
}

func (r *Runner) bestFilledPick(group OutcomeGroup) (FilledPick, bool) {
	_, best, ok := r.AllFilledPicks(group, r.limits())
	return best, ok
}

func (r *Runner) allOutcomeGroupPicks(group OutcomeGroup, lim Limits) []Pick {
	filled, _, _ := r.AllFilledPicks(group, lim)
	picks := make([]Pick, 0, len(filled))
	for _, p := range filled {
		picks = append(picks, Pick{YesSide: p.Yes.Side, NoSide: p.No.Side,
			YesBuyYes: p.YesBuyYes, NoBuyYes: p.NoBuyYes, Qty: p.Qty, Per: p.Per})
	}
	return picks
}

func (r *Runner) bestOutcomeGroupPick(group OutcomeGroup, lim Limits) (Pick, bool) {
	_, p, ok := r.AllFilledPicks(group, lim)
	return Pick{YesSide: p.Yes.Side, NoSide: p.No.Side, YesBuyYes: p.YesBuyYes, NoBuyYes: p.NoBuyYes,
		Qty: p.Qty, Per: p.Per}, ok
}

// AllFilledPicks evaluates all directions against one immutable book snapshot.
func (r *Runner) AllFilledPicks(group OutcomeGroup, lim Limits) (all []FilledPick, best FilledPick, ok bool) {
	if group.PlatformCount() < 2 || r.w.Store() == nil {
		return nil, FilledPick{}, false
	}
	view, at := r.w.StrategySnapshot()
	sides := r.groupSides(group, view)
	for i, yes := range sides {
		yesCost, yesSize, yesOK := yes.PayoutQuote(true, at)
		if !yesOK {
			continue
		}
		for j, no := range sides {
			if i == j || !SameSettlementScope(r.w, yes, no) {
				continue
			}
			noCost, noSize, noOK := no.PayoutQuote(false, at)
			per := yesCost + noCost
			// TEST: 仅阈值 config_strategy_threshold；不强制 per < 1（兑付硬顶已关）
			if !noOK || per >= lim.MaxCost {
				continue
			}
			_, yesTopSz, yesTopOK := yes.TopAskSz(true)
			_, noTopSz, noTopOK := no.TopAskSz(false)
			if !yesTopOK || !noTopOK {
				continue
			}
			qty, hit := SizeForBudget(per, math.Min(yesSize, noSize), lim.OrderAmount, lim.LevelFactor)
			if !hit {
				continue
			}
			pick := FilledPick{Yes: yes, No: no, YesBuyYes: true, Costs: [2]float64{yesCost, noCost},
				Sizes: [2]float64{yesSize, noSize}, TopSizes: [2]float64{yesTopSz, noTopSz},
				Qty: qty, Per: per, At: at, view: view}
			all = append(all, pick)
			if !ok || per < best.Per {
				best, ok = pick, true
			}
		}
	}
	return all, best, ok
}

func (r *Runner) groupSides(group OutcomeGroup, view book.Snapshot) []ExecSide {
	sides := make([]ExecSide, 0, group.PlatformCount())
	for _, platform := range group.PlatformList() {
		index, _ := group.PlatformIndex(platform)
		if side, ok := QuoteExecSide(r.w, view, index); ok {
			sides = append(sides, side)
		}
	}
	return sides
}
