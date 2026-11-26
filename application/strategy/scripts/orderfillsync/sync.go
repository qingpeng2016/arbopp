package orderfillsync

import (
	"context"
	"math"
	"runtime/debug"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const TaskName = "order_fill_sync"

const defaultBatch = 100

var terminalStatuses = map[string]struct{}{
	"FILLED": {}, "PARTIAL": {}, "NO_FILL": {}, "ERROR": {},
}

// Sync 按 orders.platform_order_id 回查各平台成交，补全 trade_qty / trade_price / trade_amount / status / fee；
// 同一 arb_opp_id 全部终态后回写 arb_binary_opportunity.arb_profit_amount。
type Sync struct {
	orderRepo   persistentrepo.ArbBinaryOrderRepo
	oppRepo     persistentrepo.ArbBinaryOpportunityRepo
	accountRepo persistentrepo.AccountRepo
	poly        httprepo.PolymarketRepo
	predict     httprepo.PredictRepo
	sxbet       httprepo.SxbetRepo
}

func NewSync(
	orderRepo persistentrepo.ArbBinaryOrderRepo,
	oppRepo persistentrepo.ArbBinaryOpportunityRepo,
	accountRepo persistentrepo.AccountRepo,
	poly httprepo.PolymarketRepo,
	predict httprepo.PredictRepo,
	sxbet httprepo.SxbetRepo,
) *Sync {
	return &Sync{
		orderRepo: orderRepo, oppRepo: oppRepo, accountRepo: accountRepo,
		poly: poly, predict: predict, sxbet: sxbet,
	}
}

func (s *Sync) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "order_fill_sync-Panic",
				zap.Any("panic", r), zap.String("stack", string(debug.Stack())))
		}
	}()
	if s == nil || s.orderRepo == nil {
		notification.SendErrorLog(ctx, "order_fill_sync-skipped", zap.String("reason", "nil orderRepo"))
		return
	}
	start := time.Now()
	creds := s.loadCreds(ctx)
	list, err := s.orderRepo.ListForFillSync(ctx, defaultBatch, true)
	if err != nil {
		notification.SendErrorLog(ctx, "order_fill_sync-list", zap.Error(err))
		return
	}
	var okN, errN, profitN, skipN int
	touchedOpp := make(map[uint]struct{})
	for _, row := range list {
		if strings.TrimSpace(row.PlatformOrderID) == "" {
			if oid := extractOrderIDFromText(row.ErrMsg); oid != "" {
				row.PlatformOrderID = oid
			} else {
				skipN++
				continue
			}
		}
		fill, fee, tradePx, exchStatus, qerr := s.queryFill(ctx, row, creds)
		if qerr != nil {
			errN++
			logger.InfoZ(ctx, "order_fill_sync-query-fail",
				zap.Uint("id", row.ID), zap.String("platform", row.Platform),
				zap.String("oid", row.PlatformOrderID), zap.Error(qerr))
			continue
		}
		qty := row.Qty.InexactFloat64()
		prev := row.TradeQty.InexactFloat64()
		prevFee := row.Fee.InexactFloat64()
		prevPx := row.TradePrice.InexactFloat64()
		// 本轮查不到 fee/均价时保留库内已有值，避免把已回补的 fee 冲成 0。
		if !(fee > 0) {
			fee = prevFee
		}
		if !(tradePx > 0) {
			tradePx = prevPx
		}
		fill, status, apply := resolveFillUpdate(prev, fill, qty, exchStatus)
		if !apply {
			// 成交状态未定稿时，仍允许补写 fee / trade_price（常见：FILLED 但 amountFilled 尚在结算）。
			if prev > 0 && (fee > prevFee+1e-12 || (tradePx > 0 && !(prevPx > 0))) {
				st := strings.TrimSpace(row.Status)
				if st == "" {
					st = statusFromFill(prev, qty)
				}
				if !(tradePx > 0) {
					tradePx = row.Price.InexactFloat64()
				}
				if err := s.orderRepo.UpdateFill(ctx, row.ID, prev, fee, tradePx, st, ""); err != nil {
					errN++
					notification.SendErrorLog(ctx, "order_fill_sync-update-meta",
						zap.Uint("id", row.ID), zap.Error(err))
				} else {
					okN++
					if row.ArbOppID > 0 {
						touchedOpp[row.ArbOppID] = struct{}{}
					}
				}
				continue
			}
			skipN++
			logger.InfoZ(ctx, "order_fill_sync-pending",
				zap.Uint("id", row.ID), zap.String("platform", row.Platform),
				zap.String("oid", row.PlatformOrderID), zap.String("exch_status", exchStatus),
				zap.Float64("fill", fill), zap.Float64("prev_fill", prev),
				zap.Float64("fee", fee))
			continue
		}
		if fill > 0 && !(tradePx > 0) {
			tradePx = row.Price.InexactFloat64() // 无平台均价时回退下单限价
		}
		if fill <= 0 {
			tradePx = 0
		}
		if err := s.orderRepo.UpdateFill(ctx, row.ID, fill, fee, tradePx, status, ""); err != nil {
			errN++
			notification.SendErrorLog(ctx, "order_fill_sync-update",
				zap.Uint("id", row.ID), zap.Error(err))
			continue
		}
		logger.InfoZ(ctx, "order_fill_sync-updated",
			zap.Uint("id", row.ID), zap.String("platform", row.Platform),
			zap.Float64("trade_qty", fill), zap.Float64("trade_price", tradePx),
			zap.Float64("fee", fee), zap.String("status", status))
		okN++
		if row.ArbOppID > 0 {
			touchedOpp[row.ArbOppID] = struct{}{}
		}
	}
	for oppID := range touchedOpp {
		if s.maybeWriteOppProfit(ctx, oppID) {
			profitN++
		}
	}
	logger.InfoZ(ctx, "order_fill_sync-done",
		zap.Int("ok", okN), zap.Int("err", errN), zap.Int("skip", skipN), zap.Int("profit", profitN),
		zap.Int("batch", len(list)), zap.Duration("elapsed", time.Since(start)))
}

