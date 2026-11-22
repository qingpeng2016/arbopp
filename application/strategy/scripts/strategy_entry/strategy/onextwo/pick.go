package onextwo

import (
	"math"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
)

func (r *Runner) bestTriplePick() (TriplePick, bool) {
	return bestTripleFrom(r.allTriplePicks())
}

func (r *Runner) tripleAt(groups [3]yesno.OutcomeGroup, buyYes bool, maxCost float64, lim Limits) (TriplePick, bool) {
	if maxCost <= 0 {
		return TriplePick{}, false
	}
	var legs [3]yesno.ExecSide
	var prices, sizes [3]float64
	var sumPx, minSz float64 = 0, math.MaxFloat64
	for i := range groups {
		side, perShare, sz, ok := r.bestLeg(groups[i], buyYes)
		if !ok {
			return TriplePick{}, false
		}
		legs[i] = side
		prices[i] = perShare
		sizes[i] = sz
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
		Kind: kind, BuyYes: buyYes, Legs: legs, Prices: prices, LegSizes: sizes, Qty: qty, Per: per,
	}, true
}