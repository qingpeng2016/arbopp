package wsbookflush

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/wsbookqueue"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const TaskName = "ws_book_flush"

const maxRoundsPerRun = 20

// Flush 从 Redis 队列批量写入 market_order_book_ws。
type Flush struct {
	repo  persistentrepo.OrderBookWSRepo
	queue *wsbookqueue.Queue
}

func NewFlush(repo persistentrepo.OrderBookWSRepo, queue *wsbookqueue.Queue) *Flush {
	return &Flush{repo: repo, queue: queue}
}

func (f *Flush) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "ws_book_flush-Panic",
				zap.Any("panic", r), zap.String("stack", string(debug.Stack())))
		}
	}()
	if f == nil || f.repo == nil || f.queue == nil {
		notification.SendErrorLog(ctx, "ws_book_flush-skipped", zap.String("reason", "repo or redis queue nil"))
		return
	}
	var total int
	for round := 0; round < maxRoundsPerRun; round++ {
		batch, err := f.queue.PopBatch(ctx, wsbookqueue.MaxPopBatch)
		if err != nil {
			notification.SendErrorLog(ctx, "ws_book_flush-pop-failed", zap.Error(err))
			return
		}
		if len(batch) == 0 {
			break
		}
		batch = wsbookqueue.DedupeBatch(batch)
		writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = f.repo.BatchUpsert(writeCtx, nil, batch)
		cancel()
		if err != nil {
			notification.SendErrorLog(ctx, "ws_book_flush-upsert-failed",
				zap.Error(err), zap.Int("rows", len(batch)))
			return
		}
		total += len(batch)
		if len(batch) < wsbookqueue.MaxPopBatch {
			break
		}
	}
	if total > 0 {
		logger.InfoZ(ctx, "ws_book_flush-done", zap.Int("rows", total))
	}
}