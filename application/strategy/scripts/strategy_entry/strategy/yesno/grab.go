package yesno

import (
	"context"
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
)

// Sender 双边发单与单腿平仓。
type Sender interface {
	Send(side Side, buyYes bool, px, qty float64) (fill float64, err error)
	Flatten(side Side, boughtYes bool, mine, other float64)
}

// Fire 并发双腿 FAK，再 flatten。
func Fire(snd Sender, a, b Side, aBuyYes, bBuyYes bool, qty float64) (fillA, fillB float64, errA, errB error) {
	pxA, pxB := SidePx(a, aBuyYes), SidePx(b, bBuyYes)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		fillA, errA = snd.Send(a, aBuyYes, pxA, qty)
	}()
	go func() {
		defer wg.Done()
		fillB, errB = snd.Send(b, bBuyYes, pxB, qty)
	}()
	wg.Wait()
	snd.Flatten(a, aBuyYes, fillA, fillB)
	snd.Flatten(b, bBuyYes, fillB, fillA)
	return fillA, fillB, errA, errB
}

// GrabHit 一次抢单结果。
type GrabHit struct {
	Pick            FilledPick
	Leg1, Leg2      watchgroup.LegPersistSnap // 同锁盘口 JSON；TopAsk/TopSz 与 pick 对齐
	MaxCost         float64
	YesFill, NoFill float64
	YesErr, NoErr   error
	DryRun          bool // true：只记机会表，不发单、不写 arb_binary_order
}

// TryPickForOutcome 盘口更新后试算最优组合（回测扫描等只读路径，不落库）。
func (r *Runner) TryPickForOutcome(outcomeID uint) (pick FilledPick, ok bool) {
	return r.tryPickForGrab(outcomeID)
}

// tryPickForGrab 盘口更新后在同一 OutcomeGroup 内选最优可成交组合（只读，不加锁）。
func (r *Runner) tryPickForGrab(outcomeID uint) (pick FilledPick, ok bool) {
	_, pick, ok = r.tryAllPicksForGrab(outcomeID)
	return pick, ok
}

func (r *Runner) tryAllPicksForGrab(outcomeID uint) (all []FilledPick, best FilledPick, ok bool) {
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
	return r.AllFilledPicks(group, r.limits())
}

// grabOne 实盘/联调：命中后异步落库（当前 DryRun，不发单）。
func (r *Runner) grabOne(outcomeID uint) {
	r.grabOneAttempt(outcomeID, false)
}

func (r *Runner) grabOneAttempt(outcomeID uint, retried bool) {
	if !r.orderMu.TryLock() {
		if !retried {
			go func(id uint) {
				time.Sleep(25 * time.Millisecond)
				r.grabOneAttempt(id, true)
			}(outcomeID)
		}
		return
	}
	defer r.orderMu.Unlock()
	all, pick, ok := r.tryAllPicksForGrab(outcomeID)
	if !ok {
		return
	}
	lim := r.limits()
	const dryRun = true
	if !dryRun && (!r.canTrade(pick.Yes.Platform) || !r.canTrade(pick.No.Platform)) {
		return
	}
	// 联调机会表：暂不发单。要恢复实盘时设 dryRun=false 并取消 Fire 注释。
	var yesFill, noFill float64
	var yesErr, noErr error
	if !dryRun {
		yesFill, noFill, yesErr, noErr = Fire(outcomeSender{r}, pick.Yes.Side, pick.No.Side, pick.YesBuyYes, pick.NoBuyYes, pick.Qty)
	}
	bestHit := GrabHit{
		Pick: pick, MaxCost: lim.MaxCost,
		YesFill: yesFill, NoFill: noFill, YesErr: yesErr, NoErr: noErr,
		DryRun: dryRun,
	}
	ctx := r.grabContext()
	mapKey := r.w.MapKey
	persist := r.grabPersist()
	for _, fp := range all {
		leg1, leg2 := r.snapPickLegs(fp)
		SaveGrabHit(ctx, persist, mapKey, GrabHit{
			Pick: fp, Leg1: leg1, Leg2: leg2, MaxCost: lim.MaxCost,
			YesFill: yesFill, NoFill: noFill, DryRun: dryRun,
		})
	}
	go func(ctx context.Context, mapKey string, best GrabHit) {
		NotifyGrab(ctx, mapKey, best)
	}(ctx, mapKey, bestHit)
}

// grabOneBacktest 历史回放：不发真实单，同步写入 backtest_arb_*（由 Ops.BacktestTaskID 决定落库表）。
func (r *Runner) grabOneBacktest(outcomeID uint) {
	if !r.orderMu.TryLock() {
		return
	}
	defer r.orderMu.Unlock()
	all, pick, ok := r.tryAllPicksForGrab(outcomeID)
	if !ok {
		return
	}
	lim := r.limits()
	ctx := r.grabContext()
	mapKey := r.w.MapKey
	persist := r.grabPersist()
	qty := pick.Qty
	for _, fp := range all {
		leg1, leg2 := r.snapPickLegs(fp)
		SaveGrabHit(ctx, persist, mapKey, GrabHit{
			Pick: fp, Leg1: leg1, Leg2: leg2, MaxCost: lim.MaxCost,
			YesFill: qty, NoFill: qty,
		})
	}
}