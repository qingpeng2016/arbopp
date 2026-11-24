package yesno

import (
	"context"
	"strings"
	"sync"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Runner 跨所二元 YES+NO 策略（由 watch 在盘口更新时调用）。
// 仅 moneyline：同一命题 YES+NO；让分/大小球等不入组（与 onextwo 一致）。
// 每个 Watch 对应一个 map_key，orderMu 即按场次互斥，防止双所 WS 同时触发重复下单。
type Runner struct {
	w             *watchgroup.Watch
	outcomeGroups map[string]OutcomeGroup
	recent        watchgroup.RecentHits
	orderMu       sync.Mutex // 仅在确认套利、发单/落库前 TryLock，不挡盘口试算
}

// Entry 绑定本组 watch，预建 OutcomeGroup 索引。
func Entry(w *watchgroup.Watch) *Runner {
	return &Runner{
		w:             w,
		recent:        make(watchgroup.RecentHits),
		outcomeGroups: buildOutcomeGroups(w),
	}
}

func buildOutcomeGroups(w *watchgroup.Watch) map[string]OutcomeGroup {
	groups := map[string]OutcomeGroup{}
	n := w.OutcomeCount()
	for i := 0; i < n; i++ {
		oc := w.OutcomeAt(i)
		if oc == nil || !outcomeIsMoneyline(oc) {
			continue
		}
		key := w.OutcomeGroupKeyAt(i)
		if key == "" {
			continue
		}
		group := groups[key]
		group.AddMarketOutcome(oc, i, w.OutcomeAt)
		groups[key] = group
	}

	// 这个地方就是outcome的映射关键了，取了outcome的name字符串匹配 队名称，或者平局的draw
	// Runner 里 buildOutcomeGroups 建出来的表
	/*outcomeGroups: map[string]OutcomeGroup{
	"v1|英超|epl-ars-sun-2026-11-21|arsenal": {
		byPlatform: map[string]int{
			"POLYMARKET": 0, // → Outcomes[0] 阿森纳赢
			"KALSHI":     3, // → Outcomes[3] 阿森纳赢
		},
		},
	"v1|英超|epl-ars-sun-2026-11-21|sunderland": {
		byPlatform: map[string]int{
			"POLYMARKET": 1,
			"KALSHI":     2,
		},
		},
	}
	*/
	return groups
}

func (r *Runner) OnBookUpdate(outcomeID uint) {
	idx, ok := r.outcomeIndex(outcomeID)
	if !ok {
		return
	}
	if oc := r.w.OutcomeAt(idx); oc == nil || !outcomeIsMoneyline(oc) {
		return
	}
	if r.w != nil && r.w.Ops.BacktestMode {
		r.grabOneBacktest(outcomeID)
		return
	}
	r.grabOne(outcomeID)
}

func outcomeIsMoneyline(oc *persistententity.MarketOutcome) bool {
	if oc == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(oc.OutcomeType), "moneyline")
}

func (r *Runner) outcomeIndex(outcomeID uint) (int, bool) {
	return r.w.OutcomeIndex(outcomeID)
}

func (r *Runner) outcomeGroupKeyAt(outcomeIndex int) string {
	if outcomeIndex < 0 {
		return ""
	}
	return r.w.OutcomeGroupKeyAt(outcomeIndex)
}

// OutcomeGroupKeyForOutcomeID 同一 OutcomeGroup 去重触发用。
func (r *Runner) OutcomeGroupKeyForOutcomeID(outcomeID uint) string {
	idx, ok := r.outcomeIndex(outcomeID)
	if !ok {
		return ""
	}
	return r.outcomeGroupKeyAt(idx)
}

func (r *Runner) sideTop(outcomeIndex int) (Side, bool) {
	s, ok := r.execSide(outcomeIndex)
	return s.Side, ok
}

func (r *Runner) canTrade(platform string) bool {
	if r.w.Ops.CanTrade == nil {
		return false
	}
	return r.w.Ops.CanTrade(platform)
}

func (r *Runner) grabContext() context.Context {
	return r.w.Context()
}

