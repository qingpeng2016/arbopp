package synthetic

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const TaskName = "synthetic_history_sync"

const dataPlatformCodeSynthetic = "synthetic"

// Sync 按 data_task(synthetic_l2) 伪造秒级对齐订单簿并批量写入 data_history_order_book。
type Sync struct {
	dataTaskRepo     persistentrepo.DataTaskRepo
	platformRepo     persistentrepo.PlatformRepo
	marketRepo       persistentrepo.MarketRepo
	outcomeRepo      persistentrepo.MarketOutcomeRepo
	dataPlatformRepo persistentrepo.DataPlatformRepo
	historyRepo      persistentrepo.HistoryDataOrderBookRepo
}

func NewSync(
	dataTaskRepo persistentrepo.DataTaskRepo,
	platformRepo persistentrepo.PlatformRepo,
	marketRepo persistentrepo.MarketRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
	dataPlatformRepo persistentrepo.DataPlatformRepo,
	historyRepo persistentrepo.HistoryDataOrderBookRepo,
) *Sync {
	return &Sync{
		dataTaskRepo:     dataTaskRepo,
		platformRepo:     platformRepo,
		marketRepo:       marketRepo,
		outcomeRepo:      outcomeRepo,
		dataPlatformRepo: dataPlatformRepo,
		historyRepo:      historyRepo,
	}
}

func (j *Sync) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "synthetic_history_sync-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()
	if j.dataTaskRepo == nil || j.historyRepo == nil || j.marketRepo == nil || j.outcomeRepo == nil || j.dataPlatformRepo == nil {
		notification.SendErrorLog(ctx, "synthetic_history_sync-skipped", zap.String("reason", "repo nil"))
		return
	}

	dp, err := j.dataPlatformRepo.FindOne(ctx, nil, map[string]interface{}{
		"code = ?":   dataPlatformCodeSynthetic,
		"status = ?": "ACTIVE",
	}, "")
	if err != nil {
		notification.SendErrorLog(ctx, "synthetic_history_sync-no-data-platform", zap.Error(err))
		return
	}
	extraRaw := ""
	if dp.ExtraConfig != nil {
		extraRaw = *dp.ExtraConfig
	}
	cfg := ParseExtraConfig(extraRaw)

	_, tasks, err := j.dataTaskRepo.List(ctx, nil, map[string]interface{}{
		"ingest_source = ?": persistententity.DataTaskIngestSynthetic,
		"status = ?":        persistententity.DataTaskStatusPending,
	}, "id asc", "")
	if err != nil {
		notification.SendErrorLog(ctx, "synthetic_history_sync-ListTasksFailed", zap.Error(err))
		return
	}
	if len(tasks) == 0 {
		logger.InfoZ(ctx, "synthetic_history_sync-no-pending-tasks")
		return
	}

	for i := range tasks {
		j.runTask(ctx, dp, cfg, &tasks[i])
	}
}

func (j *Sync) runTask(ctx context.Context, dp persistententity.DataPlatform, cfg ExtraConfig, task *persistententity.DataTask) {
	if task == nil {
		return
	}
	mapKey := strings.TrimSpace(task.MapKey)
	logger.InfoZ(ctx, "synthetic_history_sync-task-start",
		zap.Uint("task_id", task.ID),
		zap.String("map_key", mapKey))
	_ = j.dataTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":     persistententity.DataTaskStatusRunning,
		"error_text": "",
	})

	fixtures, err := marketmap.LoadMarketsByMapKey(ctx, j.marketRepo, j.platformRepo, mapKey)
	if err != nil {
		j.failTask(ctx, task, err)
		return
	}
	if len(fixtures) == 0 {
		j.failTask(ctx, task, fmt.Errorf("no market for map_key %q", mapKey))
		return
	}
	outcomes, err := marketmap.OutcomesForFixtureMarkets(ctx, j.outcomeRepo, fixtures)
	if err != nil {
		j.failTask(ctx, task, err)
		return
	}
	outcomes = filterOutcomes(cfg.OutcomeTypes, outcomes)
	if len(outcomes) == 0 {
		j.failTask(ctx, task, fmt.Errorf("no outcomes for map_key %q", mapKey))
		return
	}

	_ = j.historyRepo.DeleteWhere(ctx, nil, map[string]interface{}{
		"map_key = ?":          mapKey,
		"data_platform_id = ?": dp.ID,
		"source = ?":           persistententity.HistorySourceSynthetic,
	})

	baseTs := historydata.BaseSnapshotMsForMapKey(mapKey)
	var inserted int64
	for batchStart := 0; batchStart < cfg.TotalSeconds; batchStart += cfg.BatchSeconds {
		end := batchStart + cfg.BatchSeconds
		if end > cfg.TotalSeconds {
			end = cfg.TotalSeconds
		}
		rows := j.buildSecondBatch(dp, cfg, mapKey, outcomes, baseTs, batchStart, end)
		if len(rows) == 0 {
			continue
		}
		if err := j.historyRepo.BatchCreate(ctx, nil, rows); err != nil {
			j.failTask(ctx, task, err)
			return
		}
		inserted += int64(len(rows))
		logger.InfoZ(ctx, "synthetic_history_sync-batch-inserted",
			zap.String("map_key", mapKey),
			zap.Int("from_sec", batchStart),
			zap.Int("to_sec", end-1),
			zap.Int("rows", len(rows)))
	}

	_ = j.dataTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":     persistententity.DataTaskStatusCompleted,
		"error_text": fmt.Sprintf("inserted %d rows, %d seconds", inserted, cfg.TotalSeconds),
	})
	task.Status = persistententity.DataTaskStatusCompleted
	historydata.NotifySyntheticTaskFinished(ctx, task, inserted, cfg.TotalSeconds, len(outcomes))
	logger.InfoZ(ctx, "synthetic_history_sync-task-done",
		zap.Uint("task_id", task.ID),
		zap.Int64("rows", inserted))
}

