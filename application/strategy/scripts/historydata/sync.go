package historydata

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata/pmdata"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const TaskName = "history_data_sync"

const dataPlatformCodePMData = "pmdata"

// Sync 从 data_platform 配置拉 PMData L2，写入 data_history_order_book。
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
			notification.SendErrorLog(ctx, "history_data_sync-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()
	if j.dataPlatformRepo == nil || j.historyRepo == nil || j.marketRepo == nil || j.outcomeRepo == nil || j.dataTaskRepo == nil {
		notification.SendErrorLog(ctx, "history_data_sync-skipped", zap.String("reason", "repo nil"))
		return
	}

	dp, err := j.dataPlatformRepo.FindOne(ctx, nil, map[string]interface{}{
		"code = ?":   dataPlatformCodePMData,
		"status = ?": "ACTIVE",
	}, "")
	if err != nil {
		notification.SendErrorLog(ctx, "history_data_sync-no-data-platform", zap.Error(err))
		return
	}
	if strings.TrimSpace(dp.APIKey) == "" {
		notification.SendErrorLog(ctx, "history_data_sync-skipped", zap.String("reason", "data_platform.api_key empty"))
		return
	}
	extraRaw := ""
	if dp.ExtraConfig != nil {
		extraRaw = *dp.ExtraConfig
	}
	cfg := ParsePMDataExtraConfig(extraRaw)
	client := pmdata.NewClient(dp.APIBaseURL, dp.APIKey)

	j.reclaimStaleRunningTasks(ctx, 15*time.Minute)

	_, tasks, err := j.dataTaskRepo.List(ctx, nil, map[string]interface{}{
		"ingest_source = ?": persistententity.DataTaskIngestPmdata,
		"status = ?":        persistententity.DataTaskStatusPending,
	}, "id asc", "")
	if err != nil {
		notification.SendErrorLog(ctx, "history_data_sync-ListTasksFailed", zap.Error(err))
		return
	}
	if len(tasks) == 0 {
		logger.InfoZ(ctx, "history_data_sync-no-pending-tasks")
		return
	}
	logger.InfoZ(ctx, "history_data_sync-batch-start", zap.Int("pending_tasks", len(tasks)))

	var totalFiles, totalSnaps, totalFailed int
	for i := range tasks {
		t := tasks[i]
		files, snaps, failed := j.runDataTask(ctx, client, dp, cfg, &t)
		totalFiles += files
		totalSnaps += snaps
		totalFailed += failed
	}

	logger.InfoZ(ctx, "history_data_sync-Completed",
		zap.Int("tasks", len(tasks)),
		zap.Int("files_ok", totalFiles),
		zap.Int("snapshots_upserted", totalSnaps),
		zap.Int("failed", totalFailed))
}

