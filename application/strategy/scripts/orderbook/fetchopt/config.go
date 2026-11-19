package fetchopt

import (
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

// PlatformFetch 单平台：每批条数 + 批间并发协程数。
type PlatformFetch struct {
	BatchSize        int `json:"batch_size"`
	BatchConcurrency int `json:"batch_concurrency"`
}

// FetchConfig orderbook_sync 拉盘口参数（config_strategy.orderbook_fetch）。
type FetchConfig struct {
	Polymarket PlatformFetch `json:"polymarket"`
	Kalshi     PlatformFetch `json:"kalshi"`
	Predict    PlatformFetch `json:"predict"`
}

const (
	defaultPolyBatch        = 500
	defaultKalshiBatch      = 100
	defaultPredictBatch     = 100
	defaultBatchConcurrency = 1
	maxPolyBatch            = 500
	maxPredictBatch         = 100
	maxBatchConcurrency     = 16
)

func DefaultFetchConfig() FetchConfig {
	return FetchConfig{
		Polymarket: PlatformFetch{BatchSize: defaultPolyBatch, BatchConcurrency: defaultBatchConcurrency},
		Kalshi:     PlatformFetch{BatchSize: defaultKalshiBatch, BatchConcurrency: defaultBatchConcurrency},
		Predict:    PlatformFetch{BatchSize: defaultPredictBatch, BatchConcurrency: defaultBatchConcurrency},
	}
}

func (f PlatformFetch) Normalized(platform string) PlatformFetch {
	out := f
	if out.BatchSize <= 0 {
		switch {
		case strings.EqualFold(platform, string(cryptoPredictionEnum.PlatformPolymarket)):
			out.BatchSize = defaultPolyBatch
		case strings.EqualFold(platform, string(cryptoPredictionEnum.PlatformPredict)):
			out.BatchSize = defaultPredictBatch
		default:
			out.BatchSize = defaultKalshiBatch
		}
	}
	if strings.EqualFold(platform, string(cryptoPredictionEnum.PlatformPolymarket)) && out.BatchSize > maxPolyBatch {
		out.BatchSize = maxPolyBatch
	}
	if strings.EqualFold(platform, string(cryptoPredictionEnum.PlatformPredict)) && out.BatchSize > maxPredictBatch {
		out.BatchSize = maxPredictBatch
	}
	if out.BatchConcurrency <= 0 {
		out.BatchConcurrency = defaultBatchConcurrency
	}
	if out.BatchConcurrency > maxBatchConcurrency {
		out.BatchConcurrency = maxBatchConcurrency
	}
	return out
}

func (c FetchConfig) ForPlatform(platform string) PlatformFetch {
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case string(cryptoPredictionEnum.PlatformPolymarket):
		return c.Polymarket.Normalized(platform)
	case string(cryptoPredictionEnum.PlatformKalshi):
		return c.Kalshi.Normalized(platform)
	case string(cryptoPredictionEnum.PlatformPredict):
		return c.Predict.Normalized(platform)
	default:
		return PlatformFetch{BatchSize: defaultKalshiBatch, BatchConcurrency: defaultBatchConcurrency}
	}
}
