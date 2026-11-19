package backtest

import (
	"context"
	"fmt"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

const progressFlushStep = 10

// taskProgress 回测回放进度（内存），每跨过 10% 档位写库一次。
type taskProgress struct {
	taskID      uint
	total       int
	done        int
	lastWritten int
	repo        persistentrepo.BacktestTaskRepo
}

func newTaskProgress(repo persistentrepo.BacktestTaskRepo, taskID uint, totalSeconds int) *taskProgress {
	return &taskProgress{repo: repo, taskID: taskID, total: totalSeconds, lastWritten: -1}
}

func (p *taskProgress) advance(ctx context.Context) {
	if p == nil || p.total <= 0 {
		return
	}
	p.done++
	p.maybeFlush(ctx)
}

func (p *taskProgress) flushStart(ctx context.Context) {
	if p == nil {
		return
	}
	p.lastWritten = -1
	p.maybeFlush(ctx)
}

func (p *taskProgress) flushDone(ctx context.Context, extra string) {
	if p == nil || p.repo == nil {
		return
	}
	text := fmt.Sprintf("progress=100%% replayed=%d/%d", p.done, p.total)
	if extra != "" {
		text = text + "; " + extra
	}
	_ = p.repo.Update(ctx, nil, map[string]interface{}{"id = ?": p.taskID}, map[string]interface{}{
		"progress":   uint8(100),
		"error_text": trimErrText(text, 500),
	})
	p.lastWritten = 100
}

func (p *taskProgress) maybeFlush(ctx context.Context) {
	if p == nil || p.repo == nil || p.total <= 0 {
		return
	}
	bucket := p.percentBucket()
	if bucket <= p.lastWritten {
		return
	}
	p.lastWritten = bucket
	text := fmt.Sprintf("progress=%d%% replayed=%d/%d", bucket, p.done, p.total)
	logger.InfoZ(ctx, "backtest_replay-progress",
		zap.Uint("task_id", p.taskID),
		zap.Int("bucket_pct", bucket),
		zap.Int("replayed", p.done),
		zap.Int("total", p.total))
	_ = p.repo.Update(ctx, nil, map[string]interface{}{"id = ?": p.taskID}, map[string]interface{}{
		"progress":   uint8(bucket),
		"error_text": trimErrText(text, 500),
	})
}

func (p *taskProgress) percentBucket() int {
	if p.total <= 0 {
		return 100
	}
	if p.done >= p.total {
		return 100
	}
	pct := p.done * 100 / p.total
	bucket := (pct / progressFlushStep) * progressFlushStep
	return bucket
}

func trimErrText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}