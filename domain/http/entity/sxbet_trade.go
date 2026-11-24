package entity

// SxbetAccount SX Bet 发单身份（EOA 私钥签名 + API Key；交易走已部署 proxy）。
type SxbetAccount struct {
	APIKey       string
	PrivateKey   string
	EOAAddress   string
	ProxyAddress string
}

// SxbetMetadataObv3 GET /metadata/obv3 data。
type SxbetMetadataObv3 struct {
	ChainID     int64                `json:"chainId"`
	Domain      SxbetEIP712Domain    `json:"domain"`
	ActiveAsset SxbetActiveAssetMeta `json:"activeAsset"`
}

// SxbetEIP712Domain EIP-712 domain（直接用于签名）。
type SxbetEIP712Domain struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	ChainID           int64  `json:"chainId"`
	VerifyingContract string `json:"verifyingContract"`
}

// SxbetActiveAssetMeta 结算资产。
type SxbetActiveAssetMeta struct {
	Symbol        string `json:"symbol"`
	BaseToken     string `json:"baseToken"`
	EscrowAddress string `json:"escrowAddress"`
	Decimals      int    `json:"decimals"`
}

// SxbetMetadataResponse GET /metadata/obv3。
type SxbetMetadataResponse struct {
	Status string            `json:"status"`
	Data   SxbetMetadataObv3 `json:"data"`
}

// SxbetCreateOrderItem POST /orders-v3 单笔。
type SxbetCreateOrderItem struct {
	MarketHash              string `json:"marketHash"`
	Maker                   string `json:"maker"`
	TotalBetSize            string `json:"totalBetSize"`
	PercentageOdds          string `json:"percentageOdds"`
	Salt                    string `json:"salt"`
	Expiry                  int64  `json:"expiry"`
	BaseToken               string `json:"baseToken"`
	IsMakerBettingOutcomeOne bool  `json:"isMakerBettingOutcomeOne"`
	TimeInForce             string `json:"timeInForce"`
	OrderSignature          string `json:"orderSignature"`
	ClientOrderID           string `json:"clientOrderId,omitempty"`
}

// SxbetCreateOrdersRequest POST /orders-v3。
type SxbetCreateOrdersRequest struct {
	Orders         []SxbetCreateOrderItem `json:"orders"`
	WaitForOutcome bool                   `json:"waitForOutcome,omitempty"`
	MaxWaitTime    int                    `json:"maxWaitTime,omitempty"`
}

// SxbetCreateOrderOutcome waitForOutcome 时的匹配结果。
type SxbetCreateOrderOutcome struct {
	State           string   `json:"state"`
	RemainingAmount string   `json:"remainingAmount,omitempty"`
	FillAmount      string   `json:"fillAmount,omitempty"`
	BlendedOdds     string   `json:"blendedOdds,omitempty"`
	MatchIDs        []string `json:"matchIds,omitempty"`
	TradeID         string   `json:"tradeId,omitempty"`
	CancelReason    string   `json:"cancelReason,omitempty"`
}

// SxbetCreateOrderResult 单笔提交结果。
type SxbetCreateOrderResult struct {
	OrderID       string                   `json:"orderId"`
	Status        string                   `json:"status"`
	FailureReason string                   `json:"failureReason,omitempty"`
	CommandID     string                   `json:"commandId"`
	ClientOrderID string                   `json:"clientOrderId,omitempty"`
	Outcome       *SxbetCreateOrderOutcome `json:"outcome,omitempty"`
}

// SxbetCreateOrdersResponse POST /orders-v3。
type SxbetCreateOrdersResponse struct {
	Status string `json:"status"`
	Data   struct {
		Orders []SxbetCreateOrderResult `json:"orders"`
	} `json:"data"`
}

// SxbetPlaceOrderReq 策略侧发单请求（LIMIT + IOC ≈ FAK）。
type SxbetPlaceOrderReq struct {
	MarketHash string
	BuyYes     bool    // true=outcomeOne；false=outcomeTwo(NO)
	Price      float64 // taker 限价 0~1
	Quantity   float64 // 策略份额（≈ USDC stake）
}