func (j *Sync) runDataTask(
	ctx context.Context,
	client *pmdata.Client,
	dp persistententity.DataPlatform,
	cfg PMDataExtraConfig,
	task *persistententity.DataTask,
) (downloaded, snapshots, failed int) {
	if task == nil {
		return 0, 0, 0
	}
	taskParamsRaw := ""
	if task.Params != nil {
		taskParamsRaw = strings.TrimSpace(*task.Params)
	}
	tp, hasTaskParams := ParseDataTaskParams(taskParamsRaw)
	cfg = ApplyPMDataExtra(cfg, tp, hasTaskParams)
	fixtureSlug, err := PMDataFixtureSlug(tp, hasTaskParams)
	if err != nil {
		j.failTask(ctx, task.ID, err)
		return 0, 0, 1
	}
	logger.InfoZ(ctx, "history_data_sync-task-start",
		zap.Uint("task_id", task.ID),
		zap.String("fixture_slug", fixtureSlug),
		zap.Bool("task_params", hasTaskParams))
	_ = j.dataTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":     persistententity.DataTaskStatusRunning,
		"error_text": "",
	})

	fixtures, mapKey, err := LoadPMDataFixtures(ctx, j.marketRepo, j.platformRepo, fixtureSlug)
	if err != nil {
		j.failTask(ctx, task.ID, err)
		return 0, 0, 1
	}
	if len(fixtures) == 0 {
		j.failTask(ctx, task.ID, fmt.Errorf("no market rows for params.fixture_slug %q (check market.slug)", fixtureSlug))
		return 0, 0, 1
	}

	outcomes, err := marketmap.OutcomesForFixtureMarkets(ctx, j.outcomeRepo, fixtures)
	if err != nil {
		j.failTask(ctx, task.ID, err)
		return 0, 0, 1
	}
	if len(outcomes) == 0 {
		j.failTask(ctx, task.ID, fmt.Errorf("no market_outcome under map_key %q", mapKey))
		return 0, 0, 1
	}

	for i := range outcomes {
		oc := outcomes[i]
		if !outcomeTypeAllowed(cfg.OutcomeTypes, oc.OutcomeType) {
			continue
		}
		fx, ok := fixtures[oc.MarketRowID]
		if !ok {
			continue
		}
		platform := strings.ToUpper(strings.TrimSpace(oc.Platform))
		evStart, evEnd := "", ""
		if fx.EventStartTime != nil {
			evStart = *fx.EventStartTime
		}
		if fx.EventEndTime != nil {
			evEnd = *fx.EventEndTime
		}
		dates, useFixed := utcDatesFromTaskParams(tp, hasTaskParams)
		if !useFixed {
			dates = UTCDatesForFixture(mapKey, evStart, evEnd, cfg.UTCDaysBeforeEvent, cfg.UTCDaysAfterEvent)
		}
		dates = FilterUTCDatesByMinAge(dates, cfg.MinUTCAgeDays)
		if len(dates) == 0 {
			continue
		}

		switch platform {
		case string(cryptoPredictionEnum.PlatformPolymarket):
			slug := PolymarketVendorSlug(oc)
			if slug == "" {
				continue
			}
			for _, day := range dates {
				d, s, f := j.ingestPolymarket(ctx, client, dp, cfg, mapKey, oc, slug, day)
				downloaded += d
				snapshots += s
				failed += f
			}
		case string(cryptoPredictionEnum.PlatformPredict):
			mid := predictVendorMarketID(oc)
			if mid == "" {
				continue
			}
			for _, day := range dates {
				d, s, f := j.ingestPredict(ctx, client, dp, cfg, mapKey, oc, mid, day)
				downloaded += d
				snapshots += s
				failed += f
			}
		}
	}

	status := persistententity.DataTaskStatusCompleted
	errText := ""
	if failed > 0 && downloaded == 0 {
		status = persistententity.DataTaskStatusFailed
		errText = fmt.Sprintf("all downloads failed (%d errors)", failed)
	} else if failed > 0 {
		errText = fmt.Sprintf("partial failures: %d", failed)
	}
	_ = j.dataTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":     status,
		"error_text": errText,
	})
	task.Status = status
	NotifyTaskFinished(ctx, task, downloaded, failed, errText)
	logger.InfoZ(ctx, "history_data_sync-task-done",
		zap.Uint("task_id", task.ID),
		zap.String("map_key", mapKey),
		zap.String("status", status),
		zap.Int("files_ok", downloaded),
		zap.Int("snapshots_upserted", snapshots),
		zap.Int("failed", failed))
	return downloaded, snapshots, failed
}

func (j *Sync) failTask(ctx context.Context, id uint, err error) {
	msg := trimErr(err, 500)
	_ = j.dataTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": id}, map[string]interface{}{
		"status":     persistententity.DataTaskStatusFailed,
		"error_text": msg,
	})
	notification.SendErrorLog(ctx, "history_data_sync-task-failed", zap.Uint("task_id", id), zap.Error(err))
	row, findErr := j.dataTaskRepo.FindOne(ctx, nil, map[string]interface{}{"id = ?": id}, "")
	if findErr == nil {
		NotifyTaskFinished(ctx, &row, 0, 0, msg)
	}
}

