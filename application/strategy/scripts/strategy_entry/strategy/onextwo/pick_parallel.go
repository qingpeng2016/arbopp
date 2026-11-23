package onextwo

import (
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
)

type pricedLeg struct {
	side            yesno.ExecSide
	cost, size      float64 // size：PayoutQuote 净兑付深度
	topPx, topSize  float64 // 卖一裸价与原始挂单量
}

func (r *Runner) priceGroup(view book.Snapshot, at time.Time, group yesno.OutcomeGroup, buyYes bool) []pricedLeg {
	out := make([]pricedLeg, 0, group.PlatformCount())
	for _, platform := range group.PlatformList() {
		idx, _ := group.PlatformIndex(platform)
		if side, ok := yesno.QuoteExecSide(r.w, view, idx); ok {
			if cost, size, ok := side.PayoutQuote(buyYes, at); ok {
				topPx, topSz, topOK := side.TopAskSz(buyYes)
				if !topOK {
					continue
				}
				out = append(out, pricedLeg{side: side, cost: cost, size: size, topPx: topPx, topSize: topSz})
			}
		}
	}
	return out
}

func (r *Runner) priceTriple(legs [3]pricedLeg, at time.Time, buyYes bool, lim Limits) (TriplePick, bool) {
	a, b, c := legs[0].side, legs[1].side, legs[2].side
	if a.Platform == b.Platform && b.Platform == c.Platform {
		return TriplePick{}, false
	}
	if !yesno.SameSettlementScope(r.w, a, b) || !yesno.SameSettlementScope(r.w, a, c) ||
		!yesno.SameSettlementScope(r.w, b, c) {
		return TriplePick{}, false
	}
	kind, maxCost := OppKindNoTriple, min(lim.NoTripleMax, 2)
	if buyYes {
		kind, maxCost = OppKindYesTriple, min(lim.YesTripleMax, 1)
	}
	cost := legs[0].cost + legs[1].cost + legs[2].cost
	if cost >= maxCost {
		return TriplePick{}, false
	}
	qty, ok := yesno.SizeForBudget(cost, min(legs[0].size, legs[1].size, legs[2].size), lim.OrderAmount, lim.LevelFactor)
	if !ok {
		return TriplePick{}, false
	}
	pick := TriplePick{Kind: kind, BuyYes: buyYes, At: at, Qty: qty, Per: cost}
	for i, leg := range legs {
		pick.Legs[i] = leg.side
		pick.Prices[i] = leg.topPx
		pick.Costs[i] = leg.cost
		pick.LegSizes[i] = leg.topSize
	}
	return pick, true
}

func (r *Runner) allTriplePicks(lim Limits) []TriplePick {
	draw, teamA, teamB, ok := r.moneylineTripleGroups()
	if !ok || r.w.Store() == nil {
		return nil
	}
	view, at := r.w.StrategySnapshot()
	var out []TriplePick
	for _, buyYes := range []bool{true, false} {
		draws := r.priceGroup(view, at, draw, buyYes)
		firstTeams := r.priceGroup(view, at, teamA, buyYes)
		secondTeams := r.priceGroup(view, at, teamB, buyYes)
		for _, a := range draws {
			for _, b := range firstTeams {
				for _, c := range secondTeams {
					if pick, hit := r.priceTriple([3]pricedLeg{a, b, c}, at, buyYes, lim); hit {
						out = append(out, pick)
					}
				}
			}
		}
	}
	return out
}

func bestTripleFrom(picks []TriplePick) (TriplePick, bool) {
	var best TriplePick
	found := false
	for _, p := range picks {
		if !found || p.Per < best.Per {
			best, found = p, true
		}
	}
	return best, found
}
