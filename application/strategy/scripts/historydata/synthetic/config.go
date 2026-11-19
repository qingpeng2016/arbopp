package synthetic

import (
	"encoding/json"
	"strings"
)

// ExtraConfig data_platform（code=synthetic）extra_config。
type ExtraConfig struct {
	TotalSeconds  int      `json:"total_seconds"`
	BatchSeconds  int      `json:"batch_seconds"`
	OutcomeTypes  []string `json:"outcome_types"`
}

func ParseExtraConfig(raw string) ExtraConfig {
	cfg := ExtraConfig{
		TotalSeconds: 1000,
		BatchSeconds: 100,
		OutcomeTypes: []string{"moneyline"},
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(raw), &cfg)
	if cfg.TotalSeconds <= 0 {
		cfg.TotalSeconds = 1000
	}
	if cfg.BatchSeconds <= 0 {
		cfg.BatchSeconds = 100
	}
	if cfg.BatchSeconds > cfg.TotalSeconds {
		cfg.BatchSeconds = cfg.TotalSeconds
	}
	if len(cfg.OutcomeTypes) == 0 {
		cfg.OutcomeTypes = []string{"moneyline"}
	}
	return cfg
}