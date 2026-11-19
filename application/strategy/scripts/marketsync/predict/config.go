package predict

import (
	"encoding/json"
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

type predictSportItem struct {
	Name     string   `json:"name"`
	NameEn   string   `json:"name_en"`
	TagIDs   []string `json:"tag_ids"`
	SportID  string   `json:"sport_id"`
	NameDesc string   `json:"name_desc"`
}

type predictSportsGroup struct {
	SportCN          string             `json:"sport_cn"`
	SportEn          string             `json:"sport_en"`
	SubCategory      string             `json:"sub_category"`
	SyncAllSoccer    bool               `json:"sync_all_soccer"`
	SoccerRootTagID  string             `json:"soccer_root_tag_id"`
	SyncAllSub       bool               `json:"sync_all_sub"`
	SyncRootTagID    string             `json:"sync_root_tag_id"`
	MarketVariants   []string           `json:"market_variants"`
	Items            []predictSportItem `json:"items"`
}

// leagueNames 与 polymarket_sports / kalshi_game_series 对齐的联赛展示名。
type leagueNames struct {
	CN string
	EN string
}

// SportsBySubCategory 子类 -> 联赛配置。
type SportsBySubCategory map[cryptoPredictionEnum.SubMarketCategory][]predictSportItem

// SportsConfig market_sync.predict_sports。
type SportsConfig struct {
	BySubCategory        SportsBySubCategory
	CompetitionBySportID map[string]string
	CompetitionEnByTag   map[string]string // key: sport_id 或 tag_id
	TagToCompetition     map[string]leagueNames
	SyncRootTagBySub     map[cryptoPredictionEnum.SubMarketCategory]string
	SyncRootLabelBySub   map[cryptoPredictionEnum.SubMarketCategory]leagueNames
	MarketVariants       map[cryptoPredictionEnum.SubMarketCategory][]string
	Active               bool
}

// LoadSportsConfig 从 config_strategy 解析。
func LoadSportsConfig(raw, valueType string) SportsConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SportsConfig{}
	}
	valueType = strings.ToLower(strings.TrimSpace(valueType))
	if valueType != "" && valueType != "json" && valueType != "json_object" && !strings.HasPrefix(raw, "[") {
		return SportsConfig{}
	}
	var groups []predictSportsGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil || len(groups) == 0 {
		return SportsConfig{}
	}
	bySub := make(SportsBySubCategory)
	names := make(map[string]string)
	namesEn := make(map[string]string)
	tagToLeague := make(map[string]leagueNames)
	syncRootBySub := make(map[cryptoPredictionEnum.SubMarketCategory]string)
	syncRootLabel := make(map[cryptoPredictionEnum.SubMarketCategory]leagueNames)
	variants := make(map[cryptoPredictionEnum.SubMarketCategory][]string)
	for _, g := range groups {
		sub := parseSubMarketCategory(g.SubCategory)
		if sub == "" {
			continue
		}
		if len(g.MarketVariants) > 0 {
			variants[sub] = append(variants[sub], g.MarketVariants...)
		}
		if root := predictGroupSyncRoot(g); root != "" {
			syncRootBySub[sub] = root
			syncRootLabel[sub] = predictSyncRootLabel(g, sub)
		}
		if len(g.Items) == 0 {
			if _, ok := syncRootBySub[sub]; ok {
				bySub[sub] = nil
			}
			continue
		}
		for _, it := range g.Items {
			sid := strings.TrimSpace(it.SportID)
			if sid == "" && len(it.TagIDs) > 0 {
				sid = strings.TrimSpace(it.TagIDs[0])
			}
			cn := strings.TrimSpace(it.Name)
			en := strings.TrimSpace(it.NameEn)
			if en == "" {
				en = strings.TrimSpace(it.NameDesc)
			}
			if sid != "" {
				if cn != "" {
					names[sid] = cn
				}
				if en != "" {
					namesEn[sid] = en
				}
			}
			for _, tid := range it.TagIDs {
				tid = strings.TrimSpace(tid)
				if tid == "" || cn == "" {
					continue
				}
				if en == "" {
					en = cn
				}
				tagToLeague[tid] = leagueNames{CN: cn, EN: en}
			}
			bySub[sub] = append(bySub[sub], it)
		}
	}
	if len(bySub) == 0 && len(syncRootBySub) == 0 {
		return SportsConfig{}
	}
	return SportsConfig{
		BySubCategory:        bySub,
		CompetitionBySportID: names,
		CompetitionEnByTag:   namesEn,
		TagToCompetition:     tagToLeague,
		SyncRootTagBySub:     syncRootBySub,
		SyncRootLabelBySub:   syncRootLabel,
		MarketVariants:       variants,
		Active:               true,
	}
}

