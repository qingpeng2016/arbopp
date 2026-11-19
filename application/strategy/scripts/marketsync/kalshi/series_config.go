package kalshi

import (
	"encoding/json"
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

// kalshiGameSeriesItem 配置里的一条：ticker 才是 API series。
// name_en 与 Polymarket 同一中文联赛保持一致；name_desc 是本平台展示英文，Lark 用它。
type kalshiGameSeriesItem struct {
	Name     string `json:"name"`
	NameEn   string `json:"name_en"`
	NameDesc string `json:"name_desc"`
	Ticker   string `json:"ticker"`
	PairPoly string `json:"pair_poly_sport_id,omitempty"`
}

type kalshiGameSeriesFlatEntry struct {
	Name        string `json:"name"`
	SubCategory string `json:"sub_category"`
	Ticker      string `json:"ticker"`
}

type kalshiGameSeriesGroup struct {
	SportCN     string                 `json:"sport_cn"`
	SubCategory string                 `json:"sub_category"`
	Items       []kalshiGameSeriesItem `json:"items"`
}

// parseKalshiGameSeriesBySubCategory 解析 kalshi_game_series。
// 支持 value_type=json：分组格式（见 docs/sql/kalshi_game_series.json）或扁平 {"name","sub_category","ticker"} 数组。
// 兼容旧 csv / 字符串 ticker 列表（按内置 map 推断 sub_category）。
func parseKalshiGameSeriesBySubCategory(raw, valueType string) map[cryptoPredictionEnum.SubMarketCategory][]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	valueType = strings.ToLower(strings.TrimSpace(valueType))

	out := make(map[cryptoPredictionEnum.SubMarketCategory][]string)
	add := func(subCat cryptoPredictionEnum.SubMarketCategory, ticker string) {
		ticker = strings.ToUpper(strings.TrimSpace(ticker))
		if ticker == "" || subCat == "" {
			return
		}
		for _, existing := range out[subCat] {
			if existing == ticker {
				return
			}
		}
		out[subCat] = append(out[subCat], ticker)
	}

	if valueType == "json" || valueType == "json_array" || valueType == "json_object" || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		var flat []kalshiGameSeriesFlatEntry
		if err := json.Unmarshal([]byte(raw), &flat); err == nil && len(flat) > 0 && strings.TrimSpace(flat[0].Ticker) != "" {
			for _, e := range flat {
				add(parseSubMarketCategory(e.SubCategory), e.Ticker)
			}
			if len(out) > 0 {
				return out
			}
		}
		var groups []kalshiGameSeriesGroup
		if err := json.Unmarshal([]byte(raw), &groups); err == nil {
			for _, g := range groups {
				sub := parseSubMarketCategory(g.SubCategory)
				if sub == "" || len(g.Items) == 0 {
					continue
				}
				for _, it := range g.Items {
					add(sub, it.Ticker)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
		// 旧版 json 字符串数组：["KXNBAGAME", ...]
		var tickers []string
		if err := json.Unmarshal([]byte(raw), &tickers); err == nil {
			for _, t := range tickers {
				add(subCatForKalshiSeries(t), t)
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	// csv / string
	for _, t := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	}) {
		add(subCatForKalshiSeries(t), t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// leagueDesc 通知里的英文。优先本平台 name_desc；旧配置只有 name_en 时沿用它。
func leagueDesc(it kalshiGameSeriesItem) string {
	if desc := strings.TrimSpace(it.NameDesc); desc != "" {
		return desc
	}
	return strings.TrimSpace(it.NameEn)
}

// kalshiLeagueMeta 一条 series 的展示名，供联赛完成通知使用。
type kalshiLeagueMeta struct {
	Name     string
	NameEn   string
	NameDesc string
	PairPoly string
}

func parseKalshiLeagues(raw string) map[string]kalshiLeagueMeta {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	put := func(out map[string]kalshiLeagueMeta, ticker, name, nameEn, nameDesc, pair string) {
		ticker = strings.ToUpper(strings.TrimSpace(ticker))
		if ticker == "" {
			return
		}
		out[ticker] = kalshiLeagueMeta{
			Name:     strings.TrimSpace(name),
			NameEn:   strings.TrimSpace(nameEn),
			NameDesc: strings.TrimSpace(nameDesc),
			PairPoly: strings.ToLower(strings.TrimSpace(pair)),
		}
	}
	var groups []kalshiGameSeriesGroup
	if err := json.Unmarshal([]byte(raw), &groups); err == nil && len(groups) > 0 && len(groups[0].Items) > 0 {
		out := make(map[string]kalshiLeagueMeta)
		for _, g := range groups {
			for _, it := range g.Items {
				put(out, it.Ticker, it.Name, it.NameEn, leagueDesc(it), it.PairPoly)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var flat []kalshiGameSeriesFlatEntry
	if err := json.Unmarshal([]byte(raw), &flat); err == nil {
		out := make(map[string]kalshiLeagueMeta)
		for _, e := range flat {
			put(out, e.Ticker, e.Name, "", "", "")
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

// parseKalshiCompetitionBySeries 从 kalshi_game_series JSON 解析 ticker -> 联赛名（如英超）。
func parseKalshiCompetitionBySeries(raw, valueType string) map[string]string {
	leagues := parseKalshiLeagues(raw)
	if len(leagues) == 0 {
		_ = valueType
		return nil
	}
	out := make(map[string]string, len(leagues))
	for ticker, meta := range leagues {
		if meta.Name != "" {
			out[ticker] = meta.Name
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SeriesConfig market_sync kalshi_game_series 配置。
type SeriesConfig struct {
	BySubCategory       map[cryptoPredictionEnum.SubMarketCategory][]string
	CompetitionBySeries map[string]string
	LeagueBySeries      map[string]kalshiLeagueMeta
	Active              bool
}

// LoadSeriesConfig 从 config_strategy 行内容解析。
func LoadSeriesConfig(raw, valueType string) SeriesConfig {
	parsed := parseKalshiGameSeriesBySubCategory(raw, valueType)
	if len(parsed) == 0 {
		return SeriesConfig{}
	}
	leagues := parseKalshiLeagues(raw)
	return SeriesConfig{
		BySubCategory:       parsed,
		CompetitionBySeries: parseKalshiCompetitionBySeries(raw, valueType),
		LeagueBySeries:      leagues,
		Active:              true,
	}
}

func parseSubMarketCategory(s string) cryptoPredictionEnum.SubMarketCategory {
	s = strings.TrimSpace(s)
	if s == "" {
		return cryptoPredictionEnum.SubMarketCategoryDefault
	}
	return cryptoPredictionEnum.SubMarketCategory(s)
}
