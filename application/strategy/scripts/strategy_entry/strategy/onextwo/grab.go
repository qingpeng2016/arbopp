package onextwo

import (
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

// GrabHit 1X2 命中。
type GrabHit struct {
	Pick            TriplePick
	Legs            [3]watchgroup.LegPersistSnap
	Limits          Limits
	DryRun          bool
	ExtData         string
	SkipOrderReason string
	Fills           [3]float64
	FillErrs        [3]error
	Sends           [3]watchgroup.SendResult
}

func (r *Runner) grabOne(outcomeID uint) {
	r.grabOneAttempt(outcomeID, false)
}

func (r *Runner) grabOneAttempt(_ uint, retried bool) {
	if !r.orderMu.TryLock() {
		if !retried {
			go func() {
				time.Sleep(25 * time.Millisecond)
				r.grabOneAttempt(0, true)
			}()
			return
		}
		r.persistMissedLock()
		return
	}
	defer r.orderMu.Unlock()
	lim := r.limits()
	all := r.allTriplePicks(lim)
	if len(all) == 0 {
		return
	}

	ctx := r.w.Context()
	mapKey := r.w.MapKey
	deps := r.persistDeps()
	openBlocked := yesno.HasBlockingOpenOrders(ctx, deps.order, mapKey)

	var tradePick TriplePick
	var tradeOK bool
	for _, p := range all {
		if !p.SizeCheckOK {
			continue
		}
		okTrade := true
		for i := range p.Legs {
			if !r.canTrade(p.Legs[i].Platform) {
				okTrade = false
				break
			}
		}
		if !okTrade {
			continue
		}
		if !tradeOK || p.Per < tradePick.Per {
			tradePick, tradeOK = p, true
		}
	}

	var fills [3]float64
	var fillErrs [3]error
	var sends [3]watchgroup.SendResult
	ordered := false
	if tradeOK && !openBlocked {
		fills, fillErrs, sends = fireTriple(r, tradePick)
		ordered = true
	}

	r.recent.Prune(all[0].At)
	var bestHit GrabHit
	var saved bool
	for _, p := range all {
		key := watchgroup.HitKey{
			Kind: p.Kind, Indices: [3]int{p.Legs[0].Index, p.Legs[1].Index, p.Legs[2].Index},
			Prices: p.Prices, Depths: p.LegSizes, Cost: p.Per, Quantity: p.Qty,
		}
		isTrade := ordered && sameTriple(p, tradePick)
		hit := GrabHit{Pick: p, Legs: r.snapTripleLegs(p), Limits: lim, DryRun: true}
		canAll := true
		for i := range p.Legs {
			if !r.canTrade(p.Legs[i].Platform) {
				canAll = false
				break
			}
		}
		if !p.SizeCheckOK {
			hit.SkipOrderReason = "size_check_failed"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{"skip_order_reason": "size_check_failed"})
		} else if !canAll {
			hit.SkipOrderReason = "account_unavailable"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "account_unavailable",
				"platforms":         []string{p.Legs[0].Platform, p.Legs[1].Platform, p.Legs[2].Platform},
			})
		} else if openBlocked {
			hit.SkipOrderReason = "non_terminal_orders"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "non_terminal_orders",
				"map_key":           mapKey,
			})
		} else if isTrade {
			hit.DryRun = false
			hit.Fills, hit.FillErrs, hit.Sends = fills, fillErrs, sends
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{"ordered": true, "order_qty": p.Qty, "fills": fills})
		} else {
			hit.SkipOrderReason = "not_selected_for_order"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{"skip_order_reason": hit.SkipOrderReason})
		}
		insert := func() bool {
			if !SaveGrabHit(ctx, deps, mapKey, hit) {
				return false
			}
			if !saved || p.Per < bestHit.Pick.Per {
				bestHit, saved = hit, true
			}
			return true
		}
		if hit.DryRun {
			r.recent.Save(key, p.At, insert)
		} else {
			insert()
		}
	}
	if saved {
		go NotifyGrab(ctx, mapKey, bestHit)
	}
	if ordered {
		go NotifyOrderResult(ctx, mapKey, GrabHit{
			Pick: tradePick, Limits: lim, DryRun: false, Fills: fills, FillErrs: fillErrs, Sends: sends,
			Legs: r.snapTripleLegs(tradePick),
		})
	}
}

func sameTriple(a, b TriplePick) bool {
	if a.Kind != b.Kind || a.BuyYes != b.BuyYes {
		return false
	}
	for i := 0; i < 3; i++ {
		if a.Legs[i].Index != b.Legs[i].Index {
			return false
		}
	}
	return true
}

func fireTriple(r *Runner, pick TriplePick) (fills [3]float64, errs [3]error, sends [3]watchgroup.SendResult) {
	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		i := i
		go func() {
			defer wg.Done()
			if r.w.Ops.SendOnOutcome == nil {
				return
			}
			res, err := r.w.Ops.SendOnOutcome(pick.Legs[i].Index, pick.BuyYes, pick.Prices[i], pick.Qty)
			sends[i], fills[i], errs[i] = res, res.Fill, err
		}()
	}
	wg.Wait()
	// 单腿成交不全时 flatten
	for i := 0; i < 3; i++ {
		othersOK := true
		for j := 0; j < 3; j++ {
			if i == j {
				continue
			}
			if fills[j] <= 0 {
				othersOK = false
				break
			}
		}
		if fills[i] > 0 && !othersOK && r.w.Ops.FlattenOnOutcome != nil {
			r.w.Ops.FlattenOnOutcome(pick.Legs[i].Index, pick.BuyYes, fills[i], 0)
		}
	}
	return fills, errs, sends
}

func (r *Runner) persistMissedLock() {
	lim := r.limits()
	all := r.allTriplePicks(lim)
	if len(all) == 0 {
		return
	}
	ctx := r.w.Context()
	mapKey := r.w.MapKey
	deps := r.persistDeps()
	for _, p := range all {
		hit := GrabHit{
			Pick: p, Legs: r.snapTripleLegs(p), Limits: lim, DryRun: true,
			SkipOrderReason: "map_key_lock_missed",
			ExtData: yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "map_key_lock_missed", "map_key": mapKey,
			}),
		}
		SaveGrabHit(ctx, deps, mapKey, hit)
	}
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
	if !r.orderMu.TryLock() {
		return
	}
	defer r.orderMu.Unlock()
	lim := r.limits()
	all := r.allTriplePicks(lim)
	_, ok := bestTripleFrom(all)
	if !ok {
		return
	}
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
