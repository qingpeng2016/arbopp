package response

import (
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"time"
)

// PlaceOrderResp 预测市场单个下单响应
type PlaceOrderResp struct {
	OrderID  string                           `json:"order_id"`            // 订单ID
	Status   cryptoPredictionEnum.OrderStatus `json:"status"`              // 订单状态
	Message  string                           `json:"message,omitempty"`   // 消息
	ErrorMsg string                           `json:"error_msg,omitempty"` // 错误信息
	ExtData  map[string]interface{}           `json:"ext_data,omitempty"`
}

// CancelOrderResp 预测市场取消订单响应
type CancelOrderResp struct {
	OrderID  string                 `json:"order_id"`            // 订单ID
	Success  bool                   `json:"success"`             // 是否成功
	Message  string                 `json:"message,omitempty"`   // 消息
	ErrorMsg string                 `json:"error_msg,omitempty"` // 错误信息
	ExtData  map[string]interface{} `json:"ext_data,omitempty"`
}

// ModifyOrderResp 预测市场修改订单响应
type ModifyOrderResp struct {
	OrderID  string                           `json:"order_id"`            // 订单ID
	Status   cryptoPredictionEnum.OrderStatus `json:"status"`              // 订单状态
	Message  string                           `json:"message,omitempty"`   // 消息
	ErrorMsg string                           `json:"error_msg,omitempty"` // 错误信息
	ExtData  map[string]interface{}           `json:"ext_data,omitempty"`
}

