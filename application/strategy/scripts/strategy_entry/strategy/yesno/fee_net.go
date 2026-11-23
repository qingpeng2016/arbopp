package yesno

import (
	"encoding/json"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// TradeFeeRate 吃单费率：Predict 优先市场 feeRateBps，其余走 config_strategy_fee 系数。
func TradeFeeRate(platform, category, sub string, ocConfig *string, tradeFee func(string, string, string) float64) float64 {
	platform = strings.ToUpper(strings.TrimSpace(platform))
	if platform == appcommon.PlatformPredict && ocConfig != nil {
		var cfg struct {
			FeeRateBps *int `json:"feeRateBps"`
		}
		if json.Unmarshal([]byte(*ocConfig), &cfg) == nil && cfg.FeeRateBps != nil &&
			*cfg.FeeRateBps >= 0 && *cfg.FeeRateBps <= 1000 {
			return float64(*cfg.FeeRateBps) / 10000
		}
	}
	if tradeFee != nil {
		if r := tradeFee(platform, category, sub); r >= 0 {
			return r
		}
	}
	return 0
}

// NetCostPerShare 买 1 份净兑付所需总成本（含平台计费规则）。
func NetCostPerShare(platform string, price, rate float64) float64 {
	if !book.ValidPrice(price) || rate < 0 || rate >= 1 {
		return 0
	}
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case appcommon.PlatformPredict:
		return price / PredictNetRatio(price, rate)
	case appcommon.PlatformSxbet:
		denom := 1 - rate*(1-price)
		if denom <= 0 {
			return 0
		}
		return price / denom
	default:
		return price + Fee(price, rate)
	}
}

// PredictNetRatio 买后净兑付 / 标价（Predict 按份扣费模型）。
func PredictNetRatio(price, rate float64) float64 {
	if !book.ValidPrice(price) {
		return 0
	}
	return 1 - rate*minFloat(price, 1-price)/price
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}