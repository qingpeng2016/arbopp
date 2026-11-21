package stat

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const (
	groupMapKeyAudit          = "map_key_audit"
	itemKeyMarketRunningCount = "market_running_count_strategy"
)

func GroupMapKeyAudit() string          { return groupMapKeyAudit }
func ItemKeyMarketRunningCountStrategy() string { return itemKeyMarketRunningCount }

// Group 在跑 map_key 及关联 market.id（写入 stat_kv description）。
type Group struct {
	MapKey  string            `json:"map_key"`
	Markets map[string]uint   `json:"markets,omitempty"` // 平台 slug 大写 → market.id
}

// MarketsByPlatform 从 map_key 组内 market 行得到各平台对应的 market_row_id。
func MarketsByPlatform(markets map[uint]persistententity.Market) map[string]uint {
	if len(markets) == 0 {
		return nil
	}
	out := make(map[string]uint, len(markets))
	for id, m := range markets {
		p := strings.ToUpper(strings.TrimSpace(m.Platform))
		if p == "" {
			continue
		}
		out[p] = id
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MarketRunningOpts FlushMarketRunningCountStrategy 入参。
type MarketRunningOpts struct {
	StatRepo            persistentrepo.StatKVRepo
	PreEventScanMinutes int
	EndEventScanMinutes int
	Groups              []Group
}

// FlushMarketRunningCountStrategy 写入当前 strategy_entry 在跑场次（与 stat_collect market_running_count 同 JSON 结构）。
func FlushMarketRunningCountStrategy(ctx context.Context, o MarketRunningOpts) {
	if o.StatRepo == nil {
		notification.SendErrorLog(ctx, "strategy_entry-market_running_count_strategy-skipped",
			zap.String("reason", "statRepo nil"))
		return
	}
	groups := o.Groups
	if groups == nil {
		groups = []Group{}
	}
	desc := marketRunningDescJSON(o.PreEventScanMinutes, o.EndEventScanMinutes, groups)
	row := &persistententity.StatKV{
		GroupName:   groupMapKeyAudit,
		ItemKey:     itemKeyMarketRunningCount,
		ItemValue:   strconv.Itoa(len(groups)),
		ValueType:   "int",
		Description: desc,
		SortNo:      6,
		IsEnabled:   1,
	}
	if err := o.StatRepo.UpsertBatch(ctx, nil, []*persistententity.StatKV{row}); err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-market_running_count_strategy-stat_kv", zap.Error(err))
		return
	}
	logger.InfoZ(ctx, "strategy_entry-market_running_count_strategy-stat_kv",
		zap.Int("count", len(groups)),
		zap.String("item_key", itemKeyMarketRunningCount))
}

// SortGroups 按 map_key 排序（调用方也可自行排序）。
func SortGroups(groups []Group) {
	sort.Slice(groups, func(i, j int) bool {
		return strings.TrimSpace(groups[i].MapKey) < strings.TrimSpace(groups[j].MapKey)
	})
}

func marketRunningDescJSON(preEventScanMinutes, endEventScanMinutes int, groups []Group) string {
	if groups == nil {
		groups = []Group{}
	}
	payload := struct {
		PreEventScanMinutes int     `json:"pre_event_scan_minutes"`
		EndEventScanMinutes int     `json:"end_event_scan_minutes"`
		Groups              []Group `json:"groups"`
	}{
		PreEventScanMinutes: preEventScanMinutes,
		EndEventScanMinutes: endEventScanMinutes,
		Groups:              groups,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return `{"pre_event_scan_minutes":` + strconv.Itoa(preEventScanMinutes) + `,"groups":[]}`
	}
	return string(b)
}