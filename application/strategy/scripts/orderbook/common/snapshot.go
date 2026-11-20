package common

import (
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

// SnapshotSecondFromUnixMs 毫秒时间戳 → Unix 秒；无效时返回 0。
func SnapshotSecondFromUnixMs(ms int64) int64 {
	if ms <= 0 {
		return 0
	}
	return ms / 1000
}

// SnapshotSecondFromPolyTimestamp CLOB/WS 的 timestamp（秒、毫秒或 RFC3339）。
func SnapshotSecondFromPolyTimestamp(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n := book.ParseNum(raw)
	if n <= 0 {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return t.Unix()
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t.Unix()
		}
		return 0
	}
	if n < 1e12 {
		return int64(n)
	}
	return int64(n) / 1000
}

// SnapshotSecondOrNow 接口无时间戳时 fallback 为本地拉取秒。
func SnapshotSecondOrNow(sec int64) int64 {
	if sec > 0 {
		return sec
	}
	return time.Now().Unix()
}

// RestFetchSnapshotSecond REST 拉盘口写入 market_order_book 的快照秒（拉取时刻，不用平台返回的陈旧 timestamp）。
func RestFetchSnapshotSecond() int64 {
	return time.Now().Unix()
}