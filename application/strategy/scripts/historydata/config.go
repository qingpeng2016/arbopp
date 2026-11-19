package historydata

import (
	"encoding/json"
	"strings"
	"time"
)

// PMDataExtraConfig data_platform.extra_config（code=pmdata）。
type PMDataExtraConfig struct {
	StorageDir         string   `json:"storage_dir"`
	DataType           string   `json:"data_type"`
	UTCDaysBeforeEvent int      `json:"utc_days_before_event"`
	UTCDaysAfterEvent  int      `json:"utc_days_after_event"`
	OutcomeTypes       []string `json:"outcome_types"`
	PolymarketOutcome    string   `json:"polymarket_outcome"` // yes | no
	MaxSnapshotsPerDay   int      `json:"max_snapshots_per_day"`
	MinUTCAgeDays        int      `json:"min_utc_age_days"`
}

func ParsePMDataExtraConfig(raw string) PMDataExtraConfig {
	cfg := PMDataExtraConfig{
		StorageDir:         "var/pmdata",
		DataType:           "l2",
		UTCDaysBeforeEvent: 1,
		UTCDaysAfterEvent:  1,
		OutcomeTypes:       []string{"moneyline"},
		PolymarketOutcome:  "yes",
		MaxSnapshotsPerDay: 500,
		MinUTCAgeDays:      21,
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cfg
	}
	_ = json.Unmarshal([]byte(raw), &cfg)
	if strings.TrimSpace(cfg.StorageDir) == "" {
		cfg.StorageDir = "var/pmdata"
	}
	if strings.TrimSpace(cfg.DataType) == "" {
		cfg.DataType = "l2"
	}
	if cfg.UTCDaysBeforeEvent < 0 {
		cfg.UTCDaysBeforeEvent = 0
	}
	if cfg.UTCDaysAfterEvent < 0 {
		cfg.UTCDaysAfterEvent = 0
	}
	if len(cfg.OutcomeTypes) == 0 {
		cfg.OutcomeTypes = []string{"moneyline"}
	}
	if strings.TrimSpace(cfg.PolymarketOutcome) == "" {
		cfg.PolymarketOutcome = "yes"
	}
	if cfg.MaxSnapshotsPerDay <= 0 {
		cfg.MaxSnapshotsPerDay = 500
	}
	if cfg.MinUTCAgeDays < 0 {
		cfg.MinUTCAgeDays = 0
	}
	return cfg
}

// FilterUTCDatesByMinAge 去掉 PMData 免费档不允许的近期 UTC 日（仍可直接被 API 403）。
func FilterUTCDatesByMinAge(dates []time.Time, minAgeDays int) []time.Time {
	if minAgeDays <= 0 || len(dates) == 0 {
		return dates
	}
	cutoff := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -minAgeDays)
	out := make([]time.Time, 0, len(dates))
	for _, d := range dates {
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		if !day.After(cutoff) {
			out = append(out, day)
		}
	}
	return out
}

func outcomeTypeAllowed(types []string, ocType string) bool {
	ocType = strings.ToLower(strings.TrimSpace(ocType))
	if ocType == "" {
		return true
	}
	for _, t := range types {
		if strings.EqualFold(strings.TrimSpace(t), ocType) {
			return true
		}
	}
	return false
}
