package statcollect

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	strategyentry "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const TaskName = "stat_collect"

const (
	groupMapKeyAudit = "map_key_audit"
	groupMarketCount  = "market_count"
	groupOutcomeCount = "outcome_count"
	groupMapKeyCount  = "map_key_count"
	groupStatMeta     = "stat_meta"

	jsonTotalKey = "__total__"


)

// Collect 刷新 stat_kv：map_key 审计 + 各平台 JSON 汇总（market/outcome/map_key 按子类）。
type Collect struct {
	db                 *gorm.DB
	statRepo           persistentrepo.StatKVRepo
	marketRepo         persistentrepo.MarketRepo
	outcomeRepo        persistentrepo.MarketOutcomeRepo
	configStrategyRepo persistentrepo.ConfigStrategyRepo
	platformRepo       persistentrepo.PlatformRepo
}

func NewCollect(
	db *gorm.DB,
	statRepo persistentrepo.StatKVRepo,
	marketRepo persistentrepo.MarketRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
	configStrategyRepo persistentrepo.ConfigStrategyRepo,
	platformRepo persistentrepo.PlatformRepo,
) *Collect {
	return &Collect{
		db:                 db,
		statRepo:           statRepo,
		marketRepo:         marketRepo,
		outcomeRepo:        outcomeRepo,
		configStrategyRepo: configStrategyRepo,
		platformRepo:       platformRepo,
	}
}

