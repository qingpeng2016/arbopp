package leaguecfg

import (
	"context"
	"encoding/json"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/kalshi"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/polymarket"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/predict"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/sxbet"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

func listEnabled(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) ([]persistententity.ConfigMarketCategory, bool) {
	if repo == nil {
		return nil, false
	}
	list, err := repo.ListCategories(ctx, nil, true)
	if err != nil || len(list) == 0 {
		return nil, false
	}
	return list, true
}

func PolymarketSportsConfig(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) (polymarket.SportsConfig, bool) {
	cats, ok := listEnabled(ctx, repo)
	if !ok {
		return polymarket.SportsConfig{}, false
	}
	raw := groupsArrayForPlatformRows(cats, appcommon.PlatformPolymarket)
	if raw == "" {
		return polymarket.SportsConfig{}, false
	}
	cfg := polymarket.LoadSportsConfig(raw, "json")
	return cfg, cfg.Active && len(cfg.BySubCategory) > 0
}

func KalshiSeriesConfig(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) (kalshi.SeriesConfig, bool) {
	cats, ok := listEnabled(ctx, repo)
	if !ok {
		return kalshi.SeriesConfig{}, false
	}
	raw := groupsArrayForPlatformRows(cats, appcommon.PlatformKalshi)
	if raw == "" {
		return kalshi.SeriesConfig{}, false
	}
	cfg := kalshi.LoadSeriesConfig(raw, "json")
	return cfg, cfg.Active && len(cfg.BySubCategory) > 0
}

func SxbetLeaguesConfig(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) (sxbet.LeaguesConfig, bool) {
	cats, ok := listEnabled(ctx, repo)
	if !ok {
		return sxbet.LeaguesConfig{}, false
	}
	raw := groupsArrayForPlatformRows(cats, appcommon.PlatformSxbet)
	if raw == "" {
		return sxbet.LeaguesConfig{}, false
	}
	cfg := sxbet.LoadLeaguesConfig(raw, "json")
	return cfg, cfg.Active && len(cfg.BySubCategory) > 0
}

func PredictSportsConfig(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) (predict.SportsConfig, bool) {
	cats, ok := listEnabled(ctx, repo)
	if !ok {
		return predict.SportsConfig{}, false
	}
	raw := groupsArrayForPlatformRows(cats, appcommon.PlatformPredict)
	if raw == "" {
		return predict.SportsConfig{}, false
	}
	cfg := predict.LoadSportsConfig(raw, "json")
	return cfg, cfg.Active && len(cfg.BySubCategory) > 0
}

func SyncOutcomeTypes(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) (cryptoPredictionEnum.SyncOutcomeTypeConfig, bool) {
	cats, ok := listEnabled(ctx, repo)
	if !ok {
		return cryptoPredictionEnum.SyncOutcomeTypeConfig{}, false
	}
	seen := make(map[string]struct{})
	var groups []cryptoPredictionEnum.SyncOutcomeTypeGroup
	for i := range cats {
		c := &cats[i]
		sub := strings.TrimSpace(c.SubCategory)
		if sub == "" {
			continue
		}
		if _, dup := seen[sub]; dup {
			continue
		}
		items := parseOutcomeTypeStrings(c.OutcomeTypesJSON)
		if len(items) == 0 {
			continue
		}
		seen[sub] = struct{}{}
		groups = append(groups, cryptoPredictionEnum.SyncOutcomeTypeGroup{
			SportCN: c.SportCN, SubCategory: sub, Items: items,
		})
	}
	if len(groups) == 0 {
		return cryptoPredictionEnum.SyncOutcomeTypeConfig{}, false
	}
	b, err := json.Marshal(groups)
	if err != nil {
		return cryptoPredictionEnum.SyncOutcomeTypeConfig{}, false
	}
	return cryptoPredictionEnum.ParseSyncOutcomeTypes(string(b), "json"), true
}

func parseOutcomeTypeStrings(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
	return list
}

func LeagueFetchConfig(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) (leagueopt.Config, bool) {
	cats, ok := listEnabled(ctx, repo)
	if !ok {
		return leagueopt.Config{}, false
	}
	return mergeLeagueFetchFromRows(cats)
}