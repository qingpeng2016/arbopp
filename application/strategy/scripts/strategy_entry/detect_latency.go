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

// mapLatencyStats：ws_* = RecvAt→下单前；compute_* = 拿锁后纯试算。
type mapLatencyStats struct {
	MinMs         float64 `json:"min_ms"`
	MaxMs         float64 `json:"max_ms"`
	Samples       int64   `json:"samples"`
	ComputeMinMs  float64 `json:"compute_min_ms"`
	ComputeMaxMs  float64 `json:"compute_max_ms"`
	ComputeSamples int64  `json:"compute_samples"`
}

// detectLatencyPayload 写入 stat_kv 的单条 JSON。
type detectLatencyPayload struct {
	RefreshedAt string                     `json:"refreshed_at"`
	ByMapKey    map[string]mapLatencyStats `json:"by_map_key"`
	ByPlatform  map[string]mapLatencyStats `json:"by_platform"`
}

// DetectLatency 按 map_key / platform 累计耗时（进程内存，心跳落盘）。
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

func (d *DetectLatency) Record(mapKey, platform string, wsElapsed, computeElapsed time.Duration) {
	if d == nil {
		return
	}
	wsMs, computeMs := -1.0, -1.0
	if wsElapsed >= 0 {
		wsMs = float64(wsElapsed.Nanoseconds()) / 1e6
	}
	if computeElapsed >= 0 {
		computeMs = float64(computeElapsed.Nanoseconds()) / 1e6
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if mapKey != "" {
		d.byKey[mapKey] = bumpLatency(d.byKey[mapKey], wsMs, computeMs)
	}
	if p := strings.ToUpper(strings.TrimSpace(platform)); p != "" {
		d.byPlatform[p] = bumpLatency(d.byPlatform[p], wsMs, computeMs)
	}
}

func bumpLatency(st mapLatencyStats, wsMs, computeMs float64) mapLatencyStats {
	if wsMs >= 0 {
		if st.Samples == 0 {
			st.MinMs, st.MaxMs, st.Samples = wsMs, wsMs, 1
		} else {
			st.Samples++
			if wsMs < st.MinMs {
				st.MinMs = wsMs
			}
			if wsMs > st.MaxMs {
				st.MaxMs = wsMs
			}
		}
	}
	if computeMs >= 0 {
		if st.ComputeSamples == 0 {
			st.ComputeMinMs, st.ComputeMaxMs, st.ComputeSamples = computeMs, computeMs, 1
		} else {
			st.ComputeSamples++
			if computeMs < st.ComputeMinMs {
				st.ComputeMinMs = computeMs
			}
			if computeMs > st.ComputeMaxMs {
				st.ComputeMaxMs = computeMs
			}
		}
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
			MinMs:          roundMs(v.MinMs),
			MaxMs:          roundMs(v.MaxMs),
			Samples:        v.Samples,
			ComputeMinMs:   roundMs(v.ComputeMinMs),
			ComputeMaxMs:   roundMs(v.ComputeMaxMs),
			ComputeSamples: v.ComputeSamples,
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

func (e *Entry) recordDetectLatencyYesno(mapKey, platform string, wsElapsed, computeElapsed time.Duration) {
	if e == nil || e.detectLatYesno == nil {
		return
	}
	e.detectLatYesno.Record(mapKey, platform, wsElapsed, computeElapsed)
}

func (e *Entry) recordDetectLatencyOnextwo(mapKey, platform string, wsElapsed, computeElapsed time.Duration) {
	if e == nil || e.detectLatOnextwo == nil {
		return
	}
	e.detectLatOnextwo.Record(mapKey, platform, wsElapsed, computeElapsed)
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
		{statKeyDetectLatencyYesno, "strategy_entry yesno：min/max=WS→下单前；compute_*=纯试算(ms，不含发单)", e.detectLatYesno},
		{statKeyDetectLatencyOnextwo, "strategy_entry onextwo：min/max=WS→下单前；compute_*=纯试算(ms，不含发单)", e.detectLatOnextwo},
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
