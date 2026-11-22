package watchgroup

import (
	"context"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Ops 本组 watch 上由 strategy_entry 注入的行情以外能力（参数、发单、落库）。
type Ops struct {
	StrategyParams   func() (maxCost, orderAmount, levelFactor float64)
	OneX2Limits      func() (yesTripleMax, noTripleMax float64)
	TradeFee         func(platform, category, sub string) float64
	CanTrade         func(platform string) bool
	SendOnOutcome    func(outcomeIndex int, buyYes bool, px, qty float64) (fill float64, err error)
	FlattenOnOutcome func(outcomeIndex int, boughtYes bool, mine, other float64)
	OppRepo          func() persistentrepo.ArbBinaryOpportunityRepo
	OrderRepo        func() persistentrepo.ArbBinaryOrderRepo
	BookRepo         func() persistentrepo.OrderBookRepo
	BooksJSONForPair func(outcomeIndexA, outcomeIndexB int) string
	BookRowsForPair  func(outcomeIndexA, outcomeIndexB int) []*persistententity.OrderBook

	BacktestMode      bool
	BacktestTaskID    uint
	BacktestOppRepo   func() persistentrepo.BacktestArbBinaryOpportunityRepo
	BacktestOrderRepo func() persistentrepo.BacktestOrderRepo
	SnapshotNow       func() time.Time
	SnapshotAtMs      func() int64
}

// Watch 一组 map_key 的内存盘口与 outcome 行（yesno 策略直接读写字段/方法）。
type Watch struct {
	Ctx       context.Context
	MapKey    string
	Outcomes  []persistententity.MarketOutcome
	// MarketsByRowID 本组 outcome 关联的父场次，用于落库 competition_en 等。
	MarketsByRowID map[uint]persistententity.Market
	ByToken   map[string]int
	ByTicker  map[string]int
	ByPredict map[string]int
	BySxbet   map[string]int
	Books     *book.Store
	Ops       Ops
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