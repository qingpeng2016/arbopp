package marketmap

import (
	"context"
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// LoadMarketsByMapKey 按单个 map_key 加载场次（历史任务用，不限 ACTIVE）。
func LoadMarketsByMapKey(ctx context.Context, repo persistentrepo.MarketRepo, platformRepo persistentrepo.PlatformRepo, mapKey string) (map[uint]persistententity.Market, error) {
	mapKey = strings.TrimSpace(mapKey)
	if repo == nil || mapKey == "" {
		return map[uint]persistententity.Market{}, nil
	}
	platIN, err := activePlatformIN(ctx, platformRepo)
	if err != nil {
		return nil, err
	}
	where := map[string]interface{}{
		"map_key = ?": mapKey,
	}
	if len(platIN) > 0 {
		where["platform IN ?"] = platIN
	}
	_, rows, err := repo.List(ctx, nil, where, "platform asc, id asc", "")
	if err != nil {
		return nil, err
	}
	out := make(map[uint]persistententity.Market, len(rows))
	for i := range rows {
		out[rows[i].ID] = rows[i]
	}
	return out, nil
}

// LoadMarketsByFixtureSlug 按场次 slug 加载（PMData 任务 map_key 可直接写 slug，不必 v1|联赛|slug）。
func LoadMarketsByFixtureSlug(ctx context.Context, repo persistentrepo.MarketRepo, platformRepo persistentrepo.PlatformRepo, slug string) (map[uint]persistententity.Market, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if repo == nil || slug == "" {
		return map[uint]persistententity.Market{}, nil
	}
	platIN, err := activePlatformIN(ctx, platformRepo)
	if err != nil {
		return nil, err
	}
	where := map[string]interface{}{
		"slug = ?": slug,
	}
	if len(platIN) > 0 {
		where["platform IN ?"] = platIN
	}
	_, rows, err := repo.List(ctx, nil, where, "platform asc, id asc", "")
	if err != nil {
		return nil, err
	}
	out := make(map[uint]persistententity.Market, len(rows))
	for i := range rows {
		out[rows[i].ID] = rows[i]
	}
	return out, nil
}