// Run 执行统计并写入 stat_kv。
func (c *Collect) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "stat_collect-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()
	if c == nil || c.db == nil || c.statRepo == nil {
		notification.SendErrorLog(ctx, "stat_collect-skipped", zap.String("reason", "nil db or repo"))
		return
	}
	start := time.Now()
	badOnly, err := c.loadMapKeyAuditBadOnly(ctx)
	if err != nil {
		notification.SendErrorLog(ctx, "stat_collect-map_key_audit-failed", zap.Error(err))
		return
	}
	marketRows, err := c.loadMarketCounts(ctx)
	if err != nil {
		notification.SendErrorLog(ctx, "stat_collect-market_count-failed", zap.Error(err))
		return
	}
	outcomeRows, err := c.loadOutcomeCounts(ctx)
	if err != nil {
		notification.SendErrorLog(ctx, "stat_collect-outcome_count-failed", zap.Error(err))
		return
	}
	mapKeyRows, err := c.loadMapKeyCounts(ctx)
	if err != nil {
		notification.SendErrorLog(ctx, "stat_collect-map_key_count-failed", zap.Error(err))
		return
	}

	// 快路径 SQL 先落库；应对场次计数放在后面，避免前面全量 outcome 拖死整轮。
	err = c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, g := range []string{groupMarketCount, groupOutcomeCount, groupMapKeyCount} {
			if err := c.statRepo.DeleteByGroup(ctx, tx, g); err != nil {
				return err
			}
		}
		all := make([]*persistententity.StatKV, 0, len(marketRows)+len(outcomeRows)+len(mapKeyRows))
		all = append(all, marketRows...)
		all = append(all, outcomeRows...)
		all = append(all, mapKeyRows...)
		if err := c.statRepo.BatchCreate(ctx, tx, all); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		notification.SendErrorLog(ctx, "stat_collect-persist-counts-failed", zap.Error(err))
		return
	}

	compliant, compliantJSON, verifiedOK, verifiedJSON, invErr := c.loadCompliantGroupInventory(ctx)
	if invErr != nil {
		notification.SendErrorLog(ctx, "stat_collect-group_inventory-failed", zap.Error(invErr))
	}
	watchCounts, watchErr := c.loadWatchGroupCounts(ctx)
	if watchErr != nil {
		notification.SendErrorLog(ctx, "stat_collect-watch_groups-failed", zap.Error(watchErr))
		watchCounts = strategyentry.WatchGroupCounts{}
	}
	preRunnable, preErr := c.countPreEventRunnableGroups(ctx)
	if preErr != nil {
		notification.SendErrorLog(ctx, "stat_collect-market_running_count-failed", zap.Error(preErr))
		preRunnable = preEventRunnableStat{}
	}
	preMins := strategyentry.LoadPreEventScanMinutes(ctx, c.configStrategyRepo)
	endMins := strategyentry.LoadEndEventScanMinutes(ctx, c.configStrategyRepo)
	audit := mapKeyAudit{
		Watchable:           int64(watchCounts.Watchable),
		Matched:             int64(watchCounts.Matched),
		MaxGroups:           watchCounts.MaxGroups,
		PreEventRunnable:    preRunnable.Count,
		PreEventScanMinutes: preMins,
		EndEventScanMinutes: endMins,
		PreEventGroups:      parsePreEventGroupsFromDesc(preRunnable.DescriptionJSON),
		BadLt2:              badOnly.BadLt2,
		BadDup:              badOnly.BadDup,
		CompliantGroups:     compliant,
		CompliantByPlatform: compliantJSON,
		VerifiedCompliant:   verifiedOK,
		VerifiedByPlatform:  verifiedJSON,
	}
	auditRows := mapKeyAuditRows(audit)

	err = c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := c.statRepo.DeleteByGroupItems(ctx, tx, groupMapKeyAudit, mapKeyAuditPurgeItemKeys()); err != nil {
			return err
		}
		if err := c.statRepo.UpsertBatch(ctx, tx, auditRows); err != nil {
			return err
		}
		metaRows := []*persistententity.StatKV{
			kvStat(groupStatMeta, "last_refreshed_at", start.UTC().Format(time.RFC3339), "datetime", "最近一次统计时间", 1),
			kvStat(groupStatMeta, "refresh_ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10), "int", "本轮写入耗时（毫秒）", 2),
		}
		return c.statRepo.UpsertBatch(ctx, tx, metaRows)
	})
	if err != nil {
		notification.SendErrorLog(ctx, "stat_collect-persist-failed", zap.Error(err))
		return
	}
	logger.InfoZ(ctx, "stat_collect-done",
		zap.Int64("compliant_groups", audit.CompliantGroups),
		zap.Int64("verified_compliant_groups", audit.VerifiedCompliant),
		zap.Int64("matched_groups", audit.Matched),
		zap.Int64("watchable_groups", audit.Watchable),
		zap.Int64("market_running_count", audit.PreEventRunnable),
		zap.Int64("bad_lt2", audit.BadLt2),
		zap.Int64("bad_dup", audit.BadDup),
		zap.Bool("watch_groups_ok", watchErr == nil),
		zap.Int("market_count_rows", len(marketRows)),
		zap.Int("outcome_count_rows", len(outcomeRows)),
		zap.Int("map_key_count_rows", len(mapKeyRows)),
		zap.Int64("elapsed_ms", time.Since(start).Milliseconds()))
}

type mapKeyAudit struct {
	Matched               int64
	Watchable             int64
	MaxGroups             int
	PreEventRunnable      int64
	PreEventScanMinutes   int
	EndEventScanMinutes   int
	PreEventGroups        []preEventRunnableGroup
	BadLt2                int64
	BadDup                int64
	CompliantGroups       int64
	CompliantByPlatform   string // JSON：库内合规组按平台数分桶（ACTIVE+map_key，audit OK）
	VerifiedCompliant     int64
	VerifiedByPlatform    string // JSON：map_verified=1 且 platform 为 ACTIVE 的合规组分桶
}

// mapKeyGroupsSnapshot 写入 stat_kv map_key_audit / groups 的 item_value（JSON）。
type mapKeyGroupsSnapshot struct {
	Compliant         mapKeyGroupsLayer `json:"compliant"`
	VerifiedCompliant mapKeyGroupsLayer `json:"verified_compliant"`
	Strategy          mapKeyStrategyLayer `json:"strategy"`
	AuditBad          mapKeyAuditBad      `json:"audit_bad"`
}