func predictGroupSyncRoot(g predictSportsGroup) string {
	if g.SyncAllSoccer {
		root := strings.TrimSpace(g.SoccerRootTagID)
		if root == "" {
			root = "14"
		}
		return root
	}
	if g.SyncAllSub {
		return strings.TrimSpace(g.SyncRootTagID)
	}
	return ""
}

func predictSyncRootLabel(g predictSportsGroup, sub cryptoPredictionEnum.SubMarketCategory) leagueNames {
	cn := strings.TrimSpace(g.SportCN)
	en := strings.TrimSpace(g.SportEn)
	if cn == "" {
		cn = string(sub)
	}
	if en == "" {
		en = cn
	}
	return leagueNames{CN: cn, EN: en}
}

func parseSubMarketCategory(s string) cryptoPredictionEnum.SubMarketCategory {
	s = strings.TrimSpace(s)
	if s == "" {
		return cryptoPredictionEnum.SubMarketCategoryDefault
	}
	return cryptoPredictionEnum.SubMarketCategory(s)
}

func marketVariantsForSub(sub cryptoPredictionEnum.SubMarketCategory, cfg SportsConfig) []string {
	if v, ok := cfg.MarketVariants[sub]; ok && len(v) > 0 {
		return v
	}
	return []string{"SPORTS_MATCH", "SPORTS_PROPS"}
}

func predictLeaguesForSubCategory(sub cryptoPredictionEnum.SubMarketCategory, cfg SportsConfig) []predictSportItem {
	if root, ok := cfg.SyncRootTagBySub[sub]; ok && root != "" {
		label := cfg.SyncRootLabelBySub[sub]
		if label.CN == "" {
			label = predictSyncRootLabel(predictSportsGroup{SportCN: string(sub), SportEn: string(sub)}, sub)
		}
		return []predictSportItem{{
			Name:    label.CN,
			NameEn:  label.EN,
			TagIDs:  []string{root},
			SportID: predictSyncRootSportID(sub),
		}}
	}
	items := cfg.BySubCategory[sub]
	return collapsePredictLeagueItems(items)
}

func predictSyncRootSportID(sub cryptoPredictionEnum.SubMarketCategory) string {
	if sub == cryptoPredictionEnum.SubMarketCategoryFootball {
		return "soccer"
	}
	return strings.ToLower(string(sub))
}

func predictIsSyncRootPass(sub cryptoPredictionEnum.SubMarketCategory, item predictSportItem, cfg SportsConfig) bool {
	root, ok := cfg.SyncRootTagBySub[sub]
	if !ok || root == "" {
		return false
	}
	if len(item.TagIDs) != 1 {
		return false
	}
	return strings.TrimSpace(item.TagIDs[0]) == root
}

func predictItemHasTagIDs(it predictSportItem) bool {
	for _, id := range it.TagIDs {
		if strings.TrimSpace(id) != "" {
			return true
		}
	}
	return false
}

// collapsePredictLeagueItems 避免多个联赛 tag_ids 都为空时并发重复扫全站（只跑一轮全量）。
// 已配置 tag 的联赛照常逐个拉取并各发一条 Lark「联赛完成」。
func collapsePredictLeagueItems(items []predictSportItem) []predictSportItem {
	if len(items) == 0 {
		return nil
	}
	var withTags []predictSportItem
	for _, it := range items {
		if predictItemHasTagIDs(it) {
			withTags = append(withTags, it)
		}
	}
	if len(withTags) > 0 {
		return withTags
	}
	// 仅一次全站拉取；联赛名不从占位符写入 DB（见 resolveCompetition）。
	return []predictSportItem{{TagIDs: nil}}
}