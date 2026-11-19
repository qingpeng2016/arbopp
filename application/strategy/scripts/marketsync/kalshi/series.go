package kalshi

import (
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"strings"
)

// 各运动子类型对应的 Kalshi「单场胜负」系列（*GAME 等）；可在 config_strategy 用 kalshi_game_series 追加/覆盖列表。
var kalshiGameSeriesBySubCategory = map[cryptoPredictionEnum.SubMarketCategory][]string{
	cryptoPredictionEnum.SubMarketCategoryFootball: {
		"KXEPLGAME", "KXUCLGAME", "KXLALIGAGAME", "KXBUNDESLIGAGAME", "KXSERIEAGAME",
	},
	cryptoPredictionEnum.SubMarketCategoryBasketball: {
		"KXNBAGAME", "KXWNBAGAME", "KXNCAAMBGAME",
	},
	cryptoPredictionEnum.SubMarketCategoryNFL: {
		"KXNFLGAME", "KXNCAAFGAME",
	},
	cryptoPredictionEnum.SubMarketCategoryMLB: {"KXMLBGAME"},
	cryptoPredictionEnum.SubMarketCategoryNHL: {"KXNHLGAME"},
	cryptoPredictionEnum.SubMarketCategoryTennis: {
		"KXATPGAME", "KXATPMATCH",
	},
	cryptoPredictionEnum.SubMarketCategoryMMA: {
		"KXUFCGAME", "KXUFCFIGHT",
	},
}

func kalshiGameSeriesForSubCategory(subCat cryptoPredictionEnum.SubMarketCategory, cfg SeriesConfig) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(t string) {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t == "" {
			return
		}
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if cfg.Active {
		for _, s := range cfg.BySubCategory[subCat] {
			add(s)
		}
		return out
	}
	for _, s := range kalshiGameSeriesBySubCategory[subCat] {
		add(s)
	}
	return out
}

func subCatForKalshiSeries(series string) cryptoPredictionEnum.SubMarketCategory {
	series = strings.ToUpper(strings.TrimSpace(series))
	for sub, list := range kalshiGameSeriesBySubCategory {
		for _, s := range list {
			if s == series {
				return sub
			}
		}
	}
	return cryptoPredictionEnum.SubMarketCategoryDefault
}

// kalshiEventIsMatchMarket 只同步单场比赛，含常规时间胜负。冠军、出线、小组头名等长话题不入库。
// Spread / Total 是另外的 series（*GAME → *SPREAD / *TOTAL），不在这里当独立场次；对应类型开启时挂到这场的 outcome。
func kalshiEventIsMatchMarket(ev httpentity.KalshiEvent) bool {
	if !strings.EqualFold(strings.TrimSpace(ev.Category), "Sports") {
		return false
	}
	if ev.ProductMetadata == nil {
		return true
	}
	scope := strings.ToLower(strings.TrimSpace(ev.ProductMetadata.CompetitionScope))
	if scope == "" {
		return true
	}
	if scope == "game" || scope == "match" || scope == "regulation time moneyline" || strings.Contains(scope, "fight") {
		return true
	}
	return false
}