type mapKeyGroupsLayer struct {
	Total      int64            `json:"total"`
	ByPlatform map[string]int64 `json:"by_platform"`
}

type mapKeyStrategyLayer struct {
	Matched             int64                    `json:"matched"`
	Watchable           int64                    `json:"watchable"`
	MaxGroups           int                      `json:"max_groups"`
	PreEventRunnable    int64                    `json:"pre_event_runnable"`
	PreEventScanMinutes int                      `json:"pre_event_scan_minutes"`
	EndEventScanMinutes int                      `json:"end_event_scan_minutes"`
	PreEventGroups      []preEventRunnableGroup  `json:"pre_event_groups"`
}

type mapKeyAuditBad struct {
	Lt2PlatformGroups int64 `json:"lt2_platform_groups"`
	DupPlatformGroups int64 `json:"dup_platform_groups"`
}

const mapKeyAuditGroupsItemKey = "groups"

// 历次 map_key_audit 分项键；删除时一并清理，避免 stat_kv 堆积。
var mapKeyAuditLegacyItemKeys = []string{
	"compliant_groups", "compliant_groups_by_platform",
	"verified_compliant_groups", "verified_compliant_by_platform",
	"matched_groups", "watchable_groups", "market_running_count",
	"bad_lt2_platform_groups", "bad_dup_platform_groups",
}

type preEventRunnableStat struct {
	Count           int64
	DescriptionJSON string
}

type preEventRunnableGroup struct {
	MapKey        string `json:"map_key"`
	MarketRowIDs  []uint `json:"market_row_ids"`
}

func (c *Collect) loadMapKeyAuditBadOnly(ctx context.Context) (mapKeyAudit, error) {
	const q = `
SELECT
    COALESCE(SUM(CASE WHEN platform_cnt < 2 THEN 1 ELSE 0 END), 0) AS bad_lt2_platform_groups,
    COALESCE(SUM(CASE WHEN market_rows > platform_cnt THEN 1 ELSE 0 END), 0) AS bad_dup_platform_groups
FROM (
    SELECT
        map_key,
        COUNT(*) AS market_rows,
        COUNT(DISTINCT UPPER(TRIM(platform))) AS platform_cnt
    FROM market
    WHERE map_key IS NOT NULL
      AND TRIM(map_key) <> ''
    GROUP BY map_key
) t`
	var row struct {
		BadLt2 int64 `gorm:"column:bad_lt2_platform_groups"`
		BadDup int64 `gorm:"column:bad_dup_platform_groups"`
	}
	if err := c.db.WithContext(ctx).Raw(q).Scan(&row).Error; err != nil {
		return mapKeyAudit{}, err
	}
	return mapKeyAudit{BadLt2: row.BadLt2, BadDup: row.BadDup}, nil
}

// loadWatchGroupCounts 与 strategy_entry.loadGroups 同源（含开赛窗、outcome、ACTIVE 平台）。
func (c *Collect) loadWatchGroupCounts(ctx context.Context) (strategyentry.WatchGroupCounts, error) {
	if c.marketRepo == nil || c.outcomeRepo == nil {
		return strategyentry.WatchGroupCounts{}, nil
	}
	return strategyentry.CountWatchGroups(ctx, strategyentry.WatchGroupDeps{
		MarketRepo:         c.marketRepo,
		OutcomeRepo:        c.outcomeRepo,
		ConfigStrategyRepo: c.configStrategyRepo,
		PlatformRepo:       c.platformRepo,
	})
}

type groupInventoryBucketRow struct {
	PlatformCnt int64 `gorm:"column:platform_cnt"`
	GroupCnt    int64 `gorm:"column:group_cnt"`
}