func (s *Sync) maybeWriteOppProfit(ctx context.Context, arbOppID uint) bool {
	if s.oppRepo == nil || arbOppID == 0 {
		return false
	}
	orders, err := s.orderRepo.ListByArbOppID(ctx, arbOppID)
	if err != nil {
		notification.SendErrorLog(ctx, "order_fill_sync-list-opp",
			zap.Uint("arb_opp_id", arbOppID), zap.Error(err))
		return false
	}
	if len(orders) == 0 || !allOrdersTerminal(orders) {
		return false
	}
	profit, ok := calcArbProfitAmount(orders)
	if !ok {
		return false
	}
	if err := s.oppRepo.UpdateArbProfitAmount(ctx, arbOppID, profit); err != nil {
		notification.SendErrorLog(ctx, "order_fill_sync-opp-profit",
			zap.Uint("arb_opp_id", arbOppID), zap.Error(err))
		return false
	}
	logger.InfoZ(ctx, "order_fill_sync-opp-profit",
		zap.Uint("arb_opp_id", arbOppID), zap.Float64("arb_profit_amount", profit))
	return true
}

func allOrdersTerminal(orders []*persistententity.ArbBinaryOrder) bool {
	for _, o := range orders {
		st := strings.ToUpper(strings.TrimSpace(o.Status))
		if _, ok := terminalStatuses[st]; !ok {
			return false
		}
	}
	return true
}

