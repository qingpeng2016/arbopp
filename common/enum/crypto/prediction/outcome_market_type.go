package prediction

import (
	"encoding/json"
	"strings"
)

// OutcomeMarketType 同步/落库的盘口大类（market_sync.sync_outcome_types）。
type OutcomeMarketType string

const (
	// OutcomeMarketTypeMoneyline 胜平负 / 单场胜负（含 draw）。
	OutcomeMarketTypeMoneyline OutcomeMarketType = "moneyline"
	// OutcomeMarketTypeHandicap 让分/spread（含 ±0.5、±1.5 等）。
	OutcomeMarketTypeHandicap OutcomeMarketType = "handicap"
	// OutcomeMarketTypeTotal 大小球/总分（Over/Under，如 O 2.5 / U 2.5）。
	OutcomeMarketTypeTotal OutcomeMarketType = "total"
)

// AllSyncOutcomeMarketTypes 当前支持的 sync_outcome_types 取值。
func AllSyncOutcomeMarketTypes() []OutcomeMarketType {
	return []OutcomeMarketType{OutcomeMarketTypeMoneyline, OutcomeMarketTypeHandicap, OutcomeMarketTypeTotal}
}

// SyncOutcomeTypeGroup config_strategy.sync_outcome_types 里的一组运动。
// 与联赛配置相同：sport_cn 是中文运动名，sub_category 对应同步时的运动子类。
type SyncOutcomeTypeGroup struct {
	SportCN     string   `json:"sport_cn"`
	SubCategory string   `json:"sub_category"`
	Items       []string `json:"items"`
}

// SyncOutcomeTypeConfig 按运动子类取盘口类型。旧的扁平数组仍可用，这时每种运动都用同一份。
type SyncOutcomeTypeConfig struct {
	BySubCategory map[SubMarketCategory][]OutcomeMarketType
	SportCN       map[SubMarketCategory]string
	LegacyAll     []OutcomeMarketType
}

// For 返回该运动要同步的盘口。分组配置里没有的运动返回 nil，调用方只用胜平负。
func (c SyncOutcomeTypeConfig) For(sub SubMarketCategory) []OutcomeMarketType {
	if len(c.BySubCategory) > 0 {
		return c.BySubCategory[sub]
	}
	return c.LegacyAll
}

// ParseSyncOutcomeTypes 解析 config_strategy item_value。
// 分组：[{"sport_cn":"足球","sub_category":"Football","items":["moneyline","handicap","total"]}]
// 旧格式：["moneyline","handicap","total"] 或逗号分隔。
func ParseSyncOutcomeTypes(raw, valueType string) SyncOutcomeTypeConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SyncOutcomeTypeConfig{}
	}
	valueType = strings.ToLower(strings.TrimSpace(valueType))
	if valueType == "json" || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		if cfg, ok := parseSyncOutcomeTypeGroups(raw); ok {
			return cfg
		}
		return SyncOutcomeTypeConfig{LegacyAll: normalizeOutcomeMarketTypes(jsonStringList(raw))}
	}
	return SyncOutcomeTypeConfig{LegacyAll: normalizeOutcomeMarketTypes(strings.Split(raw, ","))}
}

func parseSyncOutcomeTypeGroups(raw string) (SyncOutcomeTypeConfig, bool) {
	var groups []SyncOutcomeTypeGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil || len(groups) == 0 {
		return SyncOutcomeTypeConfig{}, false
	}
	if groups[0].SportCN == "" && groups[0].SubCategory == "" && groups[0].Items == nil {
		return SyncOutcomeTypeConfig{}, false
	}
	cfg := SyncOutcomeTypeConfig{
		BySubCategory: make(map[SubMarketCategory][]OutcomeMarketType),
		SportCN:       make(map[SubMarketCategory]string),
	}
	for _, g := range groups {
		sub := syncOutcomeSubCategory(g.SportCN, g.SubCategory)
		if sub == "" || len(g.Items) == 0 {
			continue
		}
		types := normalizeOutcomeMarketTypes(g.Items)
		if len(types) == 0 {
			continue
		}
		cfg.BySubCategory[sub] = types
		if name := strings.TrimSpace(g.SportCN); name != "" {
			cfg.SportCN[sub] = name
		}
	}
	if len(cfg.BySubCategory) == 0 {
		return SyncOutcomeTypeConfig{}, false
	}
	return cfg, true
}

func syncOutcomeSubCategory(sportCN, sub string) SubMarketCategory {
	sub = strings.TrimSpace(sub)
	if sub != "" {
		return SubMarketCategory(sub)
	}
	switch strings.TrimSpace(sportCN) {
	case "足球":
		return SubMarketCategoryFootball
	case "篮球":
		return SubMarketCategoryBasketball
	default:
		return ""
	}
}

func jsonStringList(raw string) []string {
	var tokens []string
	if err := json.Unmarshal([]byte(raw), &tokens); err != nil {
		return nil
	}
	return tokens
}

func normalizeOutcomeMarketTypes(tokens []string) []OutcomeMarketType {
	seen := make(map[OutcomeMarketType]struct{})
	var out []OutcomeMarketType
	for _, t := range tokens {
		ot := OutcomeMarketType(strings.ToLower(strings.TrimSpace(t)))
		switch ot {
		case OutcomeMarketTypeMoneyline, OutcomeMarketTypeHandicap, OutcomeMarketTypeTotal:
			if _, ok := seen[ot]; ok {
				continue
			}
			seen[ot] = struct{}{}
			out = append(out, ot)
		}
	}
	return out
}
