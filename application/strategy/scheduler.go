package strategy

import (
	"context"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/backtest"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata"
	historysynthetic "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/historydata/synthetic"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/statcollect"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/wsbookflush"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/taskalert"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"

	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
)

const scheduleModule = "less_one"

type scheduleTask struct {
	module          string
	taskName        string
	intervalSeconds float64
	enabled         bool
}

// Scheduler 按 bot_schedule_config 表挂脚本任务。
type Scheduler struct {
	taskAlert       *taskalert.TaskAlert
	orderBook       *orderbook.OrderBook
	historyDataSync      *historydata.Sync
	syntheticHistorySync *historysynthetic.Sync
	backtestReplay       *backtest.Replay
	strategyEntry        *strategyentry.Entry
	marketSync      *marketsync.MarketSync
	marketMap       *marketmap.MarketMap
	statCollect     *statcollect.Collect
	wsBookFlush     *wsbookflush.Flush
	botScheduleRepo persistentrepo.BotScheduleConfigRepo
	ns              *gocron.Scheduler
	jobMap          map[string]*gocron.Job
	configMap       map[string]float64
	started         bool
	mu              sync.Mutex
}

func NewScheduler(
	taskAlert *taskalert.TaskAlert,
	orderBook *orderbook.OrderBook,
	historyDataSync *historydata.Sync,
	syntheticHistorySync *historysynthetic.Sync,
	backtestReplay *backtest.Replay,
	strategyEntry *strategyentry.Entry,
	marketSync *marketsync.MarketSync,
	marketMap *marketmap.MarketMap,
	statCollect *statcollect.Collect,
	wsBookFlush *wsbookflush.Flush,
	botScheduleRepo persistentrepo.BotScheduleConfigRepo,
) *Scheduler {
	return &Scheduler{
		taskAlert:       taskAlert,
		orderBook:       orderBook,
		historyDataSync:      historyDataSync,
		syntheticHistorySync: syntheticHistorySync,
		backtestReplay:       backtestReplay,
		strategyEntry:        strategyEntry,
		marketSync:      marketSync,
		marketMap:       marketMap,
		statCollect:     statCollect,
		wsBookFlush:     wsBookFlush,
		botScheduleRepo: botScheduleRepo,
		ns:              gocron.NewScheduler(time.Local),
		jobMap:          make(map[string]*gocron.Job),
		configMap:       make(map[string]float64),
	}
}

func (s *Scheduler) Handle() {
	ctx := context.Background()
	s.mu.Lock()
	machineName := conf.Get_MACHINE_NAME()
	logger.DebugZ(ctx, "scheduler-LockAcquired", zap.String("machineName", machineName))
	defer s.mu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "scheduler-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()

	if !s.started {
		s.ns.StartAsync()
		s.started = true
		logger.InfoZ(ctx, "scheduler-Started")
	}

	tasks, err := s.loadTasksFromDB(ctx)
	if err != nil {
		notification.SendErrorLog(ctx, "scheduler-LoadConfigFailed", zap.Error(err))
		return
	}

	want := make(map[string]scheduleTask)
	for _, t := range tasks {
		if !t.enabled {
			continue
		}
		want[t.module+"-"+t.taskName] = t
	}

	for jobName := range s.jobMap {
		if _, exists := want[jobName]; !exists {
			if jobName == scheduleModule+"-"+strategyentry.TaskName && s.strategyEntry != nil {
				s.strategyEntry.Stop()
			}
			if jobName == scheduleModule+"-orderbook" && s.orderBook != nil {
				s.orderBook.StopPoll()
			}
			s.ns.RemoveByTags(jobName)
			delete(s.jobMap, jobName)
			delete(s.configMap, jobName)
			logger.InfoZ(ctx, "scheduler-JobRemoved", zap.String("jobName", jobName))
		}
	}
	if len(want) == 0 {
		logger.InfoZ(ctx, "scheduler-NoEnabledTasks")
		return
	}

	for _, config := range tasks {
		if !config.enabled {
			continue
		}
		handleFunc := s.getHandleFunc(config.module, config.taskName)
		if handleFunc == nil {
			notification.SendErrorLog(ctx, "scheduler-UnknownTask",
				zap.String("module", config.module),
				zap.String("taskName", config.taskName))
			continue
		}

		jobName := config.module + "-" + config.taskName
		needUpdate := false
		if _, exists := s.jobMap[jobName]; exists {
			if savedInterval, ok := s.configMap[jobName]; !ok || savedInterval != config.intervalSeconds {
				needUpdate = true
			}
		}

		if !needUpdate && s.jobMap[jobName] != nil {
			continue
		}

		if needUpdate {
			s.ns.RemoveByTags(jobName)
			delete(s.jobMap, jobName)
			delete(s.configMap, jobName)
		}

		intervalDur := time.Duration(config.intervalSeconds * float64(time.Second))
		if intervalDur < time.Millisecond {
			intervalDur = time.Millisecond
		}
		job, err := s.ns.Every(intervalDur).
			Tag(jobName).
			Name(jobName).
			WaitForSchedule().
			SingletonMode().
			Do(func() {
				defer catch()
				handleFunc()
			})
		if err != nil {
			notification.SendErrorLog(ctx, "scheduler-FailedToCreateJob",
				zap.String("module", config.module),
				zap.String("taskName", config.taskName),
				zap.Error(err))
			continue
		}

		s.jobMap[jobName] = job
		s.configMap[jobName] = config.intervalSeconds
		if needUpdate {
			logger.InfoZ(ctx, "scheduler-JobUpdated",
				zap.String("jobName", jobName),
				zap.Float64("interval", config.intervalSeconds))
		} else {
			logger.InfoZ(ctx, "scheduler-JobCreated",
				zap.String("jobName", jobName),
				zap.Float64("interval", config.intervalSeconds))
		}
	}

	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
}