func (j *Sync) ingestPolymarket(
	ctx context.Context,
	client *pmdata.Client,
	dp persistententity.DataPlatform,
	cfg PMDataExtraConfig,
	mapKey string,
	oc persistententity.MarketOutcome,
	slug string,
	utcDay time.Time,
) (filesOK, snapshots, failed int) {
	logger.InfoZ(ctx, "history_data_sync-download",
		zap.String("exchange", "polymarket"),
		zap.String("symbol", slug),
		zap.String("utc_date", utcDay.Format("2006-01-02")),
		zap.String("map_key", mapKey))

	body, path, sha, fresh, status, err := j.parquetBodyPolymarket(ctx, client, dp, cfg, oc, slug, utcDay)
	if err != nil {
		if pmdata.IsNotFound(status) {
			j.saveSkipped(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePolymarket(), slug, utcDay, err)
			return 0, 0, 0
		}
		if pmdata.IsForbidden(status, err) {
			j.saveSkipped(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePolymarket(), slug, utcDay, fmt.Errorf("pmdata free tier: data within 3 weeks forbidden; upgrade or set min_utc_age_days"))
			return 0, 0, 0
		}
		if fresh {
			j.saveFailedFile(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePolymarket(), slug, utcDay, err)
			return 0, 0, 1
		}
		logger.InfoZ(ctx, "history_data_sync-polymarket-skip-no-body", zap.Error(err))
		return 0, 0, 0
	}
	if len(body) == 0 {
		return 0, 0, 0
	}
	if fresh {
		var werr error
		path, sha, werr = pmdata.WriteLocalFile(cfg.StorageDir, pmdata.VendorExchangePolymarket(), cfg.DataType, utcDay, slug, body)
		if werr != nil {
			j.saveFailedFile(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePolymarket(), slug, utcDay, werr)
			return 0, 0, 1
		}
	}
	filesOK = 1

	snaps, perr := pmdata.ParsePolymarketL2Parquet(body, cfg.MaxSnapshotsPerDay)
	if perr != nil {
		notification.SendErrorLog(ctx, "history_data_sync-parse-polymarket-parquet",
			zap.Error(perr), zap.String("slug", slug), zap.String("path", path))
		return filesOK, 0, 0
	}
	if err := j.upsertFileIndexRow(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePolymarket(), slug, utcDay, path, sha, body, snaps); err != nil {
		notification.SendErrorLog(ctx, "history_data_sync-upsert-file", zap.Error(err), zap.String("slug", slug))
		return filesOK, 0, 1
	}
	n, f := j.upsertL2Snapshots(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePolymarket(), slug, utcDay, path, sha, body, snaps)
	logger.InfoZ(ctx, "history_data_sync-polymarket-snapshots",
		zap.String("slug", slug),
		zap.String("utc_date", utcDay.Format("2006-01-02")),
		zap.Int("parsed", len(snaps)),
		zap.Int("inserted", n),
		zap.Int("max_bid_levels", pmdata.MaxBookDepth(snaps, true)),
		zap.Int("max_ask_levels", pmdata.MaxBookDepth(snaps, false)))
	return filesOK, n, f
}

