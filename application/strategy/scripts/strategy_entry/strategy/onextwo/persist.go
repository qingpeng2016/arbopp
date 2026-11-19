package onextwo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type persistDeps struct {
	scope    func(yesMoID, noMoID uint) watchgroup.FixtureScope
	opp      persistentrepo.ArbBinaryOpportunityRepo
	order    persistentrepo.ArbBinaryOrderRepo
	btOpp    persistentrepo.BacktestArbBinaryOpportunityRepo
	btOrd    persistentrepo.BacktestOrderRepo
	task     uint
	snap     func() time.Time
	snapMs   func() int64
}

func (r *Runner) persistDeps() persistDeps {
	ops := &r.w.Ops
	d := persistDeps{task: ops.BacktestTaskID, scope: r.w.ScopeForOutcomeIDs, snap: ops.SnapshotNow, snapMs: ops.SnapshotAtMs}
	if ops.BacktestOppRepo != nil {
		d.btOpp = ops.BacktestOppRepo()
	}
	if ops.BacktestOrderRepo != nil {
		d.btOrd = ops.BacktestOrderRepo()
	}
	if ops.OppRepo != nil {
		d.opp = ops.OppRepo()
	}
	if ops.OrderRepo != nil {
		d.order = ops.OrderRepo()
	}
	return d
}

func SaveGrabHit(ctx context.Context, d persistDeps, mapKey string, hit GrabHit) {
	if d.task > 0 && d.btOpp != nil {
		saveBacktest(ctx, d, mapKey, hit)
		return
	}
	if d.opp == nil {
		return
	}
	saveLive(ctx, d, mapKey, hit)
}

func snapshotTimes(d persistDeps) (time.Time, int64) {
	if d.snapMs != nil {
		if ms := d.snapMs(); ms > 0 {
			t := time.UnixMilli(ms).UTC()
			return t, ms
		}
	}
	now := time.Now().UTC()
	if d.snap != nil {
		now = d.snap()
	}
	return now, now.UnixMilli()
}

func buildOppBacktest(d persistDeps, mapKey string, hit GrabHit) *persistententity.BacktestArbBinaryOpportunity {
	pick := hit.Pick
	now, snapMs := snapshotTimes(d)
	yesMoID, noMoID := pick.Legs[0].OutcomeID, pick.Legs[1].OutcomeID
	leg3MoID := pick.Legs[2].OutcomeID
	scope := watchgroup.FixtureScope{}
	if d.scope != nil {
		scope = d.scope(yesMoID, noMoID)
	}
	sideWord := "YES"
	profitBase := 1.0
	thKey := ThresholdKeyYesTriple
	thVal := hit.Limits.YesTripleMax
	stratType := StrategyTypeYesTriple
	if !pick.BuyYes {
		sideWord = "NO"
		profitBase = 2.0
		thKey = ThresholdKeyNoTriple
		thVal = hit.Limits.NoTripleMax
		stratType = StrategyTypeNoTriple
	}
	outcomes := make([]map[string]any, 0, 3)
	for i, leg := range pick.Legs {
		outcomes = append(outcomes, map[string]any{
			"leg": i + 1, "platform": leg.Platform, "market_id": leg.MarketID,
			"market_outcome_id": leg.OutcomeID, "side": sideWord, "best_ask": pick.Prices[i], "size": pick.Qty,
		})
	}
	outJSON, _ := json.Marshal(outcomes)
	yesAsk := decimal.NewFromFloat(pick.Prices[0]).Round(8)
	noAsk := decimal.NewFromFloat(pick.Prices[1]).Round(8)
	mapKeyPtr := mapKey
	return &persistententity.BacktestArbBinaryOpportunity{
		BacktestTaskID: d.task,
		OppKind:        pick.Kind, MapKey: &mapKeyPtr,
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		YesMarketOutcomeID: yesMoID, NoMarketOutcomeID: noMoID, Leg3MarketOutcomeID: leg3MoID,
		OutcomesJSON: string(outJSON), OrderBooksJSON: `{"legs":3,"strategy":"` + stratType + `"}`,
		YesBestAsk: &yesAsk, NoBestAsk: &noAsk,
		TotalCost: decimal.NewFromFloat(pick.Per).Round(8),
		ProfitSpace: decimal.NewFromFloat(profitBase - pick.Per).Round(8),
		ThresholdKey: thKey, ThresholdValue: decimal.NewFromFloat(thVal).Round(8),
		BookUpdatedAt: &now, SnapshotAt: now, SnapshotAtMs: snapMs,
	}
}

