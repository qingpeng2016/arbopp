package yesno

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/noderelay"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
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
	OrderRepo          persistentrepo.ArbBinaryOrderRepo
	YesLegInfo         func(outcomeIndex int, buyYes bool) string
	NoLegInfo          func(outcomeIndex int, buyYes bool) string
	LegOrderBooks      func(outcomeIndex int, buyYes bool) string
	// OnOrdersWritten 实盘写 orders 成功后回调（刷新内存未终态集合）。
	OnOrdersWritten func(mapKey string)

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
		if hit.SkipOrderReason != "" {
			mode = hit.SkipOrderReason
		}
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

// NotifyOrderResult 下单结果专用通知（order_webhook_url）。
// 各平台成交份额 (max-min)/max ≤ FillShareMismatchPct → 套利成功；否则未套利成功。
func NotifyOrderResult(ctx context.Context, mapKey string, hit GrabHit) {
	a, b := hit.Pick.Yes, hit.Pick.No
	aPx, bPx := SidePx(a.Side, hit.Pick.YesBuyYes), SidePx(b.Side, hit.Pick.NoBuyYes)
	errA, errB := "", ""
	if hit.YesErr != nil {
		errA = hit.YesErr.Error()
	}
	if hit.NoErr != nil {
		errB = hit.NoErr.Error()
	}
	fields := []notification.FieldPair{
		{Key: "map_key", Value: mapKey},
		{Key: "side_a", Value: sideLabel(a.Platform, hit.Pick.YesBuyYes)},
		{Key: "side_b", Value: sideLabel(b.Platform, hit.Pick.NoBuyYes)},
		{Key: "limit_a", Value: strconv.FormatFloat(aPx, 'f', -1, 64)},
		{Key: "limit_b", Value: strconv.FormatFloat(bPx, 'f', -1, 64)},
		{Key: "order_qty", Value: strconv.FormatFloat(hit.Pick.Qty, 'f', -1, 64)},
		{Key: "fill_a", Value: strconv.FormatFloat(hit.YesFill, 'f', -1, 64)},
		{Key: "fill_b", Value: strconv.FormatFloat(hit.NoFill, 'f', -1, 64)},
		{Key: "err_a", Value: errA},
		{Key: "err_b", Value: errB},
	}
	matched, ok := FillsAligned([]float64{hit.YesFill, hit.NoFill}, hit.FillShareMismatchPct)
	title := "strategy_entry 下单结果：未套利成功"
	if ok {
		costSum := aPx*hit.YesFill + bPx*hit.NoFill
		profit := matched - costSum
		title = "strategy_entry 下单结果：套利成功"
		fields = append(fields,
			notification.FieldPair{Key: "matched_qty", Value: strconv.FormatFloat(matched, 'f', -1, 64)},
			notification.FieldPair{Key: "cost_sum", Value: strconv.FormatFloat(costSum, 'f', -1, 64)},
			notification.FieldPair{Key: "arb_profit", Value: strconv.FormatFloat(profit, 'f', -1, 64)},
		)
	} else {
		// 下单瞬间尚无 trade_price 回补：trade_amount ≈ limit*fill
		est := EstProfitMaxQtyMinusCost(
			[]float64{hit.YesFill, hit.NoFill},
			[]float64{aPx * hit.YesFill, bPx * hit.NoFill},
			[]float64{hit.YesSend.Fee, hit.NoSend.Fee},
		)
		fields = append(fields,
			notification.FieldPair{Key: "note", Value: "各平台成交份额差异超出容忍或未成交，未形成完整套利"},
			notification.FieldPair{Key: "预计利润", Value: strconv.FormatFloat(est, 'f', -1, 64)},
		)
	}
	_ = notification.GetGlobalNotificationManager().SendOrderAlert(ctx, title, fields)
}

// EstProfitMaxQtyMinusCost 预计利润 = max(trade_qty) - Σtrade_amount - Σfee。
func EstProfitMaxQtyMinusCost(tradeQtys, tradeAmounts, fees []float64) float64 {
	var maxQ, sumAmt, sumFee float64
	for i, q := range tradeQtys {
		if q < 0 {
			q = 0
		}
		if q > maxQ {
			maxQ = q
		}
		a := 0.0
		if i < len(tradeAmounts) {
			a = tradeAmounts[i]
		}
		if a < 0 {
			a = 0
		}
		sumAmt += a
		f := 0.0
		if i < len(fees) {
			f = fees[i]
		}
		if f < 0 {
			f = 0
		}
		sumFee += f
	}
	return maxQ - sumAmt - sumFee
}

