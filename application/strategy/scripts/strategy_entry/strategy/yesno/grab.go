package yesno

import (
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

// Sender 双边发单与单腿平仓。
type Sender interface {
	Send(side Side, buyYes bool, px, qty float64) (watchgroup.SendResult, error)
	Flatten(side Side, boughtYes bool, mine, other float64)
}

// Fire 并发双腿 LIMIT+FAK（flatten 暂时关闭）。
func Fire(snd Sender, a, b Side, aBuyYes, bBuyYes bool, qty float64) (ra, rb watchgroup.SendResult, errA, errB error) {
	pxA, pxB := SidePx(a, aBuyYes), SidePx(b, bBuyYes)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ra, errA = snd.Send(a, aBuyYes, pxA, qty)
	}()
	go func() {
		defer wg.Done()
		rb, errB = snd.Send(b, bBuyYes, pxB, qty)
	}()
	wg.Wait()
	// snd.Flatten(a, aBuyYes, ra.Fill, rb.Fill)
	// snd.Flatten(b, bBuyYes, rb.Fill, ra.Fill)
	return ra, rb, errA, errB
}

// GrabHit 一次抢单结果。
type GrabHit struct {
	Pick                 FilledPick
	Leg1, Leg2           watchgroup.LegPersistSnap
	MaxCost              float64
	FillShareMismatchPct float64
	YesFill, NoFill      float64
	YesErr, NoErr        error
	YesSend, NoSend      watchgroup.SendResult
	DryRun               bool   // true：只记机会表，不发单、不写 orders
	ExtData              string // arb_binary_opportunity.ext_data
	SkipOrderReason      string
}

// TryPickForOutcome 盘口更新后试算最优组合（回测扫描等只读路径，不落库）。
func (r *Runner) TryPickForOutcome(outcomeID uint) (pick FilledPick, ok bool) {
	return r.tryPickForGrab(outcomeID)
}

func (r *Runner) tryPickForGrab(outcomeID uint) (pick FilledPick, ok bool) {
	_, pick, ok = r.tryAllPicksForGrab(outcomeID, r.limits())
	return pick, ok
}

func (r *Runner) tryAllPicksForGrab(outcomeID uint, lim Limits) (all []FilledPick, best FilledPick, ok bool) {
	outcomeIndex, okIdx := r.outcomeIndex(outcomeID)
	if !okIdx {
		return nil, FilledPick{}, false
	}
	groupKey := r.outcomeGroupKeyAt(outcomeIndex)
	if groupKey == "" {
		return nil, FilledPick{}, false
	}
	group, okGrp := r.outcomeGroups[groupKey]
	if !okGrp {
		return nil, FilledPick{}, false
	}
	return r.AllFilledPicks(group, lim)
}

func (r *Runner) grabOne(outcomeID uint) {
	r.grabOneAttempt(outcomeID, false)
}

