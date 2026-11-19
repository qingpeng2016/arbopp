package self_mm

// Vendor 渠道
type Vendor string

const (
	VendorForegate Vendor = "foregate"
)

// OrderType 订单类型
type OrderType string

// OrderSide 买卖方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OutcomeType YES/NO
type OutcomeType string

const (
	OutcomeTypeYes OutcomeType = "YES"
	OutcomeTypeNo  OutcomeType = "NO"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusAccepted  OrderStatus = "ACCEPTED"
	OrderStatusCreate    OrderStatus = "CREATE"
	OrderStatusFilled    OrderStatus = "FILLED"
)

// DataSource 数据来源
type DataSource string
