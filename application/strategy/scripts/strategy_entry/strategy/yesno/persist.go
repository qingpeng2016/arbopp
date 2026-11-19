package yesno

import (
	"context"
	"encoding/json"
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
	thresholdKeyMaxCost  = "binary_yes_no_max_cost"
	strategyTypeYesNo    = "BINARY_YES_NO"
	emptyOrderBooksJSON = `{"outcomes":[]}`
)

// PersistGrab 抢单落库依赖。
type PersistGrab struct {
	ScopeForOutcomeIDs func(yesMoID, noMoID uint) watchgroup.FixtureScope
	OppRepo            persistentrepo.ArbBinaryOpportunityRepo
	OrderRepo persistentrepo.ArbBinaryOrderRepo
	BookRepo  persistentrepo.OrderBookRepo
	BooksJSON func(a, b ExecSide) string
	BookRows  func(a, b ExecSide) []*persistententity.OrderBook

	BacktestTaskID    uint
	BacktestOppRepo   persistentrepo.BacktestArbBinaryOpportunityRepo
	BacktestOrderRepo persistentrepo.ArbBinaryOrderRepo
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
	qty, per, maxCost := hit.Pick.Qty, hit.Pick.Per, hit.MaxCost
	aPx, bPx := SidePx(a.Side, aYes), SidePx(b.Side, bYes)
	aSide, bSide := "NO", "NO"
	if aYes {
		aSide = "YES"
	}
	if bYes {
		bSide = "YES"
	}
	yesPx, noPx := aPx, bPx
	if !aYes {
		yesPx, noPx = bPx, aPx
	}
	now, snapMs := snapshotTimes(d)
	yesMoID, noMoID := comboLegMarketOutcomeIDs(a, b, aYes)
	scope := fixtureScopeForGrab(d, yesMoID, noMoID)
	outcomes, _ := json.Marshal([]map[string]any{
		{"platform": a.Platform, "market_id": a.MarketID, "market_outcome_id": a.OutcomeID, "side": aSide, "best_ask": aPx, "size": qty, "fill": hit.YesFill},
		{"platform": b.Platform, "market_id": b.MarketID, "market_outcome_id": b.OutcomeID, "side": bSide, "best_ask": bPx, "size": qty, "fill": hit.NoFill},
	})
	booksJSON := orderBooksJSONForHit(d, a, b)
	yes := decimal.NewFromFloat(yesPx).Round(8)
	no := decimal.NewFromFloat(noPx).Round(8)
	row := &persistententity.ArbBinaryOpportunity{
		OppKind: "BINARY_YES_NO", MapKey: strPtr(mapKey),
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		YesMarketOutcomeID: yesMoID, NoMarketOutcomeID: noMoID,
		OutcomesJSON: string(outcomes), OrderBooksJSON: booksJSON,
		YesBestAsk: &yes, NoBestAsk: &no,
		TotalCost: decimal.NewFromFloat(per).Round(8), ProfitSpace: decimal.NewFromFloat(1 - per).Round(8),
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
			if err := d.BookRepo.BatchUpsert(runCtx, nil, rows); err != nil {
				notification.SendErrorLog(runCtx, "strategy_entry-BookUpsertFailed", zap.Error(err), zap.Int("rows", len(rows)))
			}
		}
	}
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

// orderBooksJSONForHit arb_binary_opportunity / backtest_arb_binary_opportunity 共用：命中时两腿内存盘口（至多 8 档）。
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

func orderBooksJSONForHit(d PersistGrab, a, b ExecSide) string {
	if d.BooksJSON == nil {
		return emptyOrderBooksJSON
	}
	s := strings.TrimSpace(d.BooksJSON(a, b))
	if s == "" {
		return emptyOrderBooksJSON
	}
	return s
}

func saveBacktestGrabHit(ctx context.Context, d PersistGrab, mapKey string, hit GrabHit) {
	a, b := hit.Pick.Yes, hit.Pick.No
	aYes, bYes := hit.Pick.YesBuyYes, hit.Pick.NoBuyYes
	qty, per, maxCost := hit.Pick.Qty, hit.Pick.Per, hit.MaxCost
	aPx, bPx := SidePx(a.Side, aYes), SidePx(b.Side, bYes)
	aSide, bSide := "NO", "NO"
	if aYes {
		aSide = "YES"
	}
	if bYes {
		bSide = "YES"
	}
	yesPx, noPx := aPx, bPx
	if !aYes {
		yesPx, noPx = bPx, aPx
	}
	now, snapMs := snapshotTimes(d)
	yesMoID, noMoID := comboLegMarketOutcomeIDs(a, b, aYes)
	scope := fixtureScopeForGrab(d, yesMoID, noMoID)
	outcomes, _ := json.Marshal([]map[string]any{
		{"platform": a.Platform, "market_id": a.MarketID, "market_outcome_id": a.OutcomeID, "side": aSide, "best_ask": aPx, "size": qty, "fill": hit.YesFill},
		{"platform": b.Platform, "market_id": b.MarketID, "market_outcome_id": b.OutcomeID, "side": bSide, "best_ask": bPx, "size": qty, "fill": hit.NoFill},
	})
	booksJSON := orderBooksJSONForHit(d, a, b)
	yes := decimal.NewFromFloat(yesPx).Round(8)
	no := decimal.NewFromFloat(noPx).Round(8)
	row := &persistententity.BacktestArbBinaryOpportunity{
		BacktestTaskID: d.BacktestTaskID,
		OppKind:        "BINARY_YES_NO", MapKey: strPtr(mapKey),
		Category: scope.Category, SubCategory: scope.SubCategory,
		Competition: scope.Competition, CompetitionEn: scope.CompetitionEn,
		YesMarketOutcomeID: yesMoID, NoMarketOutcomeID: noMoID,
		OutcomesJSON: string(outcomes), OrderBooksJSON: booksJSON,
		YesBestAsk: &yes, NoBestAsk: &no,
		TotalCost: decimal.NewFromFloat(per).Round(8), ProfitSpace: decimal.NewFromFloat(1-per).Round(8),
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
		orders := []*persistententity.ArbBinaryOrder{
			orderForSide(strategyTypeYesNo, d.BacktestTaskID, mapKey, row.ID, now, 1, a, aYes, qty, aPx, yesFill, nil),
			orderForSide(strategyTypeYesNo, d.BacktestTaskID, mapKey, row.ID, now, 2, b, bYes, qty, bPx, noFill, nil),
		}
		if err := d.BacktestOrderRepo.BatchCreate(runCtx, nil, orders); err != nil {
			notification.SendErrorLog(runCtx, "backtest-OrderCreateFailed", zap.Error(err))
		}
	}
}
