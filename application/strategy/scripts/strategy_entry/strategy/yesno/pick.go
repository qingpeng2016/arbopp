package yesno

import (
	"sync"
)

// Pick 跨平台两两套利候选。
type Pick struct {
	YesSide, NoSide           Side
	YesBuyYes, NoBuyYes       bool
	Qty, Per                  float64
}

// FilledPick 带平台信息的成交候选。
type FilledPick struct {
	Yes, No                   ExecSide
	YesBuyYes, NoBuyYes       bool
	Qty, Per                  float64
}

func considerPick(best *Pick, found *bool, pick Pick, hit bool) {
	if !hit {
		return
	}
	if !*found || pick.Per < best.Per {
		*best = pick
		*found = true
	}
}

// bestFilledPick 同一 OutcomeGroup 上最优跨所 YES+NO 组合（Runner.grabOne 调用）。
func (r *Runner) bestFilledPick(group OutcomeGroup) (FilledPick, bool) {
	lim := r.limits()
	p, ok := r.bestOutcomeGroupPick(group, lim)
	if !ok {
		return FilledPick{}, false
	}
	return r.filledPickFrom(p)
}

// pairPicksBetween 两平台间两种方向的可成交组合。
func (r *Runner) pairPicksBetween(sa, sb Side, lim Limits) []Pick {
	var out []Pick
	qtyA, perA, hitA := TopHit(sa.YesPx, sa.YesSz, sb.NoPx, sb.NoSz, sa.Fee, sb.Fee, lim.OrderAmount, lim.LevelFactor, lim.MaxCost)
	if hitA {
		out = append(out, Pick{YesSide: sa, NoSide: sb, YesBuyYes: true, NoBuyYes: false, Qty: qtyA, Per: perA})
	}
	qtyB, perB, hitB := TopHit(sb.YesPx, sb.YesSz, sa.NoPx, sa.NoSz, sb.Fee, sa.Fee, lim.OrderAmount, lim.LevelFactor, lim.MaxCost)
	if hitB {
		out = append(out, Pick{YesSide: sb, NoSide: sa, YesBuyYes: true, NoBuyYes: false, Qty: qtyB, Per: perB})
	}
	return out
}

// allOutcomeGroupPicks 同一 OutcomeGroup 上全部跨所两两套利命中（并行试算）。
func (r *Runner) allOutcomeGroupPicks(group OutcomeGroup, lim Limits) []Pick {
	if group.PlatformCount() < 2 {
		return nil
	}
	plats := group.PlatformList()
	type pairIdx struct{ i, j int }
	var jobs []pairIdx
	for i := 0; i < len(plats); i++ {
		for j := i + 1; j < len(plats); j++ {
			jobs = append(jobs, pairIdx{i, j})
		}
	}
	var (
		mu  sync.Mutex
		out []Pick
		wg  sync.WaitGroup
	)
	for _, job := range jobs {
		wg.Add(1)
		go func(job pairIdx) {
			defer wg.Done()
			ia, okA := group.PlatformIndex(plats[job.i])
			ib, okB := group.PlatformIndex(plats[job.j])
			if !okA || !okB {
				return
			}
			sa, oka := r.sideTop(ia)
			sb, okb := r.sideTop(ib)
			if !oka || !okb {
				return
			}
			local := r.pairPicksBetween(sa, sb, lim)
			if len(local) == 0 {
				return
			}
			mu.Lock()
			out = append(out, local...)
			mu.Unlock()
		}(job)
	}
	wg.Wait()
	return out
}

func bestPickFrom(picks []Pick) (Pick, bool) {
	var best Pick
	found := false
	for _, p := range picks {
		considerPick(&best, &found, p, true)
	}
	return best, found
}

func (r *Runner) bestOutcomeGroupPick(group OutcomeGroup, lim Limits) (Pick, bool) {
	return bestPickFrom(r.allOutcomeGroupPicks(group, lim))
}

// AllFilledPicks 全部可成交组合与最优（供落库全量机会）。
func (r *Runner) AllFilledPicks(group OutcomeGroup, lim Limits) (all []FilledPick, best FilledPick, ok bool) {
	picks := r.allOutcomeGroupPicks(group, lim)
	p, okPick := bestPickFrom(picks)
	if !okPick {
		return nil, FilledPick{}, false
	}
	best, ok = r.filledPickFrom(p)
	if !ok {
		return nil, FilledPick{}, false
	}
	for _, pick := range picks {
		fp, okFp := r.filledPickFrom(pick)
		if okFp {
			all = append(all, fp)
		}
	}
	return all, best, true
}

func (r *Runner) filledPickFrom(p Pick) (FilledPick, bool) {
	yes, okA := r.execSide(p.YesSide.Index)
	no, okB := r.execSide(p.NoSide.Index)
	if !okA || !okB {
		return FilledPick{}, false
	}
	// leg_*_best_ask / total_cost 必须与 TopHit 同源：用判定时的卖一，禁止二次读盘漂移。
	copyPickSidePrices(&yes.Side, p.YesSide, p.YesBuyYes)
	copyPickSidePrices(&no.Side, p.NoSide, p.NoBuyYes)
	return FilledPick{
		Yes: yes, No: no,
		YesBuyYes: p.YesBuyYes, NoBuyYes: p.NoBuyYes,
		Qty: p.Qty, Per: p.Per,
	}, true
}

// copyPickSidePrices 把 TopHit 用的卖一写回 ExecSide，落库与 JSON 对账一致。
func copyPickSidePrices(dst *Side, src Side, buyYes bool) {
	if buyYes {
		dst.YesPx, dst.YesSz = src.YesPx, src.YesSz
		return
	}
	dst.NoPx, dst.NoSz = src.NoPx, src.NoSz
}