// loadCompliantGroupInventory 库内 map_key 合规组（≥2 平台、同平台不重复），分「全 ACTIVE」与「已核对+ACTIVE 平台」两层。
func (c *Collect) loadCompliantGroupInventory(ctx context.Context) (okAll int64, jsonAll string, okVerified int64, jsonVerified string, err error) {
	if c.db == nil {
		return 0, "{}", 0, "{}", nil
	}
	okAll, jsonAll, err = c.compliantGroupsWithFilter(ctx, false)
	if err != nil {
		return 0, "", 0, "", err
	}
	okVerified, jsonVerified, err = c.compliantGroupsWithFilter(ctx, true)
	if err != nil {
		return 0, "", 0, "", err
	}
	return okAll, jsonAll, okVerified, jsonVerified, nil
}

func (c *Collect) compliantGroupsWithFilter(ctx context.Context, verifiedOnly bool) (total int64, jsonBody string, err error) {
	verifiedClause := ""
	if verifiedOnly {
		verifiedClause = `
      AND m.map_verified = 1
      AND (
        NOT EXISTS (SELECT 1 FROM platform WHERE status = 'ACTIVE' LIMIT 1)
        OR UPPER(TRIM(m.platform)) IN (
          SELECT UPPER(TRIM(slug)) FROM platform WHERE status = 'ACTIVE'
        )
      )`
	}
	q := `
SELECT platform_cnt, COUNT(*) AS group_cnt
FROM (
    SELECT
        TRIM(m.map_key) AS map_key,
        COUNT(*) AS market_rows,
        COUNT(DISTINCT UPPER(TRIM(m.platform))) AS platform_cnt
    FROM market m
    WHERE m.status = 'ACTIVE'
      AND m.map_key IS NOT NULL
      AND TRIM(m.map_key) <> ''` + verifiedClause + `
    GROUP BY TRIM(m.map_key)
    HAVING platform_cnt >= 2
       AND market_rows = platform_cnt
) t
GROUP BY platform_cnt
ORDER BY platform_cnt`
	var rows []groupInventoryBucketRow
	if err := c.db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
		return 0, "", err
	}
	by := make(map[string]int64, len(rows))
	for _, r := range rows {
		total += r.GroupCnt
		by[strconv.FormatInt(r.PlatformCnt, 10)] = r.GroupCnt
	}
	by["total"] = total
	b, err := json.Marshal(by)
	if err != nil {
		return total, "{}", nil
	}
	return total, string(b), nil
}

// countPreEventRunnableGroups 与 strategy_entry.loadGroups 的 pre_event_scan_minutes 窗口一致（默认 30 分钟）。
func (c *Collect) countPreEventRunnableGroups(ctx context.Context) (preEventRunnableStat, error) {
	empty := preEventRunnableStat{DescriptionJSON: `{"pre_event_scan_minutes":30,"end_event_scan_minutes":180,"groups":[]}`}
	if c.marketRepo == nil || c.outcomeRepo == nil {
		return empty, nil
	}
	fixtures, err := marketmap.LoadVerifiedFixturesNew(ctx, c.marketRepo, c.configStrategyRepo, c.platformRepo)
	if err != nil {
		return empty, err
	}
	preMins := strategyentry.LoadPreEventScanMinutes(ctx, c.configStrategyRepo)
	endMins := strategyentry.LoadEndEventScanMinutes(ctx, c.configStrategyRepo)
	empty.DescriptionJSON = preEventRunnableDescJSON(preMins, endMins, nil)
	if len(fixtures) == 0 {
		return empty, nil
	}
	runnableKeys := statRunnableMapKeys(fixtures, time.Now().UTC(), preMins, endMins)
	if len(runnableKeys) == 0 {
		return preEventRunnableStat{Count: 0, DescriptionJSON: preEventRunnableDescJSON(preMins, endMins, nil)}, nil
	}
	marketIDs := make([]uint, 0, len(fixtures))
	for id := range fixtures {
		marketIDs = append(marketIDs, id)
	}
	_, raw, err := c.outcomeRepo.List(ctx, nil, map[string]interface{}{
		"market_row_id IN ?": marketIDs,
	}, "id asc", "")
	if err != nil {
		return empty, err
	}
	outcomes := marketmap.FilterOutcomesUnderVerifiedFixtures(raw, fixtures)
	withOutcomes := map[string]struct{}{}
	for i := range outcomes {
		oc := outcomes[i]
		fx, ok := fixtures[oc.MarketRowID]
		if !ok || fx.MapKey == nil {
			continue
		}
		k := strings.TrimSpace(*fx.MapKey)
		if _, ok := runnableKeys[k]; !ok {
			continue
		}
		withOutcomes[k] = struct{}{}
	}
	groups := buildPreEventRunnableGroups(fixtures, withOutcomes)
	desc := preEventRunnableDescJSON(preMins, endMins, groups)
	return preEventRunnableStat{Count: int64(len(groups)), DescriptionJSON: desc}, nil
}

