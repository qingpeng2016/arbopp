package wsbookqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/redis/go-redis/v9"
)

const (
	// RedisListKey WS 盘口待落库队列（RPUSH / 左侧 LRANGE 批量弹出）。
	RedisListKey = "ob:ws:mysql_queue"
	// MaxPopBatch 单次批量弹出上限。
	MaxPopBatch = 500
)

var batchPop = redis.NewScript(`
local key = KEYS[1]
local n = tonumber(ARGV[1])
local len = redis.call('LLEN', key)
if len == 0 then return {} end
if n > len then n = len end
local items = redis.call('LRANGE', key, 0, n - 1)
redis.call('LTRIM', key, n, -1)
return items
`)

// Queue Redis 队列：strategy_entry 写入，ws_book_flush 批量落库。
type Queue struct {
	rdb redis.UniversalClient
	key string
}

func New(rdb redis.UniversalClient) *Queue {
	if rdb == nil {
		return nil
	}
	return &Queue{rdb: rdb, key: RedisListKey}
}

func (q *Queue) Push(ctx context.Context, row *persistententity.OrderBookWS) error {
	if q == nil || q.rdb == nil || row == nil {
		return nil
	}
	b, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("wsbookqueue marshal: %w", err)
	}
	return q.rdb.RPush(ctx, q.key, b).Err()
}

// PopBatch 原子弹出最多 max 条（先进先出）。
func (q *Queue) PopBatch(ctx context.Context, max int) ([]*persistententity.OrderBookWS, error) {
	if q == nil || q.rdb == nil {
		return nil, nil
	}
	if max <= 0 {
		max = MaxPopBatch
	}
	if max > MaxPopBatch {
		max = MaxPopBatch
	}
	raw, err := batchPop.Run(ctx, q.rdb, []string{q.key}, max).Slice()
	if err != nil {
		return nil, err
	}
	out := make([]*persistententity.OrderBookWS, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		var row persistententity.OrderBookWS
		if json.Unmarshal([]byte(s), &row) != nil {
			continue
		}
		out = append(out, &row)
	}
	return out, nil
}