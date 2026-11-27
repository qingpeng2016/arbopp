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

func (r *Runner) grabOne(outcomeID uint, platform string, start time.Time) {
	r.pending.Store(&pendingDetect{outcomeID: outcomeID, platform: platform, start: start})
	r.kickDetect()
}

func (r *Runner) noteDetectLatency(platform string, start, computeStart time.Time) {
	if r == nil || r.w == nil || r.w.Ops.RecordDetectLatency == nil {
		return
	}
	var wsElapsed, computeElapsed time.Duration = -1, -1
	if !start.IsZero() {
		wsElapsed = time.Since(start)
	}
	if !computeStart.IsZero() {
		computeElapsed = time.Since(computeStart)
	}
	r.w.Ops.RecordDetectLatency("onextwo", r.w.MapKey, platform, wsElapsed, computeElapsed)
}

func (r *Runner) kickDetect() {
	if !r.orderMu.TryLock() {
		return
	}
	defer func() {
		r.orderMu.Unlock()
		if r.pending.Load() != nil {
			go r.kickDetect()
		}
	}()
	for {
		p := r.pending.Swap(nil)
		if p == nil {
			return
		}
		r.grabOneLocked(p.outcomeID, p.platform, p.start)
	}
}

func (r *Runner) grabOneLocked(_ uint, platform string, start time.Time) {
	computeStart := time.Now()
	lim := r.limits()
	all := r.allTriplePicks(lim)
	if len(all) == 0 {
		r.noteDetectLatency(platform, start, computeStart)
		return
	}

	ctx := r.w.Context()
	mapKey := r.w.MapKey
	deps := r.persistDeps()
	openBlocked := yesno.HasBlockingOpenOrders(&r.w.Ops, mapKey)

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
		if !predictTripleBookOK(p) {
			continue
		}
		if !tradeOK || p.Per < tradePick.Per {
			tradePick, tradeOK = p, true
		}
	}

	r.noteDetectLatency(platform, start, computeStart)

	var fills [3]float64
	var fillErrs [3]error
	var sends [3]watchgroup.SendResult
	ordered := false
	orderCapHit := false
	if tradeOK && !openBlocked {
		if yesno.TryConsumeOrderSlot(&r.w.Ops) {
			fills, fillErrs, sends = fireTriple(r, tradePick)
			ordered = true
		} else {
			orderCapHit = true
		}
	}
	placedN, maxN := yesno.OrderCountSnapshot(&r.w.Ops)

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
		predOK, predSnap := predictTripleBookDetail(p)
		if !p.SizeCheckOK {
			hit.SkipOrderReason = "size_check_failed"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{"skip_order_reason": "size_check_failed"})
		} else if !canAll {
			hit.SkipOrderReason = "account_unavailable"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "account_unavailable",
				"platforms":         []string{p.Legs[0].Platform, p.Legs[1].Platform, p.Legs[2].Platform},
			})
		} else if !predOK {
			hit.SkipOrderReason = "predict_fee_shares_gt_book"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, predSnap, map[string]any{
				"skip_order_reason": "predict_fee_shares_gt_book",
			})
		} else if openBlocked {
			hit.SkipOrderReason = "non_terminal_orders"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "non_terminal_orders",
				"map_key":           mapKey,
			})
		} else if orderCapHit {
			hit.SkipOrderReason = "max_order_count_reached"
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "max_order_count_reached",
				"order_count":       placedN,
				"max_order_count":   maxN,
			})
		} else if isTrade {
			hit.DryRun = false
			hit.Fills, hit.FillErrs, hit.Sends = fills, fillErrs, sends
			hit.ExtData = yesno.ExtDataJSON(p.SizeCheck.Snapshot, predSnap, map[string]any{
				"ordered": true, "order_qty": p.Qty, "fills": fills,
				"order_count": placedN, "max_order_count": maxN,
			})
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

func predictTripleBookOK(p TriplePick) bool {
	ok, _ := predictTripleBookDetail(p)
	return ok
}

func predictTripleBookDetail(p TriplePick) (ok bool, snap map[string]any) {
	legs := make([]map[string]any, 0, 3)
	ok = true
	for i := range p.Legs {
		fit, need, have := p.Legs[i].PredictFeeSharesFitBook(p.BuyYes, p.Qty)
		legs = append(legs, map[string]any{
			"platform": p.Legs[i].Platform, "need": need, "book": have, "ok": fit,
		})
		if !fit {
			ok = false
		}
	}
	snap = map[string]any{"predict_fee_shares": legs}
	return ok, snap
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
	// 单腿成交不全时 flatten（暂时关闭）
	// for i := 0; i < 3; i++ {
	// 	othersOK := true
	// 	for j := 0; j < 3; j++ {
	// 		if i == j {
	// 			continue
	// 		}
	// 		if fills[j] <= 0 {
	// 			othersOK = false
	// 			break
	// 		}
	// 	}
	// 	if fills[i] > 0 && !othersOK && r.w.Ops.FlattenOnOutcome != nil {
	// 		r.w.Ops.FlattenOnOutcome(pick.Legs[i].Index, pick.BuyYes, fills[i], 0)
	// 	}
	// }
	return fills, errs, sends
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
