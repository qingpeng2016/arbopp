package sxbet

import (
	"fmt"
	"strings"
	"unicode"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

const sxbetWebBase = "https://sx.bet"

// 前台场次页：/sports/{sportSlug}/{leagueSlug}/{sportXeventId}
// 盘口页在同路径下追加 /market/{marketHash}（非 /event/{id}、/market/{hash}）
func buildSxbetEventURLFromMarket(m httpentity.SxbetMarket) string {
	eventID := strings.TrimSpace(m.SportXeventID)
	if eventID == "" {
		return sxbetWebBase
	}
	sport := sportSlug(m.SportID, m.SportLabel)
	league := leagueSlug(m.Group1, m.LeagueLabel)
	if sport == "" || league == "" {
		return sxbetWebBase
	}
	return fmt.Sprintf("%s/sports/%s/%s/%s", sxbetWebBase, sport, league, eventID)
}

func buildSxbetMarketURLFromMarket(m httpentity.SxbetMarket) string {
	hash := strings.TrimSpace(m.MarketHash)
	if hash == "" {
		return buildSxbetEventURLFromMarket(m)
	}
	base := buildSxbetEventURLFromMarket(m)
	if base == sxbetWebBase {
		return base
	}
	return base + "/market/" + hash
}

// sportSlug 与 sx.bet 侧栏路由一致（sportId 优先）。
func sportSlug(sportID int, sportLabel string) string {
	if s, ok := sxbetSportSlugByID[sportID]; ok && s != "" {
		return s
	}
	return slugify(sportLabel)
}

var sxbetSportSlugByID = map[int]string{
	1:  "basketball",
	2:  "hockey",
	3:  "baseball",
	4:  "golf",
	5:  "soccer",
	6:  "tennis",
	7:  "mixed-martial-arts",
	8:  "football",
	9:  "esports",
	11: "rugby-union",
	13: "boxing",
	15: "cricket",
	20: "rugby-league",
}

func leagueSlug(group1, leagueLabel string) string {
	if s := slugify(group1); s != "" {
		return s
	}
	return slugify(leagueLabel)
}

func slugify(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(raw) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}