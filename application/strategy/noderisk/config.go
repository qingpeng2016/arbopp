package noderisk

import (
	"context"
	"time"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"github.com/shopspring/decimal"
)

const (
	KeyEnabled            = "risk_relay_enabled"
	KeyCheckIntervalSec   = "risk_relay_check_interval_sec"
	KeyMaxRTTMs           = "risk_relay_max_rtt_ms"
	KeyBookStaleSec       = "risk_relay_book_stale_sec"
	KeyBookPushIntervalSec = "risk_relay_book_push_interval_sec"
	KeyRecoverOKSamples   = "risk_relay_recover_ok_samples"
	KeyAlertRepeatSec     = "risk_relay_alert_repeat_sec"
)

// Config 风控阈值（热路径 atomic 读）。
type Config struct {
	Enabled            bool
	CheckInterval      time.Duration
	MaxRTT             time.Duration
	BookStale          time.Duration
	BookPushInterval   time.Duration
	RecoverOKSamples   int
	AlertRepeat        time.Duration
}

func defaultConfig() Config {
	return Config{
		Enabled:          true,
		CheckInterval:    60 * time.Second,
		MaxRTT:           500 * time.Millisecond,
		BookStale:        12 * time.Second,
		BookPushInterval: 5 * time.Second,
		RecoverOKSamples: 3,
		AlertRepeat:      5 * time.Minute,
	}
}

// LoadConfig 从 config_strategy_threshold 读 risk_* 键。
func LoadConfig(ctx context.Context, repo persistentrepo.ConfigStrategyThresholdRepo) Config {
	c := defaultConfig()
	if repo == nil {
		return c
	}
	if v, ok := thresholdBool(ctx, repo, KeyEnabled); ok {
		c.Enabled = v
	}
	if v, ok := thresholdSec(ctx, repo, KeyCheckIntervalSec); ok && v > 0 {
		c.CheckInterval = time.Duration(v) * time.Second
	}
	if v, ok := thresholdMs(ctx, repo, KeyMaxRTTMs); ok && v > 0 {
		c.MaxRTT = time.Duration(v) * time.Millisecond
	}
	if v, ok := thresholdSec(ctx, repo, KeyBookStaleSec); ok && v > 0 {
		c.BookStale = time.Duration(v) * time.Second
	}
	if v, ok := thresholdSec(ctx, repo, KeyBookPushIntervalSec); ok && v > 0 {
		c.BookPushInterval = time.Duration(v) * time.Second
	}
	if v, ok := thresholdInt(ctx, repo, KeyRecoverOKSamples); ok && v > 0 {
		c.RecoverOKSamples = v
	}
	if v, ok := thresholdSec(ctx, repo, KeyAlertRepeatSec); ok && v > 0 {
		c.AlertRepeat = time.Duration(v) * time.Second
	}
	return c
}

func thresholdRow(ctx context.Context, repo persistentrepo.ConfigStrategyThresholdRepo, key string) (persistententity.ConfigStrategyThreshold, bool) {
	row, err := repo.FindOne(ctx, nil, map[string]interface{}{
		"config_key = ?": key,
		"status = ?":     1,
	}, "")
	if err != nil {
		return persistententity.ConfigStrategyThreshold{}, false
	}
	return row, true
}

func thresholdSec(ctx context.Context, repo persistentrepo.ConfigStrategyThresholdRepo, key string) (int64, bool) {
	row, ok := thresholdRow(ctx, repo, key)
	if !ok {
		return 0, false
	}
	f := row.ConfigValue.InexactFloat64()
	if f < 0 {
		return 0, false
	}
	return int64(f), true
}

func thresholdMs(ctx context.Context, repo persistentrepo.ConfigStrategyThresholdRepo, key string) (int64, bool) {
	return thresholdSec(ctx, repo, key)
}

func thresholdInt(ctx context.Context, repo persistentrepo.ConfigStrategyThresholdRepo, key string) (int, bool) {
	v, ok := thresholdSec(ctx, repo, key)
	if !ok {
		return 0, false
	}
	return int(v), true
}

func thresholdBool(ctx context.Context, repo persistentrepo.ConfigStrategyThresholdRepo, key string) (bool, bool) {
	row, ok := thresholdRow(ctx, repo, key)
	if !ok {
		return false, false
	}
	return row.ConfigValue.GreaterThanOrEqual(decimal.NewFromFloat(0.5)), true
}