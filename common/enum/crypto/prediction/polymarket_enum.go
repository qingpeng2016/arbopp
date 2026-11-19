package prediction

// PolymarketOrderSide Polymarket订单方向
type PolymarketOrderSide string

const (
	PolymarketOrderSideBuy  PolymarketOrderSide = "BUY"
	PolymarketOrderSideSell PolymarketOrderSide = "SELL"
)

// PolymarketOrderType Polymarket订单类型
type PolymarketOrderType string

const (
	PolymarketOrderTypeGTC PolymarketOrderType = "GTC" // Good Till Cancelled - 限价单，直到取消
	PolymarketOrderTypeGTD PolymarketOrderType = "GTD" // Good Till Date - 限价单，到期自动取消
	PolymarketOrderTypeFOK PolymarketOrderType = "FOK" // Fill Or Kill - 全部成交或取消
	PolymarketOrderTypeFAK PolymarketOrderType = "FAK" // Fill And Kill - 立即成交，未成交部分取消
)

// PolymarketOrderStatus Polymarket订单状态
type PolymarketOrderStatus string

const (
	PolymarketOrderStatusCreate       PolymarketOrderStatus = "CREATE"        // 创建挂单
	PolymarketOrderStatusCreateFailed PolymarketOrderStatus = "CREATE_FAILED" // 创建挂单失败
	PolymarketOrderStatusAccepted     PolymarketOrderStatus = "ACCEPTED"      // 挂单成功
	PolymarketOrderStatusPending      PolymarketOrderStatus = "PENDING"       // 挂单中
	PolymarketOrderStatusFilled       PolymarketOrderStatus = "FILLED"        // 全部成交
	PolymarketOrderStatusCancelled    PolymarketOrderStatus = "CANCELLED"     // 挂单取消
	PolymarketOrderStatusPartial      PolymarketOrderStatus = "PARTIAL"       // 部分成交
	PolymarketOrderStatusFailed       PolymarketOrderStatus = "FAILED"        // 成交失败
)
