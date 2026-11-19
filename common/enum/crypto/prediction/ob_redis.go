package prediction

import "encoding/json"

// ObRedisValue Redis 单 key 的 value：订单簿 asks/bids JSON，与 RedisKeyOB 对应
type ObRedisValue struct {
	Asks json.RawMessage `json:"asks"`
	Bids json.RawMessage `json:"bids"`
}
