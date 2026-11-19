package polymarket

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// Polymarket 上标记「单场」盘口的 tag（与 GET /sports 各联赛 tags 取交集，排除 82 等赛季/排名类 tag）。
var defaultPolymarketMatchTagIDs = []int{100350, 100639}

type polymarketSportItem struct {
	Name     string `json:"name"`
	NameEn   string `json:"name_en"`   // 跨平台统一英文名；同一中文联赛两边相同
	NameDesc string `json:"name_desc"` // 本平台展示英文，Lark 联赛完成通知用这个
	SportID  string `json:"sport_id"`
	TagIDs   []int  `json:"tag_ids"`
}

type polymarketSportsGroup struct {
	SportCN            string                `json:"sport_cn"`
	SubCategory        string                `json:"sub_category"`
	SyncAllSoccerGames bool                  `json:"sync_all_soccer_games"`
	SoccerGameTagIDs   []int                 `json:"soccer_game_tag_ids"`
	Items              []polymarketSportItem `json:"items"`
}

// SportsBySubCategory 配置解析后的子类 -> 联赛列表。
type SportsBySubCategory map[cryptoPredictionEnum.SubMarketCategory][]polymarketSportItem

// SportsConfig market_sync polymarket_sports 配置。
type SportsConfig struct {
	BySubCategory          SportsBySubCategory
	CompetitionBySportID   map[string]string
	CompetitionEnBySportID map[string]string
	Active                 bool
	SyncAllSoccerGames     bool
	SoccerGameTagIDs       []int
}

// LoadSportsConfig 从 config_strategy 行内容解析。
func LoadSportsConfig(raw, valueType string) SportsConfig {
	parsed := parsePolymarketSportsBySubCategory(raw, valueType)
	if len(parsed) == 0 {
		return SportsConfig{}
	}
	syncAll, soccerTags := parsePolymarketSoccerSyncOptions(raw, valueType)
	names, namesEn := parsePolymarketLeagueNames(raw, valueType)
	return SportsConfig{
		BySubCategory:          parsed,
		CompetitionBySportID:   names,
		CompetitionEnBySportID: namesEn,
		Active:                 true,
		SyncAllSoccerGames:     syncAll,
		SoccerGameTagIDs:       soccerTags,
	}
}

type polymarketSportsBySubCategory = SportsBySubCategory

func parseSubMarketCategory(s string) cryptoPredictionEnum.SubMarketCategory {
	s = strings.TrimSpace(s)
	if s == "" {
		return cryptoPredictionEnum.SubMarketCategoryDefault
	}
	return cryptoPredictionEnum.SubMarketCategory(s)
}

