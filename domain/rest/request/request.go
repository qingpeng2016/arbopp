package request

import (
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

// PlaceOrderReq 预测市场单个下单请求
type PlaceOrderReq struct {
	Platform cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识：POLYMARKET, FOREGATE等
	// 预测市场通用字段
	GroupID   string                         `json:"group_id,omitempty"`          // 组ID（Foregate的market_id）
	MarketID  string                         `json:"market_id,omitempty"`         // 市场ID（Foregate的outcome_id）
	TokenID   string                         `json:"token_id" binding:"required"` // 代币ID（必填，Polymarket的token_id，Foregate的option_id）
	Side      cryptoPredictionEnum.OrderSide `json:"side"`                        // 买卖方向: BUY/SELL
	OrderType cryptoPredictionEnum.OrderType `json:"order_type"`                  // 订单类型: LIMIT/MARKET
	Price     string                         `json:"price,omitempty"`             // 限价单价格
	Size      string                         `json:"size,omitempty"`              // 限价单数量
	Amount    string                         `json:"amount,omitempty"`            // 市价单金额（买入时使用）
	ExtParams map[string]interface{}         `json:"ext_params,omitempty"`
}

// CancelOrderReq 预测市场取消订单请求
type CancelOrderReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	OrderID   string                        `json:"order_id" binding:"required"` // 订单ID
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// ModifyOrderReq 预测市场修改订单请求
type ModifyOrderReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	OrderID   string                        `json:"order_id" binding:"required"` // 订单ID
	Price     string                        `json:"price,omitempty"`             // 新价格（可选）
	Size      string                        `json:"size,omitempty"`              // 新数量（可选）
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// BatchPlaceOrderReq 预测市场批量下单请求
type BatchPlaceOrderReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	Orders    []PlaceOrderReq               `json:"orders" binding:"required"`   // 订单列表
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// BatchCancelOrderReq 预测市场批量取消订单请求
type BatchCancelOrderReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"`  // 平台标识
	OrderIDs  []string                      `json:"order_ids" binding:"required"` // 订单ID列表
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// BatchModifyOrderReq 预测市场批量修改订单请求
type BatchModifyOrderReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	Orders    []ModifyOrderReq              `json:"orders" binding:"required"`   // 订单列表
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetTradesReq 预测市场查询成交记录请求
type GetTradesReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	TradeID   string                        `json:"trade_id,omitempty"`          // 成交ID（Polymarket CLOB 单笔查询）
	GroupID   string                        `json:"group_id,omitempty"`          // 组ID（可选）
	MarketID  string                        `json:"market_id,omitempty"`         // 市场ID（可选）
	TokenID   string                        `json:"token_id,omitempty"`          // 代币ID（可选）
	OrderID   string                        `json:"order_id,omitempty"`          // 订单ID（可选）
	Limit     int                           `json:"limit,omitempty"`             // 返回数量限制
	Offset    int                           `json:"offset,omitempty"`            // 分页偏移
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetPositionsReq 预测市场查询持仓记录请求
type GetPositionsReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	GroupID   string                        `json:"group_id,omitempty"`          // 组ID（可选）
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetMarketInfoReq 预测市场查询标的信息请求
type GetMarketInfoReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	GroupID   string                        `json:"group_id" binding:"required"` // 组ID（必填，Foregate的market_id）
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetUserInfoReq 预测市场查询用户信息请求
type GetUserInfoReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetAllOrdersReq 预测市场查询所有订单请求
type GetAllOrdersReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"` // 平台标识
	GroupID   string                        `json:"group_id,omitempty"`          // 组ID（可选）
	MarketID  string                        `json:"market_id,omitempty"`         // 市场ID（可选）
	TokenID   string                        `json:"token_id,omitempty"`          // 代币ID（可选）
	Status    string                        `json:"status,omitempty"`            // 订单状态（可选）
	Limit     int                           `json:"limit,omitempty"`             // 返回数量限制
	Offset    int                           `json:"offset,omitempty"`            // 分页偏移
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetMarketsByMarketAddressListReq 通过 condition_ids 批量查询市场请求（市场地址列表）
type GetMarketsByMarketAddressListReq struct {
	Platform     cryptoPredictionEnum.Platform `json:"platform" binding:"required"`
	ConditionIDs []string                      `json:"condition_ids" binding:"required"` // conditionId 数组
	ExtParams    map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetOrderBookReq 获取订单簿请求
type GetOrderBookReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"`
	TokenID   string                        `json:"token_id" binding:"required"` // 代币ID
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// BatchGetOrderBooksReqItem 批量获取订单簿请求项
type BatchGetOrderBooksReqItem struct {
	TokenID string                         `json:"token_id" binding:"required"`
	Side    cryptoPredictionEnum.OrderSide `json:"side,omitempty"` // 可选
}

// BatchGetOrderBooksReq 批量获取订单簿请求
type BatchGetOrderBooksReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"`
	Items     []BatchGetOrderBooksReqItem   `json:"items" binding:"required"`
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetOrderByIdReq 根据订单ID查询单个订单请求
type GetOrderByIdReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"`
	OrderID   string                        `json:"order_id" binding:"required"`
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetEventBySlugReq 根据 slug 获取事件请求
type GetEventBySlugReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"`
	SlugName  string                        `json:"slug_name" binding:"required"` // 事件 slug
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}

// CreateAPIKeyReq 创建 API Key 请求
type CreateAPIKeyReq struct {
	Platform      cryptoPredictionEnum.Platform `json:"platform" binding:"required"`
	PrivateKeyHex string                        `json:"private_key_hex" binding:"required"`
	SignerAddress string                        `json:"signer_address" binding:"required"`
	ExtParams     map[string]interface{}        `json:"ext_params,omitempty"`
}

// GetFeeRateReq 查询平台交易费率请求。
type GetFeeRateReq struct {
	Platform  cryptoPredictionEnum.Platform `json:"platform" binding:"required"`
	TokenID   string                        `json:"token_id" binding:"required"`
	ExtParams map[string]interface{}        `json:"ext_params,omitempty"`
}
