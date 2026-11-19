package strategyentry

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
)

const (
	statGroupDetectLatency     = "strategy_entry_detect_latency"
	statKeyDetectLatencyYesno  = "ws_to_arb_check_ms_yesno"
	statKeyDetectLatencyOnextwo = "ws_to_arb_check_ms_onextwo"
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

func (e *Entry) recordDetectLatencyYesno(mapKey string, elapsed time.Duration) {
	if e == nil || e.detectLatYesno == nil {
		return
	}
	e.detectLatYesno.Record(mapKey, elapsed)
}

func (e *Entry) recordDetectLatencyOnextwo(mapKey string, elapsed time.Duration) {
	if e == nil || e.detectLatOnextwo == nil {
		return
	}
	e.detectLatOnextwo.Record(mapKey, elapsed)
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
		{statKeyDetectLatencyYesno, "strategy_entry yesno：各 map_key WS收包→二元检测耗时(ms)", e.detectLatYesno},
		{statKeyDetectLatencyOnextwo, "strategy_entry onextwo：各 map_key WS收包→1X2检测耗时(ms)", e.detectLatOnextwo},
	} {
		if spec.lat == nil {
			continue
		}
		payload := spec.lat.snapshot()
		raw, err := json.Marshal(payload)
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