package backtest

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const TaskName = "backtest_replay"

// Replay 消费 backtest_task(pending)，按秒重放 data_history_order_book。
type Replay struct {
	backtestTaskRepo persistentrepo.BacktestTaskRepo
	historyRepo      persistentrepo.HistoryDataOrderBookRepo
	marketRepo       persistentrepo.MarketRepo
	platformRepo     persistentrepo.PlatformRepo
	outcomeRepo      persistentrepo.MarketOutcomeRepo
	thresholdRepo    persistentrepo.ConfigStrategyThresholdRepo
	feeRepo          persistentrepo.ConfigStrategyFeeRepo
	btOppRepo        persistentrepo.BacktestArbBinaryOpportunityRepo
	btOrderRepo      persistentrepo.BacktestOrderRepo
}

func NewReplay(
	backtestTaskRepo persistentrepo.BacktestTaskRepo,
	historyRepo persistentrepo.HistoryDataOrderBookRepo,
	marketRepo persistentrepo.MarketRepo,
	platformRepo persistentrepo.PlatformRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
	thresholdRepo persistentrepo.ConfigStrategyThresholdRepo,
	feeRepo persistentrepo.ConfigStrategyFeeRepo,
	btOppRepo persistentrepo.BacktestArbBinaryOpportunityRepo,
	btOrderRepo persistentrepo.BacktestOrderRepo,
) *Replay {
	return &Replay{
		backtestTaskRepo: backtestTaskRepo,
		historyRepo:      historyRepo,
		marketRepo:       marketRepo,
		platformRepo:     platformRepo,
		outcomeRepo:      outcomeRepo,
		thresholdRepo:    thresholdRepo,
		feeRepo:          feeRepo,
		btOppRepo:        btOppRepo,
		btOrderRepo:      btOrderRepo,
	}
}

func (j *Replay) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "backtest_replay-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()
	if j.backtestTaskRepo == nil || j.historyRepo == nil {
		notification.SendErrorLog(ctx, "backtest_replay-skipped", zap.String("reason", "repo nil"))
		return
	}
	_, tasks, err := j.backtestTaskRepo.List(ctx, nil, map[string]interface{}{
		"status = ?": persistententity.BacktestTaskStatusPending,
	}, "id asc", "")
	if err != nil {
		notification.SendErrorLog(ctx, "backtest_replay-ListTasksFailed", zap.Error(err))
		return
	}
	if len(tasks) == 0 {
		logger.InfoZ(ctx, "backtest_replay-no-pending-tasks")
		return
	}
	for i := range tasks {
		j.runTask(ctx, &tasks[i])
	}
}

func (j *Replay) runTask(ctx context.Context, task *persistententity.BacktestTask) {
	if task == nil {
		return
	}
	mapKey := strings.TrimSpace(task.MapKey)
	strategy := strings.TrimSpace(task.StrategyCode)
	runYesno, runOneX2 := replayFlags(strategy)
	logger.InfoZ(ctx, "backtest_replay-task-start",
		zap.Uint("task_id", task.ID),
		zap.String("map_key", mapKey),
		zap.String("strategy_code", strategy),
		zap.Bool("run_yesno", runYesno),
		zap.Bool("run_onextwo", runOneX2))
	_ = j.backtestTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":     persistententity.BacktestTaskStatusRunning,
		"progress":   uint8(0),
		"error_text": "",
	})

	if !runYesno && !runOneX2 {
		j.failTask(ctx, task, fmt.Errorf("no strategy enabled for strategy_code %q", strategy))
		return
	}

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
	if len(outcomes) == 0 {
		j.failTask(ctx, task, fmt.Errorf("no outcomes for map_key %q", mapKey))
		return
	}

	_, snaps, err := j.historyRepo.List(ctx, nil, map[string]interface{}{
		"map_key = ?":      mapKey,
		"record_kind = ?":  persistententity.HistoryRecordKindSnapshot,
		"snapshot_at_ms > ?": 0,
	}, "snapshot_at_ms asc, id asc", "")
	if err != nil {
		j.failTask(ctx, task, err)
		return
	}
	if len(snaps) == 0 {
		j.failTask(ctx, task, fmt.Errorf("no history snapshots for map_key %q", mapKey))
		return
	}

	_ = j.btOrderRepo.DeleteWhere(ctx, nil, map[string]interface{}{"backtest_task_id = ?": task.ID})
	_ = j.btOppRepo.DeleteWhere(ctx, nil, map[string]interface{}{"backtest_task_id = ?": task.ID})

	marketsByRow := map[uint]persistententity.Market{}
	for i := range fixtures {
		fx := fixtures[i]
		marketsByRow[fx.ID] = fx
	}
	params := loadStrategyParams(ctx, j.thresholdRepo, j.feeRepo)
	sess := newStrategyEntrySession(ctx, mapKey, outcomes, marketsByRow, task.ID, params, j.btOppRepo, j.btOrderRepo)

	tsList := groupSnapshotsByTs(snaps)
	prog := newTaskProgress(j.backtestTaskRepo, task.ID, len(tsList))
	prog.flushStart(ctx)

	replayed := 0
	for _, tsMs := range tsList {
		replayAt := time.UnixMilli(tsMs).UTC()
		sess.SetReplayTime(replayAt)
		batch := rowsAtTs(snaps, tsMs)
		// 先写入本秒各平台 YES 视角 L2（内存互补维护 NO，见 replay.applySnapshot）。
		groupTouch := make(map[string]uint, len(batch))
		for _, row := range batch {
			if err := applySnapshot(sess.watch, row, replayAt); err != nil {
				logger.InfoZ(ctx, "backtest_replay-apply-skip",
					zap.Error(err),
					zap.Uint64("history_id", row.ID))
				continue
			}
			if row.MarketOutcomeID == 0 {
				continue
			}
			oid := uint(row.MarketOutcomeID)
			gk := sess.runner.OutcomeGroupKeyForOutcomeID(oid)
			if gk == "" {
				if runYesno {
					sess.runner.OnBookUpdate(oid, "", time.Time{})
				}
				continue
			}
			if _, ok := groupTouch[gk]; !ok {
				groupTouch[gk] = oid
			}
		}
		for _, oid := range groupTouch {
			if runYesno {
				sess.runner.OnBookUpdate(oid, "", time.Time{})
			}
		}
		if runOneX2 && sess.onextwo != nil {
			sess.onextwo.OnBookUpdateAfterReplay()
		}
		replayed++
		prog.advance(ctx)
	}

	extra := fmt.Sprintf("%d snapshots", len(snaps))
	prog.flushDone(ctx, extra)
	sum := fmt.Sprintf("progress=100%% replayed=%d/%d; %s", replayed, len(tsList), extra)
	_ = j.backtestTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":   persistententity.BacktestTaskStatusCompleted,
		"progress": uint8(100),
	})
	logger.InfoZ(ctx, "backtest_replay-task-done",
		zap.Uint("task_id", task.ID),
		zap.String("summary", sum))
}

func (j *Replay) failTask(ctx context.Context, task *persistententity.BacktestTask, err error) {
	if task == nil {
		return
	}
	msg := trimErr(err, 500)
	_ = j.backtestTaskRepo.Update(ctx, nil, map[string]interface{}{"id = ?": task.ID}, map[string]interface{}{
		"status":     persistententity.BacktestTaskStatusFailed,
		"progress":   uint8(0),
		"error_text": msg,
	})
	notification.SendErrorLog(ctx, "backtest_replay-task-failed",
		zap.Uint("task_id", task.ID),
		zap.Error(err))
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