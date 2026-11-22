package yesno

import (
	"context"
	"strconv"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	thresholdKeyMaxCost = "binary_yes_no_max_cost"
	strategyTypeYesNo   = "BINARY_YES_NO"
	emptyLegJSON        = "{}\n"
)

// PersistGrab 抢单落库依赖。
type PersistGrab struct {
	ScopeForOutcomeIDs func(yesMoID, noMoID uint) watchgroup.FixtureScope
	OppRepo            persistentrepo.ArbBinaryOpportunityRepo
	OrderRepo persistentrepo.ArbBinaryOrderRepo
	BookRepo  persistentrepo.OrderBookRepo
	YesLegInfo    func(outcomeIndex int, buyYes bool) string
	NoLegInfo     func(outcomeIndex int, buyYes bool) string
	LegOrderBooks func(outcomeIndex int, buyYes bool) string
	BookRows      func(a, b ExecSide) []*persistententity.OrderBook

	BacktestTaskID    uint
	BacktestOppRepo   persistentrepo.BacktestArbBinaryOpportunityRepo
	BacktestOrderRepo persistentrepo.BacktestOrderRepo
	SnapshotAt        func() time.Time
	SnapshotAtMs      func() int64
}

func NotifyGrab(ctx context.Context, mapKey string, hit GrabHit) {
	a, b := hit.Pick.Yes, hit.Pick.No
	errA, errB := "", ""
	if hit.YesErr != nil {
		errA = hit.YesErr.Error()
	}
	if hit.NoErr != nil {
		errB = hit.NoErr.Error()
	}
	title := "strategy_entry 跨所下单"
	mode := "live"
	if hit.DryRun {
		title = "strategy_entry 套利命中（未发单）"
		mode = "dry_run"
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, title, []notification.FieldPair{
		{Key: "mode", Value: mode},
		{Key: "map_key", Value: mapKey},
		{Key: "side_a", Value: sideLabel(a.Platform, hit.Pick.YesBuyYes)},
		{Key: "side_b", Value: sideLabel(b.Platform, hit.Pick.NoBuyYes)},
		{Key: "qty", Value: strconv.FormatFloat(hit.Pick.Qty, 'f', -1, 64)},
		{Key: "per_share", Value: strconv.FormatFloat(hit.Pick.Per, 'f', -1, 64)},
		{Key: "fill_a", Value: strconv.FormatFloat(hit.YesFill, 'f', -1, 64)},
		{Key: "fill_b", Value: strconv.FormatFloat(hit.NoFill, 'f', -1, 64)},
		{Key: "err_a", Value: errA},
		{Key: "err_b", Value: errB},
	})
}

