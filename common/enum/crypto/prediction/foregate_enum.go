package prediction

// ForegateOrderSide Foregate订单方向
type ForegateOrderSide string

const (
	ForegateOrderSideBuy  ForegateOrderSide = "BUY"
	ForegateOrderSideSell ForegateOrderSide = "SELL"
)

// ForegateOrderType Foregate订单类型
type ForegateOrderType string

const (
	ForegateOrderTypeLimit  ForegateOrderType = "LIMIT"
	ForegateOrderTypeMarket ForegateOrderType = "MARKET"
)

// ForegateOrderStatus Foregate订单状态
type ForegateOrderStatus string

const (
	ForegateOrderStatusCreate       ForegateOrderStatus = "CREATE"        // 创建挂单
	ForegateOrderStatusCreateFailed ForegateOrderStatus = "CREATE_FAILED" // 创建挂单失败
	ForegateOrderStatusAccepted     ForegateOrderStatus = "ACCEPTED"      // 挂单成功
	ForegateOrderStatusPending      ForegateOrderStatus = "PENDING"       // 挂单中
	ForegateOrderStatusFilled       ForegateOrderStatus = "FILLED"        // 全部成交
	ForegateOrderStatusCancelled    ForegateOrderStatus = "CANCELLED"     // 挂单取消
	ForegateOrderStatusPartial      ForegateOrderStatus = "PARTIAL"       // 部分成交
	ForegateOrderStatusFailed       ForegateOrderStatus = "FAILED"        // 成交失败
)
