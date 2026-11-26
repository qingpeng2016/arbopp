package entity

// PredictEnvelope 多数 REST 响应外包。
type PredictEnvelope struct {
	Success bool   `json:"success"`
	Cursor  string `json:"cursor"`
	Code    int    `json:"code"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// PredictTag category 上的标签。
type PredictTag struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Level    *int   `json:"level"`
	ParentID string `json:"parentId"`
}

// PredictOutcome 单个 outcome（Yes/No 等）。
type PredictOutcome struct {
	Name      string `json:"name"`
	IndexSet  int    `json:"indexSet"`
	OnChainID string `json:"onChainId"`
	Status    string `json:"status"`
}

// PredictAccount Predict.fun 下单身份（Privy EOA + 可选 Predict 智能账户）。
type PredictAccount struct {
	APIKey         string
	PrivateKey     string
	PredictAccount string // 空则 EOA 直连；否则 maker/signer 为此地址
	EOAAddress     string
}

// PredictMarket REST Market。
type PredictMarket struct {
	ID               int64            `json:"id"`
	Title            string           `json:"title"`
	Question         string           `json:"question"`
	Description      string           `json:"description"`
	TradingStatus    string           `json:"tradingStatus"`
	Status           string           `json:"status"`
	CategorySlug     string           `json:"categorySlug"`
	MarketVariant    string           `json:"marketVariant"`
	MarketType       string           `json:"marketType"`
	ConditionID      string           `json:"conditionId"`
	FeeRateBps       int              `json:"feeRateBps"`
	IsNegRisk        bool             `json:"isNegRisk"`
	IsYieldBearing   bool             `json:"isYieldBearing"`
	DecimalPrecision int              `json:"decimalPrecision"`
	Outcomes         []PredictOutcome `json:"outcomes"`
	CreatedAt        string           `json:"createdAt"`
}

// PredictGetMarketResponse GET /markets/{id}。
type PredictGetMarketResponse struct {
	PredictEnvelope
	Data PredictMarket `json:"data"`
}

// PredictGetMarketOrderbookResponse GET /markets/{id}/orderbook。
type PredictGetMarketOrderbookResponse struct {
	PredictEnvelope
	Data PredictOrderbook `json:"data"`
}

// PredictCategory 含嵌套 markets 的 category（场次）。
type PredictCategory struct {
	ID            int64           `json:"id"`
	Slug          string          `json:"slug"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	MarketVariant string          `json:"marketVariant"`
	StartsAt      string          `json:"startsAt"`
	EndsAt        string          `json:"endsAt"`
	Status        string          `json:"status"`
	IsVisible     bool            `json:"isVisible"`
	Markets       []PredictMarket `json:"markets"`
	Tags          []PredictTag    `json:"tags"`
}

// PredictGetCategoriesResponse GET /categories。
type PredictGetCategoriesResponse struct {
	PredictEnvelope
	Data []PredictCategory `json:"data"`
}

// PredictOrderbook GET /markets/{id}/orderbook 与批量 /markets/orderbooks 中的一本（YES 视角）。
type PredictOrderbook struct {
	MarketID          int64       `json:"marketId"`
	UpdateTimestampMs int64       `json:"updateTimestampMs"`
	Asks              [][]float64 `json:"asks"`
	Bids              [][]float64 `json:"bids"`
}

// PredictGetOrderbooksResponse GET /v1/markets/orderbooks。
type PredictGetOrderbooksResponse struct {
	PredictEnvelope
	Data []PredictOrderbook `json:"data"`
}

// PredictSignedOrder REST 发单请求内嵌订单（EIP-712 字段）。
type PredictSignedOrder struct {
	Salt           string `json:"salt"`
	Maker          string `json:"maker"`
	Signer         string `json:"signer"`
	Taker          string `json:"taker"`
	TokenID        string `json:"tokenId"`
	MakerAmount    string `json:"makerAmount"`
	TakerAmount    string `json:"takerAmount"`
	Expiration     int64  `json:"expiration"`
	Nonce          string `json:"nonce"`
	FeeRateBps     string `json:"feeRateBps"`
	Side           int    `json:"side"`
	SignatureType  int    `json:"signatureType"`
	Signature      string `json:"signature"`
	Hash           string `json:"hash,omitempty"`
}

// PredictCreateOrderBody POST /orders 请求体。
type PredictCreateOrderBody struct {
	Data PredictCreateOrderData `json:"data"`
}

// PredictCreateOrderData 市价单字段。
type PredictCreateOrderData struct {
	Order          PredictSignedOrder `json:"order"`
	PricePerShare  string             `json:"pricePerShare"`
	Strategy       string             `json:"strategy"`
	SlippageBps    string             `json:"slippageBps,omitempty"`
	IsMinAmountOut bool               `json:"isMinAmountOut,omitempty"`
}

// PredictCreateOrderResponse POST /orders 响应。
type PredictCreateOrderResponse struct {
	PredictEnvelope
	Data struct {
		OrderID string `json:"orderId"`
		OrderHash string `json:"orderHash"`
	} `json:"data"`
}

// PredictGetOrderResponse GET /orders/{hash}。
type PredictGetOrderResponse struct {
	PredictEnvelope
	Data PredictOrderData `json:"data"`
}

// PredictOrderData 查单 data。
type PredictOrderData struct {
	Order        PredictSignedOrder `json:"order"`
	ID           string             `json:"id"`
	MarketID     int64              `json:"marketId"`
	Amount       string             `json:"amount"`
	AmountFilled string             `json:"amountFilled"`
	Strategy     string             `json:"strategy"`
	Status       string             `json:"status"`
}

// PredictGetMatchesResponse GET /orders/matches。
type PredictGetMatchesResponse struct {
	PredictEnvelope
	Data []PredictMatchEvent `json:"data"`
}

// PredictMatchEvent 撮合事件（手续费字段做宽松解析）。
type PredictMatchEvent struct {
	AmountFilled    string           `json:"amountFilled"` // 本笔撮合份额 wei
	TakerFeeAmount  string           `json:"takerFeeAmount"`
	MakerFeeAmounts []string         `json:"makerFeeAmounts"`
	TakerFee        *PredictMatchFee `json:"takerFee"`
	Makers          []struct {
		Fee *PredictMatchFee `json:"fee"`
	} `json:"makers"`
}

// PredictMatchFee 单笔 fee。
type PredictMatchFee struct {
	Amount string `json:"amount"`
	Type   string `json:"type"` // SHARES | COLLATERAL
}