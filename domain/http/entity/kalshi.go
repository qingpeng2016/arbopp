package entity

// Kalshi 平台领域实体（对齐 Kalshi 现行生产 schema，2026-06 实测）。
//
// 关键约定（与 Polymarket 同构，便于 fgmm-runtime 对冲层统一处理）：
//   - Kalshi 市场标识是 ticker（如 "KXBTCD-26JUN2205-T63999.99"），对标 Polymarket conditionId(market_id)。
//   - 对外（adapter 层）用「合成 token_id」= "{ticker}|yes" / "{ticker}|no" 标识最小交易单位；
//     一个 market_id(ticker) 对应两个 token_id，结构与 Polymarket 一致。拆分/合成在 adapter 层完成。
//
// 重要：Kalshi 已把价格从「美分整数」迁移为「美元定点字符串」(*_dollars，如 "0.5600")，
// 数量从整数迁移为「定点字符串」(*_fp，支持小数份额，如 "676.00")。
// 本文件实体直接保存这些字符串原值，0~1 小数 <-> 美元字符串 / bid-ask 换算在 adapter 层完成。
//
// 下单走 V2 单簿模型：side = "bid"(买 yes / 卖 no) / "ask"(卖 yes / 买 no)，price 为 YES 价美元字符串。
//
// 所有类型以 Kalshi 前缀命名，避免与同 package 下 Polymarket 实体冲突。

// KalshiAccount Kalshi 鉴权身份。私钥在进程内做 RSA-PSS。
type KalshiAccount struct {
	// KeyID 来自 account.api_key，写入 KALSHI-ACCESS-KEY。
	KeyID string
	// PrivateKey 来自 account.api_secret_enc，RSA 私钥 PEM，本地签名。
	PrivateKey string
	// Passphrase 来自 account.api_passphrase_enc，只在 PEM 被加密时用来解密。
	Passphrase string
}

// ===== 市场 =====

// KalshiMarket Kalshi 单个市场（GET /markets/{ticker} 的 market 字段）。价格为美元字符串，数量为 *_fp 字符串。
type KalshiMarket struct {
	Ticker             string `json:"ticker"`
	EventTicker        string `json:"event_ticker"`
	Title              string `json:"title"`
	Subtitle           string `json:"subtitle"`
	Status             string `json:"status"` // active/closed/settled 等
	YesBidDollars      string `json:"yes_bid_dollars"`
	YesAskDollars      string `json:"yes_ask_dollars"`
	NoBidDollars       string `json:"no_bid_dollars"`
	NoAskDollars       string `json:"no_ask_dollars"`
	YesBidSizeFp       string `json:"yes_bid_size_fp"`
	YesAskSizeFp       string `json:"yes_ask_size_fp"`
	LastPriceDollars   string `json:"last_price_dollars"`
	VolumeFp           string `json:"volume_fp"`
	Volume24hFp        string `json:"volume_24h_fp"`
	OpenInterestFp     string `json:"open_interest_fp"`
	LiquidityDollars   string `json:"liquidity_dollars"`
	NotionalValDollars string `json:"notional_value_dollars"`
	OpenTime           string `json:"open_time"`
	CloseTime          string `json:"close_time"`
	Result             string `json:"result"`
}

// KalshiGetMarketResponse GET /trade-api/v2/markets/{ticker}
type KalshiGetMarketResponse struct {
	Market KalshiMarket `json:"market"`
}

// KalshiGetMarketsResponse GET /trade-api/v2/markets
type KalshiGetMarketsResponse struct {
	Markets []KalshiMarket `json:"markets"`
	Cursor  string         `json:"cursor"`
}

// KalshiProductMetadata 运动等品类的元数据（用于区分单场胜负盘与 props/大小分等）。
type KalshiProductMetadata struct {
	Competition      string `json:"competition"`
	CompetitionScope string `json:"competition_scope"`
}

// KalshiEvent Kalshi 事件，对标 Polymarket event（slug -> event_ticker）。
// with_nested_markets=true 时，该事件下的市场嵌在 event.markets（不是响应顶层）。
type KalshiEvent struct {
	EventTicker     string                 `json:"event_ticker"`
	SeriesTicker    string                 `json:"series_ticker"`
	Title           string                 `json:"title"`
	SubTitle        string                 `json:"sub_title"`
	Category        string                 `json:"category"`
	ProductMetadata *KalshiProductMetadata `json:"product_metadata"`
	Markets         []KalshiMarket         `json:"markets"`
}

