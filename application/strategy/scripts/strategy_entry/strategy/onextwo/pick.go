package onextwo

import (
	"math"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
)

func (r *Runner) bestTriplePick() (TriplePick, bool) {
	draw, tA, tB, ok := r.moneylineTripleGroups()
	if !ok {
		return TriplePick{}, false
	}
	lim := r.limits()
	groups := [3]yesno.OutcomeGroup{draw, tA, tB}

	var best TriplePick
	found := false

	if pick, hit := r.tripleAt(groups, true, lim.YesTripleMax, lim); hit {
		if !found || pick.Per < best.Per {
			best, found = pick, true
		}
	}
	if pick, hit := r.tripleAt(groups, false, lim.NoTripleMax, lim); hit {
		if !found || pick.Per < best.Per {
			best, found = pick, true
		}
	}
	return best, found
}

func (r *Runner) tripleAt(groups [3]yesno.OutcomeGroup, buyYes bool, maxCost float64, lim Limits) (TriplePick, bool) {
	if maxCost <= 0 {
		return TriplePick{}, false
	}
	var legs [3]yesno.ExecSide
	var prices [3]float64
	var sumPx, minSz float64 = 0, math.MaxFloat64
	for i := range groups {
		side, perShare, sz, ok := r.bestLeg(groups[i], buyYes)
		if !ok {
			return TriplePick{}, false
		}
		legs[i] = side
		prices[i] = perShare
		sumPx += perShare
		if sz < minSz {
			minSz = sz
		}
	}
	if minSz <= 0 || sumPx <= 0 {
		return TriplePick{}, false
	}
	orderAmount := lim.OrderAmount
	if orderAmount <= 0 {
		orderAmount = 100
	}
	levelFactor := lim.LevelFactor
	if levelFactor <= 0 {
		levelFactor = 2
	}
	qty := orderAmount / sumPx
	if minSz*levelFactor+1e-9 < qty {
		return TriplePick{}, false
	}
	per := sumPx
	if per >= maxCost {
		return TriplePick{}, false
	}
	kind := OppKindYesTriple
	if !buyYes {
		kind = OppKindNoTriple
	}
	return TriplePick{
		Kind: kind, BuyYes: buyYes, Legs: legs, Prices: prices, Qty: qty, Per: per,
	}, true
}

func (r *Runner) bestLeg(g yesno.OutcomeGroup, buyYes bool) (yesno.ExecSide, float64, float64, bool) {
	var best yesno.ExecSide
	bestPx := math.MaxFloat64
	bestSz := 0.0
	found := false
	for _, plat := range g.PlatformList() {
		idx, ok := g.PlatformIndex(plat)
		if !ok {
			continue
		}
		side, ok := r.execSide(idx)
		if !ok {
			continue
		}
		px := yesno.SidePx(side.Side, buyYes)
		sz := side.Side.YesSz
		if !buyYes {
			sz = side.Side.NoSz
		}
		if !book.ValidPrice(px) || sz <= 0 {
			continue
		}
		per := px + yesno.Fee(px, side.Side.Fee)
		if per < bestPx {
			bestPx, bestSz, best, found = per, sz, side, true
		}
	}
	if !found {
		return yesno.ExecSide{}, 0, 0, false
	}
	return best, bestPx, bestSz, true
}