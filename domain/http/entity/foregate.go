package entity

import (
	"encoding/json"
	"strconv"
)

// ForegateAccount Foregate账户信息
type ForegateAccount struct {
	UserID      int    `json:"user_id"`
	APIKey      string `json:"api_key"`
	APISecret   string `json:"api_secret"`
	AccountType string `json:"account_type"` // "buy" 或 "sell"
	Email       string `json:"email"`
	Password    string `json:"password"`
}

// AccountKeyResponse foregate 账户密钥响应结构
type AccountKeyResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    string `json:"data"` // 私钥字符串（没有0x前缀）
}

// MarketOutcomeResponse 市场详情及盘口结果
type MarketOutcomeResponse struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data *MarketDetail `json:"data"`
}

type MarketDetail struct {
	MarketID          string          `json:"marketId"`
	CreatorID         string          `json:"creatorId"`
	Title             string          `json:"title"`
	Status            string          `json:"status"`
	ModelType         string          `json:"modelType"`
	MarketType        string          `json:"marketType"`
	CoinID            int64           `json:"coinId"`
	IsEnableTrans     bool            `json:"isEnableTrans"`
	DisplayText       *string         `json:"displayText"`
	DisplayTextZH     *string         `json:"displayTextZH"`
	MarketLogo        string          `json:"marketLogo"`
	StartTime         string          `json:"startTime"`
	EndTime           string          `json:"endTime"`
	IsResolveOvertime bool            `json:"isResolveOvertime"`
	Outcomes          []MarketOutcome `json:"outcomes"`
}

type MarketOutcome struct {
	OutcomeID   int64           `json:"outcomeId"`
	Title       string          `json:"title"`
	Name        string          `json:"name"`
	Chance      json.Number     `json:"chance"`
	OutcomeLogo string          `json:"outcomeLogo"`
	Description *string         `json:"description"`
	Options     []OutcomeOption `json:"options"`
}

type OutcomeOption struct {
	OptionID    int64       `json:"optionId"`
	Title       string      `json:"title"`
	Chance      json.Number `json:"chance"`
	Price       json.Number `json:"price"`
	PriceBuy    json.Number `json:"priceBuy"`
	PriceSell   json.Number `json:"priceSell"`
	IsWinFirst  *bool       `json:"isWinFirst"`
	IsWin       *bool       `json:"isWin"`
	OptionIndex int         `json:"optionIndex"`
}

// UnmarshalJSON 自定义 JSON 反序列化，处理 isWinFirst 和 isWin 字段可能是字符串 "1" 的情况
func (o *OutcomeOption) UnmarshalJSON(data []byte) error {
	// 定义一个临时结构体来处理原始 JSON
	type Alias OutcomeOption
	aux := &struct {
		IsWinFirst interface{} `json:"isWinFirst"`
		IsWin      interface{} `json:"isWin"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 处理 IsWinFirst 字段
	if aux.IsWinFirst != nil {
		switch v := aux.IsWinFirst.(type) {
		case bool:
			o.IsWinFirst = &v
		case string:
			if v == "1" || v == "true" {
				val := true
				o.IsWinFirst = &val
			} else if v == "0" || v == "false" {
				val := false
				o.IsWinFirst = &val
			} else {
				// 尝试解析为数字
				if num, err := strconv.Atoi(v); err == nil {
					val := num != 0
					o.IsWinFirst = &val
				}
			}
		case float64:
			val := v != 0
			o.IsWinFirst = &val
		}
	}

	// 处理 IsWin 字段
	if aux.IsWin != nil {
		switch v := aux.IsWin.(type) {
		case bool:
			o.IsWin = &v
		case string:
			if v == "1" || v == "true" {
				val := true
				o.IsWin = &val
			} else if v == "0" || v == "false" {
				val := false
				o.IsWin = &val
			} else {
				// 尝试解析为数字
				if num, err := strconv.Atoi(v); err == nil {
					val := num != 0
					o.IsWin = &val
				}
			}
		case float64:
			val := v != 0
			o.IsWin = &val
		}
	}

	return nil
}

// PositionsSearchResponse 持仓查询响应结构
type PositionsSearchResponse struct {
	Code     int                  `json:"code"`
	Msg      string               `json:"msg"`
	Current  int                  `json:"current"`
	Size     int                  `json:"size"`
	Total    int                  `json:"total"`
	Pages    int                  `json:"pages"`
	Data     []PositionSearchItem `json:"data"`
	TotalRow *int                 `json:"totalRow"`
	Empty    bool                 `json:"empty"`
}

// PositionSearchItem 持仓查询项
type PositionSearchItem struct {
	MarketID           int64        `json:"marketId"`
	MarketTitle        string       `json:"marketTitle"`
	MarketLogo         string       `json:"marketLogo"`
	MarketDescription  string       `json:"marketDescription"`
	ModelType          string       `json:"modelType"`
	MarketStatus       string       `json:"marketStatus"`
	IsResolveOvertime  bool         `json:"isResolveOvertime"`
	MarketStartTime    string       `json:"marketStartTime"`
	MarketEndTime      string       `json:"marketEndTime"`
	ClaimFee           json.Number  `json:"claimFee"`
	OrderID            string       `json:"orderId"`
	OrderType          string       `json:"orderType"`
	OrderTitle         string       `json:"orderTitle"`
	OrderStatus        string       `json:"orderStatus"`
	Shares             json.Number  `json:"shares"`
	AvailableShare     json.Number  `json:"availableShare"`
	AveragePrice       json.Number  `json:"averagePrice"`
	CostValue          json.Number  `json:"costValue"`
	WinPrice           json.Number  `json:"winPrice"`
	CurrentValue       json.Number  `json:"currentValue"`
	CurrentPrice       json.Number  `json:"currentPrice"`
	SettlementResult   int          `json:"settlementResult"`
	Win                bool         `json:"win"`
	Yes                bool         `json:"yes"`
	Pnl                json.Number  `json:"pnl"`
	OffWinningPool     *string      `json:"offWinningPool"`
	Fees               *json.Number `json:"fees"`
	Chance             json.Number  `json:"chance"`
	OutcomeID          int64        `json:"outcomeId"`
	OutComeName        string       `json:"outComeName"`
	OutComeDescription string       `json:"outComeDescription"`
	OutComeLogoURL     string       `json:"outComeLogoUrl"`
	OptionID           string       `json:"optionId"`
	OptionTitle        string       `json:"optionTitle"`
	OptionIndex        int          `json:"optionIndex"`
	CoinID             int64        `json:"coinId"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data *LoginData `json:"data"`
}

// LoginData 登录数据
type LoginData struct {
	FirstName       string     `json:"firstName"`
	LastName        string     `json:"lastName"`
	CreatedAt       string     `json:"createdAt"`
	Is2FAEnabled    bool       `json:"is2FAEnabled"`
	Language        string     `json:"language"`
	ID              int        `json:"id"`
	RoleList        []RoleInfo `json:"roleList"`
	UserName        string     `json:"userName"`
	IsActive        bool       `json:"isActive"`
	UserID          string     `json:"userId"`
	Email           string     `json:"email"`
	IsEmailVerified bool       `json:"isEmailVerified"`
}

// RoleInfo 角色信息
type RoleInfo struct {
	ID     int `json:"id"`
	UserID int `json:"userId"`
	RoleID int `json:"roleId"`
}

// LoginResult 登录结果（包含 Authorization token）
type LoginResult struct {
	Code          int        `json:"code"`
	Msg           string     `json:"msg"`
	Data          *LoginData `json:"data"`
	Authorization string     `json:"authorization"` // 从 Cookie 中提取的 Authorization 值
}

// WalletBalanceResponse 钱包余额响应结构
type WalletBalanceResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data *WalletBalance `json:"data"`
}