// KalshiGetEventsResponse GET /trade-api/v2/events
type KalshiGetEventsResponse struct {
	Events []KalshiEvent `json:"events"`
	Cursor string        `json:"cursor"`
}

// KalshiGetEventResponse GET /trade-api/v2/events/{event_ticker}?with_nested_markets=true
// 实测市场嵌在 event.markets（见 KalshiEvent.Markets）；顶层 Markets 仅作兜底，正常为空。
type KalshiGetEventResponse struct {
	Event   KalshiEvent    `json:"event"`
	Markets []KalshiMarket `json:"markets"`
}

// ===== 订单簿 =====

// KalshiOrderbook GET /markets/{ticker}/orderbook 的 orderbook_fp 字段。
// yes_dollars / no_dollars 均为 [[price_dollar_string, size_fp_string], ...]，表示对应 side 的「挂买单」。
// 某 side 的卖盘价 = 1 - 对手 side 的买盘价（在 adapter 层换算）。
type KalshiOrderbook struct {
	Yes [][]string `json:"yes_dollars"`
	No  [][]string `json:"no_dollars"`
}

// KalshiGetOrderbookResponse GET /trade-api/v2/markets/{ticker}/orderbook
type KalshiGetOrderbookResponse struct {
	OrderbookFp KalshiOrderbook `json:"orderbook_fp"`
}

// KalshiBatchOrderbook 批量订单簿响应中的单个市场：ticker + 该市场盘口。
// 盘口结构与单市场端点的 orderbook_fp 完全一致（yes_dollars/no_dollars 均为挂买单）。
type KalshiBatchOrderbook struct {
	Ticker      string          `json:"ticker"`       // 市场 ticker，用作按市场索引的键
	OrderbookFp KalshiOrderbook `json:"orderbook_fp"` // 该市场盘口
}

// KalshiGetOrderbooksResponse GET /trade-api/v2/markets/orderbooks?tickers=A&tickers=B
//
// 注意：
//   - 单次请求最多 100 个 ticker（Kalshi 硬限制，超出直接 400）；
//   - 响应顺序不保证与入参一致；实测对查不到的 ticker 会回一条空盘口（yes/no 均为空数组），
//     所以调用方必须按 Ticker 建索引，并对空盘口判空，不能依赖下标或「有元素就代表有盘口」；
//   - 该端点不支持 depth 参数，返回全量档位（与单市场端点不同，需要档位上限由调用方自己截）。
type KalshiGetOrderbooksResponse struct {
	Orderbooks []KalshiBatchOrderbook `json:"orderbooks"`
}

// ===== 下单（V2 单簿模型） =====

// KalshiPlaceOrderReq POST /trade-api/v2/portfolio/events/orders 请求体。
type KalshiPlaceOrderReq struct {
	Ticker                  string `json:"ticker"`
	ClientOrderID           string `json:"client_order_id"`
	Side                    string `json:"side"`  // bid(买yes/卖no) / ask(卖yes/买no)
	Count                   string `json:"count"` // 份额，定点字符串（支持小数）
	Price                   string `json:"price"` // YES 价，美元定点字符串，如 "0.5600"
	TimeInForce             string `json:"time_in_force"`
	SelfTradePreventionType string `json:"self_trade_prevention_type"`
	PostOnly                bool   `json:"post_only,omitempty"`
}

// KalshiCreateOrderResponse POST /portfolio/events/orders 的扁平 201 响应（无 status，按 fill/remaining 推断）。
type KalshiCreateOrderResponse struct {
	OrderID        string `json:"order_id"`
	ClientOrderID  string `json:"client_order_id"`
	FillCount      string `json:"fill_count"`
	RemainingCount string `json:"remaining_count"`
	TsMs           int64  `json:"ts_ms"`
}

