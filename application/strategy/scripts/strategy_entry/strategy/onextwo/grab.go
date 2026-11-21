package onextwo

import (
	"context"
)

// GrabHit 1X2 命中。
type GrabHit struct {
	Pick   TriplePick
	Limits Limits
	DryRun bool
}

func (r *Runner) grabOne(outcomeID uint) {
	_ = outcomeID
	all := r.allTriplePicks()
	pick, ok := bestTripleFrom(all)
	if !ok {
		return
	}
	for i := range pick.Legs {
		if !r.canTrade(pick.Legs[i].Platform) {
			return
		}
	}
	if !r.orderMu.TryLock() {
		return
	}
	defer r.orderMu.Unlock()
	lim := r.limits()
	bestHit := GrabHit{Pick: pick, Limits: lim, DryRun: true}
	ctx := r.w.Context()
	mapKey := r.w.MapKey
	deps := r.persistDeps()
	go func(ctx context.Context, mapKey string, picks []TriplePick, best GrabHit, d persistDeps) {
		for _, p := range picks {
			SaveGrabHit(ctx, d, mapKey, GrabHit{Pick: p, Limits: best.Limits, DryRun: best.DryRun})
		}
		NotifyGrab(ctx, mapKey, best)
	}(ctx, mapKey, all, bestHit, deps)
}

// OnBookUpdateAfterReplay 回测：本 snapshot 全部盘口写入后试一次 1X2（避免同一毫秒重复落库）。
func (r *Runner) OnBookUpdateAfterReplay() {
	if r.w == nil || !r.w.Ops.BacktestMode {
		return
	}
	r.grabOneBacktest()
}

// TryTriplePick 只读试算（历史扫描等）。
func (r *Runner) TryTriplePick() (TriplePick, bool) {
	return r.bestTriplePick()
}

func (r *Runner) grabOneBacktest() {
	all := r.allTriplePicks()
	_, ok := bestTripleFrom(all)
	if !ok {
		return
	}
	if !r.orderMu.TryLock() {
		return
	}
	defer r.orderMu.Unlock()
	lim := r.limits()
	ctx := r.w.Context()
	deps := r.persistDeps()
	mapKey := r.w.MapKey
	for _, p := range all {
		SaveGrabHit(ctx, deps, mapKey, GrabHit{Pick: p, Limits: lim})
	}
}

func (r *Runner) canTrade(platform string) bool {
	if r.w.Ops.CanTrade == nil {
		return false
	}
	return r.w.Ops.CanTrade(platform)
}