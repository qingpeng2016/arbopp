package fetchopt

import (
	"context"
	"sync"
)

// BatchTask 单批拉盘口并落库。
type BatchTask func(context.Context) (upserted, missed int, err error)

// RunConcurrent 多批任务并发，concurrency 为同时在飞的批次数。
func RunConcurrent(ctx context.Context, concurrency int, tasks []BatchTask) (upserted, missed int, err error) {
	if len(tasks) == 0 {
		return 0, 0, nil
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(tasks) {
		concurrency = len(tasks)
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, task := range tasks {
		task := task
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			u, m, e := task(ctx)
			mu.Lock()
			upserted += u
			missed += m
			if e != nil && err == nil {
				err = e
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return upserted, missed, err
}