func (j *Sync) ingestPredict(
	ctx context.Context,
	client *pmdata.Client,
	dp persistententity.DataPlatform,
	cfg PMDataExtraConfig,
	mapKey string,
	oc persistententity.MarketOutcome,
	marketID string,
	utcDay time.Time,
) (filesOK, snapshots, failed int) {
	logger.InfoZ(ctx, "history_data_sync-download",
		zap.String("exchange", "predict_fun"),
		zap.String("symbol", marketID),
		zap.String("utc_date", utcDay.Format("2006-01-02")),
		zap.String("map_key", mapKey))
	body, status, err := client.DownloadL2(ctx, pmdata.VendorExchangePredict(), cfg.DataType, marketID, utcDay, "")
	if err != nil {
		if pmdata.IsNotFound(status) {
			j.saveSkipped(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePredict(), marketID, utcDay, err)
			return 0, 0, 0
		}
		if pmdata.IsForbidden(status, err) {
			j.saveSkipped(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePredict(), marketID, utcDay, err)
			return 0, 0, 0
		}
		j.saveFailedFile(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePredict(), marketID, utcDay, err)
		return 0, 0, 1
	}
	path, sha, werr := pmdata.WriteLocalFile(cfg.StorageDir, pmdata.VendorExchangePredict(), cfg.DataType, utcDay, marketID, body)
	if werr != nil {
		j.saveFailedFile(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePredict(), marketID, utcDay, werr)
		return 0, 0, 1
	}

	filesOK = 1

	snaps, perr := pmdata.ParsePredictL2Parquet(body, cfg.MaxSnapshotsPerDay)
	if perr != nil {
		notification.SendErrorLog(ctx, "history_data_sync-parse-predict-parquet",
			zap.Error(perr), zap.String("market_id", marketID), zap.String("path", path))
		return filesOK, 0, 0
	}
	if err := j.upsertFileIndexRow(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePredict(), marketID, utcDay, path, sha, body, snaps); err != nil {
		notification.SendErrorLog(ctx, "history_data_sync-upsert-predict-file", zap.Error(err))
		return filesOK, 0, 1
	}
	n, f := j.upsertL2Snapshots(ctx, dp, cfg, mapKey, oc, pmdata.VendorExchangePredict(), marketID, utcDay, path, sha, body, snaps)
	logger.InfoZ(ctx, "history_data_sync-predict-snapshots",
		zap.String("market_id", marketID),
		zap.String("utc_date", utcDay.Format("2006-01-02")),
		zap.Int("parsed", len(snaps)),
		zap.Int("inserted", n),
		zap.Int("max_bid_levels", pmdata.MaxBookDepth(snaps, true)),
		zap.Int("max_ask_levels", pmdata.MaxBookDepth(snaps, false)))
	return filesOK, n, f
}

// upsertFileIndexRow record_kind=file：元数据 + 最后一档预览（真正回测用 snapshot 行）。
func (j *Sync) upsertFileIndexRow(
	ctx context.Context,
	dp persistententity.DataPlatform,
	cfg PMDataExtraConfig,
	mapKey string,
	oc persistententity.MarketOutcome,
	vendorExchange, vendorSymbol string,
	utcDay time.Time,
	path, sha string,
	body []byte,
	snaps []pmdata.PredictSnapshot,
) error {
	fileRow := j.baseRow(dp, cfg, mapKey, oc, vendorExchange, vendorSymbol, utcDay)
	fileRow.RecordKind = persistententity.HistoryRecordKindFile
	fileRow.SnapshotAtMs = 0
	fileRow.LocalFilePath = path
	fileRow.FileBytes = uint64(len(body))
	fileRow.FileSHA256 = sha
	fileRow.IngestStatus = persistententity.HistoryIngestOK
	fileRow.OrderBookBids = "[]"
	fileRow.OrderBookAsks = "[]"
	if len(snaps) > 0 {
		last := snaps[len(snaps)-1]
		fileRow.OrderBookBids = pmdata.LevelsJSON(last.Bids)
		fileRow.OrderBookAsks = pmdata.LevelsJSON(last.Asks)
	}
	fileRow.ErrorText = fmt.Sprintf("parquet %d bytes; snapshots=%d (see record_kind=snapshot)", len(body), len(snaps))
	return j.historyRepo.BatchUpsert(ctx, nil, []*persistententity.HistoryDataOrderBook{fileRow})
}

func (j *Sync) parquetBodyPolymarket(
	ctx context.Context,
	client *pmdata.Client,
	dp persistententity.DataPlatform,
	cfg PMDataExtraConfig,
	oc persistententity.MarketOutcome,
	slug string,
	utcDay time.Time,
) (body []byte, path, sha string, fresh bool, httpStatus int, err error) {
	if row, rerr := j.existingFileHistoryRow(ctx, dp.ID, oc, utcDay); rerr == nil && row != nil {
		path = strings.TrimSpace(row.LocalFilePath)
		sha = strings.TrimSpace(row.FileSHA256)
		if path != "" {
			body, err = os.ReadFile(path)
			if err == nil && len(body) > 0 {
				return body, path, sha, false, 0, nil
			}
		}
	}
	body, httpStatus, err = client.DownloadL2(ctx, pmdata.VendorExchangePolymarket(), cfg.DataType, slug, utcDay, cfg.PolymarketOutcome)
	if err != nil {
		return nil, "", "", true, httpStatus, err
	}
	return body, "", "", true, httpStatus, nil
}