func buildPreEventRunnableGroups(fixtures map[uint]persistententity.Market, mapKeys map[string]struct{}) []preEventRunnableGroup {
	if len(mapKeys) == 0 {
		return nil
	}
	idsByKey := make(map[string]map[uint]struct{})
	for id, fx := range fixtures {
		if fx.MapKey == nil {
			continue
		}
		k := strings.TrimSpace(*fx.MapKey)
		if _, ok := mapKeys[k]; !ok {
			continue
		}
		if idsByKey[k] == nil {
			idsByKey[k] = make(map[uint]struct{})
		}
		idsByKey[k][id] = struct{}{}
	}
	keys := make([]string, 0, len(mapKeys))
	for k := range mapKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]preEventRunnableGroup, 0, len(keys))
	for _, k := range keys {
		idSet := idsByKey[k]
		ids := make([]uint, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		out = append(out, preEventRunnableGroup{MapKey: k, MarketRowIDs: ids})
	}
	return out
}

func preEventRunnableDescJSON(preEventScanMinutes, endEventScanMinutes int, groups []preEventRunnableGroup) string {
	if groups == nil {
		groups = []preEventRunnableGroup{}
	}
	payload := struct {
		PreEventScanMinutes int                     `json:"pre_event_scan_minutes"`
		EndEventScanMinutes int                     `json:"end_event_scan_minutes"`
		Groups              []preEventRunnableGroup `json:"groups"`
	}{
		PreEventScanMinutes: preEventScanMinutes,
		EndEventScanMinutes: endEventScanMinutes,
		Groups:              groups,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return `{"pre_event_scan_minutes":` + strconv.Itoa(preEventScanMinutes) + `,"groups":[]}`
	}
	s := string(b)
	const descMax = 500
	if len(s) > descMax {
		// description 列 varchar(500)：只保留 map_key + 组数，避免截断非法 JSON
		compact := struct {
			PreEventScanMinutes int      `json:"pre_event_scan_minutes"`
			EndEventScanMinutes int      `json:"end_event_scan_minutes"`
			Count               int      `json:"count"`
			MapKeys             []string `json:"map_keys"`
			Truncated           bool     `json:"truncated"`
		}{
			PreEventScanMinutes: preEventScanMinutes,
			EndEventScanMinutes: endEventScanMinutes,
			Count:               len(groups),
			Truncated:           true,
		}
		for _, g := range groups {
			compact.MapKeys = append(compact.MapKeys, g.MapKey)
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

func statRunnableMapKeys(fixtures map[uint]persistententity.Market, now time.Time, preEventScanMinutes, endEventScanMinutes int) map[string]struct{} {
	rowsByKey := make(map[string][]persistententity.Market)
	for _, fx := range fixtures {
		if fx.MapKey == nil || fx.MapVerified != 1 {
			continue
		}
		k := strings.TrimSpace(*fx.MapKey)
		if k == "" {
			continue
		}
		rowsByKey[k] = append(rowsByKey[k], fx)
	}
	out := make(map[string]struct{})
	for k, rows := range rowsByKey {
		if dup, _, _ := marketmap.MapKeyDuplicatePlatform(rows); dup {
			continue
		}
		if !marketmap.MapKeyHasTwoPlatforms(rows) {
			continue
		}
		if !strategyentry.MapKeyWithinPreEventScan(rows, now, preEventScanMinutes, endEventScanMinutes) {
			continue
		}
		out[k] = struct{}{}
	}
	return out
}

type platformSubCountRow struct {
	Platform    string `gorm:"column:platform"`
	SubCategory string `gorm:"column:sub_category"`
	Cnt         int64  `gorm:"column:cnt"`
}

func (c *Collect) loadMarketCounts(ctx context.Context) ([]*persistententity.StatKV, error) {
	const q = `
SELECT
    UPPER(TRIM(platform)) AS platform,
    COALESCE(NULLIF(TRIM(sub_category), ''), '_unknown') AS sub_category,
    COUNT(*) AS cnt
FROM market
GROUP BY UPPER(TRIM(platform)), COALESCE(NULLIF(TRIM(sub_category), ''), '_unknown')
ORDER BY platform, sub_category`
	return c.loadPlatformJSONCounts(ctx, q, groupMarketCount, false,
		"各子类 market 场次；item_value 为 JSON，含 __total__")
}

func (c *Collect) loadOutcomeCounts(ctx context.Context) ([]*persistententity.StatKV, error) {
	const q = `
SELECT
    UPPER(TRIM(platform)) AS platform,
    COALESCE(NULLIF(TRIM(sub_category), ''), '_unknown') AS sub_category,
    COUNT(*) AS cnt
FROM market_outcome
GROUP BY UPPER(TRIM(platform)), COALESCE(NULLIF(TRIM(sub_category), ''), '_unknown')
ORDER BY platform, sub_category`
	return c.loadPlatformJSONCounts(ctx, q, groupOutcomeCount, false,
		"各子类 market_outcome 盘口；item_value 为 JSON，含 __total__")
}

func (c *Collect) loadMapKeyCounts(ctx context.Context) ([]*persistententity.StatKV, error) {
	const q = `
SELECT
    UPPER(TRIM(platform)) AS platform,
    COALESCE(NULLIF(TRIM(sub_category), ''), '_unknown') AS sub_category,
    COUNT(DISTINCT TRIM(map_key)) AS cnt
FROM market
WHERE map_key IS NOT NULL
  AND TRIM(map_key) <> ''
GROUP BY UPPER(TRIM(platform)), COALESCE(NULLIF(TRIM(sub_category), ''), '_unknown')
ORDER BY platform, sub_category`
	return c.loadPlatformJSONCounts(ctx, q, groupMapKeyCount, true,
		"各子类非空 map_key 去重数；item_value 为 JSON，含 __total__")
}

func (c *Collect) loadPlatformJSONCounts(ctx context.Context, q, group string, distinctTotal bool, description string) ([]*persistententity.StatKV, error) {
	var rows []platformSubCountRow
	if err := c.db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byPlatform := make(map[string]map[string]int64)
	for _, r := range rows {
		plat := strings.TrimSpace(r.Platform)
		sub := strings.TrimSpace(r.SubCategory)
		if plat == "" {
			continue
		}
		if byPlatform[plat] == nil {
			byPlatform[plat] = make(map[string]int64)
		}
		byPlatform[plat][sub] = r.Cnt
	}
	plats := make([]string, 0, len(byPlatform))
	for p := range byPlatform {
		plats = append(plats, p)
	}
	sort.Strings(plats)

	out := make([]*persistententity.StatKV, 0, len(plats))
	for i, plat := range plats {
		subs := byPlatform[plat]
		var total int64
		if distinctTotal {
			var err error
			total, err = c.platformDistinctMapKeyTotal(ctx, plat)
			if err != nil {
				return nil, err
			}
			subs[jsonTotalKey] = total
		} else {
			for _, n := range subs {
				total += n
			}
			subs[jsonTotalKey] = total
		}
		val, err := json.Marshal(subs)
		if err != nil {
			return nil, err
		}
		desc := fmt.Sprintf("平台 %s：%s", plat, description)
		out = append(out, kvStat(group, plat, string(val), "json", desc, i+1))
	}
	return out, nil
}

func (c *Collect) platformDistinctMapKeyTotal(ctx context.Context, platform string) (int64, error) {
	const q = `
SELECT COUNT(DISTINCT TRIM(map_key)) AS cnt
FROM market
WHERE map_key IS NOT NULL
  AND TRIM(map_key) <> ''
  AND UPPER(TRIM(platform)) = ?`
	var row struct {
		Cnt int64 `gorm:"column:cnt"`
	}
	err := c.db.WithContext(ctx).Raw(q, strings.ToUpper(strings.TrimSpace(platform))).Scan(&row).Error
	return row.Cnt, err
}

func mapKeyAuditRows(a mapKeyAudit) []*persistententity.StatKV {
	body := buildMapKeyGroupsJSON(a)
	return []*persistententity.StatKV{
		kvStat(groupMapKeyAudit, mapKeyAuditGroupsItemKey, body, "json",
			"跨平台 map_key 组分层统计（compliant / verified / strategy / audit_bad）", 1),
	}
}

func mapKeyAuditPurgeItemKeys() []string {
	keys := append([]string{mapKeyAuditGroupsItemKey}, mapKeyAuditLegacyItemKeys...)
	return keys
}

func buildMapKeyGroupsJSON(a mapKeyAudit) string {
	groups := a.PreEventGroups
	if groups == nil {
		groups = []preEventRunnableGroup{}
	}
	snap := mapKeyGroupsSnapshot{
		Compliant: mapKeyGroupsLayer{
			Total:      a.CompliantGroups,
			ByPlatform: platformCountsForSnapshot(a.CompliantByPlatform),
		},
		VerifiedCompliant: mapKeyGroupsLayer{
			Total:      a.VerifiedCompliant,
			ByPlatform: platformCountsForSnapshot(a.VerifiedByPlatform),
		},
		Strategy: mapKeyStrategyLayer{
			Matched:             a.Matched,
			Watchable:           a.Watchable,
			MaxGroups:           a.MaxGroups,
			PreEventRunnable:    a.PreEventRunnable,
			PreEventScanMinutes: a.PreEventScanMinutes,
			EndEventScanMinutes: a.EndEventScanMinutes,
			PreEventGroups:      groups,
		},
		AuditBad: mapKeyAuditBad{
			Lt2PlatformGroups: a.BadLt2,
			DupPlatformGroups: a.BadDup,
		},
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "{\n  \"error\": \"marshal_map_key_groups_snapshot\"\n}\n"
	}
	return string(b)
}

// platformCountsForSnapshot 分桶里去掉冗余的 total，仅保留 2/3/4… 平台组计数。
func platformCountsForSnapshot(raw string) map[string]int64 {
	m := parsePlatformCountJSON(raw)
	if len(m) == 0 {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(m))
	for k, v := range m {
		if k == "total" {
			continue
		}
		out[k] = v
	}
	return out
}

func parsePlatformCountJSON(raw string) map[string]int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]int64{}
	}
	out := map[string]int64{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]int64{}
	}
	return out
}

func parsePreEventGroupsFromDesc(descJSON string) []preEventRunnableGroup {
	descJSON = strings.TrimSpace(descJSON)
	if descJSON == "" {
		return nil
	}
	var payload struct {
		Groups []preEventRunnableGroup `json:"groups"`
	}
	if err := json.Unmarshal([]byte(descJSON), &payload); err != nil {
		return nil
	}
	return payload.Groups
}

func kvStat(group, key, val, valType, desc string, sortNo int) *persistententity.StatKV {
	return &persistententity.StatKV{
		GroupName:   group,
		ItemKey:     key,
		ItemValue:   val,
		ValueType:   valType,
		Description: desc,
		SortNo:      sortNo,
		IsEnabled:   1,
	}
}