func parsePolymarketSportsBySubCategory(raw, valueType string) polymarketSportsBySubCategory {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	valueType = strings.ToLower(strings.TrimSpace(valueType))
	out := make(polymarketSportsBySubCategory)

	if valueType == "json" || strings.HasPrefix(raw, "[") {
		var groups []polymarketSportsGroup
		if err := json.Unmarshal([]byte(raw), &groups); err == nil {
			for _, g := range groups {
				sub := parseSubMarketCategory(g.SubCategory)
				if sub == "" || len(g.Items) == 0 {
					continue
				}
				for _, it := range g.Items {
					sid := strings.ToLower(strings.TrimSpace(it.SportID))
					if sid == "" {
						continue
					}
					it.SportID = sid
					out[sub] = append(out[sub], it)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

// parsePolymarketSoccerSyncOptions 读取足球组的 games 全量同步开关与共用 tag。
func parsePolymarketSoccerSyncOptions(raw, valueType string) (syncAll bool, tagIDs []int) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}
	valueType = strings.ToLower(strings.TrimSpace(valueType))
	if valueType != "json" && !strings.HasPrefix(raw, "[") {
		return false, nil
	}
	var groups []polymarketSportsGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil {
		return false, nil
	}
	for _, g := range groups {
		if parseSubMarketCategory(g.SubCategory) != cryptoPredictionEnum.SubMarketCategoryFootball {
			continue
		}
		return g.SyncAllSoccerGames, append([]int(nil), g.SoccerGameTagIDs...)
	}
	return false, nil
}

// parsePolymarketLeagueNames 从 polymarket_sports JSON 解析 sport_id -> name / name_en。
func parsePolymarketLeagueNames(raw, valueType string) (names, namesEn map[string]string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var groups []polymarketSportsGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil || len(groups) == 0 {
		_ = valueType
		return nil, nil
	}
	names = make(map[string]string)
	namesEn = make(map[string]string)
	for _, g := range groups {
		for _, it := range g.Items {
			sid := strings.ToLower(strings.TrimSpace(it.SportID))
			if sid == "" {
				continue
			}
			if n := strings.TrimSpace(it.Name); n != "" {
				names[sid] = n
			}
			if n := strings.TrimSpace(it.NameEn); n != "" {
				namesEn[sid] = n
			}
		}
	}
	if len(names) == 0 {
		names = nil
	}
	if len(namesEn) == 0 {
		namesEn = nil
	}
	return names, namesEn
}

// SubCategoriesUnion 合并 Poly / Kalshi 配置里出现过的子类。
func SubCategoriesUnion(poly SportsBySubCategory, kalshi map[cryptoPredictionEnum.SubMarketCategory][]string) []cryptoPredictionEnum.SubMarketCategory {
	seen := make(map[cryptoPredictionEnum.SubMarketCategory]struct{})
	var out []cryptoPredictionEnum.SubMarketCategory
	add := func(sub cryptoPredictionEnum.SubMarketCategory) {
		if sub == "" {
			return
		}
		if _, ok := seen[sub]; ok {
			return
		}
		seen[sub] = struct{}{}
		out = append(out, sub)
	}
	for sub := range poly {
		add(sub)
	}
	for sub := range kalshi {
		add(sub)
	}
	return out
}

func tagIDsFromSportsMeta(tagsCSV string, sportID string, item polymarketSportItem) []int {
	sportID = strings.ToLower(strings.TrimSpace(sportID))
	if tagsCSV == "" {
		return nil
	}
	wantTag := item.TagIDs
	if len(wantTag) == 0 {
		wantTag = defaultPolymarketMatchTagIDs
	}
	wantSet := make(map[int]struct{}, len(wantTag))
	for _, id := range wantTag {
		wantSet[id] = struct{}{}
	}
	var sportTags []int
	for _, part := range strings.Split(tagsCSV, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id == 1 {
			continue
		}
		sportTags = append(sportTags, id)
	}
	_ = sportID
	var out []int
	seen := make(map[int]struct{})
	for _, id := range sportTags {
		if _, ok := wantSet[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// allTagIDsForLeagueFetch 联赛下所有可用 tag（含 100639/100350），按 sport 绑定拉取，避免全站混扫。
func allTagIDsForLeagueFetch(meta httpentity.SportsMetadata, item polymarketSportItem) []int {
	if len(item.TagIDs) > 0 {
		var out []int
		for _, id := range item.TagIDs {
			if _, shared := polymarketSharedTagIDs[id]; shared {
				continue
			}
			out = append(out, id)
		}
		sort.Ints(out)
		if len(out) > 0 {
			return out
		}
	}
	seen := make(map[int]struct{})
	var out []int
	add := func(id int) {
		if id <= 0 || id == 1 || id == 82 {
			return
		}
		if _, shared := polymarketSharedTagIDs[id]; shared {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, part := range strings.Split(meta.Tags, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		add(id)
	}
	if meta.PrimaryTagID > 0 {
		add(meta.PrimaryTagID)
	}
	sort.Ints(out)
	return out
}

func mergeUniqueIntIDs(a, b []int) []int {
	seen := make(map[int]struct{}, len(a)+len(b))
	var out []int
	for _, slice := range [][]int{a, b} {
		for _, id := range slice {
			if id <= 0 {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
