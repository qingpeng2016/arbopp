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
	groupMapKeyAudit  = "map_key_audit"
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

	watchCounts, watchErr := c.loadWatchGroupCounts(ctx)
	if watchErr != nil {
		notification.SendErrorLog(ctx, "stat_collect-watch_groups-failed", zap.Error(watchErr))
		watchCounts = strategyentry.WatchGroupCounts{}
	}
	audit := mapKeyAudit{
		Watchable: int64(watchCounts.Watchable),
		Matched:   int64(watchCounts.Matched),
		BadLt2:    badOnly.BadLt2,
		BadDup:    badOnly.BadDup,
	}
	auditRows := mapKeyAuditRows(audit)

	err = c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := c.statRepo.DeleteByGroup(ctx, tx, groupMapKeyAudit); err != nil {
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
		zap.Int64("matched_groups", audit.Matched),
		zap.Int64("watchable_groups", audit.Watchable),
		zap.Int64("bad_lt2", audit.BadLt2),
		zap.Int64("bad_dup", audit.BadDup),
		zap.Bool("watch_groups_ok", watchErr == nil),
		zap.Int("market_count_rows", len(marketRows)),
		zap.Int("outcome_count_rows", len(outcomeRows)),
		zap.Int("map_key_count_rows", len(mapKeyRows)),
		zap.Int64("elapsed_ms", time.Since(start).Milliseconds()))
}

type mapKeyAudit struct {
	Matched   int64
	Watchable int64
	BadLt2    int64
	BadDup    int64
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

// loadWatchGroupCounts 与 loadGroups 同源；纯 SQL，不加载 market / market_outcome 行。
func (c *Collect) loadWatchGroupCounts(ctx context.Context) (strategyentry.WatchGroupCounts, error) {
	if c.db == nil {
		return strategyentry.WatchGroupCounts{}, nil
	}
	matched, err := c.countMatchedGroupsSQL(ctx)
	if err != nil {
		return strategyentry.WatchGroupCounts{}, err
	}
	limit := strategyentry.LoadMaxGroups(ctx, c.configStrategyRepo)
	watchable := matched
	if watchable > limit {
		watchable = limit
	}
	return strategyentry.WatchGroupCounts{
		Matched:   matched,
		Watchable: watchable,
		MaxGroups: limit,
	}, nil
}

func (c *Collect) countMatchedGroupsSQL(ctx context.Context) (int, error) {
	const q = `
SELECT COUNT(*) AS cnt FROM (
	SELECT TRIM(m.map_key) AS map_key
	FROM market m
	INNER JOIN market_outcome o ON o.market_row_id = m.id
	INNER JOIN (
		SELECT TRIM(map_key) AS map_key
		FROM market
		WHERE status = 'ACTIVE'
		  AND map_verified = 1
		  AND map_key IS NOT NULL
		  AND TRIM(map_key) <> ''
		  AND (
		    NOT EXISTS (SELECT 1 FROM platform WHERE status = 'ACTIVE' LIMIT 1)
		    OR UPPER(TRIM(platform)) IN (
		      SELECT UPPER(TRIM(slug)) FROM platform WHERE status = 'ACTIVE'
		    )
		  )
		GROUP BY TRIM(map_key)
		HAVING COUNT(DISTINCT UPPER(TRIM(platform))) >= 2
		   AND COUNT(*) = COUNT(DISTINCT UPPER(TRIM(platform)))
	) rk ON rk.map_key = TRIM(m.map_key)
	WHERE m.status = 'ACTIVE'
	  AND m.map_verified = 1
	  AND (
	    NOT EXISTS (SELECT 1 FROM platform WHERE status = 'ACTIVE' LIMIT 1)
	    OR UPPER(TRIM(m.platform)) IN (
	      SELECT UPPER(TRIM(slug)) FROM platform WHERE status = 'ACTIVE'
	    )
	  )
	GROUP BY TRIM(m.map_key)
) t`
	var row struct {
		Cnt int64 `gorm:"column:cnt"`
	}
	if err := c.db.WithContext(ctx).Raw(q).Scan(&row).Error; err != nil {
		return 0, err
	}
	return int(row.Cnt), nil
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
	return []*persistententity.StatKV{
		kvStat(groupMapKeyAudit, "watchable_groups", strconv.FormatInt(a.Watchable, 10), "int",
			"strategy_entry 应对监控组数（与 loadGroups/max_groups 一致）", 1),
		kvStat(groupMapKeyAudit, "matched_groups", strconv.FormatInt(a.Matched, 10), "int",
			"strategy_entry 库内匹配组数（未截 max_groups）", 2),
		kvStat(groupMapKeyAudit, "bad_lt2_platform_groups", strconv.FormatInt(a.BadLt2, 10), "int",
			"map_key 平台数<2 的组数（audit D）", 3),
		kvStat(groupMapKeyAudit, "bad_dup_platform_groups", strconv.FormatInt(a.BadDup, 10), "int",
			"map_key 同平台重复组数（audit D）", 4),
	}
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