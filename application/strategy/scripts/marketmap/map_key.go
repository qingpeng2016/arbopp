package marketmap

import (
	"fmt"
	"strings"
)

const mapKeyVersion = "v1"

// buildFixtureMapKey 跨平台场次对齐键（≤192）：联赛 + 场次(slug)。
func buildFixtureMapKey(competition, fixtureID string) string {
	competition = strings.TrimSpace(competition)
	fixtureID = strings.TrimSpace(fixtureID)
	if competition == "" || fixtureID == "" {
		return ""
	}
	key := fmt.Sprintf("%s|%s|%s", mapKeyVersion, competition, fixtureID)
	if len(key) <= 192 {
		return key
	}
	return truncateRunes(key, 192)
}

// buildMapKey 跨平台同一腿对齐键（≤192）：联赛 + 场次 + 胜出方；arb 分组用。
func buildMapKey(competition, fixtureID, outcomeNorm string) string {
	competition = strings.TrimSpace(competition)
	fixtureID = strings.TrimSpace(fixtureID)
	outcomeNorm = strings.TrimSpace(outcomeNorm)
	if competition == "" || fixtureID == "" || outcomeNorm == "" {
		return ""
	}
	key := fmt.Sprintf("%s|%s|%s|%s", mapKeyVersion, competition, fixtureID, outcomeNorm)
	if len(key) <= 192 {
		return key
	}
	// 超长时截断 outcome，保留联赛+时间
	prefix := fmt.Sprintf("%s|%s|%s|", mapKeyVersion, competition, fixtureID)
	maxOut := 192 - len(prefix)
	if maxOut < 8 {
		return truncateRunes(key, 192)
	}
	outcomeNorm = truncateRunes(outcomeNorm, maxOut)
	return prefix + outcomeNorm
}

func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// BuildOutcomeGroupKeyFromMapKey 已核验 map_key + 归一化赛果（与 strategy_entry 跨所分组一致）。
func BuildOutcomeGroupKeyFromMapKey(mapKey, outcomeNorm string) string {
	mapKey = strings.TrimSpace(mapKey)
	outcomeNorm = strings.TrimSpace(outcomeNorm)
	if mapKey == "" || outcomeNorm == "" {
		return ""
	}
	key := mapKey + "|" + outcomeNorm
	if len(key) <= 192 {
		return key
	}
	prefix := mapKey + "|"
	maxOut := 192 - len(prefix)
	if maxOut < 8 {
		return truncateRunes(key, 192)
	}
	return prefix + truncateRunes(outcomeNorm, maxOut)
}

type fixtureGroupKey struct {
	competition   string
	competitionEn string
	fixtureID     string
}
