package sxbet

import (
	"encoding/json"
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

type sxbetLeagueItem struct {
	Name     string `json:"name"`
	NameEn   string `json:"name_en"`
	NameDesc string `json:"name_desc"`
	LeagueID int    `json:"league_id"`
	PairPoly string `json:"pair_poly_sport_id,omitempty"`
}

type sxbetLeagueFlatEntry struct {
	Name        string `json:"name"`
	SubCategory string `json:"sub_category"`
	LeagueID    int    `json:"league_id"`
}

type sxbetLeagueGroup struct {
	SportCN     string            `json:"sport_cn"`
	SubCategory string            `json:"sub_category"`
	Items       []sxbetLeagueItem `json:"items"`
}

// LeagueMeta 一条联赛的展示名。
type LeagueMeta struct {
	Name     string
	NameEn   string
	NameDesc string
	PairPoly string
}

// LeaguesConfig market_sync 中 SXBET 的 market_category / sxbet_leagues JSON。
type LeaguesConfig struct {
	BySubCategory       map[cryptoPredictionEnum.SubMarketCategory][]int
	CompetitionByLeague map[int]string
	LeagueByID          map[int]LeagueMeta
	Active              bool
}

func parseSubMarketCategory(s string) cryptoPredictionEnum.SubMarketCategory {
	s = strings.TrimSpace(s)
	if s == "" {
		return cryptoPredictionEnum.SubMarketCategoryDefault
	}
	return cryptoPredictionEnum.SubMarketCategory(s)
}

func leagueDesc(it sxbetLeagueItem) string {
	if desc := strings.TrimSpace(it.NameDesc); desc != "" {
		return desc
	}
	return strings.TrimSpace(it.NameEn)
}

func parseLeaguesBySub(raw, valueType string) map[cryptoPredictionEnum.SubMarketCategory][]int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	valueType = strings.ToLower(strings.TrimSpace(valueType))
	out := make(map[cryptoPredictionEnum.SubMarketCategory][]int)
	add := func(sub cryptoPredictionEnum.SubMarketCategory, leagueID int) {
		if sub == "" || leagueID <= 0 {
			return
		}
		for _, existing := range out[sub] {
			if existing == leagueID {
				return
			}
		}
		out[sub] = append(out[sub], leagueID)
	}
	if valueType == "json" || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		var flat []sxbetLeagueFlatEntry
		if err := json.Unmarshal([]byte(raw), &flat); err == nil && len(flat) > 0 && flat[0].LeagueID > 0 {
			for _, e := range flat {
				add(parseSubMarketCategory(e.SubCategory), e.LeagueID)
			}
			if len(out) > 0 {
				return out
			}
		}
		var groups []sxbetLeagueGroup
		if err := json.Unmarshal([]byte(raw), &groups); err == nil {
			for _, g := range groups {
				sub := parseSubMarketCategory(g.SubCategory)
				for _, it := range g.Items {
					add(sub, parseLeagueID(it))
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return out
}

func parseLeagueID(it sxbetLeagueItem) int {
	if it.LeagueID > 0 {
		return it.LeagueID
	}
	return 0
}

func parseLeagueMeta(raw string) map[int]LeagueMeta {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	put := func(out map[int]LeagueMeta, leagueID int, name, nameEn, nameDesc, pair string) {
		if leagueID <= 0 {
			return
		}
		out[leagueID] = LeagueMeta{
			Name:     strings.TrimSpace(name),
			NameEn:   strings.TrimSpace(nameEn),
			NameDesc: strings.TrimSpace(nameDesc),
			PairPoly: strings.ToLower(strings.TrimSpace(pair)),
		}
	}
	var groups []sxbetLeagueGroup
	if err := json.Unmarshal([]byte(raw), &groups); err == nil && len(groups) > 0 && len(groups[0].Items) > 0 {
		out := make(map[int]LeagueMeta)
		for _, g := range groups {
			for _, it := range g.Items {
				id := parseLeagueID(it)
				put(out, id, it.Name, it.NameEn, leagueDesc(it), it.PairPoly)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var flat []sxbetLeagueFlatEntry
	if err := json.Unmarshal([]byte(raw), &flat); err == nil {
		out := make(map[int]LeagueMeta)
		for _, e := range flat {
			put(out, e.LeagueID, e.Name, "", "", "")
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func parseCompetitionByLeague(meta map[int]LeagueMeta) map[int]string {
	if len(meta) == 0 {
		return nil
	}
	out := make(map[int]string, len(meta))
	for id, m := range meta {
		if m.Name != "" {
			out[id] = m.Name
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// LoadLeaguesConfig 从 config_strategy / market_category 解析。
func LoadLeaguesConfig(raw, valueType string) LeaguesConfig {
	parsed := parseLeaguesBySub(raw, valueType)
	if len(parsed) == 0 {
		return LeaguesConfig{}
	}
	meta := parseLeagueMeta(raw)
	return LeaguesConfig{
		BySubCategory:       parsed,
		CompetitionByLeague: parseCompetitionByLeague(meta),
		LeagueByID:          meta,
		Active:              true,
	}
}

func sxbetLeaguesForSubCategory(sub cryptoPredictionEnum.SubMarketCategory, cfg LeaguesConfig) []int {
	return cfg.BySubCategory[sub]
}