func (j *Sync) buildSecondBatch(
	dp persistententity.DataPlatform,
	cfg ExtraConfig,
	mapKey string,
	outcomes []persistententity.MarketOutcome,
	baseTs int64,
	fromSec, toSec int,
) []*persistententity.HistoryDataOrderBook {
	capN := (toSec - fromSec) * len(outcomes)
	if capN <= 0 {
		return nil
	}
	rows := make([]*persistententity.HistoryDataOrderBook, 0, capN)
	plan := buildArbPlan(outcomes)
	for sec := fromSec; sec < toSec; sec++ {
		tsMs := baseTs + int64(sec)*1000
		utcDate := persistententity.CalendarDateFrom(time.UnixMilli(tsMs).UTC()).String()
		for oi, oc := range outcomes {
			bids, asks := bookForOutcome(sec, oc, oi, plan)
			symbol := strings.TrimSpace(oc.PlatformMarketOutcomeID)
			if sp := historydata.PolymarketVendorSlug(oc); strings.EqualFold(oc.Platform, string(cryptoPredictionEnum.PlatformPolymarket)) {
				symbol = sp
			}
			rows = append(rows, &persistententity.HistoryDataOrderBook{
				DataPlatformID:  dp.ID,
				MapKey:          mapKey,
				MarketRowID:     oc.MarketRowID,
				MarketOutcomeID: oc.ID,
				VenuePlatform:   strings.ToUpper(strings.TrimSpace(oc.Platform)),
				VendorExchange:  "synthetic",
				VendorSymbol:    symbol,
				UTCDate:         utcDate,
				DataType:        "l2",
				Source:          persistententity.HistorySourceSynthetic,
				RecordKind:      persistententity.HistoryRecordKindSnapshot,
				SnapshotAtMs:    tsMs,
				OrderBookBids:   bids,
				OrderBookAsks:   asks,
				IngestStatus:    persistententity.HistoryIngestSynthetic,
				ErrorText:       fmt.Sprintf("sec=%d", sec),
			})
		}
	}
	return rows
}

func filterOutcomes(types []string, list []persistententity.MarketOutcome) []persistententity.MarketOutcome {
	typeSet := map[string]struct{}{}
	for _, t := range types {
		typeSet[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	allowedPlat := map[string]struct{}{
		string(cryptoPredictionEnum.PlatformPolymarket): {},
		string(cryptoPredictionEnum.PlatformPredict):    {},
		string(cryptoPredictionEnum.PlatformKalshi):   {},
	}
	out := make([]persistententity.MarketOutcome, 0, len(list))
	for i := range list {
		oc := list[i]
		p := strings.ToUpper(strings.TrimSpace(oc.Platform))
		if _, ok := allowedPlat[p]; !ok {
			continue
		}
		if len(typeSet) > 0 {
			ot := strings.ToLower(strings.TrimSpace(oc.OutcomeType))
			if ot != "" {
				if _, ok := typeSet[ot]; !ok {
					continue
				}
			}
		}
		out = append(out, oc)
	}
	return out
}

func (j *Sync) failTask(ctx context.Context, task *persistententity.DataTask, err error) {
	if task == nil {
		return
	}
	msg := trimErr(err, 500)
	_ = j.dataTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":     persistententity.DataTaskStatusFailed,
		"error_text": msg,
	})
	task.Status = persistententity.DataTaskStatusFailed
	notification.SendErrorLog(ctx, "synthetic_history_sync-task-failed",
		zap.Uint("task_id", task.ID),
		zap.Error(err))
	historydata.NotifySyntheticTaskFinished(ctx, task, 0, 0, 0)
}

func trimErr(err error, max int) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) > max {
		return s[:max]
	}
	return s
}