// calcArbProfitAmount：total_sum_amount = Σ(trade_amount 或 trade_price*trade_qty 或 price*trade_qty)；
// total_sum_min_qty = min(各平台 Σtrade_qty)；利润 = min_qty - amount。
func calcArbProfitAmount(orders []*persistententity.ArbBinaryOrder) (float64, bool) {
	if len(orders) == 0 {
		return 0, false
	}
	var totalSumAmount float64
	byPlatform := make(map[string]float64)
	for _, o := range orders {
		fill := o.TradeQty.InexactFloat64()
		if fill < 0 {
			fill = 0
		}
		amount := o.TradeAmount.InexactFloat64()
		if amount <= 0 {
			tp := o.TradePrice.InexactFloat64()
			if tp > 0 && fill > 0 {
				amount = tp * fill
			} else {
				px := o.Price.InexactFloat64()
				if px < 0 {
					px = 0
				}
				amount = px * fill
			}
		}
		totalSumAmount += amount
		plat := strings.ToUpper(strings.TrimSpace(o.Platform))
		if plat == "" {
			plat = "_"
		}
		byPlatform[plat] += fill
	}
	if len(byPlatform) == 0 {
		return 0, false
	}
	totalSumMinQty := math.Inf(1)
	for _, q := range byPlatform {
		if q < totalSumMinQty {
			totalSumMinQty = q
		}
	}
	if math.IsInf(totalSumMinQty, 1) {
		return 0, false
	}
	profit := totalSumMinQty - totalSumAmount
	if math.IsNaN(profit) || math.IsInf(profit, 0) {
		return 0, false
	}
	return profit, true
}

type activeCreds struct {
	poly    httpentity.PolymarketAccount
	polyOK  bool
	predict httpentity.PredictAccount
	predOK  bool
	sxbet   httpentity.SxbetAccount
	sxOK    bool
}

func (s *Sync) loadCreds(ctx context.Context) activeCreds {
	var c activeCreds
	if s.accountRepo == nil {
		return c
	}
	_, list, err := s.accountRepo.List(ctx, nil, map[string]interface{}{"status = ?": "ACTIVE"}, "id asc", "")
	if err != nil {
		return c
	}
	for i := range list {
		a := list[i]
		switch strings.ToUpper(strings.TrimSpace(a.Platform)) {
		case appcommon.PlatformPolymarket:
			if !c.polyOK {
				if acc, ok := polyAccount(a); ok {
					c.poly, c.polyOK = acc, true
				}
			}
		case appcommon.PlatformPredict:
			if !c.predOK {
				if acc, ok := predictAccount(a); ok {
					c.predict, c.predOK = acc, true
				}
			}
		case appcommon.PlatformSxbet:
			if !c.sxOK {
				if acc, ok := sxbetAccount(a); ok {
					c.sxbet, c.sxOK = acc, true
				}
			}
		}
	}
	return c
}

func (s *Sync) queryFill(ctx context.Context, row *persistententity.ArbBinaryOrder, creds activeCreds) (fill, fee, tradePrice float64, status string, err error) {
	oid := strings.TrimSpace(row.PlatformOrderID)
	platform := strings.ToUpper(strings.TrimSpace(row.Platform))
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	switch platform {
	case appcommon.PlatformPolymarket:
		if s.poly == nil || !creds.polyOK {
			return 0, 0, 0, "", errUnavailable("polymarket")
		}
		detail, err := s.poly.GetOrderById(ctx, httpentity.GetOrderReq{OrderID: oid}, creds.poly)
		if err != nil {
			return 0, 0, 0, "", err
		}
		fill = book.ParseNum(detail.Order.SizeMatched)
		trades := s.polyTradesForOrder(ctx, creds.poly, oid, row.TokenID)
		// size_matched 为空/0 时用 trades 按 taker_order_id 汇总（官方也建议如此兜底）
		if fill <= 0 {
			for _, t := range trades {
				fill += book.ParseNum(t.Size)
			}
			if math.IsNaN(fill) || math.IsInf(fill, 0) || fill < 0 {
				fill = 0
			}
		}
		fee = s.polyFeeFromTrades(trades)
		tradePrice = polyVWAP(trades)
		if !(tradePrice > 0) {
			tradePrice = book.ParseNum(detail.Order.Price)
		}
		st := strings.TrimSpace(string(detail.Order.Status))
		logger.InfoZ(ctx, "order_fill_sync-poly-query",
			zap.Uint("id", row.ID), zap.String("oid", oid),
			zap.String("exch_status", st), zap.Float64("size_matched", book.ParseNum(detail.Order.SizeMatched)),
			zap.Float64("fill", fill), zap.Float64("trade_price", tradePrice),
			zap.Float64("fee", fee), zap.Int("trades", len(trades)))
		return fill, fee, tradePrice, st, nil

	case appcommon.PlatformPredict:
		if s.predict == nil || !creds.predOK {
			return 0, 0, 0, "", errUnavailable("predict")
		}
		d, err := s.predict.GetOrderByHash(ctx, creds.predict, oid)
		if err != nil {
			return 0, 0, 0, "", err
		}
		logger.InfoZ(ctx, "order_fill_sync-predict-query",
			zap.Uint("id", row.ID), zap.String("oid", oid),
			zap.String("exch_status", d.Status), zap.Float64("amount_filled", d.AmountFilled),
			zap.Float64("amount", d.Amount), zap.Float64("avg_price", d.AvgPrice),
			zap.Float64("fee", d.FeeUSDT))
		return d.AmountFilled, d.FeeUSDT, d.AvgPrice, d.Status, nil

	case appcommon.PlatformSxbet:
		if s.sxbet == nil || !creds.sxOK {
			return 0, 0, 0, "", errUnavailable("sxbet")
		}
		// fills-v3 当前无成交价字段；均价由上层回退下单限价。
		if f, err := s.sxbet.GetFillsByOrderID(ctx, creds.sxbet, oid); err == nil && f != nil {
			if f.FillQty > 0 || f.FeeUSDT > 0 {
				return f.FillQty, f.FeeUSDT, 0, f.Status, nil
			}
		}
		o, err := s.sxbet.GetOrderByID(ctx, creds.sxbet, oid)
		if err != nil {
			return 0, 0, 0, "", err
		}
		return o.FillQty, o.FeeUSDT, 0, o.Status, nil

	default:
		return 0, 0, 0, "", errUnavailable(platform)
	}
}

