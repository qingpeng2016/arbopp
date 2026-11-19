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

// PredictMarketOrderReq 市价买入指定 outcome token（FAK 语义由 MARKET 策略承担）。
type PredictMarketOrderReq struct {
	MarketID   string  // numeric market id
	TokenID    string  // outcome onChainId
	BuyYes     bool    // YES 视角盘口换算
	Sell       bool    // false=市价买入 token，true=市价卖出
	PriceLimit float64 // 策略侧卖一价（用于校验，不单独限价发单）
	Quantity   float64 // 份额
}

// PredictPlaceOrderResult 发单结果。
type PredictPlaceOrderResult struct {
	OrderID   string
	OrderHash string
	FillQty   float64
}

// PredictRepo Predict.fun REST（x-api-key）。
type PredictRepo interface {
	GetCategories(ctx context.Context, params PredictListCategoriesParams) (*entity.PredictGetCategoriesResponse, error)
	// GetOrderbooks 批量订单簿，单次最多 100 个 market id；无盘口的市场不会出现在结果里。
	GetOrderbooks(ctx context.Context, marketIDs []string) ([]entity.PredictOrderbook, error)
	GetMarket(ctx context.Context, marketID string) (*entity.PredictMarket, error)
	PlaceMarketOrder(ctx context.Context, acc entity.PredictAccount, req PredictMarketOrderReq) (*PredictPlaceOrderResult, error)
}