func (r *Runner) grabOneAttempt(outcomeID uint, retried bool) {
	if !r.orderMu.TryLock() {
		// Lock held (usually Fire in flight). Do not write map_key_lock_missed:
		// every book tick would otherwise flood dry-run rows; the holder already persists.
		if !retried {
			go func(id uint) {
				time.Sleep(25 * time.Millisecond)
				r.grabOneAttempt(id, true)
			}(outcomeID)
		}
		return
	}
	defer r.orderMu.Unlock()

	lim := r.limits()
	all, _, ok := r.tryAllPicksForGrab(outcomeID, lim)
	if !ok {
		return
	}

	ctx := r.grabContext()
	mapKey := r.w.MapKey
	persist := r.grabPersist()
	openBlocked := r.blockingOpenOrders(mapKey)

	// 选一份可下单的最优（成本最低、份额检查通过、账号可用、Predict 费后份额盖得住内存簿）
	var tradePick FilledPick
	var tradeOK bool
	for _, fp := range all {
		if !fp.SizeCheckOK {
			continue
		}
		if !r.canTrade(fp.Yes.Platform) || !r.canTrade(fp.No.Platform) {
			continue
		}
		if yesOK, _, _ := fp.Yes.PredictFeeSharesFitBook(fp.YesBuyYes, fp.Qty); !yesOK {
			continue
		}
		if noOK, _, _ := fp.No.PredictFeeSharesFitBook(fp.NoBuyYes, fp.Qty); !noOK {
			continue
		}
		if !tradeOK || fp.Per < tradePick.Per {
			tradePick, tradeOK = fp, true
		}
	}

	var yesRes, noRes watchgroup.SendResult
	var yesErr, noErr error
	ordered := false
	if tradeOK && !openBlocked {
		yesRes, noRes, yesErr, noErr = Fire(outcomeSender{r}, tradePick.Yes.Side, tradePick.No.Side, tradePick.YesBuyYes, tradePick.NoBuyYes, tradePick.Qty)
		ordered = true
	}
	yesFill, noFill := yesRes.Fill, noRes.Fill

	r.recent.Prune(all[0].At)
	var bestHit GrabHit
	var saved bool
	for _, fp := range all {
		key := watchgroup.HitKey{
			Kind: strategyTypeYesNo, Indices: [3]int{fp.Yes.Index, fp.No.Index},
			Prices: [3]float64{fp.Yes.YesPx, fp.No.NoPx}, Depths: [3]float64{fp.Sizes[0], fp.Sizes[1]},
			Cost: fp.Per, Quantity: fp.Qty,
		}
		isTradeLeg := ordered && fp.Yes.Index == tradePick.Yes.Index && fp.No.Index == tradePick.No.Index &&
			fp.YesBuyYes == tradePick.YesBuyYes && fp.NoBuyYes == tradePick.NoBuyYes

		hit := r.buildGrabHit(fp, lim)
		yesFit, yesNeed, yesHave := fp.Yes.PredictFeeSharesFitBook(fp.YesBuyYes, fp.Qty)
		noFit, noNeed, noHave := fp.No.PredictFeeSharesFitBook(fp.NoBuyYes, fp.Qty)
		predBookSnap := map[string]any{
			"predict_fee_shares": map[string]any{
				"yes": map[string]any{"platform": fp.Yes.Platform, "need": yesNeed, "book": yesHave, "ok": yesFit},
				"no":  map[string]any{"platform": fp.No.Platform, "need": noNeed, "book": noHave, "ok": noFit},
			},
		}
		if isTradeLeg {
			hit.DryRun = false
			hit.YesFill, hit.NoFill, hit.YesErr, hit.NoErr = yesFill, noFill, yesErr, noErr
			hit.YesSend, hit.NoSend = yesRes, noRes
			hit.SkipOrderReason = ""
			hit.ExtData = ExtDataJSON(fp.SizeCheck.Snapshot, predBookSnap, map[string]any{
				"ordered": true, "order_qty": fp.Qty, "fill_yes": yesFill, "fill_no": noFill,
			})
		} else if !fp.SizeCheckOK {
			// already set in buildGrabHit
		} else if !r.canTrade(fp.Yes.Platform) || !r.canTrade(fp.No.Platform) {
			hit.DryRun = true
			hit.SkipOrderReason = "account_unavailable"
			hit.ExtData = ExtDataJSON(fp.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "account_unavailable",
				"platforms":         []string{fp.Yes.Platform, fp.No.Platform},
			})
		} else if !yesFit || !noFit {
			hit.DryRun = true
			hit.SkipOrderReason = "predict_fee_shares_gt_book"
			hit.ExtData = ExtDataJSON(fp.SizeCheck.Snapshot, predBookSnap, map[string]any{
				"skip_order_reason": "predict_fee_shares_gt_book",
			})
		} else if openBlocked {
			hit.DryRun = true
			hit.SkipOrderReason = "non_terminal_orders"
			hit.ExtData = ExtDataJSON(fp.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "non_terminal_orders",
				"map_key":           mapKey,
			})
		} else {
			// 有更优腿已下单，本组合仅记机会
			hit.DryRun = true
			hit.SkipOrderReason = "not_selected_for_order"
			hit.ExtData = ExtDataJSON(fp.SizeCheck.Snapshot, map[string]any{
				"skip_order_reason": "not_selected_for_order",
			})
		}

		insert := func() bool {
			if !SaveGrabHit(ctx, persist, mapKey, hit) {
				return false
			}
			if !saved || fp.Per < bestHit.Pick.Per {
				bestHit, saved = hit, true
			}
			return true
		}
		if hit.DryRun {
			r.recent.Save(key, fp.At, insert)
		} else {
			insert()
		}
	}
	if saved {
		go NotifyGrab(ctx, mapKey, bestHit)
	}
	if ordered {
		go NotifyOrderResult(ctx, mapKey, GrabHit{
			Pick: tradePick, MaxCost: lim.MaxCost, FillShareMismatchPct: lim.FillShareMismatchPct,
			YesFill: yesFill, NoFill: noFill, YesErr: yesErr, NoErr: noErr,
			YesSend: yesRes, NoSend: noRes, DryRun: false,
			Leg1: r.mustSnap(tradePick).l1, Leg2: r.mustSnap(tradePick).l2,
		})
	}
}

type snapPair struct{ l1, l2 watchgroup.LegPersistSnap }

func (r *Runner) mustSnap(fp FilledPick) snapPair {
	a, b := r.snapPickLegs(fp)
	return snapPair{a, b}
}

func (r *Runner) buildGrabHit(fp FilledPick, lim Limits) GrabHit {
	leg1, leg2 := r.snapPickLegs(fp)
	hit := GrabHit{Pick: fp, Leg1: leg1, Leg2: leg2, MaxCost: lim.MaxCost, DryRun: true}
	if !fp.SizeCheckOK {
		hit.SkipOrderReason = "size_check_failed"
		hit.ExtData = ExtDataJSON(fp.SizeCheck.Snapshot, map[string]any{
			"skip_order_reason": "size_check_failed",
		})
	} else if fp.SizeCheck.Snapshot != nil {
		hit.ExtData = ExtDataJSON(fp.SizeCheck.Snapshot)
	}
	return hit
}

func (r *Runner) grabOneBacktest(outcomeID uint) {
	if !r.orderMu.TryLock() {
		return
	}
	defer r.orderMu.Unlock()
	lim := r.limits()
	all, _, ok := r.tryAllPicksForGrab(outcomeID, lim)
	if !ok {
		return
	}
	ctx := r.grabContext()
	mapKey := r.w.MapKey
	persist := r.grabPersist()
	for _, fp := range all {
		leg1, leg2 := r.snapPickLegs(fp)
		SaveGrabHit(ctx, persist, mapKey, GrabHit{
			Pick: fp, Leg1: leg1, Leg2: leg2, MaxCost: lim.MaxCost,
			YesFill: fp.Qty, NoFill: fp.Qty,
		})
	}
}