func (s *Sync) polyFee(ctx context.Context, acc httpentity.PolymarketAccount, orderID, tokenID string) float64 {
	return s.polyFeeFromTrades(s.polyTradesForOrder(ctx, acc, orderID, tokenID))
}

func (s *Sync) polyFeeFromTrades(tr []httpentity.Trade) float64 {
	var fee float64
	for _, t := range tr {
		sz := book.ParseNum(t.Size)
		px := book.ParseNum(t.Price)
		bps := book.ParseNum(t.FeeRateBps)
		if sz > 0 && px > 0 && bps > 0 {
			fee += sz * px * bps / 10000
		}
	}
	if math.IsNaN(fee) || math.IsInf(fee, 0) {
		return 0
	}
	return fee
}

func polyVWAP(tr []httpentity.Trade) float64 {
	var notional, qty float64
	for _, t := range tr {
		sz := book.ParseNum(t.Size)
		px := book.ParseNum(t.Price)
		if sz > 0 && px > 0 {
			notional += sz * px
			qty += sz
		}
	}
	if qty <= 0 {
		return 0
	}
	avg := notional / qty
	if avg <= 0 || avg >= 1 || math.IsNaN(avg) || math.IsInf(avg, 0) {
		return 0
	}
	return avg
}

func (s *Sync) polyTradesForOrder(ctx context.Context, acc httpentity.PolymarketAccount, orderID, tokenID string) []httpentity.Trade {
	if s.poly == nil || strings.TrimSpace(orderID) == "" {
		return nil
	}
	req := httpentity.GetTradesReq{Limit: 100}
	if strings.TrimSpace(tokenID) != "" {
		req.TokenID = tokenID
	}
	tr, err := s.poly.GetTrades(ctx, req, acc)
	if err != nil || tr == nil {
		return nil
	}
	out := make([]httpentity.Trade, 0)
	for _, t := range tr.Trades {
		if strings.EqualFold(t.TakerOrderID, orderID) {
			out = append(out, t)
		}
	}
	return out
}

func statusFromFill(fill, qty float64) string {
	if fill <= 0 {
		return "NO_FILL"
	}
	if qty > 0 && fill+1e-8 < qty {
		return "PARTIAL"
	}
	return "FILLED"
}

