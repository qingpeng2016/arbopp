package common

import (
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// SnapshotAtMsFromUnixMs 毫秒时间戳；无效时返回 0。
func SnapshotAtMsFromUnixMs(ms int64) int64 {
	if ms <= 0 {
		return 0
	}
	return ms
}

// SnapshotAtMsFromPolyTimestamp CLOB/WS 的 timestamp（秒、毫秒或 RFC3339）→ 毫秒。
func SnapshotAtMsFromPolyTimestamp(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n := book.ParseNum(raw)
	if n <= 0 {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return t.UnixMilli()
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t.UnixMilli()
		}
		return 0
	}
	if n < 1e12 {
		return int64(n) * 1000
	}
	return int64(n)
}

// SnapshotAtMsOrNow 无有效时间戳时用本机毫秒。
func SnapshotAtMsOrNow(atMs int64) int64 {
	if atMs > 0 {
		return atMs
	}
	return time.Now().UnixMilli()
}

// RestFetchSnapshotAtMs REST 拉盘口写入快照毫秒（本机拉取时刻）。
func RestFetchSnapshotAtMs() int64 {
	return time.Now().UnixMilli()
}