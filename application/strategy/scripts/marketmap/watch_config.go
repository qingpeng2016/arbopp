package marketmap

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	ConfigGroupStrategyEntry = "strategy_entry"
	ConfigKeyWatchMapKeys      = "watch_map_keys"
	ConfigKeyMaxGroups         = "max_groups"
)

// ParseWatchMapKeys 解析 watch_map_keys：JSON 字符串数组，或单个 map_key 明文。
func ParseWatchMapKeys(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		return uniqueNonEmpty(arr)
	}
	return []string{raw}
}

func uniqueNonEmpty(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// ParseMaxGroups 同时监控的 map_key 组数上限；无效或未配置时返回 defaultVal。
func ParseMaxGroups(raw string, defaultVal int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}