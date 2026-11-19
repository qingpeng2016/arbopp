package leagueopt

import (
	"encoding/json"
	"strings"
)

// Platform 单平台同时拉多少个联赛。两边都没有「一次请求多个联赛」的批量接口，只能按联赛开协程。
type Platform struct {
	LeagueConcurrency int `json:"concurrency"`
}

// Config market_sync.concurrency（fallback；优先 market_category.concurrency 列）。
type Config struct {
	Polymarket Platform `json:"polymarket"`
	Kalshi     Platform `json:"kalshi"`
	Predict    Platform `json:"predict"`
}

const (
	defaultLeagueConcurrency = 1
	maxLeagueConcurrency     = 8
)

func Parse(raw, valueType string) Config {
	raw = strings.TrimSpace(raw)
	valueType = strings.ToLower(strings.TrimSpace(valueType))
	if raw == "" || (valueType != "" && valueType != "json" && valueType != "json_object") {
		return Config{}
	}
	var wire struct {
		Polymarket platformWire `json:"polymarket"`
		Kalshi     platformWire `json:"kalshi"`
		Predict    platformWire `json:"predict"`
	}
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return Config{}
	}
	return Config{
		Polymarket: Platform{LeagueConcurrency: wire.Polymarket.concurrency()},
		Kalshi:     Platform{LeagueConcurrency: wire.Kalshi.concurrency()},
		Predict:    Platform{LeagueConcurrency: wire.Predict.concurrency()},
	}
}

type platformWire struct {
	Concurrency       int `json:"concurrency"`
	LeagueConcurrency int `json:"league_concurrency"`
}

func (p platformWire) concurrency() int {
	if p.Concurrency > 0 {
		return p.Concurrency
	}
	return p.LeagueConcurrency
}

func (p Platform) Concurrency() int {
	n := p.LeagueConcurrency
	if n <= 0 {
		return defaultLeagueConcurrency
	}
	if n > maxLeagueConcurrency {
		return maxLeagueConcurrency
	}
	return n
}
