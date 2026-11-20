package strategyentry

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
)

const (
	statGroupMapKeyAudit           = "map_key_audit"
	statKeyMarketRunningCountEntry = "market_running_count_strategy"
)

type marketRunningGroup struct {
	MapKey       string `json:"map_key"`
	MarketRowIDs []uint `json:"market_row_ids"`
}

// flushMarketRunningCountStrategy 写入当前 strategy_entry 在跑场次（与 stat_collect market_running_count 同 JSON 结构）。
func (e *Entry) flushMarketRunningCountStrategy(ctx context.Context, wantByKey map[string]group) {
	if e == nil || e.statRepo == nil {
		if e != nil && e.statRepo == nil {
			notification.SendErrorLog(ctx, "strategy_entry-market_running_count_strategy-skipped",
				zap.String("reason", "statRepo nil"))
		}
		return
	}
	preMins := LoadPreEventScanMinutes(ctx, e.configStrategyRepo)
	groups := e.runningMarketGroups(wantByKey)
	desc := marketRunningDescJSON(preMins, groups)
	row := &persistententity.StatKV{
		GroupName:   statGroupMapKeyAudit,
		ItemKey:     statKeyMarketRunningCountEntry,
		ItemValue:   strconv.Itoa(len(groups)),
		ValueType:   "int",
		Description: desc,
		SortNo:      6,
		IsEnabled:   1,
	}
	if err := e.statRepo.UpsertBatch(ctx, nil, []*persistententity.StatKV{row}); err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-market_running_count_strategy-stat_kv", zap.Error(err))
		return
	}
	logger.InfoZ(ctx, "strategy_entry-market_running_count_strategy-stat_kv",
		zap.Int("count", len(groups)),
		zap.String("item_key", statKeyMarketRunningCountEntry))
}

func (e *Entry) runningMarketGroups(wantByKey map[string]group) []marketRunningGroup {
	e.mu.Lock()
	keys := make([]string, 0, len(e.watches))
	for k := range e.watches {
		keys = append(keys, k)
	}
	e.mu.Unlock()
	sort.Strings(keys)
	out := make([]marketRunningGroup, 0, len(keys))
	for _, k := range keys {
		g, ok := wantByKey[k]
		if !ok {
			out = append(out, marketRunningGroup{MapKey: k, MarketRowIDs: nil})
			continue
		}
		ids := marketRowIDsFromGroup(g)
		out = append(out, marketRunningGroup{MapKey: k, MarketRowIDs: ids})
	}
	return out
}

func marketRowIDsFromGroup(g group) []uint {
	if len(g.MarketsByRowID) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(g.MarketsByRowID))
	for id := range g.MarketsByRowID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func marketRunningDescJSON(preEventScanMinutes int, groups []marketRunningGroup) string {
	if groups == nil {
		groups = []marketRunningGroup{}
	}
	payload := struct {
		PreEventScanMinutes int                  `json:"pre_event_scan_minutes"`
		Groups              []marketRunningGroup `json:"groups"`
	}{
		PreEventScanMinutes: preEventScanMinutes,
		Groups:              groups,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return `{"pre_event_scan_minutes":` + strconv.Itoa(preEventScanMinutes) + `,"groups":[]}`
	}
	s := string(b)
	const descMax = 500
	if len(s) > descMax {
		compact := struct {
			PreEventScanMinutes int      `json:"pre_event_scan_minutes"`
			Count               int      `json:"count"`
			MapKeys             []string `json:"map_keys"`
			Truncated           bool     `json:"truncated"`
		}{
			PreEventScanMinutes: preEventScanMinutes,
			Count:               len(groups),
			Truncated:           true,
		}
		for _, g := range groups {
			compact.MapKeys = append(compact.MapKeys, strings.TrimSpace(g.MapKey))
		}
		b, err = json.Marshal(compact)
		if err != nil {
			return `{"truncated":true,"groups":[]}`
		}
		s = string(b)
		if len(s) > descMax {
			s = s[:descMax]
		}
	}
	return s
}