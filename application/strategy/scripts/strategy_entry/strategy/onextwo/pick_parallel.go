package onextwo

import (
	"math"
	"sync"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
)

// legAt 指定平台单腿报价。
func (r *Runner) legAt(g yesno.OutcomeGroup, platform string, buyYes bool) (yesno.ExecSide, float64, float64, bool) {
	idx, ok := g.PlatformIndex(platform)
	if !ok {
		return yesno.ExecSide{}, 0, 0, false
	}
	side, ok := r.execSide(idx)
	if !ok {
		return yesno.ExecSide{}, 0, 0, false
	}
	px := yesno.SidePx(side.Side, buyYes)
	sz := side.Side.YesSz
	if !buyYes {
		sz = side.Side.NoSz
	}
	if !book.ValidPrice(px) || sz <= 0 {
		return yesno.ExecSide{}, 0, 0, false
	}
	per := px + yesno.Fee(px, side.Side.Fee)
	return side, per, sz, true
}

func (r *Runner) bestLeg(g yesno.OutcomeGroup, buyYes bool) (yesno.ExecSide, float64, float64, bool) {
	plats := g.PlatformList()
	if len(plats) == 0 {
		return yesno.ExecSide{}, 0, 0, false
	}
	if len(plats) == 1 {
		return r.legAt(g, plats[0], buyYes)
	}
	type legHit struct {
		side   yesno.ExecSide
		per    float64
		sz     float64
		ok     bool
	}
	hits := make([]legHit, len(plats))
	var wg sync.WaitGroup
	for i, plat := range plats {
		wg.Add(1)
		go func(i int, plat string) {
			defer wg.Done()
			side, per, sz, ok := r.legAt(g, plat, buyYes)
			hits[i] = legHit{side: side, per: per, sz: sz, ok: ok}
		}(i, plat)
	}
	wg.Wait()
	var best yesno.ExecSide
	bestPx := math.MaxFloat64
	bestSz := 0.0
	found := false
	for _, h := range hits {
		if !h.ok || h.per >= bestPx {
			continue
		}
		bestPx, bestSz, best, found = h.per, h.sz, h.side, true
	}
	if !found {
		return yesno.ExecSide{}, 0, 0, false
	}
	return best, bestPx, bestSz, true
}

func (r *Runner) tripleForPlatforms(groups [3]yesno.OutcomeGroup, plats [3]string, buyYes bool, maxCost float64, lim Limits) (TriplePick, bool) {
	if maxCost <= 0 {
		return TriplePick{}, false
	}
	var legs [3]yesno.ExecSide
	var prices [3]float64
	var sumPx, minSz float64 = 0, math.MaxFloat64
	for i := range groups {
		side, perShare, sz, ok := r.legAt(groups[i], plats[i], buyYes)
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

// allTriplePicks 三路各平台组合全部命中（并行试算）。
func (r *Runner) allTriplePicks() []TriplePick {
	draw, tA, tB, ok := r.moneylineTripleGroups()
	if !ok {
		return nil
	}
	lim := r.limits()
	groups := [3]yesno.OutcomeGroup{draw, tA, tB}
	lists := [3][]string{draw.PlatformList(), tA.PlatformList(), tB.PlatformList()}
	type tripleJob struct {
		buyYes  bool
		maxCost float64
		plats   [3]string
	}
	var jobs []tripleJob
	for _, buyYes := range []bool{true, false} {
		maxCost := lim.YesTripleMax
		if !buyYes {
			maxCost = lim.NoTripleMax
		}
		for _, pd := range lists[0] {
			for _, pa := range lists[1] {
				for _, pb := range lists[2] {
					jobs = append(jobs, tripleJob{buyYes: buyYes, maxCost: maxCost, plats: [3]string{pd, pa, pb}})
				}
			}
		}
	}
	var (
		mu  sync.Mutex
		out []TriplePick
		wg  sync.WaitGroup
	)
	for _, j := range jobs {
		wg.Add(1)
		go func(j tripleJob) {
			defer wg.Done()
			if pick, hit := r.tripleForPlatforms(groups, j.plats, j.buyYes, j.maxCost, lim); hit {
				mu.Lock()
				out = append(out, pick)
				mu.Unlock()
			}
		}(j)
	}
	wg.Wait()
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