func (r *Runner) limits() Limits {
	if r.w.Ops.StrategyParams == nil {
		return Limits{}
	}
	maxCost, minAmount, maxAmount := r.w.Ops.StrategyParams()
	return Limits{MaxCost: maxCost, MinAmount: minAmount, MaxAmount: maxAmount}
}

func (r *Runner) execSide(outcomeIndex int) (ExecSide, bool) {
	if r.w.Store() == nil {
		return ExecSide{}, false
	}
	view, _ := r.w.StrategySnapshot()
	return QuoteExecSide(r.w, view, outcomeIndex)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type outcomeSender struct{ r *Runner }

func (s outcomeSender) Send(side Side, buyYes bool, px, qty float64) (watchgroup.SendResult, error) {
	if s.r.w.Ops.SendOnOutcome == nil {
		return watchgroup.SendResult{}, nil
	}
	return s.r.w.Ops.SendOnOutcome(side.Index, buyYes, px, qty)
}

func (s outcomeSender) Flatten(side Side, boughtYes bool, mine, other float64) {
	if s.r.w.Ops.FlattenOnOutcome == nil {
		return
	}
	s.r.w.Ops.FlattenOnOutcome(side.Index, boughtYes, mine, other)
}

func (r *Runner) snapPickLegs(fp FilledPick) (watchgroup.LegPersistSnap, watchgroup.LegPersistSnap) {
	return fp.Yes.PersistQuote(fp.YesBuyYes, fp.Costs[0], fp.Sizes[0]),
		fp.No.PersistQuote(fp.NoBuyYes, fp.Costs[1], fp.Sizes[1])
}

func (r *Runner) grabPersist() PersistGrab {
	ops := &r.w.Ops
	pg := PersistGrab{
		ScopeForOutcomeIDs: r.w.ScopeForOutcomeIDs,
		OppRepo:            safeOppRepo(ops),
		OrderRepo:          safeOrderRepo(ops),
		BookRepo:           safeBookRepo(ops),
		YesLegInfo:         func(i int, buyYes bool) string { return r.w.YesLegInfoJSON(i, buyYes) },
		NoLegInfo:          func(i int, buyYes bool) string { return r.w.NoLegInfoJSON(i, buyYes) },

		BookRows: r.bookRowsForPick,
	}
	if ops.BacktestTaskID > 0 {
		pg.BacktestTaskID = ops.BacktestTaskID
		if ops.BacktestOppRepo != nil {
			pg.BacktestOppRepo = ops.BacktestOppRepo()
		}
		if ops.BacktestOrderRepo != nil {
			pg.BacktestOrderRepo = ops.BacktestOrderRepo()
		}
	}
	if ops.SnapshotNow != nil {
		pg.SnapshotAt = ops.SnapshotNow
	}
	if ops.SnapshotAtMs != nil {
		pg.SnapshotAtMs = ops.SnapshotAtMs
	}
	return pg
}

func safeOppRepo(ops *watchgroup.Ops) persistentrepo.ArbBinaryOpportunityRepo {
	if ops.OppRepo == nil {
		return nil
	}
	return ops.OppRepo()
}

func safeOrderRepo(ops *watchgroup.Ops) persistentrepo.ArbBinaryOrderRepo {
	if ops.OrderRepo == nil {
		return nil
	}
	return ops.OrderRepo()
}

func safeBookRepo(ops *watchgroup.Ops) persistentrepo.OrderBookRepo {
	if ops.BookRepo == nil {
		return nil
	}
	return ops.BookRepo()
}

func (r *Runner) booksJSONForPick(a, b ExecSide) string {
	if r.w.Ops.BooksJSONForPair == nil {
		return ""
	}
	return r.w.Ops.BooksJSONForPair(a.Index, b.Index)
}

func (r *Runner) bookRowsForPick(pick FilledPick) []*persistententity.OrderBook {
	if r.w.Ops.BookRowsForPair == nil {
		return nil
	}
	return r.w.Ops.BookRowsForPair(pick.view, pick.Yes.Index, pick.No.Index)
}
