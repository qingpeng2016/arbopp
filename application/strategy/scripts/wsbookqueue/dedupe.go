package wsbookqueue

import (
	"strconv"
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// DedupeBatch 同批内按 (platform, outcome, token, snapshot_at_ms) 去重（当前 flush 未调用；保留备用）。
func DedupeBatch(list []*persistententity.OrderBookWS) []*persistententity.OrderBookWS {
	if len(list) <= 1 {
		return list
	}
	best := make(map[string]*persistententity.OrderBookWS, len(list))
	for _, row := range list {
		if row == nil {
			continue
		}
		k := dedupeKey(row)
		best[k] = row
	}
	out := make([]*persistententity.OrderBookWS, 0, len(best))
	for _, row := range best {
		out = append(out, row)
	}
	return out
}

func dedupeKey(b *persistententity.OrderBookWS) string {
	return strings.ToUpper(strings.TrimSpace(b.Platform)) + "\x00" +
		strings.TrimSpace(b.PlatformMarketOutcomeID) + "\x00" +
		strings.TrimSpace(b.TokenID) + "\x00" +
		strconv.FormatInt(b.SnapshotAtMs, 10)
}