func (s *Scheduler) loadTasksFromDB(ctx context.Context) ([]scheduleTask, error) {
	if s.botScheduleRepo == nil {
		return nil, nil
	}
	_, list, err := s.botScheduleRepo.List(ctx, nil, map[string]interface{}{
		"module = ?":     scheduleModule,
		"is_enabled = ?": 1,
	}, "exe_sort asc", "")
	if err != nil {
		return nil, err
	}
	out := make([]scheduleTask, 0, len(list))
	for _, row := range list {
		if row.IsStrategyEnabled != nil && *row.IsStrategyEnabled != 1 {
			continue
		}
		out = append(out, scheduleTask{
			module:          row.Module,
			taskName:        row.TaskName,
			intervalSeconds: row.IntervalSeconds,
			enabled:         row.IsEnabled == 1,
		})
	}
	return out, nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for jobName := range s.jobMap {
		_ = s.ns.RemoveByTags(jobName)
	}
	s.jobMap = make(map[string]*gocron.Job)
	s.configMap = make(map[string]float64)
	s.started = false
	if s.strategyEntry != nil {
		s.strategyEntry.Stop()
	}
	if s.orderBook != nil {
		s.orderBook.StopPoll()
	}
	s.ns.Stop()
	return nil
}

func (s *Scheduler) getHandleFunc(module, taskName string) func() {
	if module != scheduleModule {
		return nil
	}
	switch taskName {
	case "task_alert":
		return func() {
			if s.taskAlert != nil {
				s.taskAlert.Run(context.Background())
			}
		}
	case "orderbook":
		return func() {
			if s.orderBook != nil {
				s.orderBook.RunPoll(context.Background())
			}
		}
	case "orderbook_sync":
		return func() {
			if s.orderBook != nil {
				s.orderBook.Run(context.Background())
			}
		}
	case historydata.TaskName:
		return func() {
			if s.historyDataSync != nil {
				s.historyDataSync.Run(context.Background())
			}
		}
	case historysynthetic.TaskName:
		return func() {
			if s.syntheticHistorySync != nil {
				s.syntheticHistorySync.Run(context.Background())
			}
		}
	case backtest.TaskName:
		return func() {
			if s.backtestReplay != nil {
				s.backtestReplay.Run(context.Background())
			}
		}
	case "market_sync":
		return func() {
			if s.marketSync != nil {
				s.marketSync.Run(context.Background())
			}
		}
	case "market_map":
		return func() {
			if s.marketMap != nil {
				s.marketMap.Run(context.Background())
			}
		}
	case statcollect.TaskName:
		return func() {
			if s.statCollect != nil {
				s.statCollect.Run(context.Background())
			}
		}
	case strategyentry.TaskName:
		return func() {
			if s.strategyEntry != nil {
				s.strategyEntry.Run(context.Background())
			}
		}
	case wsbookflush.TaskName:
		return func() {
			if s.wsBookFlush != nil {
				s.wsBookFlush.Run(context.Background())
			}
		}
	default:
		return nil
	}
}
