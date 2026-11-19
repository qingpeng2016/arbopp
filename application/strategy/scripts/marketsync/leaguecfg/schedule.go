package leaguecfg

import (
	"context"
	"sort"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// CategoryRunPlan 单平台下一次 market_sync 要按序执行的一个子类。
type CategoryRunPlan struct {
	SubCategory cryptoPredictionEnum.SubMarketCategory
	Concurrency int
	SortNo      int
}

// ListEnabled 启用中的 market_category 行。
func ListEnabled(ctx context.Context, repo persistentrepo.ConfigMarketCategoryRepo) ([]persistententity.ConfigMarketCategory, bool) {
	return listEnabled(ctx, repo)
}

// FilterByActivePlatforms 仅保留 platform 表中 status=ACTIVE 的平台行（slug 与 market_category.platform 对齐）。
func FilterByActivePlatforms(cats []persistententity.ConfigMarketCategory, active map[string]struct{}) []persistententity.ConfigMarketCategory {
	if len(cats) == 0 || len(active) == 0 {
		return nil
	}
	out := make([]persistententity.ConfigMarketCategory, 0, len(cats))
	for i := range cats {
		if _, ok := active[platformKeyNorm(cats[i].Platform)]; ok {
			out = append(out, cats[i])
		}
	}
	return out
}

// PlatformCategoryRuns 同一平台内按 sort_no 升序；并发数取行的 concurrency（1–8）。
// 调用方应保证 platform 已在 ACTIVE 集合内（见 market_sync.resolveSyncPlatforms）。
func PlatformCategoryRuns(cats []persistententity.ConfigMarketCategory, platform string) []CategoryRunPlan {
	plat := platformKeyNorm(platform)
	var rows []persistententity.ConfigMarketCategory
	for i := range cats {
		if platformKeyNorm(cats[i].Platform) != plat {
			continue
		}
		sub := strings.TrimSpace(cats[i].SubCategory)
		if sub == "" {
			continue
		}
		rows = append(rows, cats[i])
	}
	if len(rows) == 0 {
		return nil
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SortNo != rows[j].SortNo {
			return rows[i].SortNo < rows[j].SortNo
		}
		return rows[i].SubCategory < rows[j].SubCategory
	})
	out := make([]CategoryRunPlan, 0, len(rows))
	for i := range rows {
		conc := rows[i].Concurrency
		if conc <= 0 {
			_, conc = sportGroupFromConfig(rows[i].ConfigJSON)
		}
		conc = leagueopt.Platform{LeagueConcurrency: conc}.Concurrency()
		out = append(out, CategoryRunPlan{
			SubCategory: cryptoPredictionEnum.SubMarketCategory(strings.TrimSpace(rows[i].SubCategory)),
			Concurrency: conc,
			SortNo:      rows[i].SortNo,
		})
	}
	return out
}