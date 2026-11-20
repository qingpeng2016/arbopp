package historydata

import (
	"encoding/json"
	"strings"
	"time"
)

// DataTaskParams data_task.params（JSON，可选）。
type DataTaskParams struct {
	UTCDaysBefore   *int     `json:"utc_days_before"`
	UTCDaysAfter    *int     `json:"utc_days_after"`
	MinUTCAgeDays   *int     `json:"min_utc_age_days"`
	UTCDates        []string `json:"utc_dates"` // YYYY-MM-DD，指定日列表时忽略 before/after 推算
	OutcomeTypes    []string `json:"outcome_types"`
}

func ParseDataTaskParams(raw string) (DataTaskParams, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return DataTaskParams{}, false
	}
	var p DataTaskParams
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return DataTaskParams{}, false
	}
	return p, true
}

// ApplyPMDataExtra 任务 params 覆盖 data_platform.pmdata extra_config。
func ApplyPMDataExtra(cfg PMDataExtraConfig, p DataTaskParams, hasParams bool) PMDataExtraConfig {
	if !hasParams {
		return cfg
	}
	if p.UTCDaysBefore != nil && *p.UTCDaysBefore >= 0 {
		cfg.UTCDaysBeforeEvent = *p.UTCDaysBefore
	}
	if p.UTCDaysAfter != nil && *p.UTCDaysAfter >= 0 {
		cfg.UTCDaysAfterEvent = *p.UTCDaysAfter
	}
	if p.MinUTCAgeDays != nil && *p.MinUTCAgeDays >= 0 {
		cfg.MinUTCAgeDays = *p.MinUTCAgeDays
	}
	if len(p.OutcomeTypes) > 0 {
		cfg.OutcomeTypes = p.OutcomeTypes
	}
	return cfg
}

func utcDatesFromTaskParams(p DataTaskParams, hasParams bool) ([]time.Time, bool) {
	if !hasParams || len(p.UTCDates) == 0 {
		return nil, false
	}
	out := make([]time.Time, 0, len(p.UTCDates))
	for _, s := range p.UTCDates {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
		if err != nil {
			continue
		}
		out = append(out, time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC))
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}