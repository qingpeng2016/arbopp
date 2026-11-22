package onextwo

import (
	"context"
	"strconv"
	"strings"
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
	scope       func(yesMoID, noMoID uint) watchgroup.FixtureScope
	drawLegInfo func(outcomeIndex int, buyYes bool) string
	yesLegInfo  func(outcomeIndex int, buyYes bool) string
	noLegInfo   func(outcomeIndex int, buyYes bool) string
	legBooks    func(outcomeIndex int, buyYes bool) string
	opp         persistentrepo.ArbBinaryOpportunityRepo
	order       persistentrepo.ArbBinaryOrderRepo
	btOpp       persistentrepo.BacktestArbBinaryOpportunityRepo
	btOrd       persistentrepo.BacktestOrderRepo
	task        uint
	snap        func() time.Time
	snapMs      func() int64
}

func (r *Runner) persistDeps() persistDeps {
	ops := &r.w.Ops
	d := persistDeps{
		task: ops.BacktestTaskID, scope: r.w.ScopeForOutcomeIDs, snap: ops.SnapshotNow, snapMs: ops.SnapshotAtMs,
		drawLegInfo: func(i int, buy bool) string { return r.w.DrawLegInfoJSON(i, buy) },
		yesLegInfo:  func(i int, buy bool) string { return r.w.YesLegInfoJSON(i, buy) },
		noLegInfo:   func(i int, buy bool) string { return r.w.NoLegInfoJSON(i, buy) },
		legBooks:    func(i int, buy bool) string { return r.w.FullOrderBooksJSON(i, buy) },
	}
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

func NotifyGrab(ctx context.Context, mapKey string, hit GrabHit) {
	title := "strategy_entry 跨所下单"
	mode := "live"
	if hit.DryRun {
		title = "strategy_entry 套利命中（未发单）"
		mode = "dry_run"
	}
	pick := hit.Pick
	tripleSide := "NO"
	if pick.BuyYes {
		tripleSide = "YES"
	}
	fields := []notification.FieldPair{
		{Key: "mode", Value: mode},
		{Key: "strategy", Value: pick.Kind},
		{Key: "map_key", Value: mapKey},
		{Key: "triple_side", Value: tripleSide},
		{Key: "qty", Value: strconv.FormatFloat(pick.Qty, 'f', -1, 64)},
		{Key: "per_share", Value: strconv.FormatFloat(pick.Per, 'f', -1, 64)},
	}
	for i, leg := range pick.Legs {
		px := pick.Prices[i]
		fields = append(fields,
			notification.FieldPair{Key: "leg" + strconv.Itoa(i+1), Value: leg.Platform + " " + tripleSide},
			notification.FieldPair{Key: "leg" + strconv.Itoa(i+1) + "_ask", Value: strconv.FormatFloat(px, 'f', -1, 64)},
		)
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, title, fields)
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
	drawMoID, yesMoID, noMoID := pick.Legs[0].OutcomeID, pick.Legs[1].OutcomeID, pick.Legs[2].OutcomeID
	scope := watchgroup.FixtureScope{}
	if d.scope != nil {
		scope = d.scope(yesMoID, noMoID)
	}
	_ = drawMoID
	profitBase := 1.0
	thKey := ThresholdKeyYesTriple
	thVal := hit.Limits.YesTripleMax
	if !pick.BuyYes {
		profitBase = 2.0
		thKey = ThresholdKeyNoTriple
		thVal = hit.Limits.NoTripleMax
	}
	leg1Info, leg1Books, leg2Info, leg2Books, leg3Info, leg3Books := tripleLegFields(d, pick)
	leg1Ask := decimal.NewFromFloat(pick.Prices[0]).Round(8)
	leg2Ask := decimal.NewFromFloat(pick.Prices[1]).Round(8)
	leg3Ask := decimal.NewFromFloat(pick.Prices[2]).Round(8)
	mapKeyPtr := mapKey
	return &persistententity.BacktestArbBinaryOpportunity{
		BacktestTaskID: d.task,
		OppKind:        pick.Kind, MapKey: &mapKeyPtr,
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		Leg1Info: leg1Info, Leg1OrderBooks: leg1Books, Leg1BestAsk: &leg1Ask,
		Leg2Info: leg2Info, Leg2OrderBooks: leg2Books, Leg2BestAsk: &leg2Ask,
		Leg3Info: leg3Info, Leg3OrderBooks: leg3Books, Leg3BestAsk: &leg3Ask,
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
		Leg1Info: bt.Leg1Info, Leg1OrderBooks: bt.Leg1OrderBooks, Leg1BestAsk: bt.Leg1BestAsk,
		Leg2Info: bt.Leg2Info, Leg2OrderBooks: bt.Leg2OrderBooks, Leg2BestAsk: bt.Leg2BestAsk,
		Leg3Info: bt.Leg3Info, Leg3OrderBooks: bt.Leg3OrderBooks, Leg3BestAsk: bt.Leg3BestAsk,
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

func tripleLegFields(d persistDeps, pick TriplePick) (leg1Info, leg1Books, leg2Info, leg2Books, leg3Info, leg3Books string) {
	leg1Info, leg1Books, leg2Info, leg2Books, leg3Info, leg3Books = "{}\n", "{}\n", "{}\n", "{}\n", "{}\n", "{}\n"
	buy := pick.BuyYes
	if d.drawLegInfo != nil {
		leg1Info = d.drawLegInfo(pick.Legs[0].Index, buy)
	}
	if d.yesLegInfo != nil {
		leg2Info = d.yesLegInfo(pick.Legs[1].Index, buy)
	}
	if d.noLegInfo != nil {
		leg3Info = d.noLegInfo(pick.Legs[2].Index, buy)
	}
	if d.legBooks != nil {
		leg1Books = d.legBooks(pick.Legs[0].Index, buy)
		leg2Books = d.legBooks(pick.Legs[1].Index, buy)
		leg3Books = d.legBooks(pick.Legs[2].Index, buy)
	}
	for _, p := range []*string{&leg1Info, &leg1Books, &leg2Info, &leg2Books, &leg3Info, &leg3Books} {
		if strings.TrimSpace(*p) == "" {
			*p = "{}\n"
		}
	}
	return
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