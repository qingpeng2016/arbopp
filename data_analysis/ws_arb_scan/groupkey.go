package main

import (
	"regexp"
	"strings"
)

var (
	reDraw   = regexp.MustCompile(`(?i)(draw|tie|end in a (?:draw|tie))`)
	reKalWin = regexp.MustCompile(`(?i)^(.+?)\s+wins?\s*$`)
	reVs     = regexp.MustCompile(`(?i)^(.+?)\s+vs\.?\s+(.+?)(\s*\[|$)`)
)

// outcomeGroupKey 对齐线上一杯键：v1|联赛|场次slug|赛果norm。
// 若 JSONL 已带 outcome_group_key 则直接用（推荐导出时写入）。
func outcomeGroupKey(o outcomeRec) string {
	if k := strings.TrimSpace(o.OutcomeGroupKey); k != "" {
		return k
	}
	fixture := strings.TrimSpace(o.MapKey)
	if fixture == "" {
		return ""
	}
	norm := normOutcome(o)
	if norm == "" {
		return ""
	}
	parts := strings.Split(fixture, "|")
	if len(parts) >= 3 {
		return fixture + "|" + norm
	}
	return "v1||" + fixture + "|" + norm
}

func normOutcome(o outcomeRec) string {
	name := strings.TrimSpace(o.OutcomeName)
	slug := strings.ToLower(strings.TrimSpace(o.Slug))
	plat := strings.ToUpper(strings.TrimSpace(o.Platform))

	if name != "" && reDraw.MatchString(name) {
		return "draw"
	}
	if m := reKalWin.FindStringSubmatch(name); len(m) > 1 {
		return slugifyTeam(m[1])
	}
	if strings.Contains(strings.ToLower(name), " vs not ") {
		left := strings.Split(name, " vs ")[0]
		return slugifyTeam(left)
	}
	if plat == platPoly && slug != "" {
		segs := strings.Split(slug, "-")
		if len(segs) > 0 {
			code := segs[len(segs)-1]
			if n := polyCodeNorm(code); n != "" {
				return n
			}
			return slugifyTeam(code)
		}
	}
	if m := reVs.FindStringSubmatch(name); len(m) > 1 {
		return slugifyTeam(m[1])
	}
	return slugifyTeam(name)
}

func slugifyTeam(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "will ")
	if idx := strings.Index(s, "["); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.Trim(s, "_")
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

// 常见 Poly slug 尾码（分析用子集；不全时靠 outcome_name / 自备 outcome_group_key）
func polyCodeNorm(code string) string {
	code = strings.ToLower(code)
	table := map[string]string{
		"ast": "aston_villa", "sun": "sunderland", "ars": "arsenal", "new": "newcastle",
		"tot": "tottenham", "che": "chelsea", "liv": "liverpool", "mun": "manchester_united",
		"mil": "milan", "int": "inter", "dor": "dortmund", "bay": "bayern_munich",
	}
	if v, ok := table[code]; ok {
		return v
	}
	return ""
}

func lastSegmentGroupKey(gk string) string {
	parts := strings.Split(gk, "|")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}