package predict

import (
	"strings"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// resolveCompetition 单联赛拉取时用配置名；全站时 tag 映射 → slug 前缀 sport_id（同 Poly）→ 最后才用 API tag 名。
func resolveCompetition(cat httpentity.PredictCategory, cfgCn, cfgEn string, useConfigLeague bool, cfg SportsConfig) (cn, en string) {
	if useConfigLeague {
		return strings.TrimSpace(cfgCn), strings.TrimSpace(cfgEn)
	}
	tagToCompetition := cfg.TagToCompetition
	if len(tagToCompetition) > 0 {
		bestLevel := -1
		var pick leagueNames
		for _, t := range cat.Tags {
			tid := strings.TrimSpace(t.ID)
			if tid == "" {
				continue
			}
			ln, ok := tagToCompetition[tid]
			if !ok || strings.TrimSpace(ln.CN) == "" {
				continue
			}
			lvl := -1
			if t.Level != nil {
				lvl = *t.Level
			}
			if lvl >= bestLevel {
				bestLevel = lvl
				pick = ln
			}
		}
		if pick.CN != "" {
			en := strings.TrimSpace(pick.EN)
			if en == "" {
				en = pick.CN
			}
			return pick.CN, en
		}
	}
	if fid := predictMainFixtureID(cat.Slug); fid != "" {
		if cn, en, ok := leagueFromSportIDPrefix(fid, cfg); ok {
			return cn, en
		}
	}
	return competitionFromCategoryTags(cat.Tags)
}

// leagueFromSportIDPrefix 与 polymarketCompetitionName 一致：slug 前缀命中 predict_sports sport_id。
func leagueFromSportIDPrefix(fixtureSlug string, cfg SportsConfig) (cn, en string, ok bool) {
	fixtureSlug = strings.ToLower(strings.TrimSpace(fixtureSlug))
	if fixtureSlug == "" || len(cfg.CompetitionBySportID) == 0 {
		return "", "", false
	}
	for sid, name := range cfg.CompetitionBySportID {
		sid = strings.ToLower(strings.TrimSpace(sid))
		if sid == "" || strings.TrimSpace(name) == "" {
			continue
		}
		if fixtureSlug == sid || strings.HasPrefix(fixtureSlug, sid+"-") {
			cn = strings.TrimSpace(name)
			en = strings.TrimSpace(cfg.CompetitionEnByTag[sid])
			if en == "" {
				en = cn
			}
			return cn, en, true
		}
	}
	return "", "", false
}

func competitionFromCategoryTags(tags []httpentity.PredictTag) (cn, en string) {
	if len(tags) == 0 {
		return "", ""
	}
	// 取 level 最大（越具体）的 tag 名作为联赛/赛事类型展示。
	bestLevel := -1
	var name string
	for _, t := range tags {
		n := strings.TrimSpace(t.Name)
		if n == "" {
			continue
		}
		lvl := -1
		if t.Level != nil {
			lvl = *t.Level
		}
		if lvl >= bestLevel {
			bestLevel = lvl
			name = n
		}
	}
	if name == "" {
		name = strings.TrimSpace(tags[0].Name)
	}
	return name, name
}