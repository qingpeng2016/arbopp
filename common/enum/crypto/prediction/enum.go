package prediction

import "fmt"

type Lang int8

const (
	LangEn Lang = 1 // 英文
	LangCh Lang = 2 // 中文
	LangHk Lang = 3 // 繁体中文
)

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType 订单类型
type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
)

type OrderStatus string

const (
	OrderStatusCreate       OrderStatus = "CREATE"        // 创建挂单
	OrderStatusCreateFailed OrderStatus = "CREATE_FAILED" // 创建挂单失败
	OrderStatusAccepted     OrderStatus = "ACCEPTED"      // 挂单成功
	OrderStatusPending      OrderStatus = "PENDING"       // 挂单中
	OrderStatusFilled       OrderStatus = "FILLED"        // 全部成交
	OrderStatusCancelled    OrderStatus = "CANCELLED"     // 挂单取消
	OrderStatusPartial      OrderStatus = "PARTIAL"       // 部分成交
	OrderStatusFailed       OrderStatus = "FAILED"        // 成交失败
)

// PlatformType 平台大类：预测市场 vs 赔率(BC)
type PlatformType string

const (
	PlatformTypePrediction PlatformType = "PREDICTION"
	PlatformTypeOdds       PlatformType = "ODDS"
)

// QuoteStyle 报价展示/归一化口径（套利前需转为可比较的「成本」）
type QuoteStyle string

const (
	QuoteStyleProbability  QuoteStyle = "PROBABILITY"   // 0~1，预测市场
	QuoteStyleDecimalOdds  QuoteStyle = "DECIMAL_ODDS"  // 欧赔
	QuoteStyleAmericanOdds QuoteStyle = "AMERICAN_ODDS" // 美式赔率
)

// Platform 交易所 / 盘口平台 slug（与 platform.slug 一致）
type Platform string

const (
	PlatformPolymarket Platform = "POLYMARKET"
	PlatformKalshi     Platform = "KALSHI"
	PlatformPredict    Platform = "PREDICT"
	PlatformForegate   Platform = "FOREGATE"
	PlatformPinnacle   Platform = "PINNACLE"
	PlatformBet365     Platform = "BET365"
)

// Type 返回平台大类
func (p Platform) Type() PlatformType {
	switch p {
	case PlatformPolymarket, PlatformKalshi, PlatformPredict, PlatformForegate:
		return PlatformTypePrediction
	case PlatformPinnacle, PlatformBet365:
		return PlatformTypeOdds
	default:
		return ""
	}
}

// DefaultQuoteStyle 默认报价口径
func (p Platform) DefaultQuoteStyle() QuoteStyle {
	switch p.Type() {
	case PlatformTypeOdds:
		return QuoteStyleDecimalOdds
	default:
		return QuoteStyleProbability
	}
}

// IsPrediction 是否预测市场
func (p Platform) IsPrediction() bool {
	return p.Type() == PlatformTypePrediction
}

// IsOdds 是否赔率平台
func (p Platform) IsOdds() bool {
	return p.Type() == PlatformTypeOdds
}

type MarketStatus string

const (
	MarketStatusActive MarketStatus = "ACTIVE" // 启用
	MarketStatusClose  MarketStatus = "CLOSE"  // 禁用
)

// MarketCategory 市场分类（用于 ListMarkets 过滤及写入 market.category）
type MarketCategory string

const (
	MarketCategoryDefault MarketCategory = ""       // 默认/全部市场
	MarketCategorySports  MarketCategory = "Sports" // 运动市场
)

// SubMarketCategory 运动子类型（用于 ListMarkets 过滤，如仅拉取足球市场）
type SubMarketCategory string

const (
	SubMarketCategoryDefault        SubMarketCategory = ""               // 默认/全部运动子类型
	SubMarketCategoryFootball       SubMarketCategory = "Football"       // 足球（epl、lal、ucl、mls 等）
	SubMarketCategoryBasketball     SubMarketCategory = "Basketball"     // 篮球（nba、ncaa、euroleague 等）
	SubMarketCategoryNFL            SubMarketCategory = "NFL"            // 美式橄榄球（nfl）
	SubMarketCategoryMLB            SubMarketCategory = "MLB"            // 棒球（mlb）
	SubMarketCategoryNHL            SubMarketCategory = "NHL"            // 冰球（nhl）
	SubMarketCategoryTennis         SubMarketCategory = "Tennis"         // 网球
	SubMarketCategoryMMA            SubMarketCategory = "MMA"            // 综合格斗（ufc 等）
	SubMarketCategoryBoxing         SubMarketCategory = "Boxing"         // 拳击
	SubMarketCategoryGolf           SubMarketCategory = "Golf"           // 高尔夫
	SubMarketCategoryEsports        SubMarketCategory = "Esports"        // 电竞/游戏（Dota 2、LoL、CS2、Valorant 等）
	SubMarketCategoryRugby          SubMarketCategory = "Rugby"          // 橄榄球（Six Nations、Premiership 等）
	SubMarketCategoryCricket        SubMarketCategory = "Cricket"        // 板球（ICC、IPL、T20 等）
	SubMarketCategoryWinterOlympics SubMarketCategory = "WinterOlympics" // 冬季奥运会
	SubMarketCategoryTableTennis    SubMarketCategory = "TableTennis"    // 乒乓球（WTT）
	SubMarketCategoryFormula1       SubMarketCategory = "Formula1"       // F1
	SubMarketCategoryChess          SubMarketCategory = "Chess"          // 国际象棋
	SubMarketCategoryPickleball     SubMarketCategory = "Pickleball"     // 匹克球
)

// AllSubMarketCategories 返回除 Default 外所有运动子类型，用于「全部」时按子类循环拉取
func AllSubMarketCategories(marketCategory MarketCategory) []SubMarketCategory {
	if marketCategory == MarketCategorySports {
		return []SubMarketCategory{
			SubMarketCategoryFootball,
			SubMarketCategoryBasketball,
			SubMarketCategoryNFL,
			SubMarketCategoryMLB,
			SubMarketCategoryNHL,
			SubMarketCategoryTennis,
			SubMarketCategoryMMA,
			SubMarketCategoryBoxing,
			SubMarketCategoryGolf,
			SubMarketCategoryEsports,
			SubMarketCategoryRugby,
			SubMarketCategoryCricket,
			SubMarketCategoryWinterOlympics,
			SubMarketCategoryTableTennis,
			SubMarketCategoryFormula1,
			SubMarketCategoryChess,
			SubMarketCategoryPickleball,
		}
	}
	return []SubMarketCategory{}
}

// Redis key 规则：订单簿索引与单 key 格式，策略用 keylist 做 MGET
// RedisKeyFormatOB 单 token 订单簿 key 格式，参数：platform, market_id, token_id
const RedisKeyFormatOB = "ob:%s:%s:%s"

// RedisKeyFormatOBKeyListPlatform 按平台的 key 列表 key 格式，参数：platform；value 为 JSON 数组 []string
const RedisKeyFormatOBKeyListPlatform = "ob:keylist:%s"

// RedisKeyOB 生成单 token 订单簿 Redis key
func RedisKeyOB(platform, marketID, tokenID string) string {
	return fmt.Sprintf(RedisKeyFormatOB, platform, marketID, tokenID)
}

// RedisKeyOBKeyListForPlatform 生成某平台的 keylist key
func RedisKeyOBKeyListForPlatform(platform string) string {
	return fmt.Sprintf(RedisKeyFormatOBKeyListPlatform, platform)
}
