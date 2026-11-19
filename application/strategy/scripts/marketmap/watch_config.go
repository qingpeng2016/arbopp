package marketmap

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	ConfigGroupStrategyEntry = "strategy_entry"
	ConfigKeyWatchMapKeys      = "watch_map_keys"
	ConfigKeyMaxGroups           = "max_groups"
	ConfigKeyWsConnectStaggerMs  = "ws_connect_stagger_ms"
)

const (
	DefaultWsStaggerMinMs = 0
	DefaultWsStaggerMaxMs = 2000
)

// WsStaggerMs 首次建连前随机等待区间 [MinMs, MaxMs]（毫秒）。
type WsStaggerMs struct {
	MinMs int `json:"min_ms"`
	MaxMs int `json:"max_ms"`
}

// ParseWsStaggerMs 解析 ws_connect_stagger_ms；无效字段回退默认 0～2000。
func ParseWsStaggerMs(raw string) WsStaggerMs {
	out := WsStaggerMs{MinMs: DefaultWsStaggerMinMs, MaxMs: DefaultWsStaggerMaxMs}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	var j WsStaggerMs
	if err := json.Unmarshal([]byte(raw), &j); err == nil {
		return j.Normalized()
	}
	parts := strings.Split(raw, ",")
	if len(parts) == 2 {
		min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil {
			return WsStaggerMs{MinMs: min, MaxMs: max}.Normalized()
		}
	}
	return out
}

func (w WsStaggerMs) Normalized() WsStaggerMs {
	min, max := w.MinMs, w.MaxMs
	if min < 0 {
		min = 0
	}
	if max < 0 {
		max = DefaultWsStaggerMaxMs
	}
	if max < min {
		max = min
	}
	return WsStaggerMs{MinMs: min, MaxMs: max}
}

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