package marketmap

import (
	"context"
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// LoadVerifiedFixtures 按 watch_map_keys 加载各 map_key 下 status=ACTIVE 且 platform IN (ACTIVE 平台) 的场次。
func LoadVerifiedFixtures(ctx context.Context, repo persistentrepo.MarketRepo, cfgRepo persistentrepo.ConfigStrategyRepo, platformRepo persistentrepo.PlatformRepo) (map[uint]persistententity.Market, error) {
	if repo == nil {
		return map[uint]persistententity.Market{}, nil
	}
	keys := watchMapKeysFromConfig(ctx, cfgRepo)
	if len(keys) == 0 {
		return map[uint]persistententity.Market{}, nil
	}
	platIN, err := activePlatformIN(ctx, platformRepo)
	if err != nil {
		return nil, err
	}
	if platformRepo != nil && len(platIN) == 0 {
		return map[uint]persistententity.Market{}, nil
	}
	marketActive := string(cryptoPredictionEnum.MarketStatusActive)
	out := make(map[uint]persistententity.Market)
	for _, mapKey := range keys {
		where := map[string]interface{}{
			"map_key = ?": mapKey,
			"status = ?":  marketActive,
		}
		if len(platIN) > 0 {
			where["platform IN ?"] = platIN
		}
		_, rows, err := repo.List(ctx, nil, where, "platform asc, id asc", "")
		if err != nil {
			return nil, err
		}
		if !MapKeyHasTwoPlatforms(rows) {
			continue
		}
		for i := range rows {
			out[rows[i].ID] = rows[i]
		}
	}
	return out, nil
}

// LoadVerifiedFixturesNew 不读 watch_map_keys：取 ACTIVE 平台下 status=ACTIVE、map_verified=1 且 map_key 非空的场次。
func LoadVerifiedFixturesNew(ctx context.Context, repo persistentrepo.MarketRepo, _ persistentrepo.ConfigStrategyRepo, platformRepo persistentrepo.PlatformRepo) (map[uint]persistententity.Market, error) {
	if repo == nil {
		return map[uint]persistententity.Market{}, nil
	}
	platIN, err := activePlatformIN(ctx, platformRepo)
	if err != nil {
		return nil, err
	}
	if platformRepo != nil && len(platIN) == 0 {
		return map[uint]persistententity.Market{}, nil
	}
	marketActive := string(cryptoPredictionEnum.MarketStatusActive)
	where := map[string]interface{}{
		"status = ?":       marketActive,
		"map_verified = ?": 1,
		"map_key <> ?":     "",
	}
	if len(platIN) > 0 {
		where["platform IN ?"] = platIN
	}
	_, rows, err := repo.List(ctx, nil, where, "map_key asc, platform asc, id asc", "")
	if err != nil {
		return nil, err
	}
	out := make(map[uint]persistententity.Market, len(rows))
	for i := range rows {
		if rows[i].MapKey == nil || strings.TrimSpace(*rows[i].MapKey) == "" {
			continue
		}
		out[rows[i].ID] = rows[i]
	}
	return out, nil
}

func activePlatformIN(ctx context.Context, repo persistentrepo.PlatformRepo) ([]string, error) {
	if repo == nil {
		return nil, nil
	}
	_, list, err := repo.List(ctx, nil, map[string]interface{}{
		"status = ?": "ACTIVE",
	}, "sort asc", "")
	if err != nil {
		return nil, err
	}
	slugs := make([]string, 0, len(list))
	for i := range list {
		s := strings.ToUpper(strings.TrimSpace(list[i].Slug))
		if s != "" {
			slugs = append(slugs, s)
		}
	}
	return slugs, nil
}

// MapKeyHasTwoPlatforms 该 map_key 下至少两条 market 且平台互不相同（跨所套利前提）。
func MapKeyHasTwoPlatforms(rows []persistententity.Market) bool {
	if len(rows) < 2 {
		return false
	}
	plat := make(map[string]struct{})
	for i := range rows {
		p := strings.ToUpper(strings.TrimSpace(rows[i].Platform))
		if p == "" {
			continue
		}
		plat[p] = struct{}{}
	}
	return len(plat) >= 2
}

// MapKeyDuplicatePlatform 同一 map_key 下是否出现重复 platform（第二条起算重复）。
func MapKeyDuplicatePlatform(rows []persistententity.Market) (dup bool, platform string, dupIDs []uint) {
	seen := map[string]uint{}
	for i := range rows {
		p := strings.ToUpper(strings.TrimSpace(rows[i].Platform))
		if p == "" {
			continue
		}
		if first, ok := seen[p]; ok {
			dupIDs = append(dupIDs, first, rows[i].ID)
			return true, p, dupIDs
		}
		seen[p] = rows[i].ID
	}
	return false, "", nil
}

// MapKeyGroupRunnable fixtures 已限定 ACTIVE 时：≥2 平台且每平台至多 1 条。
func MapKeyGroupRunnable(rows []persistententity.Market) bool {
	if dup, _, _ := MapKeyDuplicatePlatform(rows); dup {
		return false
	}
	return MapKeyHasTwoPlatforms(rows)
}

func watchMapKeysFromConfig(ctx context.Context, cfgRepo persistentrepo.ConfigStrategyRepo) []string {
	if cfgRepo == nil {
		return nil
	}
	row, err := cfgRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": ConfigGroupStrategyEntry,
		"item_key = ?":   ConfigKeyWatchMapKeys,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		return nil
	}
	return ParseWatchMapKeys(row.ItemValue)
}

// FilterOutcomesUnderVerifiedFixtures 保留父场次已核对且 map_key 非空的 ACTIVE 盘口。
func FilterOutcomesUnderVerifiedFixtures(
	outcomes []persistententity.MarketOutcome,
	fixtures map[uint]persistententity.Market,
) []persistententity.MarketOutcome {
	if len(outcomes) == 0 || len(fixtures) == 0 {
		return nil
	}
	out := make([]persistententity.MarketOutcome, 0, len(outcomes))
	for i := range outcomes {
		oc := outcomes[i]
		fx, ok := fixtures[oc.MarketRowID]
		if !ok || fx.MapVerified != 1 {
			continue
		}
		if fx.MapKey == nil || strings.TrimSpace(*fx.MapKey) == "" {
			continue
		}
		out = append(out, oc)
	}
	return out
}

// OutcomeGroupKey 跨平台同一赛果分组键（联赛+场次+归一化赛果，或 map_key|norm）。
// 有 map_key 时用核验场次键，避免 SX 等平台的 canonicalFixture 与 Poly 不一致而无法跨所组队。
func OutcomeGroupKey(m *persistententity.MarketOutcome) string {
	if m == nil {
		return ""
	}
	p := parseMarketRow(m, nil)
	if !p.ParseOK {
		return ""
	}
	if m.MapKey != nil {
		if k := BuildOutcomeGroupKeyFromMapKey(*m.MapKey, p.OutcomeNorm); k != "" {
			return k
		}
	}
	return buildMapKey(p.Competition, p.CanonicalFixture, p.OutcomeNorm)
}

// OutcomeGroupKeyForWatch 与 OutcomeGroupKey 相同；outcome 未写 map_key 时用 watch 场次键兜底。
func OutcomeGroupKeyForWatch(m *persistententity.MarketOutcome, watchMapKey string) string {
	if k := OutcomeGroupKey(m); k != "" {
		return k
	}
	if m == nil {
		return ""
	}
	p := parseMarketRow(m, nil)
	if !p.ParseOK {
		return ""
	}
	return BuildOutcomeGroupKeyFromMapKey(watchMapKey, p.OutcomeNorm)
}