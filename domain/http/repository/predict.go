package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// PredictListCategoriesParams GET /v1/categories 查询参数。
type PredictListCategoriesParams struct {
	First          int      // 1–100，默认 20
	After          string   // 翻页 cursor
	Status         string   // 非 RESOLVED 表示未结算；空为全部
	TagIDs         []string // 逗号拼接 tag id
	MarketVariants []string // 如 SPORTS_MATCH
}

// PredictMarketMeta 本地缓存的市场签名参数（来自 market_outcome.config / 同步写入）。
// 齐全时 PlaceMarketOrder 不再 GET /markets/{id}。
type PredictMarketMeta struct {
	FeeRateBps     int
	IsNegRisk      bool
	IsYieldBearing bool
}

// PredictMarketOrderReq 市价买入指定 outcome token（FAK 语义由 MARKET 策略承担）。
// Predict 的 LIMIT 会挂单；套利路径用 MARKET + PriceLimit，金额按限价直接签名，不再 REST 拉盘口。
type PredictMarketOrderReq struct {
	MarketID       string  // numeric market id
	TokenID        string  // outcome onChainId
	BuyYes         bool    // YES 视角盘口换算（保留字段；金额按限价计算后不再换算盘口）
	Sell           bool    // false=市价买入 token，true=市价卖出
	PriceLimit     float64 // 策略侧卖一价 → pricePerShare
	Quantity       float64 // 份额
	SlippageBps    int64   // 滑点 bps（100=1%）；≤0 时客户端用默认
	IsMinAmountOut bool    // true：买方向固定 USDT、少收份额（官方推荐）
	Market         *PredictMarketMeta
}

// PredictPlaceOrderResult 发单结果。
type PredictPlaceOrderResult struct {
	OrderID   string
	OrderHash string
	FillQty   float64
}

// PredictOrderFill 查单回补结果。
type PredictOrderFill struct {
	OrderID      string
	OrderHash    string
	Status       string  // OPEN / FILLED / …
	Amount       float64 // 下单份额
	AmountFilled float64
	FeeUSDT      float64
	AvgPrice     float64 // 签名金额推算均价（买=maker/taker）；0=未知
}

// PredictRepo Predict.fun REST（x-api-key）。
type PredictRepo interface {
	GetCategories(ctx context.Context, params PredictListCategoriesParams) (*entity.PredictGetCategoriesResponse, error)
	// GetOrderbooks 批量订单簿，单次最多 100 个 market id；无盘口的市场不会出现在结果里。
	GetOrderbooks(ctx context.Context, marketIDs []string) ([]entity.PredictOrderbook, error)
	GetMarket(ctx context.Context, marketID string) (*entity.PredictMarket, error)
	PlaceMarketOrder(ctx context.Context, acc entity.PredictAccount, req PredictMarketOrderReq) (*PredictPlaceOrderResult, error)
	GetOrderByHash(ctx context.Context, acc entity.PredictAccount, orderHash string) (*PredictOrderFill, error)
}