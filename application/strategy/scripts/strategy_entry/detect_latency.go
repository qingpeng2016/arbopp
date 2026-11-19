package strategyentry

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"sync"
	"time"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
)

const (
	statGroupDetectLatency = "strategy_entry_detect_latency"
	statKeyDetectLatency   = "ws_to_arb_check_ms"
)

// mapLatencyStats 单场 map_key：WS 收到消息 → 套利检测结束的耗时（毫秒）。
type mapLatencyStats struct {
	MinMs    float64 `json:"min_ms"`
	MaxMs    float64 `json:"max_ms"`
	Samples  int64   `json:"samples"`
}

// detectLatencyPayload 写入 stat_kv 的单条 JSON。
type detectLatencyPayload struct {
	RefreshedAt string                       `json:"refreshed_at"`
	ByMapKey    map[string]mapLatencyStats   `json:"by_map_key"`
}

// DetectLatency 按 map_key 累计最快/最慢检测耗时（进程内存，心跳时落盘）。
type DetectLatency struct {
	mu    sync.Mutex
	byKey map[string]mapLatencyStats
}

func NewDetectLatency() *DetectLatency {
	return &DetectLatency{byKey: map[string]mapLatencyStats{}}
}

func (d *DetectLatency) Record(mapKey string, elapsed time.Duration) {
	if d == nil || mapKey == "" || elapsed < 0 {
		return
	}
	ms := float64(elapsed.Nanoseconds()) / 1e6
	d.mu.Lock()
	defer d.mu.Unlock()
	st, ok := d.byKey[mapKey]
	if !ok {
		d.byKey[mapKey] = mapLatencyStats{MinMs: ms, MaxMs: ms, Samples: 1}
		return
	}
	st.Samples++
	if ms < st.MinMs {
		st.MinMs = ms
	}
	if ms > st.MaxMs {
		st.MaxMs = ms
	}
	d.byKey[mapKey] = st
}

func (d *DetectLatency) snapshot() detectLatencyPayload {
	if d == nil {
		return detectLatencyPayload{ByMapKey: map[string]mapLatencyStats{}}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make(map[string]mapLatencyStats, len(d.byKey))
	for k, v := range d.byKey {
		out[k] = mapLatencyStats{
			MinMs:   roundMs(v.MinMs),
			MaxMs:   roundMs(v.MaxMs),
			Samples: v.Samples,
		}
	}
	return detectLatencyPayload{
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
		ByMapKey:    out,
	}
}

func roundMs(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*1000) / 1000
}

func (e *Entry) recordDetectLatency(mapKey string, elapsed time.Duration) {
	if e == nil || e.detectLat == nil {
		return
	}
	e.detectLat.Record(mapKey, elapsed)
}

func (e *Entry) flushDetectLatencyStat(ctx context.Context) {
	if e == nil || e.statRepo == nil || e.detectLat == nil {
		return
	}
	payload := e.detectLat.snapshot()
	keys := make([]string, 0, len(payload.ByMapKey))
	for k := range payload.ByMapKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	raw, err := json.Marshal(payload)
	if err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-detect-latency-marshal", zap.Error(err))
		return
	}
	row := &persistententity.StatKV{
		GroupName:   statGroupDetectLatency,
		ItemKey:     statKeyDetectLatency,
		ItemValue:   string(raw),
		ValueType:   "json",
		Description: "strategy_entry：各 map_key WS收包→套利检测耗时(ms) 最快/最慢，心跳周期刷入",
		SortNo:      0,
		IsEnabled:   1,
	}
	if err := e.statRepo.UpsertBatch(ctx, nil, []*persistententity.StatKV{row}); err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-detect-latency-stat_kv", zap.Error(err))
		return
	}
	logger.GetLogger().Info("strategy_entry-detect-latency-flushed",
		zap.Int("map_keys", len(keys)),
		zap.String("group", statGroupDetectLatency))
}