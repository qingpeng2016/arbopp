package repository

import (
	"context"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// KalshiListParams 列表类查询的通用过滤参数（订单/持仓/成交/市场）。
type KalshiListParams struct {
	Limit   int    // 单页条数，<=0 时由 infra 使用默认值
	Cursor  string // 翻页游标
	Ticker  string // 按市场 ticker 过滤（可选）
	Status  string // 按状态过滤（可选，如 orders 的 resting）
	OrderID string // 按订单 ID 过滤（仅 fills 端点使用，用于回填某笔订单的逐笔成交）
	// MveFilter 仅 /markets 使用：exclude 排除多腿组合盘（KXMVECROSS…），include 只返回组合盘。
	// 组合盘的 ticker 在单市场端点会 404，按 ticker 取盘口前必须排除。
	MveFilter string
	// SeriesTicker 仅 /markets、/events：按系列 ticker 过滤（如 KXNBAGAME）。
	SeriesTicker string
	// Category 仅 /events：如 Sports。
	Category string
	// WithNestedMarkets 仅 /events：true 时 markets 嵌在 event 内。
	WithNestedMarkets bool
}

// KalshiRepo Kalshi 平台基础设施层接口（HTTP + RSA-PSS 签名）。
// 公开行情类接口（市场/事件/订单簿）无需账户；账户私有接口（下单/撤单/改单/持仓/成交/余额）需要 KalshiAccount。
type KalshiRepo interface {
	// ===== 公开行情 =====

	// GetMarket 按 ticker 查询单个市场
	GetMarket(ctx context.Context, ticker string) (*entity.KalshiMarket, error)

	// GetMarkets 查询市场列表（params.Ticker 复用为 event_ticker 过滤时由 infra 处理）
	GetMarkets(ctx context.Context, params KalshiListParams) (*entity.KalshiGetMarketsResponse, error)

	// GetEvents 查询事件列表（可带 nested markets，用于 sports 比赛类同步）
	GetEvents(ctx context.Context, params KalshiListParams) (*entity.KalshiGetEventsResponse, error)

	// GetEvent 按 event_ticker 查询事件（含其下市场）
	GetEvent(ctx context.Context, eventTicker string) (*entity.KalshiGetEventResponse, error)

	// GetOrderbook 查询某市场订单簿，depth<=0 表示不限制档数
	GetOrderbook(ctx context.Context, ticker string, depth int) (*entity.KalshiOrderbook, error)

	// GetOrderbooks 批量查询多个市场订单簿（Kalshi 原生批量端点，单次最多 100 个 ticker）。
	// tickers 超过 100 个时由实现内部自动分批；返回 ticker -> 订单簿 的映射。
	// 注意：map 里可能有「空盘口」（查不到/已停牌的市场，yes/no 均为空），调用方需判空后再用。
	GetOrderbooks(ctx context.Context, tickers []string) (map[string]*entity.KalshiOrderbook, error)

	// SignHeaders 为 method+path 生成 Kalshi 鉴权头。path 不含 query，例如 /trade-api/ws/v2。
	SignHeaders(ctx context.Context, account entity.KalshiAccount, method, path string) (keyID, timestamp, signature string, err error)

	// ===== 账户私有 =====

	// PlaceOrder 下单
	PlaceOrder(ctx context.Context, req entity.KalshiPlaceOrderReq, account entity.KalshiAccount) (*entity.KalshiOrder, error)

	// CancelOrder 撤单
	CancelOrder(ctx context.Context, orderID string, account entity.KalshiAccount) (*entity.KalshiOrder, error)

	// AmendOrder 改单（改价/改量）
	AmendOrder(ctx context.Context, orderID string, req entity.KalshiAmendOrderReq, account entity.KalshiAccount) (*entity.KalshiOrder, error)

	// GetOrder 按 order_id 查询单个订单
	GetOrder(ctx context.Context, orderID string, account entity.KalshiAccount) (*entity.KalshiOrder, error)

	// GetOrders 查询订单列表
	GetOrders(ctx context.Context, params KalshiListParams, account entity.KalshiAccount) (*entity.KalshiGetOrdersResponse, error)

	// GetPositions 查询持仓
	GetPositions(ctx context.Context, params KalshiListParams, account entity.KalshiAccount) (*entity.KalshiGetPositionsResponse, error)

	// GetFills 查询成交
	GetFills(ctx context.Context, params KalshiListParams, account entity.KalshiAccount) (*entity.KalshiGetFillsResponse, error)

	// GetBalance 查询账户余额
	GetBalance(ctx context.Context, account entity.KalshiAccount) (*entity.KalshiBalance, error)
}