func (j *Sync) existingFileHistoryRow(ctx context.Context, dpID uint, oc persistententity.MarketOutcome, utcDay time.Time) (*persistententity.HistoryDataOrderBook, error) {
	if j.historyRepo == nil {
		return nil, fmt.Errorf("history repo nil")
	}
	d := time.Date(utcDay.Year(), utcDay.Month(), utcDay.Day(), 0, 0, 0, 0, time.UTC)
	cond := map[string]interface{}{
		"market_outcome_id = ?": oc.ID,
		"utc_date = ?":          d,
		"record_kind = ?":       persistententity.HistoryRecordKindFile,
		"snapshot_at_ms = ?":    0,
	}
	if dpID > 0 {
		cond["data_platform_id = ?"] = dpID
	}
	row, err := j.historyRepo.FindOne(ctx, nil, cond, "")
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (j *Sync) upsertL2Snapshots(
	ctx context.Context,
	dp persistententity.DataPlatform,
	cfg PMDataExtraConfig,
	mapKey string,
	oc persistententity.MarketOutcome,
	vendorExchange, vendorSymbol string,
	utcDay time.Time,
	path, sha string,
	body []byte,
	snaps []pmdata.PredictSnapshot,
) (inserted, failed int) {
	existing, err := j.existingSnapshotMs(ctx, dp.ID, oc.ID, utcDay)
	if err != nil {
		notification.SendErrorLog(ctx, "history_data_sync-list-existing-snapshots", zap.Error(err))
		return 0, 1
	}
	var batch []*persistententity.HistoryDataOrderBook
	for _, s := range snaps {
		if _, ok := existing[s.TsMs]; ok {
			continue
		}
		row := j.baseRow(dp, cfg, mapKey, oc, vendorExchange, vendorSymbol, utcDay)
		row.RecordKind = persistententity.HistoryRecordKindSnapshot
		row.SnapshotAtMs = s.TsMs
		row.LocalFilePath = path
		row.FileBytes = uint64(len(body))
		row.FileSHA256 = sha
		row.OrderBookBids = pmdata.LevelsJSON(s.Bids)
		row.OrderBookAsks = pmdata.LevelsJSON(s.Asks)
		row.IngestStatus = persistententity.HistoryIngestOK
		if row.OrderBookBids == "[]" && row.OrderBookAsks == "[]" {
			continue
		}
		batch = append(batch, row)
	}
	if len(batch) == 0 {
		return 0, 0
	}
	const chunk = 100
	for i := 0; i < len(batch); i += chunk {
		end := i + chunk
		if end > len(batch) {
			end = len(batch)
		}
		part := batch[i:end]
		if err := j.historyRepo.BatchUpsert(ctx, nil, part); err != nil {
			notification.SendErrorLog(ctx, "history_data_sync-upsert-snapshots", zap.Error(err),
				zap.Int("offset", i), zap.Int("batch_len", len(part)))
			return inserted, 1
		}
		inserted += len(part)
	}
	return inserted, 0
}

func (j *Sync) existingSnapshotMs(ctx context.Context, platformID, outcomeID uint, utcDay time.Time) (map[int64]struct{}, error) {
	out := make(map[int64]struct{})
	if j.historyRepo == nil {
		return out, fmt.Errorf("history repo nil")
	}
	d := time.Date(utcDay.Year(), utcDay.Month(), utcDay.Day(), 0, 0, 0, 0, time.UTC)
	_, rows, err := j.historyRepo.List(ctx, nil, map[string]interface{}{
		"data_platform_id = ?":  platformID,
		"market_outcome_id = ?": outcomeID,
		"utc_date = ?":          d,
		"record_kind = ?":       persistententity.HistoryRecordKindSnapshot,
	}, "id asc", "")
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].SnapshotAtMs > 0 {
			out[rows[i].SnapshotAtMs] = struct{}{}
		}
	}
	return out, nil
}