func SaveGrabHit(ctx context.Context, d PersistGrab, mapKey string, hit GrabHit) {
	if d.BacktestTaskID > 0 && d.BacktestOppRepo != nil {
		saveBacktestGrabHit(ctx, d, mapKey, hit)
		return
	}
	if d.OppRepo == nil {
		return
	}
	a, b := hit.Pick.Yes, hit.Pick.No
	aYes, bYes := hit.Pick.YesBuyYes, hit.Pick.NoBuyYes
	qty, maxCost := hit.Pick.Qty, hit.MaxCost
	aPx, bPx := SidePx(a.Side, aYes), SidePx(b.Side, bYes)
	yesPx, noPx := legAskPrices(hit, a, b, aYes, bYes, aPx, bPx)
	now, snapMs := snapshotTimes(d)
	yesMoID, noMoID := comboLegMarketOutcomeIDs(a, b, aYes)
	scope := fixtureScopeForGrab(d, yesMoID, noMoID)
	yesInfo, noInfo, yesBooks, noBooks := legPersistFields(d, a, b, aYes, bYes, hit.Leg1, hit.Leg2)
	perShare := hit.Pick.Per
	if !persistableBinarySnap(perShare, maxCost) {
		return
	}
	profitSpace := 1 - perShare
	yesLeg, noLeg, yesBuy, noBuy := comboLegSides(a, b, aYes, bYes)
	arbQty, arbProfit := arbQtyProfitFromBinaryPick(profitSpace, yesLeg.Side, noLeg.Side, yesBuy, noBuy)
	yes := decimal.NewFromFloat(yesPx).Round(8)
	no := decimal.NewFromFloat(noPx).Round(8)
	row := &persistententity.ArbBinaryOpportunity{
		OppKind: "BINARY_YES_NO", MapKey: strPtr(mapKey),
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		Leg1Info: yesInfo, Leg1OrderBooks: yesBooks, Leg1BestAsk: &yes,
		Leg2Info: noInfo, Leg2OrderBooks: noBooks, Leg2BestAsk: &no,
		Leg3Info: emptyLegJSON, Leg3OrderBooks: emptyLegJSON,
		TotalCost: decimal.NewFromFloat(perShare).Round(8), ProfitSpace: decimal.NewFromFloat(profitSpace).Round(8),
		ArbQty: arbQty, ArbProfit: arbProfit,
		ThresholdKey: thresholdKeyMaxCost, ThresholdValue: decimal.NewFromFloat(maxCost).Round(8),
		BookUpdatedAt: &now, SnapshotAt: now, SnapshotAtMs: snapMs,
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := d.OppRepo.BatchCreate(runCtx, nil, []*persistententity.ArbBinaryOpportunity{row}); err != nil {
		notification.SendErrorLog(runCtx, "strategy_entry-OppCreateFailed", zap.Error(err))
		return
	}
	if !hit.DryRun && d.OrderRepo != nil && row.ID > 0 {
		orders := []*persistententity.ArbBinaryOrder{
			orderForSide(strategyTypeYesNo, 0, mapKey, row.ID, now, 1, a, aYes, qty, aPx, hit.YesFill, hit.YesErr),
			orderForSide(strategyTypeYesNo, 0, mapKey, row.ID, now, 2, b, bYes, qty, bPx, hit.NoFill, hit.NoErr),
		}
		if err := d.OrderRepo.BatchCreate(runCtx, nil, orders); err != nil {
			notification.SendErrorLog(runCtx, "strategy_entry-OrderCreateFailed", zap.Error(err))
		}
	}
	if d.BookRepo != nil && d.BookRows != nil {
		if rows := d.BookRows(a, b); len(rows) > 0 {
			if err := d.BookRepo.BatchCreate(runCtx, nil, rows); err != nil {
				notification.SendErrorLog(runCtx, "strategy_entry-BookCreateFailed", zap.Error(err), zap.Int("rows", len(rows)))
			}
		}
	}
}

// BacktestOrderForSide 回测组合腿（写 backtest_orders）。
func BacktestOrderForSide(strategyType string, backtestTaskID uint, mapKey string, oppID uint, sentAt time.Time, sideNo uint8, s ExecSide, buyYes bool, qty, px, fill float64, sendErr error) *persistententity.BacktestOrder {
	return backtestOrderForSide(strategyType, backtestTaskID, mapKey, oppID, sentAt, sideNo, s, buyYes, qty, px, fill, sendErr)
}

func backtestOrderForSide(strategyType string, backtestTaskID uint, mapKey string, oppID uint, sentAt time.Time, sideNo uint8, s ExecSide, buyYes bool, qty, px, fill float64, sendErr error) *persistententity.BacktestOrder {
	side := "NO"
	if buyYes {
		side = "YES"
	}
	status, errMsg := orderStatus(fill, sendErr)
	return &persistententity.BacktestOrder{
		BacktestTaskID: backtestTaskID, StrategyType: strategyType,
		ArbOppID: oppID, MapKey: mapKey, SideNo: sideNo,
		Platform: s.Platform, PlatformMarketOutcomeID: s.MarketID,
		TokenID: sideToken(s.Platform, s.YesToken, s.NoToken, buyYes),
		Side: side, Action: "BUY",
		Price: decimal.NewFromFloat(px).Round(8), Qty: decimal.NewFromFloat(qty).Round(8),
		FillQty: decimal.NewFromFloat(fill).Round(8), Status: status, ErrMsg: errMsg, SentAt: sentAt,
	}
}

// OrderForSide 实盘组合腿（写 orders）。
func OrderForSide(strategyType string, backtestTaskID uint, mapKey string, oppID uint, sentAt time.Time, sideNo uint8, s ExecSide, buyYes bool, qty, px, fill float64, sendErr error) *persistententity.ArbBinaryOrder {
	return orderForSide(strategyType, backtestTaskID, mapKey, oppID, sentAt, sideNo, s, buyYes, qty, px, fill, sendErr)
}

func orderForSide(strategyType string, backtestTaskID uint, mapKey string, oppID uint, sentAt time.Time, sideNo uint8, s ExecSide, buyYes bool, qty, px, fill float64, sendErr error) *persistententity.ArbBinaryOrder {
	side := "NO"
	if buyYes {
		side = "YES"
	}
	status, errMsg := orderStatus(fill, sendErr)
	return &persistententity.ArbBinaryOrder{
		BacktestTaskID: backtestTaskID, StrategyType: strategyType,
		ArbOppID: oppID, MapKey: mapKey, SideNo: sideNo,
		Platform: s.Platform, PlatformMarketOutcomeID: s.MarketID,
		TokenID: sideToken(s.Platform, s.YesToken, s.NoToken, buyYes),
		Side: side, Action: "BUY",
		Price: decimal.NewFromFloat(px).Round(8), Qty: decimal.NewFromFloat(qty).Round(8),
		FillQty: decimal.NewFromFloat(fill).Round(8), Status: status, ErrMsg: errMsg, SentAt: sentAt,
	}
}

func sideLabel(platform string, buyYes bool) string {
	s := "NO"
	if buyYes {
		s = "YES"
	}
	return strings.ToUpper(strings.TrimSpace(platform)) + " " + s
}

func sideToken(platform, yesToken, noToken string, buyYes bool) string {
	if !strings.EqualFold(platform, appcommon.PlatformPolymarket) {
		return ""
	}
	if buyYes {
		return strings.TrimSpace(yesToken)
	}
	return strings.TrimSpace(noToken)
}

func orderStatus(fill float64, sendErr error) (status, errMsg string) {
	if sendErr != nil {
		msg := sendErr.Error()
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		return "ERROR", msg
	}
	if fill <= 0 {
		return "NO_FILL", ""
	}
	return "FILLED", ""
}

func strPtr(s string) *string { return &s }

func fixtureScopeForGrab(d PersistGrab, yesMoID, noMoID uint) watchgroup.FixtureScope {
	if d.ScopeForOutcomeIDs == nil {
		return watchgroup.FixtureScope{}
	}
	return d.ScopeForOutcomeIDs(yesMoID, noMoID)
}

// comboLegMarketOutcomeIDs 组合语义：YES 价腿 / NO 价腿对应的 market_outcome.id。
func comboLegMarketOutcomeIDs(a, b ExecSide, aBuysYes bool) (yesLegID, noLegID uint) {
	if aBuysYes {
		return a.OutcomeID, b.OutcomeID
	}
	return b.OutcomeID, a.OutcomeID
}

func snapshotTimes(d PersistGrab) (time.Time, int64) {
	if d.SnapshotAtMs != nil {
		if ms := d.SnapshotAtMs(); ms > 0 {
			t := time.UnixMilli(ms).UTC()
			return t, ms
		}
	}
	now := time.Now()
	if d.SnapshotAt != nil {
		now = d.SnapshotAt()
	}
	return now, now.UnixMilli()
}

func comboLegSides(a, b ExecSide, aBuysYes, bBuysYes bool) (yesLeg, noLeg ExecSide, yesBuy, noBuy bool) {
	if aBuysYes {
		return a, b, true, bBuysYes
	}
	return b, a, true, aBuysYes
}

func legAskPrices(_ GrabHit, a, b ExecSide, aYes, bYes bool, aPx, bPx float64) (yesPx, noPx float64) {
	if aYes {
		return aPx, bPx
	}
	return bPx, aPx
}

func arbQtyProfitFromBinaryPick(profitSpace float64, yesSide, noSide Side, yesBuy, noBuy bool) (decimal.Decimal, decimal.Decimal) {
	_, yesSz := SideTopAskSz(yesSide, yesBuy)
	_, noSz := SideTopAskSz(noSide, noBuy)
	minQ := yesSz
	if noSz > 0 && (minQ <= 0 || noSz < minQ) {
		minQ = noSz
	}
	if minQ <= 0 {
		return decimal.Zero, decimal.Zero
	}
	return decimal.NewFromFloat(minQ).Round(8), decimal.NewFromFloat(minQ * profitSpace).Round(8)
}

func legPersistFields(d PersistGrab, a, b ExecSide, aBuysYes, bBuysYes bool, snap1, snap2 watchgroup.LegPersistSnap) (yesInfo, noInfo, yesBooks, noBooks string) {
	yesLeg, noLeg, yesBuy, noBuy := comboLegSides(a, b, aBuysYes, bBuysYes)
	yesInfo, noInfo, yesBooks, noBooks = emptyLegJSON, emptyLegJSON, emptyLegJSON, emptyLegJSON
	if d.YesLegInfo != nil {
		yesInfo = d.YesLegInfo(yesLeg.Index, yesBuy)
	}
	if d.NoLegInfo != nil {
		noInfo = d.NoLegInfo(noLeg.Index, noBuy)
	}
	if strings.TrimSpace(snap1.OrderBooks) != "" {
		yesBooks = snap1.OrderBooks
	} else if d.LegOrderBooks != nil {
		yesBooks = d.LegOrderBooks(yesLeg.Index, yesBuy)
	}
	if strings.TrimSpace(snap2.OrderBooks) != "" {
		noBooks = snap2.OrderBooks
	} else if d.LegOrderBooks != nil {
		noBooks = d.LegOrderBooks(noLeg.Index, noBuy)
	}
	if strings.TrimSpace(yesInfo) == "" {
		yesInfo = emptyLegJSON
	}
	if strings.TrimSpace(noInfo) == "" {
		noInfo = emptyLegJSON
	}
	if strings.TrimSpace(yesBooks) == "" {
		yesBooks = emptyLegJSON
	}
	if strings.TrimSpace(noBooks) == "" {
		noBooks = emptyLegJSON
	}
	return yesInfo, noInfo, yesBooks, noBooks
}

func saveBacktestGrabHit(ctx context.Context, d PersistGrab, mapKey string, hit GrabHit) {
	a, b := hit.Pick.Yes, hit.Pick.No
	aYes, bYes := hit.Pick.YesBuyYes, hit.Pick.NoBuyYes
	qty, maxCost := hit.Pick.Qty, hit.MaxCost
	aPx, bPx := SidePx(a.Side, aYes), SidePx(b.Side, bYes)
	yesPx, noPx := legAskPrices(hit, a, b, aYes, bYes, aPx, bPx)
	now, snapMs := snapshotTimes(d)
	yesMoID, noMoID := comboLegMarketOutcomeIDs(a, b, aYes)
	scope := fixtureScopeForGrab(d, yesMoID, noMoID)
	yesInfo, noInfo, yesBooks, noBooks := legPersistFields(d, a, b, aYes, bYes, hit.Leg1, hit.Leg2)
	perShare := hit.Pick.Per
	if !persistableBinarySnap(perShare, maxCost) {
		return
	}
	profitSpace := 1 - perShare
	yesLeg, noLeg, yesBuy, noBuy := comboLegSides(a, b, aYes, bYes)
	arbQty, arbProfit := arbQtyProfitFromBinaryPick(profitSpace, yesLeg.Side, noLeg.Side, yesBuy, noBuy)
	yes := decimal.NewFromFloat(yesPx).Round(8)
	no := decimal.NewFromFloat(noPx).Round(8)
	row := &persistententity.BacktestArbBinaryOpportunity{
		BacktestTaskID: d.BacktestTaskID,
		OppKind:        "BINARY_YES_NO", MapKey: strPtr(mapKey),
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		Leg1Info: yesInfo, Leg1OrderBooks: yesBooks, Leg1BestAsk: &yes,
		Leg2Info: noInfo, Leg2OrderBooks: noBooks, Leg2BestAsk: &no,
		Leg3Info: emptyLegJSON, Leg3OrderBooks: emptyLegJSON,
		TotalCost: decimal.NewFromFloat(perShare).Round(8), ProfitSpace: decimal.NewFromFloat(profitSpace).Round(8),
		ArbQty: arbQty, ArbProfit: arbProfit,
		ThresholdKey: thresholdKeyMaxCost, ThresholdValue: decimal.NewFromFloat(maxCost).Round(8),
		BookUpdatedAt: &now, SnapshotAt: now, SnapshotAtMs: snapMs,
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := d.BacktestOppRepo.BatchCreate(runCtx, nil, []*persistententity.BacktestArbBinaryOpportunity{row}); err != nil {
		notification.SendErrorLog(runCtx, "backtest-OppCreateFailed", zap.Error(err))
		return
	}
	if d.BacktestOrderRepo != nil && row.ID > 0 {
		yesFill, noFill := hit.YesFill, hit.NoFill
		if yesFill <= 0 {
			yesFill = qty
		}
		if noFill <= 0 {
			noFill = qty
		}
		orders := []*persistententity.BacktestOrder{
			backtestOrderForSide(strategyTypeYesNo, d.BacktestTaskID, mapKey, row.ID, now, 1, a, aYes, qty, aPx, yesFill, nil),
			backtestOrderForSide(strategyTypeYesNo, d.BacktestTaskID, mapKey, row.ID, now, 2, b, bYes, qty, bPx, noFill, nil),
		}
		if err := d.BacktestOrderRepo.BatchCreate(runCtx, nil, orders); err != nil {
			notification.SendErrorLog(runCtx, "backtest-OrderCreateFailed", zap.Error(err))
		}
	}
}

// persistableBinarySnap TopHit 的 pick.Per 仍满足阈值（与落库 total_cost 同源）。
func persistableBinarySnap(perShare, maxCost float64) bool {
	if 1-perShare <= 0 {
		return false
	}
	if maxCost > 0 && perShare >= maxCost {
		return false
	}
	return true
}

// ArbQtyProfitFromSnaps 各腿卖一深度最小值及 arb_qty * profit_space（与 leg_*_best_ask 同源快照）。
func ArbQtyProfitFromSnaps(profitSpace float64, legs ...watchgroup.LegPersistSnap) (decimal.Decimal, decimal.Decimal) {
	qty := watchgroup.MinLegTopAskQty(legs...)
	return decimal.NewFromFloat(qty).Round(8), decimal.NewFromFloat(qty * profitSpace).Round(8)
}
