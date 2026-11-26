package broker

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

var errEmptyOrder = errors.New("empty order response")

// PolyCreds Polymarket 发单身份。
type PolyCreds struct {
	Account httpentity.PolymarketAccount
	OK      bool
}

// KalshiCreds Kalshi 发单身份。
type KalshiCreds struct {
	Account httpentity.KalshiAccount
	OK      bool
}

// PredictCreds Predict.fun 发单身份。
type PredictCreds struct {
	Account httpentity.PredictAccount
	OK      bool
}

// SxbetCreds SX Bet 发单身份。
type SxbetCreds struct {
	Account httpentity.SxbetAccount
	OK      bool
}

func newClientOrderID() string {
	return "se" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// SendPoly 向 Polymarket 发 LIMIT + FAK；下单后尽量 GetOrder 取 size_matched。
func SendPoly(poly httprepo.PolymarketRepo, cred PolyCreds, token string, price, qty float64, buy, neg bool) (watchgroup.SendResult, error) {
	out := watchgroup.SendResult{ClientOrderID: newClientOrderID()}
	if poly == nil || token == "" || qty <= 0 || !cred.OK {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req := httpentity.PlaceOrderReq{
		TokenID:    token,
		Side:       cryptoPredictionEnum.OrderSideBuy,
		OrderType:  cryptoPredictionEnum.PolymarketOrderTypeFAK,
		IsMarket:   false,
		NegRisk:    neg,
		LimitPrice: strconv.FormatFloat(price, 'f', 4, 64),
		LimitSize:  strconv.FormatFloat(qty, 'f', 4, 64),
	}
	if !buy {
		req.Side = cryptoPredictionEnum.OrderSideSell
		req.LimitPrice = "0.01"
	}
	resp, err := poly.PlaceOrder(ctx, req, cred.Account)
	if err != nil || resp == nil {
		if err == nil {
			err = errEmptyOrder
		}
		// FAK 400 常仍带 orderID，留给 order_fill_sync 回查 size_matched。
		if oid := orderIDFromErrMsg(err.Error()); oid != "" {
			out.PlatformOrderID = oid
		}
		return out, err
	}
	out.PlatformOrderID = strings.TrimSpace(resp.OrderID)
	fill := 0.0
	switch strings.ToLower(string(resp.Status)) {
	case "filled", "matched":
		fill = qty
	}
	// 下单响应无成交量：立刻查单拿 size_matched（部分成交也能拿到）。
	if out.PlatformOrderID != "" {
		if detail, err2 := poly.GetOrderById(ctx, httpentity.GetOrderReq{OrderID: out.PlatformOrderID}, cred.Account); err2 == nil && detail != nil {
			if m := book.ParseNum(detail.Order.SizeMatched); m > 0 {
				fill = m
			}
		}
	}
	out.Fill = fill
	return out, nil
}

// SendKalshi 向 Kalshi 发 IOC。
func SendKalshi(kalshi httprepo.KalshiRepo, cred KalshiCreds, oc *persistententity.MarketOutcome, yes bool, price, qty float64, sell bool) (watchgroup.SendResult, error) {
	out := watchgroup.SendResult{ClientOrderID: newClientOrderID()}
	if kalshi == nil || oc == nil || qty <= 0 || !cred.OK {
		return out, nil
	}
	ticker := strings.TrimSpace(oc.PlatformMarketOutcomeID)
	side := "bid"
	px := price
	if yes && sell {
		side = "ask"
		px = 0.01
	}
	if !yes && !sell {
		side = "ask"
		px = 1 - price
	}
	if !yes && sell {
		side = "bid"
		px = 0.99
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := kalshi.PlaceOrder(ctx, httpentity.KalshiPlaceOrderReq{
		Ticker:                  ticker,
		ClientOrderID:           out.ClientOrderID,
		Side:                    side,
		Count:                   strconv.FormatFloat(qty, 'f', 2, 64),
		Price:                   strconv.FormatFloat(px, 'f', 4, 64),
		TimeInForce:             "immediate_or_cancel",
		SelfTradePreventionType: "taker_at_cross",
	}, cred.Account)
	if err != nil || resp == nil {
		if err == nil {
			err = errEmptyOrder
		}
		return out, err
	}
	out.PlatformOrderID = strings.TrimSpace(resp.OrderID)
	if cid := strings.TrimSpace(resp.ClientOrderID); cid != "" {
		out.ClientOrderID = cid
	}
	out.Fill = book.ParseNum(resp.FillCountFp)
	return out, nil
}

// PredictSlippage MARKET 滑点（来自 config_strategy_threshold）。
type PredictSlippage struct {
	Bps            int64
	IsMinAmountOut bool
}

// SendPredict Predict.fun MARKET（未成交取消≈FAK）；price=限价，qty=平台份额。
// 深度由策略内存簿 SizeCheck 保证；此处把 outcome.config 里的签名参数带上，避免发单再 REST 拉盘口/市场。
func SendPredict(predict httprepo.PredictRepo, cred PredictCreds, oc *persistententity.MarketOutcome, yes bool, price, qty float64, sell bool, slip PredictSlippage) (watchgroup.SendResult, error) {
	out := watchgroup.SendResult{ClientOrderID: newClientOrderID()}
	if predict == nil || oc == nil || qty <= 0 || !cred.OK {
		return out, nil
	}
	token := strings.TrimSpace(oc.YesTokenID)
	if !yes {
		token = strings.TrimSpace(oc.NoTokenID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := predict.PlaceMarketOrder(ctx, cred.Account, httprepo.PredictMarketOrderReq{
		MarketID:       strings.TrimSpace(oc.PlatformMarketOutcomeID),
		TokenID:        token,
		BuyYes:         yes,
		Sell:           sell,
		PriceLimit:     price,
		Quantity:       qty,
		SlippageBps:    slip.Bps,
		IsMinAmountOut: slip.IsMinAmountOut,
		Market:         predictMarketMeta(oc),
	})
	if err != nil || resp == nil {
		if err == nil {
			err = errEmptyOrder
		}
		return out, err
	}
	// Predict 查单走 hash；成交以 GetOrderByHash.amountFilled 为准（受理≠成交）。
	out.PlatformOrderID = strings.TrimSpace(resp.OrderHash)
	if out.PlatformOrderID == "" {
		out.PlatformOrderID = strings.TrimSpace(resp.OrderID)
	}
	if out.PlatformOrderID != "" {
		out.Fill = pollPredictFill(ctx, predict, cred.Account, out.PlatformOrderID, qty)
	}
	return out, nil
}

// pollPredictFill 下单后短轮询查单：FILLED/有 amountFilled 即返回；CANCELLED 等终态停。
func pollPredictFill(ctx context.Context, predict httprepo.PredictRepo, acc httpentity.PredictAccount, orderHash string, qty float64) float64 {
	if predict == nil || strings.TrimSpace(orderHash) == "" {
		return 0
	}
	var fill float64
	for i := 0; i < 6; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return fill
			case <-time.After(500 * time.Millisecond):
			}
		}
		d, err := predict.GetOrderByHash(ctx, acc, orderHash)
		if err != nil || d == nil {
			continue
		}
		if d.AmountFilled > 0 {
			fill = d.AmountFilled
			if qty > 0 && fill > qty {
				fill = qty
			}
		}
		st := strings.ToUpper(strings.TrimSpace(d.Status))
		switch st {
		case "FILLED":
			if fill > 0 {
				return fill
			}
		case "CANCELLED", "CANCELED", "EXPIRED", "INVALIDATED":
			return fill
		}
	}
	return fill
}

func orderIDFromErrMsg(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	// body: {"error":"...","orderID":"0x..."}
	const key = `"orderID"`
	i := strings.Index(msg, key)
	if i < 0 {
		i = strings.Index(msg, `"orderId"`)
		if i < 0 {
			return ""
		}
	}
	rest := msg[i:]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end <= 0 {
		return ""
	}
	oid := strings.TrimSpace(rest[:end])
	if oid == "" || strings.EqualFold(oid, "null") {
		return ""
	}
	return oid
}

// SendSxbet SX Bet LIMIT+IOC（waitForOutcome）；qty=USDC stake。
func SendSxbet(sxbet httprepo.SxbetRepo, cred SxbetCreds, oc *persistententity.MarketOutcome, yes bool, price, qty float64) (watchgroup.SendResult, error) {
	out := watchgroup.SendResult{ClientOrderID: newClientOrderID()}
	if sxbet == nil || oc == nil || qty <= 0 || !cred.OK {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := sxbet.PlaceIOCOrder(ctx, cred.Account, httpentity.SxbetPlaceOrderReq{
		MarketHash:    strings.TrimSpace(oc.PlatformMarketOutcomeID),
		BuyYes:        yes,
		Price:         price,
		Quantity:      qty,
		ClientOrderID: out.ClientOrderID,
	})
	if err != nil || resp == nil {
		if err == nil {
			err = errEmptyOrder
		}
		return out, err
	}
	out.PlatformOrderID = strings.TrimSpace(resp.OrderID)
	if resp.FillQty > 0 {
		out.Fill = resp.FillQty
	}
	return out, nil
}

// SendOnOutcome 在一条 market_outcome 上买 YES 或 NO（LIMIT+立刻成交）。
func SendOnOutcome(
	poly httprepo.PolymarketRepo, kalshi httprepo.KalshiRepo, predict httprepo.PredictRepo, sxbet httprepo.SxbetRepo,
	polyCred PolyCreds, kalshiCred KalshiCreds, predictCred PredictCreds, sxbetCred SxbetCreds,
	oc *persistententity.MarketOutcome, yes bool, price, qty float64, negRisk bool, predictSlip PredictSlippage,
) (watchgroup.SendResult, error) {
	if oc == nil {
		return watchgroup.SendResult{}, nil
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformKalshi) {
		return SendKalshi(kalshi, kalshiCred, oc, yes, price, qty, false)
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
		return SendPredict(predict, predictCred, oc, yes, price, qty, false, predictSlip)
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformSxbet) {
		return SendSxbet(sxbet, sxbetCred, oc, yes, price, qty)
	}
	token := strings.TrimSpace(oc.YesTokenID)
	if !yes {
		token = strings.TrimSpace(oc.NoTokenID)
	}
	return SendPoly(poly, polyCred, token, price, qty, true, negRisk)
}

// FlattenOnOutcome 单边成交后在该 outcome 上反向平仓。
func FlattenOnOutcome(
	poly httprepo.PolymarketRepo, kalshi httprepo.KalshiRepo, predict httprepo.PredictRepo, sxbet httprepo.SxbetRepo,
	polyCred PolyCreds, kalshiCred KalshiCreds, predictCred PredictCreds, sxbetCred SxbetCreds,
	oc *persistententity.MarketOutcome, boughtYes bool, mine, other float64, negRisk bool, predictSlip PredictSlippage,
) {
	if oc == nil || mine <= 0 || other > 0 {
		return
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformKalshi) {
		_, _ = SendKalshi(kalshi, kalshiCred, oc, boughtYes, 0.01, mine, true)
		return
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
		_, _ = SendPredict(predict, predictCred, oc, boughtYes, 0.01, mine, true, predictSlip)
		return
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformSxbet) {
		_, _ = SendSxbet(sxbet, sxbetCred, oc, !boughtYes, 0.99, mine)
		return
	}
	token := strings.TrimSpace(oc.YesTokenID)
	if !boughtYes {
		token = strings.TrimSpace(oc.NoTokenID)
	}
	_, _ = SendPoly(poly, polyCred, token, 0.01, mine, false, negRisk)
}

// ReadNegRisk 从盘口配置读出 negRisk。
func ReadNegRisk(cfg *string) bool {
	if cfg == nil || strings.TrimSpace(*cfg) == "" {
		return false
	}
	var body struct {
		NegRisk bool `json:"negRisk"`
	}
	if err := json.Unmarshal([]byte(*cfg), &body); err != nil {
		return false
	}
	return body.NegRisk
}

// predictMarketMeta 从 market_outcome.config 读签名参数；字段不全则返回 nil，由客户端走缓存 GetMarket。
func predictMarketMeta(oc *persistententity.MarketOutcome) *httprepo.PredictMarketMeta {
	if oc == nil || oc.Config == nil || strings.TrimSpace(*oc.Config) == "" {
		return nil
	}
	var cfg struct {
		FeeRateBps     *int  `json:"feeRateBps"`
		IsNegRisk      *bool `json:"isNegRisk"`
		IsYieldBearing *bool `json:"isYieldBearing"`
		NegRisk        *bool `json:"negRisk"`
	}
	if err := json.Unmarshal([]byte(*oc.Config), &cfg); err != nil || cfg.FeeRateBps == nil {
		return nil
	}
	neg := false
	if cfg.IsNegRisk != nil {
		neg = *cfg.IsNegRisk
	} else if cfg.NegRisk != nil {
		neg = *cfg.NegRisk
	} else {
		return nil
	}
	yield := false
	if cfg.IsYieldBearing != nil {
		yield = *cfg.IsYieldBearing
	} else {
		return nil
	}
	return &httprepo.PredictMarketMeta{
		FeeRateBps:     *cfg.FeeRateBps,
		IsNegRisk:      neg,
		IsYieldBearing: yield,
	}
}