// BatchPlaceOrderResp 预测市场批量下单响应
type BatchPlaceOrderResp struct {
	Orders  []PlaceOrderResp       `json:"orders"`  // 订单列表
	Total   int                    `json:"total"`   // 总数
	Success int                    `json:"success"` // 成功数量
	Failed  int                    `json:"failed"`  // 失败数量
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// BatchCancelOrderResp 预测市场批量取消订单响应
type BatchCancelOrderResp struct {
	OrderIDs []string               `json:"order_ids"` // 订单ID列表
	Success  int                    `json:"success"`   // 成功数量
	Failed   int                    `json:"failed"`    // 失败数量
	Results  []CancelOrderResp      `json:"results"`   // 详细结果
	ExtData  map[string]interface{} `json:"ext_data,omitempty"`
}

// BatchModifyOrderResp 预测市场批量修改订单响应
type BatchModifyOrderResp struct {
	Orders  []ModifyOrderResp      `json:"orders"`  // 订单列表
	Total   int                    `json:"total"`   // 总数
	Success int                    `json:"success"` // 成功数量
	Failed  int                    `json:"failed"`  // 失败数量
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// Trade 预测市场成交记录
type Trade struct {
	TradeID   string                 `json:"trade_id"`   // 成交ID
	OrderID   string                 `json:"order_id"`   // 订单ID
	Symbol    string                 `json:"symbol"`     // 交易对
	Side      string                 `json:"side"`       // 买卖方向
	Price     string                 `json:"price"`      // 成交价格
	Size      string                 `json:"size"`       // 成交数量
	Amount    string                 `json:"amount"`     // 成交金额
	Fee       string                 `json:"fee"`        // 手续费
	TradeTime time.Time              `json:"trade_time"` // 成交时间
	ExtData   map[string]interface{} `json:"ext_data,omitempty"`
}

// GetTradesResp 预测市场查询成交记录响应
type GetTradesResp struct {
	Trades  []Trade                `json:"trades"` // 成交记录列表
	Total   int                    `json:"total"`  // 总数
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// Position 预测市场持仓记录
type Position struct {
	Symbol        string                 `json:"symbol"`         // 交易对
	Side          string                 `json:"side"`           // 方向（LONG/SHORT）
	Size          string                 `json:"size"`           // 持仓数量
	AvgPrice      string                 `json:"avg_price"`      // 平均价格
	UnrealizedPnl string                 `json:"unrealized_pnl"` // 未实现盈亏
	RealizedPnl   string                 `json:"realized_pnl"`   // 已实现盈亏
	ExtData       map[string]interface{} `json:"ext_data,omitempty"`
}

// GetPositionsResp 预测市场查询持仓记录响应
type GetPositionsResp struct {
	Positions []Position             `json:"positions"` // 持仓列表
	Total     int                    `json:"total"`     // 总数
	ExtData   map[string]interface{} `json:"ext_data,omitempty"`
}

// MarketInfo 预测市场标的信息
type MarketInfo struct {
	Symbol    string                 `json:"symbol"`     // 交易对
	MarketID  string                 `json:"market_id"`  // 市场ID
	Name      string                 `json:"name"`       // 名称
	Status    string                 `json:"status"`     // 状态
	MinPrice  string                 `json:"min_price"`  // 最小价格
	MaxPrice  string                 `json:"max_price"`  // 最大价格
	PriceStep string                 `json:"price_step"` // 价格步长
	MinSize   string                 `json:"min_size"`   // 最小数量
	SizeStep  string                 `json:"size_step"`  // 数量步长
	ExtData   map[string]interface{} `json:"ext_data,omitempty"`
}

// GetMarketInfoResp 预测市场查询标的信息响应
type GetMarketInfoResp struct {
	Markets []MarketInfo           `json:"markets"` // 市场列表
	Total   int                    `json:"total"`   // 总数
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// UserInfo 预测市场用户信息
type UserInfo struct {
	UserID   string                 `json:"user_id"`  // 用户ID
	Username string                 `json:"username"` // 用户名
	Email    string                 `json:"email"`    // 邮箱
	Balance  string                 `json:"balance"`  // 余额
	ExtData  map[string]interface{} `json:"ext_data,omitempty"`
}

// GetUserInfoResp 预测市场查询用户信息响应
type GetUserInfoResp struct {
	UserInfo UserInfo               `json:"user_info"` // 用户信息
	ExtData  map[string]interface{} `json:"ext_data,omitempty"`
}

// Order 预测市场订单信息
type Order struct {
	OrderID       string                           `json:"order_id"`       // 订单ID
	Symbol        string                           `json:"symbol"`         // 交易对
	Side          string                           `json:"side"`           // 买卖方向
	OrderType     string                           `json:"order_type"`     // 订单类型
	Price         string                           `json:"price"`          // 价格
	Size          string                           `json:"size"`           // 数量
	FilledSize    string                           `json:"filled_size"`    // 已成交数量
	RemainingSize string                           `json:"remaining_size"` // 剩余数量
	Status        cryptoPredictionEnum.OrderStatus `json:"status"`         // 订单状态
	CreatedAt     time.Time                        `json:"created_at"`     // 创建时间
	UpdatedAt     time.Time                        `json:"updated_at"`     // 更新时间
	ExtData       map[string]interface{}           `json:"ext_data,omitempty"`
}

// GetAllOrdersResp 预测市场查询所有订单响应
type GetAllOrdersResp struct {
	Orders  []Order                `json:"orders"` // 订单列表
	Total   int                    `json:"total"`  // 总数
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// GetFeeRateResp 查询平台交易费率响应，费率单位为基点（bps）。
type GetFeeRateResp struct {
	FeeRateBps string                 `json:"fee_rate_bps"`
	ExtData    map[string]interface{} `json:"ext_data,omitempty"`
}

// GetMarketsByMarketAddressListResp 通过市场地址列表批量查询市场响应
type GetMarketsByMarketAddressListResp struct {
	Markets []MarketInfo           `json:"markets"`
	Total   int                    `json:"total"`
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// OrderBookEntry 订单簿档位
type OrderBookEntry struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// GetOrderBookResp 获取订单簿响应
type GetOrderBookResp struct {
	Market       string                 `json:"market"`
	AssetID      string                 `json:"asset_id"`
	TokenID      string                 `json:"token_id"`
	Bids         []OrderBookEntry       `json:"bids"`
	Asks         []OrderBookEntry       `json:"asks"`
	MinOrderSize string                 `json:"min_order_size,omitempty"`
	TickSize     string                 `json:"tick_size,omitempty"`
	ExtData      map[string]interface{} `json:"ext_data,omitempty"`
}

// BatchGetOrderBooksResp 批量获取订单簿响应
type BatchGetOrderBooksResp struct {
	OrderBooks []GetOrderBookResp     `json:"order_books"`
	ExtData    map[string]interface{} `json:"ext_data,omitempty"`
}

// GetOrderByIdResp 根据订单ID查询单个订单响应
type GetOrderByIdResp struct {
	Order   Order                  `json:"order"`
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// GetEventBySlugResp 根据 slug 获取事件响应（Event 为原始结构，便于多平台扩展）
type GetEventBySlugResp struct {
	Event   map[string]interface{} `json:"event"` // 或具体 Event 结构
	ExtData map[string]interface{} `json:"ext_data,omitempty"`
}

// CreateAPIKeyResp 创建 API Key 响应
type CreateAPIKeyResp struct {
	APIKey     string                 `json:"api_key"`
	Secret     string                 `json:"secret"`
	Passphrase string                 `json:"passphrase"`
	ExtData    map[string]interface{} `json:"ext_data,omitempty"`
}
