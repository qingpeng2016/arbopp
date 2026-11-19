package polymarket

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func polymarketSlugMatchesSportID(m *httpentity.Market, sportID string) bool {
	if m == nil {
		return false
	}
	slug := polySlugLower(m)
	sportID = strings.ToLower(strings.TrimSpace(sportID))
	if slug == "" || sportID == "" {
		return false
	}
	return slug == sportID || strings.HasPrefix(slug, sportID+"-")
}

// polymarketMatchesConfiguredLeague slug 前缀须落在 polymarket_sports 配置的 sport_id（如 epl-、ucl-）。
func polymarketMatchesConfiguredLeague(m *httpentity.Market, bySport map[string]string) bool {
	if m == nil || len(bySport) == 0 {
		return true
	}
	slug := polySlugLower(m)
	if slug == "" {
		return false
	}
	for sid := range bySport {
		sid = strings.ToLower(strings.TrimSpace(sid))
		if sid == "" {
			continue
		}
		if slug == sid || strings.HasPrefix(slug, sid+"-") {
			return true
		}
	}
	return false
}

// polymarketSubMatchFilter 各运动子类独立实现，互改时不影响其他文件。
type polymarketSubMatchFilter interface {
	MarketForSync(m *httpentity.Market, types syncopt.OutcomeTypeSet) (bool, cryptoPredictionEnum.OutcomeMarketType)
}

func polymarketMatchFilterFor(sub cryptoPredictionEnum.SubMarketCategory) polymarketSubMatchFilter {
	switch sub {
	case cryptoPredictionEnum.SubMarketCategoryFootball:
		return footballMatchFilter{}
	case cryptoPredictionEnum.SubMarketCategoryBasketball:
		return basketballMatchFilter{}
	case cryptoPredictionEnum.SubMarketCategoryNFL:
		return nflMatchFilter{}
	case cryptoPredictionEnum.SubMarketCategoryNHL:
		return nhlMatchFilter{}
	case cryptoPredictionEnum.SubMarketCategoryCricket:
		return cricketMatchFilter{}
	case cryptoPredictionEnum.SubMarketCategoryTennis:
		return tennisMatchFilter{}
	case cryptoPredictionEnum.SubMarketCategoryMMA:
		return mmaMatchFilter{}
	default:
		return defaultMatchFilter{}
	}
}

// polymarketMarketForSync 按 sub_category 路由到对应过滤实现（禁止用布尔开关混写多类逻辑）。
func polymarketMarketForSync(m *httpentity.Market, types syncopt.OutcomeTypeSet, sub cryptoPredictionEnum.SubMarketCategory) (bool, cryptoPredictionEnum.OutcomeMarketType) {
	return polymarketMatchFilterFor(sub).MarketForSync(m, types)
}