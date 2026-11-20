package historydata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// DataTaskParams data_task.params（JSON，可选）。
type DataTaskParams struct {
	FixtureSlug   string            `json:"fixture_slug"` // pmdata 必填：按 market.slug 找 Poly/Predict
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

// PMDataFixtureSlug history_data_sync 只认 params.fixture_slug（不认 data_task.map_key）。
func PMDataFixtureSlug(p DataTaskParams, hasParams bool) (string, error) {
	if !hasParams {
		return "", fmt.Errorf("pmdata task requires params JSON")
	}
	slug := strings.TrimSpace(p.FixtureSlug)
	if slug == "" {
		return "", fmt.Errorf("params.fixture_slug is required for history_data_sync")
	}
	return slug, nil
}

// LoadPMDataFixtures 按 fixture_slug 加载 market；history 落库 map_key 优先用 market.map_key，否则用 slug。
func LoadPMDataFixtures(
	ctx context.Context,
	marketRepo persistentrepo.MarketRepo,
	platformRepo persistentrepo.PlatformRepo,
	fixtureSlug string,
) (map[uint]persistententity.Market, string, error) {
	fixtureSlug = strings.TrimSpace(fixtureSlug)
	fixtures, err := marketmap.LoadMarketsByFixtureSlug(ctx, marketRepo, platformRepo, fixtureSlug)
	if err != nil {
		return nil, "", err
	}
	storageKey := fixtureSlug
	for _, fx := range fixtures {
		if fx.MapKey != nil && strings.TrimSpace(*fx.MapKey) != "" {
			storageKey = strings.TrimSpace(*fx.MapKey)
			break
		}
	}
	return fixtures, storageKey, nil
}