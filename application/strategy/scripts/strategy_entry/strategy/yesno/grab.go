package yesno

import (
	"context"
	"sync"
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
	outcomeIndex, okIdx := r.outcomeIndex(outcomeID)
	if !okIdx {
		return FilledPick{}, false
	}
	groupKey := r.outcomeGroupKeyAt(outcomeIndex)
	if groupKey == "" {
		return FilledPick{}, false
	}
	group, okGrp := r.outcomeGroups[groupKey]
	if !okGrp {
		return FilledPick{}, false
	}
	return r.bestFilledPick(group)
}

// grabOne 实盘/联调：命中后异步落库（当前 DryRun，不发单）。
func (r *Runner) grabOne(outcomeID uint) {
	pick, ok := r.tryPickForGrab(outcomeID)
	if !ok {
		return
	}
	lim := r.limits()
	if !r.canTrade(pick.Yes.Platform) || !r.canTrade(pick.No.Platform) {
		return
	}
	if !r.orderMu.TryLock() {
		return
	}
	defer r.orderMu.Unlock()
	// 联调机会表：暂不发单。要恢复实盘时删掉 DryRun 块并取消 Fire 注释。
	var yesFill, noFill float64
	var yesErr, noErr error
	// yesFill, noFill, yesErr, noErr = Fire(outcomeSender{r}, pick.Yes.Side, pick.No.Side, pick.YesBuyYes, pick.NoBuyYes, pick.Qty)
	hit := GrabHit{
		Pick: pick, MaxCost: lim.MaxCost,
		YesFill: yesFill, NoFill: noFill, YesErr: yesErr, NoErr: noErr,
		DryRun: true,
	}
	ctx := r.grabContext()
	mapKey := r.w.MapKey
	persist := r.grabPersist()
	go func(ctx context.Context, mapKey string, hit GrabHit, persist PersistGrab) {
		NotifyGrab(ctx, mapKey, hit)
		SaveGrabHit(ctx, persist, mapKey, hit)
	}(ctx, mapKey, hit, persist)
}

// grabOneBacktest 历史回放：不发真实单，同步写入 backtest_arb_*（由 Ops.BacktestTaskID 决定落库表）。
func (r *Runner) grabOneBacktest(outcomeID uint) {
	pick, ok := r.tryPickForGrab(outcomeID)
	if !ok {
		return
	}
	lim := r.limits()
	ctx := r.grabContext()
	mapKey := r.w.MapKey
	persist := r.grabPersist()
	qty := pick.Qty
	hit := GrabHit{
		Pick: pick, MaxCost: lim.MaxCost,
		YesFill: qty, NoFill: qty,
	}
	if !r.orderMu.TryLock() {
		return
	}
	defer r.orderMu.Unlock()
	SaveGrabHit(ctx, persist, mapKey, hit)
}