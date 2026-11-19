package leagueopt

import (
	"context"
	"sync"
)

// Each 按 concurrency 同时执行 n 个联赛任务。concurrency<=0 时一次只跑一个。
func Each(ctx context.Context, concurrency, n int, fn func(i int)) {
	if n <= 0 || fn == nil {
		return
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > maxLeagueConcurrency {
		concurrency = maxLeagueConcurrency
	}
	if concurrency > n {
		concurrency = n
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			fn(i)
		}(i)
	}
	wg.Wait()
}
