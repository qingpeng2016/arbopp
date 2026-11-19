package backtest

import (
	"context"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	strategyentry "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/onextwo"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

type strategyEntrySession struct {
	watch         *watchgroup.Watch
	runner        *yesno.Runner
	onextwo       *onextwo.Runner
	replayNow     time.Time
	replayAtMs    int64
	replayNowFunc func() time.Time
}

func newStrategyEntrySession(
	ctx context.Context,
	mapKey string,
	outcomes []persistententity.MarketOutcome,
	marketsByRow map[uint]persistententity.Market,
	taskID uint,
	params strategyentry.Params,
	btOpp persistentrepo.BacktestArbBinaryOpportunityRepo,
	btOrd persistentrepo.BacktestOrderRepo,
) *strategyEntrySession {
	if marketsByRow == nil {
		marketsByRow = map[uint]persistententity.Market{}
	}
	wg := &watchgroup.Watch{
		Ctx:            ctx,
		MapKey:         mapKey,
		Outcomes:       append([]persistententity.MarketOutcome(nil), outcomes...),
		MarketsByRowID: marketsByRow,
		ByToken:        map[string]int{},
		ByTicker:       map[string]int{},
		ByPredict:      map[string]int{},
		Books:          book.NewStore(),
	}
	for i := range wg.Outcomes {
		oc := &wg.Outcomes[i]
		if strings.EqualFold(oc.Platform, appcommon.PlatformPolymarket) {
			if id := strings.TrimSpace(oc.YesTokenID); id != "" {
				wg.ByToken[id] = i
			}
			if id := strings.TrimSpace(oc.NoTokenID); id != "" {
				wg.ByToken[id] = i
			}
		}
		if strings.EqualFold(oc.Platform, appcommon.PlatformKalshi) {
			if id := strings.TrimSpace(oc.PlatformMarketOutcomeID); id != "" {
				wg.ByTicker[id] = i
			}
		}
		if strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
			if id := strings.TrimSpace(oc.PlatformMarketOutcomeID); id != "" {
				wg.ByPredict[id] = i
			}
		}
	}
	sess := &strategyEntrySession{watch: wg}
	sess.replayNowFunc = func() time.Time {
		if sess.replayNow.IsZero() {
			return time.Now().UTC()
		}
		return sess.replayNow
	}
	wg.Ops = watchgroup.Ops{
		BacktestMode:   true,
		BacktestTaskID: taskID,
		BacktestOppRepo: func() persistentrepo.BacktestArbBinaryOpportunityRepo {
			return btOpp
		},
		BacktestOrderRepo: func() persistentrepo.BacktestOrderRepo {
			return btOrd
		},
		SnapshotNow: sess.replayNowFunc,
		SnapshotAtMs: func() int64 {
			if sess.replayAtMs > 0 {
				return sess.replayAtMs
			}
			if !sess.replayNow.IsZero() {
				return sess.replayNow.UnixMilli()
			}
			return time.Now().UTC().UnixMilli()
		},
		StrategyParams: func() (float64, float64, float64) {
			return params.MaxCost, params.OrderAmount, params.LevelFactor
		},
		OneX2Limits: func() (float64, float64) {
			return params.OneX2YesMax, params.OneX2NoMax
		},
		TradeFee: func(platform, category, sub string) float64 {
			return params.Fee(platform, category, sub)
		},
		CanTrade: func(platform string) bool {
			return true
		},
		SendOnOutcome: func(outcomeIndex int, buyYes bool, px, qty float64) (float64, error) {
			return qty, nil
		},
		BooksJSONForPair: func(outcomeIndexA, outcomeIndexB int) string {
			return wg.BooksJSONForOutcomePair(outcomeIndexA, outcomeIndexB)
		},
	}
	sess.runner = yesno.Entry(wg)
	sess.onextwo = onextwo.Entry(wg)
	return sess
}

func (s *strategyEntrySession) SetReplayTime(t time.Time) {
	if s == nil {
		return
	}
	s.replayNow = t
	s.replayAtMs = t.UnixMilli()
}