func saveBacktest(ctx context.Context, d persistDeps, mapKey string, hit GrabHit) {
	row := buildOppBacktest(d, mapKey, hit)
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := d.btOpp.BatchCreate(runCtx, nil, []*persistententity.BacktestArbBinaryOpportunity{row}); err != nil {
		notification.SendErrorLog(runCtx, "onextwo-backtest-OppCreateFailed", zap.Error(err))
		return
	}
	if d.btOrd != nil && row.ID > 0 {
		strat := StrategyTypeYesTriple
		if !hit.Pick.BuyYes {
			strat = StrategyTypeNoTriple
		}
		orders := ordersForHit(d.task, strat, mapKey, row.ID, hit, row.SnapshotAt)
		if err := d.btOrd.BatchCreate(runCtx, nil, orders); err != nil {
			notification.SendErrorLog(runCtx, "onextwo-backtest-OrderCreateFailed", zap.Error(err))
		}
	}
}

func saveLive(ctx context.Context, d persistDeps, mapKey string, hit GrabHit) {
	bt := buildOppBacktest(d, mapKey, hit)
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	live := &persistententity.ArbBinaryOpportunity{
		OppKind: bt.OppKind, MapKey: bt.MapKey,
		Category: bt.Category, SubCategory: bt.SubCategory,
		Competition: bt.Competition, CompetitionEn: bt.CompetitionEn,
		YesMarketOutcomeID: bt.YesMarketOutcomeID, NoMarketOutcomeID: bt.NoMarketOutcomeID,
		Leg3MarketOutcomeID: bt.Leg3MarketOutcomeID,
		OutcomesJSON: bt.OutcomesJSON, OrderBooksJSON: bt.OrderBooksJSON,
		YesBestAsk: bt.YesBestAsk, NoBestAsk: bt.NoBestAsk,
		TotalCost: bt.TotalCost, ProfitSpace: bt.ProfitSpace,
		ThresholdKey: bt.ThresholdKey, ThresholdValue: bt.ThresholdValue,
		BookUpdatedAt: bt.BookUpdatedAt, SnapshotAt: bt.SnapshotAt, SnapshotAtMs: bt.SnapshotAtMs,
	}
	if err := d.opp.BatchCreate(runCtx, nil, []*persistententity.ArbBinaryOpportunity{live}); err != nil {
		notification.SendErrorLog(runCtx, "onextwo-OppCreateFailed", zap.Error(err))
		return
	}
	if !hit.DryRun && d.order != nil && live.ID > 0 {
		strat := StrategyTypeYesTriple
		if !hit.Pick.BuyYes {
			strat = StrategyTypeNoTriple
		}
		orders := liveOrdersForHit(strat, mapKey, live.ID, hit, live.SnapshotAt)
		if err := d.order.BatchCreate(runCtx, nil, orders); err != nil {
			notification.SendErrorLog(runCtx, "onextwo-OrderCreateFailed", zap.Error(err))
		}
	}
}

func ordersForHit(taskID uint, strategyType, mapKey string, oppID uint, hit GrabHit, sentAt time.Time) []*persistententity.BacktestOrder {
	pick := hit.Pick
	out := make([]*persistententity.BacktestOrder, 0, 3)
	for i, leg := range pick.Legs {
		px := pick.Prices[i]
		fill := pick.Qty
		if hit.DryRun {
			fill = 0
		}
		out = append(out, yesno.BacktestOrderForSide(strategyType, taskID, mapKey, oppID, sentAt, uint8(i+1), leg, pick.BuyYes, pick.Qty, px, fill, nil))
	}
	return out
}

func liveOrdersForHit(strategyType, mapKey string, oppID uint, hit GrabHit, sentAt time.Time) []*persistententity.ArbBinaryOrder {
	pick := hit.Pick
	out := make([]*persistententity.ArbBinaryOrder, 0, 3)
	for i, leg := range pick.Legs {
		px := pick.Prices[i]
		fill := pick.Qty
		if hit.DryRun {
			fill = 0
		}
		out = append(out, yesno.OrderForSide(strategyType, 0, mapKey, oppID, sentAt, uint8(i+1), leg, pick.BuyYes, pick.Qty, px, fill, nil))
	}
	return out
}