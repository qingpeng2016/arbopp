package predict

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func predictMarketForSync(m httpentity.PredictMarket, types syncopt.OutcomeTypeSet) (ok bool, kind cryptoPredictionEnum.OutcomeMarketType) {
	if types.Empty() {
		return false, ""
	}
	mt := strings.ToUpper(strings.TrimSpace(m.MarketType))
	switch mt {
	case "SPORTS_SPREADS", "SPORTS_MAP_HANDICAP", "SPORTS_ROUND_HANDICAP":
		if types.Handicap {
			return true, cryptoPredictionEnum.OutcomeMarketTypeHandicap
		}
	case "SPORTS_TOTALS", "SPORTS_TEAM_TOTALS", "SPORTS_KILL_OVER_UNDER_GAME", "SPORTS_ROUND_TOTALS",
		"SPORTS_FIRST_HALF_TOTALS", "SPORTS_SECOND_HALF_TOTALS":
		if types.Total {
			return true, cryptoPredictionEnum.OutcomeMarketTypeTotal
		}
	case "SPORTS_MONEYLINE", "SPORTS_CHILD_MONEYLINE", "SPORTS_HALFTIME_RESULT", "SPORTS_SECOND_HALF_RESULT":
		if types.Moneyline {
			return true, cryptoPredictionEnum.OutcomeMarketTypeMoneyline
		}
	default:
		if types.Moneyline && (mt == "" || strings.HasPrefix(mt, "SPORTS_")) {
			// 未知体育盘型且只开 moneyline 时跳过，避免 props 混入
			return false, ""
		}
	}
	return false, ""
}

func predictCategoryIsSportsFixture(cat httpentity.PredictCategory) bool {
	if !cat.IsVisible {
		return false
	}
	st := strings.ToUpper(strings.TrimSpace(cat.Status))
	if st == "RESOLVED" || st == "REMOVED" {
		return false
	}
	mv := strings.ToUpper(strings.TrimSpace(cat.MarketVariant))
	if mv == "" {
		return len(cat.Markets) > 0
	}
	return strings.HasPrefix(mv, "SPORTS_") || mv == "ESPORTS_LOL" || mv == "ESPORTS_CS2" || mv == "ESPORTS_DOTA2"
}

func predictMarketTradable(m httpentity.PredictMarket) bool {
	if ts := strings.ToUpper(strings.TrimSpace(m.TradingStatus)); ts != "" && ts != "OPEN" {
		return false
	}
	st := strings.ToUpper(strings.TrimSpace(m.Status))
	return st != "RESOLVED" && st != "REMOVED"
}