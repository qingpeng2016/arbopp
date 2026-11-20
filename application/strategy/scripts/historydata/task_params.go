package historydata

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// DataTaskParams data_task.params（JSON，可选）。
type DataTaskParams struct {
	FixtureSlug   string            `json:"fixture_slug"` // 场次 slug；可与 data_task.map_key 相同，不必 v1|联赛|slug
	UTCDaysBefore *int               `json:"utc_days_before"`
	UTCDaysAfter  *int               `json:"utc_days_after"`
	MinUTCAgeDays *int               `json:"min_utc_age_days"`
	UTCDates      []string           `json:"utc_dates"` // YYYY-MM-DD，指定日列表时忽略 before/after 推算
	OutcomeTypes  []string           `json:"outcome_types"`
	MarketURL     map[string]string  `json:"market_url"` // 仅核对用，history_data_sync 不读
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

// FixtureSlugForTask PMData：map_key 可直接写 epl-xxx-2026-10-17；兼容旧 v1|联赛|slug。
func FixtureSlugForTask(taskMapKey string, p DataTaskParams, hasParams bool) string {
	if hasParams && strings.TrimSpace(p.FixtureSlug) != "" {
		return strings.TrimSpace(p.FixtureSlug)
	}
	mk := strings.TrimSpace(taskMapKey)
	if mk == "" {
		return ""
	}
	if strings.HasPrefix(mk, "v1|") {
		if i := strings.LastIndex(mk, "|"); i >= 0 && i+1 < len(mk) {
			return mk[i+1:]
		}
	}
	return mk
}

// LoadHistoryFixtures 先 map_key，再 slug。
func LoadHistoryFixtures(
	ctx context.Context,
	marketRepo persistentrepo.MarketRepo,
	platformRepo persistentrepo.PlatformRepo,
	taskMapKey, fixtureSlug string,
) (map[uint]persistententity.Market, string, error) {
	storageKey := strings.TrimSpace(taskMapKey)
	fixtures, err := marketmap.LoadMarketsByMapKey(ctx, marketRepo, platformRepo, storageKey)
	if err != nil {
		return nil, "", err
	}
	if len(fixtures) == 0 && fixtureSlug != "" {
		fixtures, err = marketmap.LoadMarketsByFixtureSlug(ctx, marketRepo, platformRepo, fixtureSlug)
		if err != nil {
			return nil, "", err
		}
	}
	if len(fixtures) == 0 {
		return fixtures, storageKey, nil
	}
	for _, fx := range fixtures {
		if fx.MapKey != nil && strings.TrimSpace(*fx.MapKey) != "" {
			storageKey = strings.TrimSpace(*fx.MapKey)
			break
		}
	}
	return fixtures, storageKey, nil
}