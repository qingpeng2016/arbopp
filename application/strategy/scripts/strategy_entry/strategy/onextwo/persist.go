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

func SaveGrabHit(ctx context.Context, d persistDeps, mapKey string, hit GrabHit) bool {
	if d.task > 0 && d.btOpp != nil {
		return saveBacktest(ctx, d, mapKey, hit)
	}
	if d.opp == nil {
		return false
	}
	return saveLive(ctx, d, mapKey, hit)
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
	if !pick.At.IsZero() && (!hit.Legs[0].OK || !hit.Legs[1].OK || !hit.Legs[2].OK) {
		return nil
	}
	now := pick.At
	if now.IsZero() {
		now, _ = snapshotTimes(d)
	}
	now = now.UTC()
	snapMs := now.UnixMilli()
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
	leg1Info, leg1Books, leg2Info, leg2Books, leg3Info, leg3Books := tripleLegFields(d, pick, hit.Legs)
	asks := pick.Prices
	leg1Ask := decimal.NewFromFloat(asks[0]).Round(8)
	leg2Ask := decimal.NewFromFloat(asks[1]).Round(8)
	leg3Ask := decimal.NewFromFloat(asks[2]).Round(8)
	perShare := pick.Per
	profitSpace := profitBase - perShare
	if profitSpace <= 0 || (thVal > 0 && perShare >= thVal) {
		return nil
	}
	arbQty, arbProfit := tripleArbQtyProfit(profitSpace, pick)
	mapKeyPtr := mapKey
	return &persistententity.BacktestArbBinaryOpportunity{
		BacktestTaskID: d.task,
		OppKind:        pick.Kind, MapKey: &mapKeyPtr,
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		Leg1Platform: watchgroup.LegPlatform(pick.Legs[0].Platform),
		Leg1Info:     leg1Info, Leg1OrderBooks: leg1Books, Leg1BestAsk: &leg1Ask,
		Leg2Platform: watchgroup.LegPlatform(pick.Legs[1].Platform),
		Leg2Info:     leg2Info, Leg2OrderBooks: leg2Books, Leg2BestAsk: &leg2Ask,
		Leg3Platform: watchgroup.LegPlatform(pick.Legs[2].Platform),
		Leg3Info:     leg3Info, Leg3OrderBooks: leg3Books, Leg3BestAsk: &leg3Ask,
		TotalCost:   decimal.NewFromFloat(perShare).Round(8),
		ProfitSpace: decimal.NewFromFloat(profitSpace).Round(8),
		ArbQty:      arbQty, ArbProfit: arbProfit,
		ThresholdKey: thKey, ThresholdValue: decimal.NewFromFloat(thVal).Round(8),
		BookUpdatedAt: &now, SnapshotAt: now, SnapshotAtMs: snapMs,
	}
}

func saveBacktest(ctx context.Context, d persistDeps, mapKey string, hit GrabHit) bool {
	row := buildOppBacktest(d, mapKey, hit)
	if row == nil {
		return false
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := d.btOpp.BatchCreate(runCtx, nil, []*persistententity.BacktestArbBinaryOpportunity{row}); err != nil {
		notification.SendErrorLog(runCtx, "onextwo-backtest-OppCreateFailed", zap.Error(err))
		return false
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
	return true
}

func saveLive(ctx context.Context, d persistDeps, mapKey string, hit GrabHit) bool {
	bt := buildOppBacktest(d, mapKey, hit)
	if bt == nil {
		return false
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	live := &persistententity.ArbBinaryOpportunity{
		OppKind: bt.OppKind, MapKey: bt.MapKey,
		Category: bt.Category, SubCategory: bt.SubCategory,
		Competition: bt.Competition, CompetitionEn: bt.CompetitionEn,
		Leg1Platform: bt.Leg1Platform,
		Leg1Info:     bt.Leg1Info, Leg1OrderBooks: bt.Leg1OrderBooks, Leg1BestAsk: bt.Leg1BestAsk,
		Leg2Platform: bt.Leg2Platform,
		Leg2Info:     bt.Leg2Info, Leg2OrderBooks: bt.Leg2OrderBooks, Leg2BestAsk: bt.Leg2BestAsk,
		Leg3Platform: bt.Leg3Platform,
		Leg3Info:     bt.Leg3Info, Leg3OrderBooks: bt.Leg3OrderBooks, Leg3BestAsk: bt.Leg3BestAsk,
		TotalCost: bt.TotalCost, ProfitSpace: bt.ProfitSpace,
		ArbQty: bt.ArbQty, ArbProfit: bt.ArbProfit,
		ThresholdKey: bt.ThresholdKey, ThresholdValue: bt.ThresholdValue,
		BookUpdatedAt: bt.BookUpdatedAt, SnapshotAt: bt.SnapshotAt, SnapshotAtMs: bt.SnapshotAtMs,
	}
	if err := d.opp.BatchCreate(runCtx, nil, []*persistententity.ArbBinaryOpportunity{live}); err != nil {
		notification.SendErrorLog(runCtx, "onextwo-OppCreateFailed", zap.Error(err))
		return false
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
	return true
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

func tripleArbQtyProfit(profitSpace float64, pick TriplePick) (decimal.Decimal, decimal.Decimal) {
	minQ := pick.LegSizes[0]
	for i := 1; i < 3; i++ {
		if pick.LegSizes[i] > 0 && (minQ <= 0 || pick.LegSizes[i] < minQ) {
			minQ = pick.LegSizes[i]
		}
	}
	if minQ <= 0 {
		return decimal.Zero, decimal.Zero
	}
	return decimal.NewFromFloat(minQ).Round(8), decimal.NewFromFloat(minQ * profitSpace).Round(8)
}

func tripleLegFields(d persistDeps, pick TriplePick, snaps [3]watchgroup.LegPersistSnap) (leg1Info, leg1Books, leg2Info, leg2Books, leg3Info, leg3Books string) {
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
	for i := range snaps {
		if strings.TrimSpace(snaps[i].OrderBooks) == "" {
			continue
		}
		switch i {
		case 0:
			leg1Books = snaps[i].OrderBooks
		case 1:
			leg2Books = snaps[i].OrderBooks
		case 2:
			leg3Books = snaps[i].OrderBooks
		}
	}
	if d.legBooks != nil {
		if strings.TrimSpace(leg1Books) == "" || leg1Books == "{}\n" {
			leg1Books = d.legBooks(pick.Legs[0].Index, buy)
		}
		if strings.TrimSpace(leg2Books) == "" || leg2Books == "{}\n" {
			leg2Books = d.legBooks(pick.Legs[1].Index, buy)
		}
		if strings.TrimSpace(leg3Books) == "" || leg3Books == "{}\n" {
			leg3Books = d.legBooks(pick.Legs[2].Index, buy)
		}
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
