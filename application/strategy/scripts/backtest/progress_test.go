package backtest

import (
	"context"
	"testing"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

func TestPercentBucket(t *testing.T) {
	cases := []struct {
		done, total, want int
	}{
		{0, 1000, 0},
		{99, 1000, 0},
		{100, 1000, 10},
		{500, 1000, 50},
		{999, 1000, 90},
		{1000, 1000, 100},
	}
	for _, c := range cases {
		p := &taskProgress{total: c.total, done: c.done}
		if got := p.percentBucket(); got != c.want {
			t.Fatalf("done=%d total=%d got=%d want=%d", c.done, c.total, got, c.want)
		}
	}
}

func TestProgressFlushEvery10Pct(t *testing.T) {
	var writes []string
	repo := &stubBacktestTaskRepo{
		onUpdate: func(text string) { writes = append(writes, text) },
	}
	p := newTaskProgress(repo, 1, 100)
	p.flushStart(context.Background())
	for i := 0; i < 100; i++ {
		p.done = i + 1
		p.maybeFlush(context.Background())
	}
	if len(writes) != 11 {
		t.Fatalf("writes=%d want 11 (0,10,...,100)", len(writes))
	}
	if writes[0] != "progress=0% replayed=0/100" {
		t.Fatalf("start %q", writes[0])
	}
	if writes[len(writes)-1] != "progress=100% replayed=100/100" {
		t.Fatalf("last %q", writes[len(writes)-1])
	}
}

type stubBacktestTaskRepo struct {
	onUpdate func(string)
}

func (s *stubBacktestTaskRepo) List(_ context.Context, _ *gorm.DB, _ map[string]interface{}, _, _ string) (int64, []persistententity.BacktestTask, error) {
	return 0, nil, nil
}

func (s *stubBacktestTaskRepo) Update(_ context.Context, _ *gorm.DB, _ map[string]interface{}, updateData map[string]interface{}) error {
	if s.onUpdate != nil {
		if v, ok := updateData["error_text"].(string); ok {
			s.onUpdate(v)
		}
	}
	return nil
}