// KalshiCancelOrderResponse DELETE /portfolio/events/orders/{id}（V2 撤单）的扁平响应。
// 注意：V2 撤单响应无 status 字段，仅返回撤掉的份额 reduced_by；成功即 HTTP 200。
type KalshiCancelOrderResponse struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"client_order_id"`
	ReducedBy     string `json:"reduced_by"` // 本次撤掉的份额（定点字符串）
	TsMs          int64  `json:"ts_ms"`
}

// KalshiOrder GET/DELETE /portfolio/orders/{id} 的 order 字段（统一订单视图）。
type KalshiOrder struct {
	OrderID         string `json:"order_id"`
	ClientOrderID   string `json:"client_order_id"`
	Ticker          string `json:"ticker"`
	Status          string `json:"status"` // resting/canceled/executed/pending（create 响应无此字段）
	Action          string `json:"action"` // buy/sell
	Side            string `json:"side"`   // yes/no（outcome side）
	BookSide        string `json:"book_side"`
	Type            string `json:"type"`
	YesPriceDollars string `json:"yes_price_dollars"`
	NoPriceDollars  string `json:"no_price_dollars"`
	InitialCountFp  string `json:"initial_count_fp"`
	FillCountFp     string `json:"fill_count_fp"`
	RemainingCntFp  string `json:"remaining_count_fp"`
	CreatedTime     string `json:"created_time"`
	LastUpdateTime  string `json:"last_update_time"`
}

// KalshiOrderResponse GET/DELETE 单订单响应
type KalshiOrderResponse struct {
	Order KalshiOrder `json:"order"`
}

// KalshiGetOrdersResponse GET /trade-api/v2/portfolio/orders
type KalshiGetOrdersResponse struct {
	Orders []KalshiOrder `json:"orders"`
	Cursor string        `json:"cursor"`
}

// KalshiAmendOrderReq 改单请求（V2）：重述订单关键字段，改价/改量。
type KalshiAmendOrderReq struct {
	Ticker               string `json:"ticker"`
	Side                 string `json:"side"`  // bid/ask
	Price                string `json:"price"` // YES 价美元字符串
	Count                string `json:"count"` // 定点字符串
	ClientOrderID        string `json:"client_order_id,omitempty"`
	UpdatedClientOrderID string `json:"updated_client_order_id"`
}

// ===== 持仓 =====

// KalshiMarketPosition 单市场持仓。PositionFp 为带符号定点字符串：>0 持有 yes 份额，<0 持有 no（做空 yes）。
type KalshiMarketPosition struct {
	Ticker                string `json:"ticker"`
	PositionFp            string `json:"position_fp"`
	MarketExposureDollars string `json:"market_exposure_dollars"`
	RealizedPnlDollars    string `json:"realized_pnl_dollars"`
	TotalTradedDollars    string `json:"total_traded_dollars"`
	FeesPaidDollars       string `json:"fees_paid_dollars"`
	RestingOrdersCount    int    `json:"resting_orders_count"`
}

// KalshiGetPositionsResponse GET /trade-api/v2/portfolio/positions
type KalshiGetPositionsResponse struct {
	MarketPositions []KalshiMarketPosition `json:"market_positions"`
	Cursor          string                 `json:"cursor"`
}

// ===== 成交 =====

// KalshiFill 单笔成交。价格为美元字符串；份额 CountFp 优先，回退整数 Count。
type KalshiFill struct {
	FillID          string `json:"fill_id"`  // 单笔撮合的唯一 ID（一个 trade 可能拆成多条 fill）
	TradeID         string `json:"trade_id"` // 撮合 ID（同一次撮合的两侧共用）
	OrderID         string `json:"order_id"` // 本方订单 ID
	Ticker          string `json:"ticker"`   // 市场 ticker
	Side            string `json:"side"`     // yes/no（outcome 侧）
	Action          string `json:"action"`   // buy/sell
	Count           int64  `json:"count"`    // 成交量（整数份额，旧字段）
	CountFp         string `json:"count_fp"` // 成交量（定点字符串，优先用这个）
	YesPriceDollars string `json:"yes_price_dollars"`
	NoPriceDollars  string `json:"no_price_dollars"`
	FeeCost         string `json:"fee_cost"` // 本笔成交实际手续费（美元字符串），用于 real_trade_amount，无需再估算
	IsTaker         bool   `json:"is_taker"`
	CreatedTime     string `json:"created_time"`
}

// KalshiGetFillsResponse GET /trade-api/v2/portfolio/fills
type KalshiGetFillsResponse struct {
	Fills  []KalshiFill `json:"fills"`
	Cursor string       `json:"cursor"`
}

// ===== 账户余额 =====

// KalshiBalance GET /trade-api/v2/portfolio/balance。Balance 为美分整数，BalanceDollars 为美元字符串（二者并存）。
type KalshiBalance struct {
	Balance        int64  `json:"balance"`
	BalanceDollars string `json:"balance_dollars"`
}
