package watchgroup

// SendResult 单腿发单回报（fill 已折成策略净兑付口径）。
type SendResult struct {
	Fill            float64
	PlatformOrderID string // Poly orderID / Predict orderHash / SX orderId
	ClientOrderID   string
	Fee             float64 // 结算币手续费；下单瞬间常为 0，由 sync_order_fills 回补
}
