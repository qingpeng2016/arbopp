package onextwo

import (
	"context"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

// GrabHit 1X2 命中。
type GrabHit struct {
	Pick   TriplePick
	Legs   [3]watchgroup.LegPersistSnap // 落库盘口与 leg_*_best_ask 同源快照
	Limits Limits
	DryRun bool
}

func (r *Runner) grabOne(outcomeID uint) {
	r.grabOneAttempt(outcomeID, false)
}

func (r *Runner) grabOneAttempt(_ uint, retried bool) {
	all := r.allTriplePicks()
	pick, ok := bestTripleFrom(all)
	if !ok {
		return
	}
	const dryRun = true
	if !dryRun {
		for i := range pick.Legs {
			if !r.canTrade(pick.Legs[i].Platform) {
				return
			}
		}
	}
	if !r.orderMu.TryLock() {
		if !retried {
			go func() {
				time.Sleep(25 * time.Millisecond)
				r.grabOneAttempt(0, true)
			}()
		}
		return
	}
	defer r.orderMu.Unlock()
	lim := r.limits()
	bestHit := GrabHit{Pick: pick, Limits: lim, DryRun: dryRun}
	ctx := r.w.Context()
	mapKey := r.w.MapKey
	deps := r.persistDeps()
	for _, p := range all {
		SaveGrabHit(ctx, deps, mapKey, GrabHit{Pick: p, Legs: r.snapTripleLegs(p), Limits: lim, DryRun: dryRun})
	}
	go func(ctx context.Context, mapKey string, best GrabHit) {
		NotifyGrab(ctx, mapKey, best)
	}(ctx, mapKey, bestHit)
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
		SaveGrabHit(ctx, deps, mapKey, GrabHit{Pick: p, Legs: r.snapTripleLegs(p), Limits: lim})
	}
}

func (r *Runner) canTrade(platform string) bool {
	if r.w.Ops.CanTrade == nil {
		return false
	}
	return r.w.Ops.CanTrade(platform)
}