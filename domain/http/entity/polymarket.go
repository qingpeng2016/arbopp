package entity

import (
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

// PolymarketAccount Polymarket 账户（API Key 或代理钱包）
type PolymarketAccount struct {
	APIKey              string `json:"api_key"`
	APISecret           string `json:"api_secret"`
	APIPassphrase       string `json:"api_passphrase"`
	EOAAddress          string `json:"eoa_address"`
	ProxyAddress        string `json:"proxy_address"`
	PrivateKey          string `json:"private_key"`
	SignatureType       int    `json:"signature_type"`
	UseMagicAccount     bool   `json:"use_magic_account"`
	OrderMaxPendingTime int    `json:"order_max_pending_time"`
}

// Market 市场信息结构
type Market struct {
	ID                       string             `json:"id"`
	Question                 string             `json:"question"`
	Description              string             `json:"description"`
	Slug                     string             `json:"slug"`
	StartDate                string             `json:"startDate"`
	StartDateIso             string             `json:"startDateIso"`
	EndDate                  time.Time          `json:"endDate"`
	EndDateIso               string             `json:"endDateIso"`
	GameStartTime            string             `json:"gameStartTime"`
	EventStartTime           string             `json:"eventStartTime"`
	Volume                   string             `json:"volume"`
	VolumeNum                float64            `json:"volumeNum"`
	Volume24hr               float64            `json:"volume24hr"`
	Volume1wk                float64            `json:"volume1wk"`
	Volume1mo                float64            `json:"volume1mo"`
	Volume1yr                float64            `json:"volume1yr"`
	Volume24hrClob           float64            `json:"volume24hrClob"`
	Volume1wkClob            float64            `json:"volume1wkClob"`
	Volume1moClob            float64            `json:"volume1moClob"`
	Volume1yrClob            float64            `json:"volume1yrClob"`
	VolumeClob               float64            `json:"volumeClob"`
	Liquidity                string             `json:"liquidity"`
	LiquidityNum             float64            `json:"liquidityNum"`
	Outcomes                 string             `json:"outcomes"`
	OutcomePrices            string             `json:"outcomePrices"`
	ClobTokenIds             string             `json:"clobTokenIds"`
	Active                   bool               `json:"active"`
	Archived                 bool               `json:"archived"`
	Closed                   bool               `json:"closed"`
	ClosedTime               string             `json:"closedTime"`
	CreatedAt                time.Time          `json:"createdAt"`
	UpdatedAt                time.Time          `json:"updatedAt"`
	Category                 string             `json:"category"`
	SubCategory              string             `json:"subcategory"`
	ConditionId              string             `json:"conditionId"`
	MarketType               string             `json:"marketType"`
	MarketMakerAddress       string             `json:"marketMakerAddress"`
	BestBid                  float64            `json:"bestBid"`
	BestAsk                  float64            `json:"bestAsk"`
	LastTradePrice           float64            `json:"lastTradePrice"`
	Image                    string             `json:"image"`
	Icon                     string             `json:"icon"`
	TwitterCardImage         string             `json:"twitterCardImage"`
	Approved                 bool               `json:"approved"`
	Ready                    bool               `json:"ready"`
	Funded                   bool               `json:"funded"`
	Competitive              float64            `json:"competitive"`
	Creator                  string             `json:"creator"`
	UpdatedBy                int                `json:"updatedBy"`
	NegRisk                  bool               `json:"negRisk"`
	SportsMarketType         string             `json:"sportsMarketType"`         // 体育盘类型（足球点球盘识别用）
	UMAResolutionStatus      string             `json:"umaResolutionStatus"`      // UMA 结算状态
	AcceptingOrdersTimestamp string             `json:"acceptingOrdersTimestamp"` // 市场开始时间
	Events                   []MarketEventEmbed `json:"events"`                   // 用于解析 league、finishedTimestamp、closedTime 及实时赛事状态
}

// MarketEventEmbed 用于解析 Market.Events[]，提取联赛/赛事及时间，并透传实时赛事状态
type MarketEventEmbed struct {
	Title  string `json:"title"`
	Slug   string `json:"slug"`
	Series []struct {
		Title string `json:"title"`
		Slug  string `json:"slug"`
	} `json:"series"`
	FinishedTimestamp string `json:"finishedTimestamp"` // 比赛结束时间
	ClosedTime        string `json:"closedTime"`        // 市场关闭时间
	// 体育盘实时状态（源 MarketLiveEvent）：period/elapsed 仅比赛进行中有值
	Period    string `json:"period"`    // 阶段码：NS/1H/HT/2H/ET/VFT
	Elapsed   string `json:"elapsed"`   // 实际分钟(字符串)，仅 1H/2H 存在
	Live      bool   `json:"live"`      // 是否进行中
	Ended     bool   `json:"ended"`     // 是否真正结束
	Score     string `json:"score"`     // 比分，如 "1-1"
	StartTime string `json:"startTime"` // 计划开球
}

// MarketsResponse 市场列表响应
type MarketsResponse struct {
	Data []Market `json:"data"`
	Meta struct {
		Total int `json:"total"`
		Page  int `json:"page"`
		Limit int `json:"limit"`
	} `json:"meta"`
}

// OrderBookResponse 订单簿响应
type OrderBookResponse struct {
	Market       string           `json:"market"`
	AssetID      string           `json:"asset_id"`
	Timestamp    string           `json:"timestamp"`
	Hash         string           `json:"hash"`
	Bids         []OrderBookEntry `json:"bids"`
	Asks         []OrderBookEntry `json:"asks"`
	MinOrderSize string           `json:"min_order_size"`
	TickSize     string           `json:"tick_size"`
	NegRisk      bool             `json:"neg_risk"`
}

// OrderBookEntry 订单簿条目
type OrderBookEntry struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// SportsMetadata 运动元数据（GET /sports 返回，参考 https://docs.polymarket.com/api-reference/sports/get-sports-metadata-information）
type SportsMetadata struct {
	Sport        string `json:"sport"`
	Name         string `json:"name"`
	Image        string `json:"image"`
	Resolution   string `json:"resolution"`
	Ordering     string `json:"ordering"`
	Tags         string `json:"tags"` // 逗号分隔的 tag ID 列表，用于分类和过滤
	PrimaryTagID int    `json:"primaryTagId"`
	Series       string `json:"series"`
}

// GetMarketsRequest 获取市场列表请求
type GetMarketsRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ListMarketsRequest 列出市场请求（支持 closed、tag_id 过滤，参考 https://docs.polymarket.com/api-reference/markets/list-markets）
type ListMarketsRequest struct {
	Limit     int    `json:"limit"`            // 每页数量，默认 100
	Offset    int    `json:"offset"`           // 分页偏移
	Closed    *bool  `json:"closed"`           // 是否已关闭，nil=不过滤，true=仅关闭，false=仅活跃
	Order     string `json:"order"`            // 排序字段，如 volumeNum
	Ascending bool   `json:"ascending"`        // 是否升序
	TagID     int    `json:"tag_id,omitempty"` // 标签ID过滤，如 tag_id=1 表示运动市场（sportsMarketTypes 被 API 忽略，改用 tag_id）
}

// ListEventsPaginationRequest Gamma GET /events/pagination（需 Accept-Encoding: gzip；offset 过大需改 keyset）
type ListEventsPaginationRequest struct {
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	Closed  *bool  `json:"closed"`
	TagID   int    `json:"tag_id,omitempty"`
	TagSlug string `json:"tag_slug,omitempty"`
}

// ListMarketsKeysetRequest Gamma GET /markets/keyset。sports_market_types=moneyline 时欧冠 tag 100977 第一页即单场（events/keyset 同 tag 前几页是期货且体积过大）。
type ListMarketsKeysetRequest struct {
	Limit             int
	Closed            *bool
	TagID             int
	SportsMarketTypes []string
	AfterCursor       string
	Order             string
	Ascending         *bool
}

// PolymarketMarketsKeyset GET /markets/keyset 响应
type PolymarketMarketsKeyset struct {
	Markets    []Market `json:"markets"`
	NextCursor string   `json:"next_cursor"`
}

// ListEventsKeysetRequest Gamma GET /events/keyset（见 https://docs.polymarket.com/api-reference/events/list-events-keyset-pagination）
type ListEventsKeysetRequest struct {
	Limit       int    `json:"limit"`
	Closed      *bool  `json:"closed"`
	TagID       int    `json:"tag_id,omitempty"`
	SeriesID    int    `json:"series_id,omitempty"`
	AfterCursor string `json:"after_cursor,omitempty"`
	Order       string `json:"order,omitempty"` // 建议 id
	Ascending   *bool  `json:"ascending,omitempty"`
}

// PolymarketEventsKeyset GET /events/keyset 响应
type PolymarketEventsKeyset struct {
	Events     []PolymarketEvent `json:"events"`
	NextCursor string            `json:"next_cursor"`
}

// PolymarketEventsPagination Gamma /events/pagination 响应
type PolymarketEventsPagination struct {
	Data       []PolymarketEvent `json:"data"`
	Pagination struct {
		HasMore      bool `json:"hasMore"`
		TotalResults int  `json:"totalResults"`
	} `json:"pagination"`
}

// PolymarketEvent 事件（含嵌套 markets，与官网 sports/games 同源）
type PolymarketEvent struct {
	ID        string   `json:"id"`
	Slug      string   `json:"slug"`
	Title     string   `json:"title"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	Closed    bool     `json:"closed"`
	Markets   []Market `json:"markets"`
}

// GetMarketRequest 获取单个市场请求
type GetMarketRequest struct {
	MarketID string `json:"market_id"`
}

// GetMarketsByConditionIdsRequest 通过conditionIds批量查询市场请求
type GetMarketsByConditionIdsRequest struct {
	ConditionIDs []string `json:"condition_ids"` // conditionId数组
}

// GetOrderBookRequest 获取订单簿请求
type GetOrderBookRequest struct {
	TokenID string `json:"token_id"`
}

// BatchGetOrderBooksRequest 批量获取订单簿请求项
type BatchGetOrderBooksRequestItem struct {
	TokenID string               `json:"token_id"`       // 必需：代币ID
	Side    prediction.OrderSide `json:"side,omitempty"` // 可选：买卖方向 (BUY/SELL)
}

// BatchGetOrderBooksRequest 批量获取订单簿请求
type BatchGetOrderBooksRequest struct {
	Items []BatchGetOrderBooksRequestItem `json:"items"`
}

// GetMarketResponse 获取单个市场响应
type GetMarketResponse struct {
	Market Market `json:"market"`
}

// GetOrderBookResponse 获取订单簿响应
type GetOrderBookResponse struct {
	OrderBook OrderBookResponse `json:"order_book"`
}

// BatchGetOrderBooksResponse 批量获取订单簿响应
type BatchGetOrderBooksResponse struct {
	OrderBooks []OrderBookResponse `json:"order_books"`
}

// PlaceOrderRequest 下单请求（旧批量接口用）
type PlaceOrderRequest struct {
	TokenID string               `json:"token_id"` // 代币ID
	Price   string               `json:"price"`    // 价格（限价单必填）
	Size    string               `json:"size"`     // 数量
	Side    prediction.OrderSide `json:"side"`     // 买卖方向
	Type    prediction.OrderType `json:"type"`     // 订单类型
	Nonce   int64                `json:"nonce"`    // 随机数
}

// PlaceOrderReq 下单请求（CLOB 单笔/限价市价）
type PlaceOrderReq struct {
	TokenID      string                         `json:"token_id"`      // 代币ID
	LimitPrice   string                         `json:"limit_price"`   // 限价单价格
	LimitSize    string                         `json:"limit_size"`    // 限价单数量
	MarketAmount string                         `json:"market_amount"` // 市价单金额（买入时使用）
	MarketSize   string                         `json:"market_size"`   // 市价单数量（卖出时使用）
	Side         prediction.OrderSide           `json:"side"`          // 买卖方向: BUY/SELL
	OrderType    prediction.PolymarketOrderType `json:"order_type"`    // 订单类型: GTC/GTD/FOK/FAK
	PostOnly     bool                           `json:"post_only"`     // 是否只挂单不立即成交
	NegRisk      bool                           `json:"neg_risk"`      // 是否为 Neg Risk 市场
	IsMarket     bool                           `json:"is_market"`     // 是否为市价单
	// FeeAmount/FeeRateBps 仅兼容旧调用方，V2 起不再进入订单签名
	FeeAmount  string `json:"fee_amount,omitempty"`
	FeeRateBps string `json:"fee_rate_bps,omitempty"`
	// BuilderCode V2 builder code，bytes32，可选
	BuilderCode string `json:"builder_code,omitempty"`
	// MarketWorstPrice 市价单最差成交价：买用 best ask、卖用 best bid；超出 [0.01,0.99] 时用实际价
	MarketWorstPrice string `json:"market_worst_price"`
}

// PlaceOrderResp 下单响应（CLOB）
type PlaceOrderResp struct {
	OrderID            string                 `json:"order_id"`
	Status             prediction.OrderStatus `json:"status"`
	Message            string                 `json:"message"`
	ErrorMsg           string                 `json:"error_msg,omitempty"`
	FeeRateBps         string                 `json:"fee_rate_bps,omitempty"`
	TransactionsHashes []string               `json:"transactions_hashes,omitempty"` // 链上结算 tx hash（市价单可能多个）
}

// GetActiveOrdersReq 查询挂单请求（CLOB）
type GetActiveOrdersReq struct {
	Market  string `json:"market,omitempty"`
	TokenID string `json:"token_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

// GetActiveOrdersResp 查询挂单响应（CLOB）
type GetActiveOrdersResp struct {
	Orders []ActiveOrder `json:"orders"`
	Total  int           `json:"total"`
}

// ActiveOrder 活跃订单（CLOB）
type ActiveOrder struct {
	ID            string                 `json:"id"`
	TokenID       string                 `json:"token_id"`
	AssetID       string                 `json:"asset_id"`
	Market        string                 `json:"market"`
	Side          prediction.OrderSide   `json:"side"`
	Price         string                 `json:"price"`
	Size          string                 `json:"size"`
	OriginalSize  string                 `json:"original_size"`
	SizeMatched   string                 `json:"size_matched"`
	RemainingSize string                 `json:"remaining_size"`
	Status        prediction.OrderStatus `json:"status"`
	OrderType     string                 `json:"order_type"`
	CreatedAt     time.Time              `json:"created_at"`
	Expiration    time.Time              `json:"expiration,omitempty"`
	Outcome       string                 `json:"outcome"`
	Owner         string                 `json:"owner"`
}

// GetTradesReq 查询成交记录请求（CLOB）
type GetTradesReq struct {
	TradeID string    `json:"trade_id,omitempty"`
	Market  string    `json:"market,omitempty"`
	TokenID string    `json:"token_id,omitempty"`
	Limit   int       `json:"limit,omitempty"`
	Offset  int       `json:"offset,omitempty"`
	Before  time.Time `json:"before,omitempty"`
	After   time.Time `json:"after,omitempty"`
}

// GetTradesResp 查询成交记录响应（CLOB）
type GetTradesResp struct {
	Trades []Trade `json:"trades"`
	Total  int     `json:"total"`
}

// Trade 成交记录（CLOB）
type Trade struct {
	ID              string    `json:"id"`
	TakerOrderID    string    `json:"taker_order_id"`
	Market          string    `json:"market"`
	AssetID         string    `json:"asset_id"`
	Side            string    `json:"side"`
	Size            string    `json:"size"`
	Price           string    `json:"price"`
	FeeRateBps      string    `json:"fee_rate_bps"`
	Status          string    `json:"status"`
	MatchTime       time.Time `json:"match_time"`
	LastUpdate      time.Time `json:"last_update"`
	Outcome         string    `json:"outcome"`
	MakerAddress    string    `json:"maker_address"`
	Owner           string    `json:"owner"`
	TransactionHash string    `json:"transaction_hash"`
	BucketIndex     int       `json:"bucket_index"`
}

// GetOrderReq 查询单个订单请求（CLOB）
type GetOrderReq struct {
	OrderID string `json:"order_id"`
}

// GetOrderResp 查询单个订单响应（CLOB）
type GetOrderResp struct {
	Order OpenOrder `json:"order"`
}

// OpenOrder 订单信息（CLOB API）
type OpenOrder struct {
	AssociateTrades []string               `json:"associate_trades"`
	ID              string                 `json:"id"`
	Status          prediction.OrderStatus `json:"status"`
	Market          string                 `json:"market"`
	OriginalSize    string                 `json:"original_size"`
	Outcome         string                 `json:"outcome"`
	MakerAddress    string                 `json:"maker_address"`
	Owner           string                 `json:"owner"`
	Price           string                 `json:"price"`
	Side            string                 `json:"side"`
	SizeMatched     string                 `json:"size_matched"`
	AssetID         string                 `json:"asset_id"`
	Expiration      string                 `json:"expiration"`
	Type            string                 `json:"type"`
	OrderType       string                 `json:"order_type"`
	CreatedAt       int64                  `json:"created_at"` // CLOB 返回 Unix 时间戳（数字）
}

// Order 订单结构体
type Order struct {
	ID              string                 `json:"id"`
	TokenID         string                 `json:"token_id"`
	AssetID         string                 `json:"asset_id"` // 根据官方文档添加
	Market          string                 `json:"market"`   // 根据官方文档添加
	Price           string                 `json:"price"`
	Size            string                 `json:"size"`
	OriginalSize    string                 `json:"original_size"` // 根据官方文档添加
	SizeMatched     string                 `json:"size_matched"`  // 根据官方文档添加
	Side            prediction.OrderSide   `json:"side"`
	Type            prediction.OrderType   `json:"type"`
	Status          prediction.OrderStatus `json:"status"`
	FilledSize      string                 `json:"filled_size"`
	RemainingSize   string                 `json:"remaining_size"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Signature       string                 `json:"signature"`
	Nonce           int64                  `json:"nonce"`
	Expiration      int64                  `json:"expiration"`
	MakerAddress    string                 `json:"maker_address"`
	TakerAddress    string                 `json:"taker_address"`
	Salt            string                 `json:"salt"`
	Outcome         string                 `json:"outcome"`          // 根据官方文档添加
	Owner           string                 `json:"owner"`            // 根据官方文档添加
	AssociateTrades []string               `json:"associate_trades"` // 根据官方文档添加
}

// PlaceOrderResponse 下单响应
type PlaceOrderResponse struct {
	OrderID            string                 `json:"orderID"`
	Status             prediction.OrderStatus `json:"status"`
	Message            string                 `json:"message"`
	TransactionsHashes []string               `json:"transactionsHashes"`
	TakingAmount       string                 `json:"takingAmount"`
	MakingAmount       string                 `json:"makingAmount"`
	ErrorMsg           string                 `json:"errorMsg"`
}

// GetOrderRequest 获取订单请求
type GetOrderRequest struct {
	OrderID string `json:"order_id"`
}

// GetActiveOrdersRequest 获取活跃订单请求
type GetActiveOrdersRequest struct {
	TokenID string `json:"token_id,omitempty"` // 可选，过滤特定代币
	Limit   int    `json:"limit,omitempty"`    // 可选，限制返回数量
	Offset  int    `json:"offset,omitempty"`   // 可选，分页偏移
}

// GetActiveOrdersResponse 获取活跃订单响应
type GetActiveOrdersResponse struct {
	Orders []Order `json:"orders"`
	Total  int     `json:"total"`
}

// CancelOrderRequest 取消订单请求
type CancelOrderRequest struct {
	OrderID string `json:"order_id"`
}

// CancelOrderResponse 取消订单响应
type CancelOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// CreateAPIKeyRequest 创建API Key请求
type CreateAPIKeyRequest struct {
	// 空结构体，创建API Key只需要私钥签名认证
}

// CreateAPIKeyResponse 创建API Key响应
type CreateAPIKeyResponse struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// MagicAccountInfo Magic账户信息响应
type MagicAccountInfo struct {
	Success         bool   `json:"success"`
	EOAAddress      string `json:"eoa_address"`
	ProxyAddress    string `json:"proxy_address"`
	SignatureType   int    `json:"signature_type"`
	UseMagicAccount bool   `json:"use_magic_account"`
	Host            string `json:"host"`
	ChainID         int    `json:"chain_id"`
	ClientStatus    string `json:"client_status"`
	ClientError     string `json:"client_error"`
	USDCBalance     string `json:"usdc_balance"` // 抵押币余额（pUSD）
	ErrorMessage    string `json:"errorMsg,omitempty"`
}

// Position 持仓信息
type Position struct {
	// 基础信息
	TokenID      string  `json:"token_id"`      // 代币ID (asset)
	AssetID      string  `json:"asset_id"`      // 资产ID (asset)
	Market       string  `json:"market"`        // 市场ID (conditionId)
	Outcome      string  `json:"outcome"`       // 结果选项
	Balance      string  `json:"balance"`       // 持仓数量 (size)
	BalanceFloat float64 `json:"balance_float"` // 持仓数量（浮点数）
	MarketName   string  `json:"market_name"`   // 市场名称 (title)
	OutcomeName  string  `json:"outcome_name"`  // 结果名称 (outcome)
	CurrentPrice string  `json:"current_price"` // 当前价格 (curPrice)
	ValueUSD     string  `json:"value_usd"`     // 美元价值 (currentValue)
	PnlUSD       string  `json:"pnl_usd"`       // 盈亏（美元）(cashPnl)
	LastUpdated  string  `json:"last_updated"`  // 最后更新时间 (endDate)

	// Polymarket特有字段
	ProxyWallet        string  `json:"proxy_wallet"`         // 代理钱包地址
	ConditionID        string  `json:"condition_id"`         // 条件ID
	AvgPrice           float64 `json:"avg_price"`            // 平均价格
	InitialValue       float64 `json:"initial_value"`        // 初始价值
	CurrentValue       float64 `json:"current_value"`        // 当前价值
	CashPnl            float64 `json:"cash_pnl"`             // 现金盈亏
	PercentPnl         float64 `json:"percent_pnl"`          // 百分比盈亏
	TotalBought        float64 `json:"total_bought"`         // 总购买量
	RealizedPnl        float64 `json:"realized_pnl"`         // 已实现盈亏
	PercentRealizedPnl float64 `json:"percent_realized_pnl"` // 已实现盈亏百分比
	Redeemable         bool    `json:"redeemable"`           // 是否可赎回
	Mergeable          bool    `json:"mergeable"`            // 是否可合并
	Slug               string  `json:"slug"`                 // URL slug
	Icon               string  `json:"icon"`                 // 图标URL
	EventSlug          string  `json:"event_slug"`           // 事件slug
	OutcomeIndex       int     `json:"outcome_index"`        // 结果索引
	OppositeOutcome    string  `json:"opposite_outcome"`     // 相反结果
	OppositeAsset      string  `json:"opposite_asset"`       // 相反资产
	EndDate            string  `json:"end_date"`             // 结束日期
	NegativeRisk       bool    `json:"negative_risk"`        // 负风险
}

// GetMagicAccountPositionsResponse Magic账户持仓响应
type GetMagicAccountPositionsResponse struct {
	Positions []Position `json:"positions"`
	Total     int        `json:"total"`
	Success   bool       `json:"success"`
	Message   string     `json:"message,omitempty"`
	ErrorMsg  string     `json:"errorMsg,omitempty"`
}

// BatchOrderRequest 批量下单请求
type BatchOrderRequest struct {
	TokenID   string                         `json:"token_id"`
	Price     string                         `json:"price"`
	Size      string                         `json:"size"`
	Side      prediction.OrderSide           `json:"side"`
	OrderType prediction.PolymarketOrderType `json:"order_type,omitempty"` // 可选，默认为 GTC
}

// SingleOrderRequest 单笔下单请求（市价单传金额+份额，限价单传价格+份额）
type SingleOrderRequest struct {
	TokenID  string               `json:"token_id"`
	Price    string               `json:"price"`
	Size     string               `json:"size"`
	Amount   string               `json:"amount,omitempty"` // 市价单时必传，与 Size 同时传表示金额+份额
	Side     prediction.OrderSide `json:"side"`
	IsMarket bool                 `json:"is_market"` // true=市价单（需 Amount+Size），false=限价单（需 Price+Size）
}

// BatchPlaceOrderRequest 批量下单请求
type BatchPlaceOrderRequest struct {
	Orders []BatchOrderRequest `json:"orders"`
}

// BatchPlaceOrderResponse 批量下单响应
type BatchPlaceOrderResponse struct {
	Orders     []PlaceOrderResponse `json:"orders"`
	Total      int                  `json:"total"`
	Successful int                  `json:"successful"`
	Failed     int                  `json:"failed"`
	Success    bool                 `json:"success"`
	Message    string               `json:"message,omitempty"`
	ErrorMsg   string               `json:"errorMsg,omitempty"`
}

// BatchCancelOrderRequest 批量取消订单请求
type BatchCancelOrderRequest struct {
	OrderIDs []string `json:"order_ids"`
}

// BatchCancelOrderResponse 批量取消订单响应
type BatchCancelOrderResponse struct {
	Orders     []CancelOrderResponse `json:"orders"`
	Total      int                   `json:"total"`
	Successful int                   `json:"successful"`
	Failed     int                   `json:"failed"`
	Success    bool                  `json:"success"`
	Message    string                `json:"message,omitempty"`
	ErrorMsg   string                `json:"errorMsg,omitempty"`
}

// GetTradeHistoryRequest 获取成交历史请求
type GetTradeHistoryRequest struct {
	Before  *int64   `json:"before"`            // Unix时间戳，获取此时间之前的成交记录
	After   *int64   `json:"after"`             // Unix时间戳，获取此时间之后的成交记录
	Market  *string  `json:"market"`            // 市场ID（condition ID），可选
	Markets []string `json:"markets,omitempty"` // 多个 condition ID；Data API v2 单次最多支持20个
	Type    *string  `json:"type,omitempty"`    // activity 类型过滤，如 TRADE
	Limit   *int     `json:"limit,omitempty"`   // 单页大小（1..1000）；支持 cursor 翻页
}

// TradeRecord 成交记录
type TradeRecord struct {
	OrderID         string                 `json:"order_id"`
	TradeID         string                 `json:"trade_id"`
	TokenID         string                 `json:"token_id"`
	AssetID         string                 `json:"asset_id"`
	Market          string                 `json:"market"`
	Side            prediction.OrderSide   `json:"side"`
	Price           string                 `json:"price"`
	Size            string                 `json:"size"`
	FilledSize      string                 `json:"filled_size"`
	Status          prediction.OrderStatus `json:"status"`
	Outcome         string                 `json:"outcome"`
	OrderType       string                 `json:"order_type"`
	CreatedAt       time.Time              `json:"created_at"`
	FilledAt        time.Time              `json:"filled_at"`
	TransactionHash string                 `json:"transaction_hash"`
	Fee             string                 `json:"fee"`
	TakerOrderId    string                 `json:"taker_order_id"` // Taker订单ID
	MakerOrderId    string                 `json:"maker_order_id"` // Maker订单ID
}

// RawTradeRecord 原始成交记录（保持API原始结构）
type RawTradeRecord struct {
	ID              string       `json:"id"`
	TakerOrderId    string       `json:"taker_order_id"`
	Market          string       `json:"market"`
	AssetId         string       `json:"asset_id"`
	Side            string       `json:"side"`
	Size            string       `json:"size"`
	UsdcSize        string       `json:"usdc_size,omitempty"`
	FeeRateBps      string       `json:"fee_rate_bps"`
	Price           string       `json:"price"`
	Status          string       `json:"status"`
	MatchTime       string       `json:"match_time"`
	LastUpdate      string       `json:"last_update"`
	Outcome         string       `json:"outcome"`
	MakerAddress    string       `json:"maker_address"`
	Owner           string       `json:"owner"`
	TransactionHash string       `json:"transaction_hash"`
	BucketIndex     int          `json:"bucket_index"`
	MakerOrders     []MakerOrder `json:"maker_orders"`
	Type            string       `json:"type"`
}

// MakerOrder 原始成交记录中的Maker订单
type MakerOrder struct {
	OrderId       string `json:"order_id"`
	MakerAddress  string `json:"maker_address"`
	Owner         string `json:"owner"`
	MatchedAmount string `json:"matched_amount"`
	FeeRateBps    string `json:"fee_rate_bps"`
	Price         string `json:"price"`
	AssetId       string `json:"asset_id"`
	Outcome       string `json:"outcome"`
	Side          string `json:"side"`
}

// GetTradeHistoryResponse 获取成交历史响应（返回原始结构）
type GetTradeHistoryResponse struct {
	Trades   []RawTradeRecord `json:"trades"`
	Total    int              `json:"total"`
	Success  bool             `json:"success"`
	Message  string           `json:"message,omitempty"`
	ErrorMsg string           `json:"errorMsg,omitempty"`
}

// TokenSupply 代币供应量信息
type TokenSupply struct {
	TokenID     string `json:"token_id"`     // 代币ID
	ConditionID string `json:"condition_id"` // 条件ID
	TotalSupply string `json:"total_supply"` // 总供应量
	Decimals    int    `json:"decimals"`     // 小数位数
	Symbol      string `json:"symbol"`       // 代币符号
	MarketID    string `json:"market_id"`    // 市场ID
}

// GetTokenSupplyRequest 获取代币供应量请求
type GetTokenSupplyRequest struct {
	TokenID string `json:"token_id"` // 代币ID
}

// GetTokenSupplyResponse 获取代币供应量响应
type GetTokenSupplyResponse struct {
	TokenSupply *TokenSupply `json:"token_supply"`
	Success     bool         `json:"success"`
	Message     string       `json:"message,omitempty"`
	ErrorMsg    string       `json:"errorMsg,omitempty"`
}

// GetEventBySlugRequest 根据slug获取事件请求
type GetEventBySlugRequest struct {
	SlugName string `json:"slug_name"` // 事件slug名称，例如：oscars-2026-best-picture-winner
}

// EventMarket 事件中的市场信息
type EventMarket struct {
	ID                           string    `json:"id"`
	Question                     string    `json:"question"`
	ConditionID                  string    `json:"conditionId"`
	Slug                         string    `json:"slug"`
	ResolutionSource             string    `json:"resolutionSource"`
	EndDate                      string    `json:"endDate"`
	Liquidity                    string    `json:"liquidity"`
	StartDate                    string    `json:"startDate"`
	Image                        string    `json:"image"`
	Icon                         string    `json:"icon"`
	Description                  string    `json:"description"`
	Outcomes                     string    `json:"outcomes"`
	OutcomePrices                string    `json:"outcomePrices"`
	Volume                       string    `json:"volume"`
	Active                       bool      `json:"active"`
	Closed                       bool      `json:"closed"`
	MarketMakerAddress           string    `json:"marketMakerAddress"`
	CreatedAt                    time.Time `json:"createdAt"`
	UpdatedAt                    time.Time `json:"updatedAt"`
	New                          bool      `json:"new"`
	Featured                     bool      `json:"featured"`
	SubmittedBy                  string    `json:"submitted_by"`
	Archived                     bool      `json:"archived"`
	ResolvedBy                   string    `json:"resolvedBy"`
	Restricted                   bool      `json:"restricted"`
	GroupItemTitle               string    `json:"groupItemTitle"`
	GroupItemThreshold           string    `json:"groupItemThreshold"`
	QuestionID                   string    `json:"questionID"`
	EnableOrderBook              bool      `json:"enableOrderBook"`
	OrderPriceMinTickSize        float64   `json:"orderPriceMinTickSize"`
	OrderMinSize                 float64   `json:"orderMinSize"`
	VolumeNum                    float64   `json:"volumeNum"`
	LiquidityNum                 float64   `json:"liquidityNum"`
	EndDateIso                   string    `json:"endDateIso"`
	StartDateIso                 string    `json:"startDateIso"`
	HasReviewedDates             bool      `json:"hasReviewedDates"`
	Volume24hr                   float64   `json:"volume24hr"`
	Volume1wk                    float64   `json:"volume1wk"`
	Volume1mo                    float64   `json:"volume1mo"`
	Volume1yr                    float64   `json:"volume1yr"`
	ClobTokenIds                 string    `json:"clobTokenIds"`
	UmaBond                      string    `json:"umaBond"`
	UmaReward                    string    `json:"umaReward"`
	Volume24hrClob               float64   `json:"volume24hrClob"`
	Volume1wkClob                float64   `json:"volume1wkClob"`
	Volume1moClob                float64   `json:"volume1moClob"`
	Volume1yrClob                float64   `json:"volume1yrClob"`
	VolumeClob                   float64   `json:"volumeClob"`
	LiquidityClob                float64   `json:"liquidityClob"`
	CustomLiveness               int       `json:"customLiveness"`
	AcceptingOrders              bool      `json:"acceptingOrders"`
	NegRisk                      bool      `json:"negRisk"`
	NegRiskMarketID              string    `json:"negRiskMarketID"`
	NegRiskRequestID             string    `json:"negRiskRequestID"`
	Ready                        bool      `json:"ready"`
	Funded                       bool      `json:"funded"`
	AcceptingOrdersTimestamp     string    `json:"acceptingOrdersTimestamp"`
	Cyom                         bool      `json:"cyom"`
	Competitive                  float64   `json:"competitive"`
	PagerDutyNotificationEnabled bool      `json:"pagerDutyNotificationEnabled"`
	Approved                     bool      `json:"approved"`
	RewardsMinSize               float64   `json:"rewardsMinSize"`
	RewardsMaxSpread             float64   `json:"rewardsMaxSpread"`
	Spread                       float64   `json:"spread"`
	OneDayPriceChange            float64   `json:"oneDayPriceChange,omitempty"`
	OneHourPriceChange           float64   `json:"oneHourPriceChange,omitempty"`
	OneWeekPriceChange           float64   `json:"oneWeekPriceChange,omitempty"`
	OneMonthPriceChange          float64   `json:"oneMonthPriceChange,omitempty"`
	OneYearPriceChange           float64   `json:"oneYearPriceChange,omitempty"`
	LastTradePrice               float64   `json:"lastTradePrice"`
	BestBid                      float64   `json:"bestBid"`
	BestAsk                      float64   `json:"bestAsk"`
	AutomaticallyActive          bool      `json:"automaticallyActive"`
	ClearBookOnStart             bool      `json:"clearBookOnStart"`
	SeriesColor                  string    `json:"seriesColor"`
	ShowGmpSeries                bool      `json:"showGmpSeries"`
	ShowGmpOutcome               bool      `json:"showGmpOutcome"`
	ManualActivation             bool      `json:"manualActivation"`
	NegRiskOther                 bool      `json:"negRiskOther"`
	UmaResolutionStatuses        string    `json:"umaResolutionStatuses"`
	PendingDeployment            bool      `json:"pendingDeployment"`
	Deploying                    bool      `json:"deploying"`
	DeployingTimestamp           string    `json:"deployingTimestamp"`
	RfqEnabled                   bool      `json:"rfqEnabled"`
	HoldingRewardsEnabled        bool      `json:"holdingRewardsEnabled"`
	FeesEnabled                  bool      `json:"feesEnabled"`
}

// EventTag 事件标签
type EventTag struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	Slug        string    `json:"slug"`
	ForceShow   bool      `json:"forceShow"`
	PublishedAt string    `json:"publishedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   int       `json:"updatedBy,omitempty"`
}

// Event 事件信息
type Event struct {
	ID                  string        `json:"id"`
	Ticker              string        `json:"ticker"`
	Slug                string        `json:"slug"`
	Title               string        `json:"title"`
	Description         string        `json:"description"`
	ResolutionSource    string        `json:"resolutionSource"`
	StartDate           string        `json:"startDate"`
	CreationDate        string        `json:"creationDate"`
	EndDate             string        `json:"endDate"`
	Image               string        `json:"image"`
	Icon                string        `json:"icon"`
	Active              bool          `json:"active"`
	Closed              bool          `json:"closed"`
	Archived            bool          `json:"archived"`
	New                 bool          `json:"new"`
	Featured            bool          `json:"featured"`
	Restricted          bool          `json:"restricted"`
	Liquidity           float64       `json:"liquidity"`
	Volume              float64       `json:"volume"`
	OpenInterest        float64       `json:"openInterest"`
	SortBy              string        `json:"sortBy"`
	CreatedAt           time.Time     `json:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt"`
	Competitive         float64       `json:"competitive"`
	Volume24hr          float64       `json:"volume24hr"`
	Volume1wk           float64       `json:"volume1wk"`
	Volume1mo           float64       `json:"volume1mo"`
	Volume1yr           float64       `json:"volume1yr"`
	EnableOrderBook     bool          `json:"enableOrderBook"`
	LiquidityClob       float64       `json:"liquidityClob"`
	NegRisk             bool          `json:"negRisk"`
	NegRiskMarketID     string        `json:"negRiskMarketID"`
	CommentCount        int           `json:"commentCount"`
	Markets             []EventMarket `json:"markets"`
	Tags                []EventTag    `json:"tags"`
	Cyom                bool          `json:"cyom"`
	ShowAllOutcomes     bool          `json:"showAllOutcomes"`
	ShowMarketImages    bool          `json:"showMarketImages"`
	EnableNegRisk       bool          `json:"enableNegRisk"`
	AutomaticallyActive bool          `json:"automaticallyActive"`
	GmpChartMode        string        `json:"gmpChartMode"`
	NegRiskAugmented    bool          `json:"negRiskAugmented"`
	PendingDeployment   bool          `json:"pendingDeployment"`
	Deploying           bool          `json:"deploying"`
	DeployingTimestamp  string        `json:"deployingTimestamp"`
}

// GetEventBySlugResponse 根据slug获取事件响应
type GetEventBySlugResponse struct {
	Event *Event `json:"event"`
}