func (j *Sync) baseRow(
	dp persistententity.DataPlatform,
	cfg PMDataExtraConfig,
	mapKey string,
	oc persistententity.MarketOutcome,
	vendorExchange, vendorSymbol string,
	utcDay time.Time,
) *persistententity.HistoryDataOrderBook {
	utcDate := time.Date(utcDay.Year(), utcDay.Month(), utcDay.Day(), 0, 0, 0, 0, time.UTC)
	return &persistententity.HistoryDataOrderBook{
		DataPlatformID:  dp.ID,
		MapKey:          mapKey,
		MarketRowID:     oc.MarketRowID,
		MarketOutcomeID: oc.ID,
		VenuePlatform:   strings.ToUpper(strings.TrimSpace(oc.Platform)),
		VendorExchange:  vendorExchange,
		VendorSymbol:    vendorSymbol,
		UTCDate:         utcDate,
		DataType:        cfg.DataType,
	}
}

func (j *Sync) alreadyHave(ctx context.Context, platformID, outcomeID uint, utcDay time.Time, kind string, snapshotMs int64) bool {
	d := time.Date(utcDay.Year(), utcDay.Month(), utcDay.Day(), 0, 0, 0, 0, time.UTC)
	_, err := j.historyRepo.FindOne(ctx, nil, map[string]interface{}{
		"data_platform_id = ?":  platformID,
		"market_outcome_id = ?": outcomeID,
		"utc_date = ?":          d,
		"record_kind = ?":       kind,
		"snapshot_at_ms = ?":    snapshotMs,
	}, "")
	return err == nil
}

func (j *Sync) saveSkipped(ctx context.Context, dp persistententity.DataPlatform, cfg PMDataExtraConfig, mapKey string, oc persistententity.MarketOutcome, exchange, symbol string, utcDay time.Time, err error) {
	row := j.baseRow(dp, cfg, mapKey, oc, exchange, symbol, utcDay)
	row.RecordKind = persistententity.HistoryRecordKindFile
	row.SnapshotAtMs = 0
	row.IngestStatus = persistententity.HistoryIngestSkipped
	row.OrderBookBids = "[]"
	row.OrderBookAsks = "[]"
	row.ErrorText = trimErr(err, 500)
	_ = j.historyRepo.BatchUpsert(ctx, nil, []*persistententity.HistoryDataOrderBook{row})
}

func (j *Sync) saveFailedFile(ctx context.Context, dp persistententity.DataPlatform, cfg PMDataExtraConfig, mapKey string, oc persistententity.MarketOutcome, exchange, symbol string, utcDay time.Time, err error) {
	row := j.baseRow(dp, cfg, mapKey, oc, exchange, symbol, utcDay)
	row.RecordKind = persistententity.HistoryRecordKindFile
	row.SnapshotAtMs = 0
	row.IngestStatus = persistententity.HistoryIngestFailed
	row.OrderBookBids = "[]"
	row.OrderBookAsks = "[]"
	row.ErrorText = trimErr(err, 500)
	_ = j.historyRepo.BatchUpsert(ctx, nil, []*persistententity.HistoryDataOrderBook{row})
	notification.SendErrorLog(ctx, "history_data_sync-download-failed",
		zap.Error(err),
		zap.String("exchange", exchange),
		zap.String("symbol", symbol),
		zap.String("map_key", mapKey))
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

// reclaimStaleRunningTasks 进程中断后残留的 running 恢复为 pending。
func (j *Sync) reclaimStaleRunningTasks(ctx context.Context, maxAge time.Duration) {
	if j.dataTaskRepo == nil || maxAge <= 0 {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	_, stale, err := j.dataTaskRepo.List(ctx, nil, map[string]interface{}{
		"ingest_source = ?":  persistententity.DataTaskIngestPmdata,
		"status = ?":         persistententity.DataTaskStatusRunning,
		"updated_at < ?":     cutoff,
	}, "id asc", "")
	if err != nil || len(stale) == 0 {
		return
	}
	for i := range stale {
		_ = j.dataTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": stale[i].ID}, map[string]interface{}{
			"status":     persistententity.DataTaskStatusPending,
			"error_text": "reclaimed stale running",
		})
		logger.InfoZ(ctx, "history_data_sync-reclaimed-stale",
			zap.Uint("task_id", stale[i].ID),
			zap.String("map_key", stale[i].MapKey))
	}
}