// resolveFillUpdate 结合交易所状态与成交量；未结算/仍挂单时不落库，避免把真实成交冲成 NO_FILL。
// apply=false：本轮跳过更新（保持原行，下次再查）。
func resolveFillUpdate(prevFill, fill, qty float64, exchangeStatus string) (outFill float64, status string, apply bool) {
	st := strings.ToUpper(strings.TrimSpace(exchangeStatus))
	switch st {
	case "OPEN", "LIVE", "PENDING", "ACCEPTED", "CREATE":
		if fill > 0 {
			return fill, statusFromFill(fill, qty), true
		}
		// 仍挂着且尚未成交：不写成 NO_FILL，留给下次回查。
		return prevFill, "PENDING", false
	case "FILLED", "MATCHED":
		if fill <= 0 {
			// Predict：FILLED 后 amountFilled 可能尚在链上结算中。
			if prevFill > 0 {
				return prevFill, "FILLED", false
			}
			return 0, "PENDING", false
		}
		return fill, statusFromFill(fill, qty), true
	case "CANCELLED", "CANCELED", "CANCELED_MARKET_RESOLVED", "EXPIRED", "INVALIDATED":
		// 市价单未成交取消 → NO_FILL；有成交再取消剩余 → PARTIAL/FILLED。
		return fill, statusFromFill(fill, qty), true
	case "FAILED", "CREATE_FAILED", "REJECTED", "INVALID":
		if fill > 0 {
			return fill, statusFromFill(fill, qty), true
		}
		return 0, "ERROR", true
	default:
		// 未知状态：有成交量就写；0 成交且库里已有成交则不冲掉。
		if fill > 0 {
			return fill, statusFromFill(fill, qty), true
		}
		if prevFill > 0 {
			return prevFill, statusFromFill(prevFill, qty), false
		}
		return 0, statusFromFill(0, qty), true
	}
}

func extractOrderIDFromText(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	for _, key := range []string{`"orderID"`, `"orderId"`} {
		i := strings.Index(msg, key)
		if i < 0 {
			continue
		}
		rest := msg[i:]
		colon := strings.Index(rest, ":")
		if colon < 0 {
			continue
		}
		rest = strings.TrimSpace(rest[colon+1:])
		if !strings.HasPrefix(rest, `"`) {
			continue
		}
		rest = rest[1:]
		end := strings.Index(rest, `"`)
		if end <= 0 {
			continue
		}
		oid := strings.TrimSpace(rest[:end])
		if oid != "" && !strings.EqualFold(oid, "null") {
			return oid
		}
	}
	return ""
}

func errUnavailable(platform string) error {
	return &syncErr{msg: platform + " creds/client unavailable"}
}

type syncErr struct{ msg string }

func (e *syncErr) Error() string { return e.msg }

func polyAccount(a persistententity.Account) (httpentity.PolymarketAccount, bool) {
	sig := 0
	if a.SignatureType != nil {
		sig = *a.SignatureType
	}
	acc := httpentity.PolymarketAccount{
		APIKey: deref(a.APIKey), APISecret: deref(a.APISecretEnc), APIPassphrase: deref(a.APIPassphraseEnc),
		EOAAddress: deref(a.EoaAddress), ProxyAddress: deref(a.ProxyAddress), PrivateKey: deref(a.PrivateKeyEnc),
		SignatureType: sig,
	}
	ok := acc.APIKey != "" && acc.APISecret != "" && acc.APIPassphrase != "" && acc.PrivateKey != "" &&
		(acc.ProxyAddress != "" || acc.EOAAddress != "")
	return acc, ok
}

func predictAccount(a persistententity.Account) (httpentity.PredictAccount, bool) {
	acc := httpentity.PredictAccount{
		APIKey: deref(a.APIKey), PrivateKey: deref(a.PrivateKeyEnc),
		PredictAccount: deref(a.ProxyAddress), EOAAddress: deref(a.EoaAddress),
	}
	return acc, acc.APIKey != "" && acc.PrivateKey != ""
}

func sxbetAccount(a persistententity.Account) (httpentity.SxbetAccount, bool) {
	acc := httpentity.SxbetAccount{
		APIKey: deref(a.APIKey), PrivateKey: deref(a.PrivateKeyEnc),
		EOAAddress: deref(a.EoaAddress), ProxyAddress: deref(a.ProxyAddress),
	}
	return acc, acc.APIKey != "" && acc.PrivateKey != "" && acc.ProxyAddress != ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
