package strategyentry

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
)

const (
	statGroupDetectLatency      = "strategy_entry_detect_latency"
	statKeyDetectLatencyYesno   = "ws_to_arb_check_ms_yesno"
	statKeyDetectLatencyOnextwo = "ws_to_arb_check_ms_onextwo"
)

// mapLatencyStats 单维度：WS收包→下单前（ms），不含发单网络。
type mapLatencyStats struct {
	MinMs   float64 `json:"min_ms"`
	MaxMs   float64 `json:"max_ms"`
	Samples int64   `json:"samples"`
}

// detectLatencyPayload 写入 stat_kv 的单条 JSON。
type detectLatencyPayload struct {
	RefreshedAt string                     `json:"refreshed_at"`
	ByMapKey    map[string]mapLatencyStats `json:"by_map_key"`
	ByPlatform  map[string]mapLatencyStats `json:"by_platform"`
}

// DetectLatency 按 map_key / platform 累计 WS→下单前耗时（进程内存，心跳落盘）。
type DetectLatency struct {
	mu         sync.Mutex
	byKey      map[string]mapLatencyStats
	byPlatform map[string]mapLatencyStats
}

func NewDetectLatency() *DetectLatency {
	return &DetectLatency{
		byKey:      map[string]mapLatencyStats{},
		byPlatform: map[string]mapLatencyStats{},
	}
}

func (d *DetectLatency) Record(mapKey, platform string, elapsed time.Duration) {
	if d == nil || elapsed < 0 {
		return
	}
	ms := float64(elapsed.Nanoseconds()) / 1e6
	d.mu.Lock()
	defer d.mu.Unlock()
	if mapKey != "" {
		d.byKey[mapKey] = bumpLatency(d.byKey[mapKey], ms)
	}
	if p := strings.ToUpper(strings.TrimSpace(platform)); p != "" {
		d.byPlatform[p] = bumpLatency(d.byPlatform[p], ms)
	}
}

func bumpLatency(st mapLatencyStats, ms float64) mapLatencyStats {
	if st.Samples == 0 {
		return mapLatencyStats{MinMs: ms, MaxMs: ms, Samples: 1}
	}
	st.Samples++
	if ms < st.MinMs {
		st.MinMs = ms
	}
	if ms > st.MaxMs {
		st.MaxMs = ms
	}
	return st
}

func (d *DetectLatency) snapshot() detectLatencyPayload {
	if d == nil {
		return detectLatencyPayload{
			ByMapKey:   map[string]mapLatencyStats{},
			ByPlatform: map[string]mapLatencyStats{},
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return detectLatencyPayload{
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
		ByMapKey:    copyLatencyMap(d.byKey),
		ByPlatform:  copyLatencyMap(d.byPlatform),
	}
}

func copyLatencyMap(in map[string]mapLatencyStats) map[string]mapLatencyStats {
	out := make(map[string]mapLatencyStats, len(in))
	for k, v := range in {
		out[k] = mapLatencyStats{
			MinMs:   roundMs(v.MinMs),
			MaxMs:   roundMs(v.MaxMs),
			Samples: v.Samples,
		}
	}
	return out
}

func roundMs(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*1000) / 1000
}

func (e *Entry) recordDetectLatencyYesno(mapKey, platform string, elapsed time.Duration) {
	if e == nil || e.detectLatYesno == nil {
		return
	}
	e.detectLatYesno.Record(mapKey, platform, elapsed)
}

func (e *Entry) recordDetectLatencyOnextwo(mapKey, platform string, elapsed time.Duration) {
	if e == nil || e.detectLatOnextwo == nil {
		return
	}
	e.detectLatOnextwo.Record(mapKey, platform, elapsed)
}

func (e *Entry) flushDetectLatencyStat(ctx context.Context) {
	if e == nil || e.statRepo == nil {
		return
	}
	rows := make([]*persistententity.StatKV, 0, 2)
	for _, spec := range []struct {
		key  string
		desc string
		lat  *DetectLatency
	}{
		{statKeyDetectLatencyYesno, "strategy_entry yesno：WS收包→下单前(ms，不含发单)；by_map_key + by_platform", e.detectLatYesno},
		{statKeyDetectLatencyOnextwo, "strategy_entry onextwo：WS收包→下单前(ms，不含发单)；by_map_key + by_platform", e.detectLatOnextwo},
	} {
		if spec.lat == nil {
			continue
		}
		payload := spec.lat.snapshot()
		raw, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			notification.SendErrorLog(ctx, "strategy_entry-detect-latency-marshal", zap.String("item_key", spec.key), zap.Error(err))
			continue
		}
		rows = append(rows, &persistententity.StatKV{
			GroupName:   statGroupDetectLatency,
			ItemKey:     spec.key,
			ItemValue:   string(raw),
			ValueType:   "json",
			Description: spec.desc,
			SortNo:      0,
			IsEnabled:   1,
		})
	}
	if len(rows) == 0 {
		return
	}
	if err := e.statRepo.UpsertBatch(ctx, nil, rows); err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-detect-latency-stat_kv", zap.Error(err))
		return
	}
	yesnoN, onextwoN := 0, 0
	if e.detectLatYesno != nil {
		yesnoN = len(e.detectLatYesno.snapshot().ByMapKey)
	}
	if e.detectLatOnextwo != nil {
		onextwoN = len(e.detectLatOnextwo.snapshot().ByMapKey)
	}
	logger.GetLogger().Info("strategy_entry-detect-latency-flushed",
		zap.Int("yesno_map_keys", yesnoN),
		zap.Int("onextwo_map_keys", onextwoN),
		zap.String("group", statGroupDetectLatency))
}