// FillsAligned 各腿成交份额是否对齐：均 >0 且 (max-min)/max ≤ mismatchPct。
// mismatchPct≤0 时要求严格相等（eps）。返回匹配份额 = min(fills)。
func FillsAligned(fills []float64, mismatchPct float64) (matched float64, ok bool) {
	const eps = 1e-6
	if len(fills) == 0 {
		return 0, false
	}
	minF, maxF := fills[0], fills[0]
	for _, f := range fills {
		if f <= eps {
			return 0, false
		}
		if f < minF {
			minF = f
		}
		if f > maxF {
			maxF = f
		}
	}
	if mismatchPct <= 0 {
		if maxF-minF > eps {
			return 0, false
		}
		return minF, true
	}
	if (maxF-minF)/maxF > mismatchPct+1e-12 {
		return 0, false
	}
	return minF, true
}

// SaveGrabHitAsync Fire 后异步落库，不占用 WS→下单热路径。
func SaveGrabHitAsync(d PersistGrab, mapKey string, hit GrabHit) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if !SaveGrabHit(ctx, d, mapKey, hit) {
			notification.SendErrorLog(ctx, "strategy_entry-SaveGrabHitAsyncFailed",
				zap.String("map_key", mapKey),
				zap.Bool("dry_run", hit.DryRun),
				zap.String("skip", hit.SkipOrderReason))
		}
	}()
}

// SaveInboundHedgeAsync 对端 TCP HEDGE 在本机 broker 成交后异步落库（单腿 + 关联 arb_id）。
func SaveInboundHedgeAsync(d PersistGrab, mapKey string, cmd noderelay.HedgeCmd, send watchgroup.SendResult, sendErr error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := saveInboundHedge(ctx, d, mapKey, cmd, send, sendErr); err != nil {
			notification.SendErrorLog(ctx, "strategy_entry-SaveInboundHedgeFailed",
				zap.String("map_key", mapKey),
				zap.String("arb_id", cmd.ArbID),
				zap.Error(err))
		}
	}()
}

