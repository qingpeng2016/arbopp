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

// SendPoly 向 Polymarket 发一笔 FAK。
func SendPoly(poly httprepo.PolymarketRepo, cred PolyCreds, token string, price, qty float64, buy, neg bool) (float64, error) {
	if poly == nil || token == "" || qty <= 0 || !cred.OK {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req := httpentity.PlaceOrderReq{
		TokenID:          token,
		Side:             cryptoPredictionEnum.OrderSideBuy,
		OrderType:        cryptoPredictionEnum.PolymarketOrderTypeFAK,
		IsMarket:         true,
		NegRisk:          neg,
		MarketWorstPrice: strconv.FormatFloat(price, 'f', 4, 64),
		MarketAmount:     strconv.FormatFloat(qty*price, 'f', 4, 64),
	}
	if !buy {
		req.Side = cryptoPredictionEnum.OrderSideSell
		req.MarketAmount = ""
		req.MarketSize = strconv.FormatFloat(qty, 'f', 4, 64)
		req.MarketWorstPrice = "0.01"
	}
	resp, err := poly.PlaceOrder(ctx, req, cred.Account)
	if err != nil || resp == nil {
		if err == nil {
			err = errEmptyOrder
		}
		return 0, err
	}
	switch strings.ToLower(string(resp.Status)) {
	case "filled", "partial", "matched":
		return qty, nil
	default:
		return 0, nil
	}
}

// SendKalshi 向 Kalshi 发 IOC。
func SendKalshi(kalshi httprepo.KalshiRepo, cred KalshiCreds, oc *persistententity.MarketOutcome, yes bool, price, qty float64, sell bool) (float64, error) {
	if kalshi == nil || oc == nil || qty <= 0 || !cred.OK {
		return 0, nil
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
		ClientOrderID:           strconv.FormatInt(time.Now().UnixNano(), 36),
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
		return 0, err
	}
	return book.ParseNum(resp.FillCountFp), nil
}

// SendPredict Predict.fun 市价 BUY outcome token。
func SendPredict(predict httprepo.PredictRepo, cred PredictCreds, oc *persistententity.MarketOutcome, yes bool, price, qty float64, sell bool) (float64, error) {
	if predict == nil || oc == nil || qty <= 0 || !cred.OK {
		return 0, nil
	}
	token := strings.TrimSpace(oc.YesTokenID)
	if !yes {
		token = strings.TrimSpace(oc.NoTokenID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := predict.PlaceMarketOrder(ctx, cred.Account, httprepo.PredictMarketOrderReq{
		MarketID:   strings.TrimSpace(oc.PlatformMarketOutcomeID),
		TokenID:    token,
		BuyYes:     yes,
		Sell:       sell,
		PriceLimit: price,
		Quantity:   qty,
	})
	if err != nil || resp == nil {
		if err == nil {
			err = errEmptyOrder
		}
		return 0, err
	}
	if resp.FillQty > 0 {
		return resp.FillQty, nil
	}
	return qty, nil
}

// SendOnOutcome 在一条 market_outcome 上买 YES 或 NO。
func SendOnOutcome(
	poly httprepo.PolymarketRepo, kalshi httprepo.KalshiRepo, predict httprepo.PredictRepo,
	polyCred PolyCreds, kalshiCred KalshiCreds, predictCred PredictCreds,
	oc *persistententity.MarketOutcome, yes bool, price, qty float64, negRisk bool,
) (float64, error) {
	if strings.EqualFold(oc.Platform, appcommon.PlatformKalshi) {
		return SendKalshi(kalshi, kalshiCred, oc, yes, price, qty, false)
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
		return SendPredict(predict, predictCred, oc, yes, price, qty, false)
	}
	token := strings.TrimSpace(oc.YesTokenID)
	if !yes {
		token = strings.TrimSpace(oc.NoTokenID)
	}
	return SendPoly(poly, polyCred, token, price, qty, true, negRisk)
}

// FlattenOnOutcome 单边成交后在该 outcome 上反向平仓。
func FlattenOnOutcome(
	poly httprepo.PolymarketRepo, kalshi httprepo.KalshiRepo, predict httprepo.PredictRepo,
	polyCred PolyCreds, kalshiCred KalshiCreds, predictCred PredictCreds,
	oc *persistententity.MarketOutcome, boughtYes bool, mine, other float64, negRisk bool,
) {
	if oc == nil || mine <= 0 || other > 0 {
		return
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformKalshi) {
		_, _ = SendKalshi(kalshi, kalshiCred, oc, boughtYes, 0.01, mine, true)
		return
	}
	if strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
		_, _ = SendPredict(predict, predictCred, oc, boughtYes, 0.01, mine, true)
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