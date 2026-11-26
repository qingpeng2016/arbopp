package watchgroup

import (
	"context"
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Ops 本组 watch 上由 strategy_entry 注入的行情以外能力（参数、发单、落库）。
// 约定：WS 热路径（盘口→选腿→发单）只读内存；唯一允许的 DB 是命中后写机会/订单。
type Ops struct {
	StrategyParams   func() (maxCost, minAmount, maxAmount float64)
	OneX2Limits      func() (yesTripleMax, noTripleMax float64)
	// FillShareMismatchPct 下单结果判定：各平台成交份额 (max-min)/max 上限。
	FillShareMismatchPct func() float64
	TradeFee             func(platform, category, sub string) float64
	CanTrade             func(platform string) bool
	// MapKeyHasOpenOrders 未终态订单阻塞（内存，reconcile 刷库）。
	MapKeyHasOpenOrders func(mapKey string) bool
	// NoteOpenOrdersWritten 实盘落单后立刻记入内存。
	NoteOpenOrdersWritten func(mapKey string)
	SendOnOutcome         func(outcomeIndex int, buyYes bool, px, qty float64) (SendResult, error)
	FlattenOnOutcome      func(outcomeIndex int, boughtYes bool, mine, other float64)
	OppRepo               func() persistentrepo.ArbBinaryOpportunityRepo
	OrderRepo             func() persistentrepo.ArbBinaryOrderRepo
	BookRepo              func() persistentrepo.OrderBookRepo
	BooksJSONForPair      func(outcomeIndexA, outcomeIndexB int) string
	BookRowsForPair       func(view book.Snapshot, outcomeIndexA, outcomeIndexB int) []*persistententity.OrderBook

	BacktestMode      bool
	BacktestTaskID    uint
	BacktestOppRepo   func() persistentrepo.BacktestArbBinaryOpportunityRepo
	BacktestOrderRepo func() persistentrepo.BacktestOrderRepo
	SnapshotNow       func() time.Time
	SnapshotAtMs      func() int64
}

// Watch 一组 map_key 的内存盘口与 outcome 行（yesno 策略直接读写字段/方法）。
type Watch struct {
	Ctx      context.Context
	MapKey   string
	Outcomes []persistententity.MarketOutcome
	// MarketsByRowID 本组 outcome 关联的父场次，用于落库 competition_en 等。
	MarketsByRowID map[uint]persistententity.Market
	ByToken        map[string]int
	ByTicker       map[string]int
	ByPredict      map[string]int
	BySxbet        map[string]int
	Books          *book.Store
	Ops            Ops
	bookEventMu    sync.Mutex
}

// ApplyBookEvent publishes one feed event, including any derived NO books.
func (w *Watch) ApplyBookEvent(apply func() bool) bool {
	w.bookEventMu.Lock()
	defer w.bookEventMu.Unlock()
	return apply()
}

func (w *Watch) StrategySnapshot() (book.Snapshot, time.Time) {
	w.bookEventMu.Lock()
	view := w.Books.Snapshot()
	w.bookEventMu.Unlock()
	at := view.At
	if w.Ops.BacktestMode && w.Ops.SnapshotNow != nil {
		at = w.Ops.SnapshotNow()
	}
	return view, at
}

func (w *Watch) Context() context.Context { return w.Ctx }

func (w *Watch) Store() *book.Store { return w.Books }

func (w *Watch) OutcomeCount() int { return len(w.Outcomes) }

func (w *Watch) OutcomeAt(outcomeIndex int) *persistententity.MarketOutcome {
	if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
		return nil
	}
	return &w.Outcomes[outcomeIndex]
}

func (w *Watch) OutcomeGroupKeyAt(outcomeIndex int) string {
	if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
		return ""
	}
	return marketmap.OutcomeGroupKeyForWatch(&w.Outcomes[outcomeIndex], w.MapKey)
}

func (w *Watch) OutcomeIndex(outcomeID uint) (int, bool) {
	if outcomeID == 0 {
		return 0, false
	}
	for i := range w.Outcomes {
		if w.Outcomes[i].ID == outcomeID {
			return i, true
		}
	}
	return 0, false
}