func saveInboundHedge(ctx context.Context, d PersistGrab, mapKey string, cmd noderelay.HedgeCmd, send watchgroup.SendResult, sendErr error) error {
	if d.OppRepo == nil || d.OrderRepo == nil {
		return nil
	}
	now := time.Now().UTC()
	ext := ExtDataJSON(nil, map[string]any{
		"leg_role":   "hedge_recv",
		"arb_id":     strings.TrimSpace(cmd.ArbID),
		"node":       conf.GetRunNode(),
		"initiator":  "peer",
		"platform":   cmd.Platform,
		"outcome_id": cmd.OutcomeID,
	})
	row := &persistententity.ArbBinaryOpportunity{
		OppKind: "HEDGE_RECV", MapKey: strPtr(mapKey),
		Leg1Platform: watchgroup.LegPlatform(cmd.Platform),
		Leg1Info:     emptyLegJSON, Leg1OrderBooks: emptyLegJSON,
		Leg2Info: emptyLegJSON, Leg2OrderBooks: emptyLegJSON, Leg3Info: emptyLegJSON, Leg3OrderBooks: emptyLegJSON,
		TotalCost:     decimal.NewFromFloat(cmd.Px).Round(8),
		ProfitSpace:   decimal.Zero,
		ArbQty:        decimal.NewFromFloat(cmd.Qty).Round(8),
		ThresholdKey:  thresholdKeyMaxCost,
		BookUpdatedAt: &now, SnapshotAt: now, SnapshotAtMs: now.UnixMilli(),
		ExtData:       extDataPtr(ext),
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := d.OppRepo.BatchCreate(runCtx, nil, []*persistententity.ArbBinaryOpportunity{row}); err != nil {
		return err
	}
	status, errMsg := orderStatus(send.Fill, cmd.Qty, sendErr, send)
	side := "NO"
	if cmd.BuyYes {
		side = "YES"
	}
	ord := &persistententity.ArbBinaryOrder{
		StrategyType: strategyTypeYesNo, ArbOppID: row.ID, MapKey: mapKey, SideNo: 1,
		Platform:                strings.ToUpper(strings.TrimSpace(cmd.Platform)),
		PlatformMarketOutcomeID: strconv.FormatUint(uint64(cmd.OutcomeID), 10),
		ClientOrderID:             strings.TrimSpace(send.ClientOrderID),
		PlatformOrderID:           strings.TrimSpace(send.PlatformOrderID),
		Side: side, Action: "BUY",
		Price: decimal.NewFromFloat(cmd.Px).Round(8), Qty: decimal.NewFromFloat(cmd.Qty).Round(8),
		TradeQty: decimal.NewFromFloat(send.Fill).Round(8), Fee: decimal.NewFromFloat(send.Fee).Round(8),
		Status: status, ErrMsg: errMsg, SentAt: now,
	}
	if err := d.OrderRepo.BatchCreate(runCtx, nil, []*persistententity.ArbBinaryOrder{ord}); err != nil {
		return err
	}
	if d.OnOrdersWritten != nil {
		d.OnOrdersWritten(mapKey)
	}
	return nil
}

func SaveGrabHit(ctx context.Context, d PersistGrab, mapKey string, hit GrabHit) bool {
	if !hit.Pick.At.IsZero() && (!hit.Leg1.OK || !hit.Leg2.OK) {
		return false
	}
	if d.BacktestTaskID > 0 && d.BacktestOppRepo != nil {
		return saveBacktestGrabHit(ctx, d, mapKey, hit)
	}
	if d.OppRepo == nil {
		return false
	}
	a, b := hit.Pick.Yes, hit.Pick.No
	aYes, bYes := hit.Pick.YesBuyYes, hit.Pick.NoBuyYes
	qty, maxCost := hit.Pick.Qty, hit.MaxCost
	aPx, bPx := SidePx(a.Side, aYes), SidePx(b.Side, bYes)
	yesPx, noPx := legAskPrices(hit, a, b, aYes, bYes, aPx, bPx)
	now := hit.Pick.At
	if now.IsZero() {
		now, _ = snapshotTimes(d)
	}
	now = now.UTC()
	snapMs := now.UnixMilli()
	yesMoID, noMoID := comboLegMarketOutcomeIDs(a, b, aYes)
	scope := fixtureScopeForGrab(d, yesMoID, noMoID)
	yesInfo, noInfo, yesBooks, noBooks := legPersistFields(d, a, b, aYes, bYes, hit.Leg1, hit.Leg2)
	perShare := hit.Pick.Per
	if !persistableBinarySnap(perShare, maxCost) {
		return false
	}
	profitSpace := 1 - perShare
	yesLeg, noLeg, yesBuy, noBuy := comboLegSides(a, b, aYes, bYes)
	arbQty, arbProfit := binaryPickProfit(hit.Pick, profitSpace, yesLeg.Side, noLeg.Side, yesBuy, noBuy)
	yes := decimal.NewFromFloat(yesPx).Round(8)
	no := decimal.NewFromFloat(noPx).Round(8)
	row := &persistententity.ArbBinaryOpportunity{
		OppKind: "BINARY_YES_NO", MapKey: strPtr(mapKey),
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		Leg1Platform: watchgroup.LegPlatform(yesLeg.Platform),
		Leg1Info:     yesInfo, Leg1OrderBooks: yesBooks, Leg1BestAsk: &yes,
		Leg2Platform: watchgroup.LegPlatform(noLeg.Platform),
		Leg2Info:     noInfo, Leg2OrderBooks: noBooks, Leg2BestAsk: &no,
		Leg3Info: emptyLegJSON, Leg3OrderBooks: emptyLegJSON,
		TotalCost: decimal.NewFromFloat(perShare).Round(8), ProfitSpace: decimal.NewFromFloat(profitSpace).Round(8),
		ArbQty: arbQty, ArbProfit: arbProfit,
		ThresholdKey: thresholdKeyMaxCost, ThresholdValue: decimal.NewFromFloat(maxCost).Round(8),
		BookUpdatedAt: &now, SnapshotAt: now, SnapshotAtMs: snapMs,
		ExtData: extDataPtr(hit.ExtData),
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := d.OppRepo.BatchCreate(runCtx, nil, []*persistententity.ArbBinaryOpportunity{row}); err != nil {
		notification.SendErrorLog(runCtx, "strategy_entry-OppCreateFailed", zap.Error(err))
		return false
	}
	if !hit.DryRun && d.OrderRepo != nil && row.ID > 0 {
		orders := []*persistententity.ArbBinaryOrder{
			orderForSide(strategyTypeYesNo, 0, mapKey, row.ID, now, 1, a, aYes, qty, aPx, hit.YesFill, hit.YesErr, hit.YesSend),
			orderForSide(strategyTypeYesNo, 0, mapKey, row.ID, now, 2, b, bYes, qty, bPx, hit.NoFill, hit.NoErr, hit.NoSend),
		}
		if err := d.OrderRepo.BatchCreate(runCtx, nil, orders); err != nil {
			notification.SendErrorLog(runCtx, "strategy_entry-OrderCreateFailed", zap.Error(err))
		} else if d.OnOrdersWritten != nil {
			d.OnOrdersWritten(mapKey)
		}
	}
	return true
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
	status, errMsg := orderStatus(fill, qty, sendErr, watchgroup.SendResult{})
	return &persistententity.BacktestOrder{
		BacktestTaskID: backtestTaskID, StrategyType: strategyType,
		ArbOppID: oppID, MapKey: mapKey, SideNo: sideNo,
		Platform: s.Platform, PlatformMarketOutcomeID: s.MarketID,
		TokenID: sideToken(s.Platform, s.YesToken, s.NoToken, buyYes),
		Side:    side, Action: "BUY",
		Price: decimal.NewFromFloat(px).Round(8), Qty: decimal.NewFromFloat(qty).Round(8),
		FillQty: decimal.NewFromFloat(fill).Round(8), Status: status, ErrMsg: errMsg, SentAt: sentAt,
	}
}

// OrderForSide 实盘组合腿（写 orders）。
func OrderForSide(strategyType string, backtestTaskID uint, mapKey string, oppID uint, sentAt time.Time, sideNo uint8, s ExecSide, buyYes bool, qty, px, fill float64, sendErr error, send watchgroup.SendResult) *persistententity.ArbBinaryOrder {
	return orderForSide(strategyType, backtestTaskID, mapKey, oppID, sentAt, sideNo, s, buyYes, qty, px, fill, sendErr, send)
}

func orderForSide(strategyType string, backtestTaskID uint, mapKey string, oppID uint, sentAt time.Time, sideNo uint8, s ExecSide, buyYes bool, qty, px, fill float64, sendErr error, send watchgroup.SendResult) *persistententity.ArbBinaryOrder {
	side := "NO"
	if buyYes {
		side = "YES"
	}
	status, errMsg := orderStatus(fill, qty, sendErr, send)
	return &persistententity.ArbBinaryOrder{
		BacktestTaskID: backtestTaskID, StrategyType: strategyType,
		ArbOppID: oppID, MapKey: mapKey, SideNo: sideNo,
		Platform: s.Platform, PlatformMarketOutcomeID: s.MarketID,
		TokenID: sideToken(s.Platform, s.YesToken, s.NoToken, buyYes),
		ClientOrderID: send.ClientOrderID, PlatformOrderID: send.PlatformOrderID,
		Side: side, Action: "BUY",
		Price: decimal.NewFromFloat(px).Round(8), Qty: decimal.NewFromFloat(qty).Round(8),
		TradeQty: decimal.NewFromFloat(fill).Round(8), Fee: decimal.NewFromFloat(send.Fee).Round(8),
		Status: status, ErrMsg: errMsg, SentAt: sentAt,
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

func orderStatus(fill, qty float64, sendErr error, send watchgroup.SendResult) (status, errMsg string) {
	if sendErr != nil {
		msg := sendErr.Error()
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		return "ERROR", msg
	}
	if send.RoutedPeer {
		return "ROUTED_PEER", ""
	}
	if fill <= 0 {
		return "NO_FILL", ""
	}
	if qty > 0 && fill+1e-8 < qty {
		return "PARTIAL", ""
	}
	return "FILLED", ""
}

func strPtr(s string) *string { return &s }

func extDataPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

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

func binaryPickProfit(pick FilledPick, profitSpace float64, _ Side, _ Side, _, _ bool) (decimal.Decimal, decimal.Decimal) {
	q := arbQtyFromPick(pick)
	if q <= 0 {
		return decimal.Zero, decimal.Zero
	}
	return decimal.NewFromFloat(q).Round(8), decimal.NewFromFloat(q * profitSpace).Round(8)
}

// arbQtyFromPick 各腿卖一原始挂单量的最小值（多腿对齐可下单上限）。
func arbQtyFromPick(pick FilledPick) float64 {
	if pick.TopSizes[0] > 0 && pick.TopSizes[1] > 0 {
		return math.Min(pick.TopSizes[0], pick.TopSizes[1])
	}
	_, yesSz := SideTopAskSz(pick.Yes.Side, pick.YesBuyYes)
	_, noSz := SideTopAskSz(pick.No.Side, pick.NoBuyYes)
	if yesSz <= 0 || noSz <= 0 {
		return 0
	}
	return math.Min(yesSz, noSz)
}

func legPersistFields(d PersistGrab, a, b ExecSide, aBuysYes, bBuysYes bool, snap1, snap2 watchgroup.LegPersistSnap) (yesInfo, noInfo, yesBooks, noBooks string) {
	yesLeg, noLeg, yesBuy, noBuy := comboLegSides(a, b, aBuysYes, bBuysYes)
	if !aBuysYes {
		snap1, snap2 = snap2, snap1
	}
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

func saveBacktestGrabHit(ctx context.Context, d PersistGrab, mapKey string, hit GrabHit) bool {
	a, b := hit.Pick.Yes, hit.Pick.No
	aYes, bYes := hit.Pick.YesBuyYes, hit.Pick.NoBuyYes
	qty, maxCost := hit.Pick.Qty, hit.MaxCost
	aPx, bPx := SidePx(a.Side, aYes), SidePx(b.Side, bYes)
	yesPx, noPx := legAskPrices(hit, a, b, aYes, bYes, aPx, bPx)
	now := hit.Pick.At
	if now.IsZero() {
		now, _ = snapshotTimes(d)
	}
	now = now.UTC()
	snapMs := now.UnixMilli()
	yesMoID, noMoID := comboLegMarketOutcomeIDs(a, b, aYes)
	scope := fixtureScopeForGrab(d, yesMoID, noMoID)
	yesInfo, noInfo, yesBooks, noBooks := legPersistFields(d, a, b, aYes, bYes, hit.Leg1, hit.Leg2)
	perShare := hit.Pick.Per
	if !persistableBinarySnap(perShare, maxCost) {
		return false
	}
	profitSpace := 1 - perShare
	yesLeg, noLeg, yesBuy, noBuy := comboLegSides(a, b, aYes, bYes)
	arbQty, arbProfit := binaryPickProfit(hit.Pick, profitSpace, yesLeg.Side, noLeg.Side, yesBuy, noBuy)
	yes := decimal.NewFromFloat(yesPx).Round(8)
	no := decimal.NewFromFloat(noPx).Round(8)
	row := &persistententity.BacktestArbBinaryOpportunity{
		BacktestTaskID: d.BacktestTaskID,
		OppKind:        "BINARY_YES_NO", MapKey: strPtr(mapKey),
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		Leg1Platform: watchgroup.LegPlatform(yesLeg.Platform),
		Leg1Info:     yesInfo, Leg1OrderBooks: yesBooks, Leg1BestAsk: &yes,
		Leg2Platform: watchgroup.LegPlatform(noLeg.Platform),
		Leg2Info:     noInfo, Leg2OrderBooks: noBooks, Leg2BestAsk: &no,
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
		return false
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
	return true
}

// persistableBinarySnap TopHit 的 pick.Per 仍满足阈值（与落库 total_cost 同源）。
func persistableBinarySnap(perShare, maxCost float64) bool {
	// TEST: 不强制 profit_space>0（原 1-perShare<=0 为兑付硬顶）
	if maxCost > 0 && perShare >= maxCost {
		return false
	}
	return true
}

// ArbQtyProfitFromSnaps 回填脚本：各腿快照 TopSz 最小值（与落库 arb_qty 口径一致）。
func ArbQtyProfitFromSnaps(profitSpace float64, legs ...watchgroup.LegPersistSnap) (decimal.Decimal, decimal.Decimal) {
	qty := watchgroup.MinLegTopAskQty(legs...)
	return decimal.NewFromFloat(qty).Round(8), decimal.NewFromFloat(qty * profitSpace).Round(8)
}