// WalletBalance 钱包余额数据
type WalletBalance struct {
	UserBalance          json.Number      `json:"userBalance"`
	UserID               string           `json:"userId"`
	AvailableAmount      json.Number      `json:"availableAmount"`
	FreezeAmount         json.Number      `json:"freezeAmount"`
	TrialUserBalance     json.Number      `json:"trialUserBalance"`
	TrialAvailableAmount json.Number      `json:"trialAvailableAmount"`
	TrialFreezeAmount    json.Number      `json:"trialFreezeAmount"`
	WMSBalanceList       []WMSBalanceItem `json:"wmsBalanceList"`
}

// WMSBalanceItem WMS余额项
type WMSBalanceItem struct {
	UserBalance     json.Number `json:"userBalance"`
	UserID          string      `json:"userId"`
	CoinID          int64       `json:"coinId"`
	AvailableAmount json.Number `json:"availableAmount"`
	FreezeAmount    json.Number `json:"freezeAmount"`
	ID              int64       `json:"id"`
}

// BatchModifyOrderOpenResponse 批量修改订单响应结构
type BatchModifyOrderOpenResponse struct {
	Code int                       `json:"code"`
	Msg  string                    `json:"msg"`
	Data *BatchModifyOrderOpenData `json:"data"`
}

// BatchModifyOrderOpenData 批量修改订单数据
type BatchModifyOrderOpenData struct {
	SuccessOrders []BatchModifyOrderResult `json:"successOrders"`
	FailedOrders  []BatchModifyOrderResult `json:"failedOrders"`
	SuccessCount  int                      `json:"successCount"`
	FailedCount   int                      `json:"failedCount"`
}

// BatchModifyOrderResult 批量修改订单结果项
type BatchModifyOrderResult struct {
	OrderOpenID int64  `json:"orderOpenId"`
	SerialNo    int64  `json:"serialNo"`
	Success     bool   `json:"success"`
	ErrorMsg    string `json:"errorMsg"`
}

// MarketInitShareRequest 初始化share请求结构
type MarketInitShareRequest struct {
	MarketID int64  `json:"marketId"`
	OptionID string `json:"optionId"`
	UserID   string `json:"userId"`
}

// MarketInitShareResponse 初始化share响应结构
type MarketInitShareResponse struct {
	Code     int                   `json:"code"`
	Msg      string                `json:"msg"`
	Current  int                   `json:"current"`
	Size     int                   `json:"size"`
	Total    int                   `json:"total"`
	Pages    int                   `json:"pages"`
	Data     []MarketInitShareItem `json:"data"`
	TotalRow *int                  `json:"totalRow"`
	Empty    bool                  `json:"empty"`
}

// MarketInitShareItem 初始化share数据项
type MarketInitShareItem struct {
	MarketID      int64  `json:"marketId"`
	UserID        int64  `json:"userId"`
	UserAccountID string `json:"userAccountId"`
	UserName      string `json:"userName"`
	UserType      string `json:"userType"`
	OptionID      int64  `json:"optionId"`
	Share         string `json:"share"`
	CreatedAt     string `json:"createdAt"`
}

// TradeOrderRequest 创建交易订单请求结构
type TradeOrderRequest struct {
	MarketID   int64   `json:"marketId"`
	OutcomeID  int64   `json:"outcomeId"`
	OptionID   int64   `json:"optionId"`
	Direction  string  `json:"direction"` // BUY or SELL
	LimitPrice float64 `json:"limitPrice"`
	Share      float64 `json:"share"`
	UserID     int64   `json:"userId"`
	CoinID     int64   `json:"coinId"`
}

// TradeOrderResponse 创建交易订单响应结构
type TradeOrderResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"` // 通常为 null
}
