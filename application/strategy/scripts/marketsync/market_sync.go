package marketsync

import (
	"context"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/kalshi"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leaguecfg"
	leagueopt "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/predict"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/polymarket"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"go.uber.org/zap"
)

// MarketSync 拉取预测平台活跃场次与盘口，写入 market + market_outcome。
type MarketSync struct {
	polymarket         httprepo.PolymarketRepo
	kalshi             httprepo.KalshiRepo
	predict            httprepo.PredictRepo
	platformRepo       persistentrepo.PlatformRepo
	marketRepo         persistentrepo.MarketRepo
	outcomeRepo        persistentrepo.MarketOutcomeRepo
	configStrategyRepo persistentrepo.ConfigStrategyRepo
	marketCategoryRepo persistentrepo.ConfigMarketCategoryRepo
	poly               *polymarket.Syncer
	kal                *kalshi.Syncer
	prd                *predict.Syncer
}

func NewMarketSync(
	polyRepo httprepo.PolymarketRepo,
	kalRepo httprepo.KalshiRepo,
	predictRepo httprepo.PredictRepo,
	platformRepo persistentrepo.PlatformRepo,
	marketRepo persistentrepo.MarketRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
	configStrategyRepo persistentrepo.ConfigStrategyRepo,
	marketCategoryRepo persistentrepo.ConfigMarketCategoryRepo,
) *MarketSync {
	j := &MarketSync{
		polymarket:         polyRepo,
		kalshi:             kalRepo,
		predict:            predictRepo,
		platformRepo:       platformRepo,
		marketRepo:         marketRepo,
		outcomeRepo:        outcomeRepo,
		configStrategyRepo: configStrategyRepo,
		marketCategoryRepo: marketCategoryRepo,
	}
	j.poly = polymarket.NewSyncer(polyRepo, j)
	j.kal = kalshi.NewSyncer(kalRepo, j)
	j.prd = predict.NewSyncer(predictRepo, j)
	return j
}

// Run：仅 platform.status=ACTIVE 的平台参与；平台间各起一个协程；同平台内 market_category 按 sort_no 升序串行；
// 子类内联赛并发=该行的 concurrency。
func (j *MarketSync) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "market_sync-Panic",
				zap.Any("panic", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()

	platforms, err := j.resolveSyncPlatforms(ctx)
	if err != nil {
		notification.SendErrorLog(ctx, "market_sync-ResolvePlatformsFailed", zap.Error(err))
		return
	}
	if len(platforms) == 0 {
		notification.SendErrorLog(ctx, "market_sync-no-platforms-to-sync")
		return
	}

	polyCfg := j.resolvePolymarketSportsConfig(ctx)
	kalshiCfg := j.resolveKalshiGameSeriesConfig(ctx)
	predictCfg := j.resolvePredictSportsConfig(ctx)
	outcomeTypes := j.resolveSyncOutcomeTypes(ctx)
	leagueFetch := j.resolveLeagueFetchConfig(ctx)
	catalogRows, catalogOK := leaguecfg.ListEnabled(ctx, j.marketCategoryRepo)
	if catalogOK {
		catalogRows = leaguecfg.FilterByActivePlatforms(catalogRows, platforms)
		catalogOK = len(catalogRows) > 0
	}

	polyKey := string(cryptoPredictionEnum.PlatformPolymarket)
	kalKey := string(cryptoPredictionEnum.PlatformKalshi)
	predictKey := string(cryptoPredictionEnum.PlatformPredict)
	_, syncPoly := platforms[polyKey]
	_, syncKal := platforms[kalKey]
	_, syncPredict := platforms[predictKey]

	var wg sync.WaitGroup
	var polySaved, kalshiSaved, predictSaved leagueopt.SaveStats
	if syncPoly {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer recoverSyncPlatform(ctx, "polymarket", "")
			plan := platformCategoryPlan(catalogOK, catalogRows, polyKey, polyCfg.BySubCategory, leagueFetch.Polymarket.Concurrency())
			logger.InfoZ(ctx, "market_sync-platform-start",
				zap.String("platform", polyKey),
				zap.Strings("sub_categories", categoryRunSubStrings(plan)))
			start := time.Now()
			polySaved = j.syncPlatformCategoryRuns(ctx, polyKey, catalogRows, plan, func(subCat cryptoPredictionEnum.SubMarketCategory, concurrency int) (leagueopt.SaveStats, error) {
				j.poly.LeagueConcurrency = concurrency
				return j.poly.Sync(ctx, subCat, polyCfg, outcomeTypesForSub(outcomeTypes, subCat))
			})
			notifyPlatformDone(ctx, "market_sync Polymarket 完成", polyKey, polySaved, categoryRunSubs(plan), time.Since(start))
		}()
	}
	if syncKal {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer recoverSyncPlatform(ctx, "kalshi", "")
			plan := platformCategoryPlan(catalogOK, catalogRows, kalKey, kalshiCfg.BySubCategory, leagueFetch.Kalshi.Concurrency())
			logger.InfoZ(ctx, "market_sync-platform-start",
				zap.String("platform", kalKey),
				zap.Strings("sub_categories", categoryRunSubStrings(plan)))
			start := time.Now()
			kalshiSaved = j.syncPlatformCategoryRuns(ctx, kalKey, catalogRows, plan, func(subCat cryptoPredictionEnum.SubMarketCategory, concurrency int) (leagueopt.SaveStats, error) {
				j.kal.LeagueConcurrency = concurrency
				return j.kal.Sync(ctx, subCat, kalshiCfg, outcomeTypesForSub(outcomeTypes, subCat))
			})
			notifyPlatformDone(ctx, "market_sync Kalshi 完成", kalKey, kalshiSaved, categoryRunSubs(plan), time.Since(start))
		}()
	}
	if syncPredict {
		if j.predict == nil {
			notification.SendErrorLog(ctx, "market_sync-predict-skipped", zap.String("reason", "no Predict API client (api_key)"))
		} else if len(predictCfg.BySubCategory) == 0 {
			notification.SendErrorLog(ctx, "market_sync-predict-skipped", zap.String("reason", "predict_sports config empty or disabled"))
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer recoverSyncPlatform(ctx, "predict", "")
				plan := platformCategoryPlan(catalogOK, catalogRows, predictKey, predictCfg.BySubCategory, leagueFetch.Predict.Concurrency())
				logger.InfoZ(ctx, "market_sync-platform-start",
					zap.String("platform", predictKey),
					zap.Strings("sub_categories", categoryRunSubStrings(plan)))
				start := time.Now()
				predictSaved = j.syncPlatformCategoryRuns(ctx, predictKey, catalogRows, plan, func(subCat cryptoPredictionEnum.SubMarketCategory, concurrency int) (leagueopt.SaveStats, error) {
					j.prd.LeagueConcurrency = concurrency
					return j.prd.Sync(ctx, subCat, predictCfg, outcomeTypesForSub(outcomeTypes, subCat))
				})
				notifyPlatformDone(ctx, "market_sync Predict 完成", predictKey, predictSaved, categoryRunSubs(plan), time.Since(start))
			}()
		}
	}
	wg.Wait()

	totalSaved := polySaved.Add(kalshiSaved).Add(predictSaved)
	logger.InfoZ(ctx, "market_sync-Completed",
		zap.Int("fixtures", totalSaved.Fixtures),
		zap.Int("outcomes", totalSaved.Outcomes),
		zap.Int("polymarket_fixtures", polySaved.Fixtures),
		zap.Int("polymarket_outcomes", polySaved.Outcomes),
		zap.Int("kalshi_fixtures", kalshiSaved.Fixtures),
		zap.Int("kalshi_outcomes", kalshiSaved.Outcomes),
		zap.Int("predict_fixtures", predictSaved.Fixtures),
		zap.Int("predict_outcomes", predictSaved.Outcomes))
}

func subCategoryStrings(cats []cryptoPredictionEnum.SubMarketCategory) []string {
	out := make([]string, len(cats))
	for i, c := range cats {
		out[i] = string(c)
	}
	return out
}

func subCategoriesFromMap[V any](m map[cryptoPredictionEnum.SubMarketCategory]V) []cryptoPredictionEnum.SubMarketCategory {
	if len(m) == 0 {
		return cryptoPredictionEnum.AllSubMarketCategories(cryptoPredictionEnum.MarketCategorySports)
	}
	out := make([]cryptoPredictionEnum.SubMarketCategory, 0, len(m))
	for sub := range m {
		if sub == "" {
			continue
		}
		out = append(out, sub)
	}
	if len(out) == 0 {
		return cryptoPredictionEnum.AllSubMarketCategories(cryptoPredictionEnum.MarketCategorySports)
	}
	return out
}

// platformCategoryPlan catalog 优先：同平台按 sort_no 顺序；无 catalog 时退化为 config 子类列表 + 统一 concurrency。
func platformCategoryPlan[V any](
	catalogOK bool,
	catalogRows []persistententity.ConfigMarketCategory,
	platform string,
	bySub map[cryptoPredictionEnum.SubMarketCategory]V,
	defaultConcurrency int,
) []leaguecfg.CategoryRunPlan {
	if catalogOK {
		if plan := leaguecfg.PlatformCategoryRuns(catalogRows, platform); len(plan) > 0 {
			return plan
		}
	}
	subs := subCategoriesFromMap(bySub)
	def := leagueopt.Platform{LeagueConcurrency: defaultConcurrency}.Concurrency()
	plan := make([]leaguecfg.CategoryRunPlan, 0, len(subs))
	for i, sub := range subs {
		plan = append(plan, leaguecfg.CategoryRunPlan{
			SubCategory: sub,
			Concurrency: def,
			SortNo:      i + 1,
		})
	}
	return plan
}

func categoryRunSubs(plan []leaguecfg.CategoryRunPlan) []cryptoPredictionEnum.SubMarketCategory {
	out := make([]cryptoPredictionEnum.SubMarketCategory, len(plan))
	for i, p := range plan {
		out[i] = p.SubCategory
	}
	return out
}

func categoryRunSubStrings(plan []leaguecfg.CategoryRunPlan) []string {
	return subCategoryStrings(categoryRunSubs(plan))
}

// syncPlatformCategoryRuns 单平台内子类按 sort_no 串行；每个子类内部联赛并发由 concurrency 控制（在 syncer 上设置）。
func (j *MarketSync) syncPlatformCategoryRuns(
	ctx context.Context,
	platform string,
	catalogRows []persistententity.ConfigMarketCategory,
	plan []leaguecfg.CategoryRunPlan,
	syncOne func(cryptoPredictionEnum.SubMarketCategory, int) (leagueopt.SaveStats, error),
) leagueopt.SaveStats {
	var saved leagueopt.SaveStats
	planTotal := len(plan)
	for i, step := range plan {
		subCat := step.SubCategory
		func() {
			defer recoverSyncPlatform(ctx, platform, subCat)
			logger.InfoZ(ctx, "market_sync-category-start",
				zap.String("platform", platform),
				zap.String("sub_category", string(subCat)),
				zap.Int("sort_no", step.SortNo),
				zap.Int("plan_index", i+1),
				zap.Int("plan_total", planTotal),
				zap.Int("concurrency", step.Concurrency))
			stepStart := time.Now()
			stats, err := syncOne(subCat, step.Concurrency)
			notifyCategoryDone(ctx, platform, step, catalogRows, stats, time.Since(stepStart), err)
			if err != nil {
				notification.SendErrorLog(ctx, "market_sync-"+platform+"-failed",
					zap.String("sub_category", string(subCat)), zap.Error(err))
				logger.InfoZ(ctx, "market_sync-category-step-end",
					zap.String("platform", platform),
					zap.String("sub_category", string(subCat)),
					zap.Int("plan_index", i+1),
					zap.Int("plan_total", planTotal),
					zap.Bool("ok", false))
				return
			}
			saved = saved.Add(stats)
			logger.InfoZ(ctx, "market_sync-category-step-end",
				zap.String("platform", platform),
				zap.String("sub_category", string(subCat)),
				zap.Int("plan_index", i+1),
				zap.Int("plan_total", planTotal),
				zap.Int("fixtures", stats.Fixtures),
				zap.Int("outcomes", stats.Outcomes),
				zap.Bool("ok", true))
		}()
	}
	return saved
}

func marketCategorySportCN(rows []persistententity.ConfigMarketCategory, platform, sub string) string {
	plat := strings.ToUpper(strings.TrimSpace(platform))
	sub = strings.TrimSpace(sub)
	for i := range rows {
		c := &rows[i]
		if strings.ToUpper(strings.TrimSpace(c.Platform)) != plat {
			continue
		}
		if strings.TrimSpace(c.SubCategory) != sub {
			continue
		}
		if cn := strings.TrimSpace(c.SportCN); cn != "" {
			return cn
		}
	}
	return sub
}

// notifyCategoryDone 每个 market_category 子类跑完后单独一条 Lark 摘要（不再把整平台子类拼在一条里）。
func notifyCategoryDone(
	ctx context.Context,
	platform string,
	step leaguecfg.CategoryRunPlan,
	catalogRows []persistententity.ConfigMarketCategory,
	stats leagueopt.SaveStats,
	elapsed time.Duration,
	syncErr error,
) {
	sub := string(step.SubCategory)
	sportCN := marketCategorySportCN(catalogRows, platform, sub)
	title := "market_sync 分类完成"
	fields := []notification.FieldPair{
		{Key: "平台", Value: strings.ToUpper(strings.TrimSpace(platform))},
		{Key: "分类", Value: sportCN},
		{Key: "子类", Value: sub},
		{Key: "场次数", Value: strconv.Itoa(stats.Fixtures)},
		{Key: "盘口数", Value: strconv.Itoa(stats.Outcomes)},
		{Key: "耗时", Value: elapsed.Round(time.Second).String()},
		{Key: "sort_no", Value: strconv.Itoa(step.SortNo)},
	}
	if syncErr != nil {
		fields = append(fields, notification.FieldPair{Key: "状态", Value: "失败"}, notification.FieldPair{Key: "错误", Value: syncErr.Error()})
	} else if stats.Empty() {
		fields = append(fields, notification.FieldPair{Key: "状态", Value: "成功"},
			notification.FieldPair{Key: "说明", Value: "本次无新开仓写入（检查 tag_ids 是否主网、市场 OPEN、outcome 过滤）"})
	} else {
		fields = append(fields, notification.FieldPair{Key: "状态", Value: "成功"})
	}
	logger.InfoZ(ctx, "market_sync-category-done",
		zap.String("platform", platform),
		zap.String("sub_category", sub),
		zap.String("sport_cn", sportCN),
		zap.Int("fixtures", stats.Fixtures),
		zap.Int("outcomes", stats.Outcomes),
		zap.String("elapsed", elapsed.Round(time.Second).String()),
		zap.Error(syncErr))
	if err := notification.GetGlobalNotificationManager().SendInfoAlert(ctx, title, fields); err != nil {
		logger.ErrorZ(ctx, "market_sync-category-notify-failed",
			zap.String("platform", platform),
			zap.String("sub_category", sub),
			zap.Error(err))
	}
}

func notifyPlatformDone(ctx context.Context, title, platform string, saved leagueopt.SaveStats, subCats []cryptoPredictionEnum.SubMarketCategory, elapsed time.Duration) {
	logger.InfoZ(ctx, "market_sync-platform-done",
		zap.String("title", title),
		zap.String("platform", platform),
		zap.Int("fixtures", saved.Fixtures),
		zap.Int("outcomes", saved.Outcomes),
		zap.Strings("sub_categories", subCategoryStrings(subCats)),
		zap.String("elapsed", elapsed.Round(time.Second).String()))
}

func recoverSyncPlatform(ctx context.Context, platform string, subCat cryptoPredictionEnum.SubMarketCategory) {
	r := recover()
	if r == nil {
		return
	}
	notification.SendErrorLog(ctx, "market_sync-"+platform+"-panic",
		zap.String("sub_category", string(subCat)),
		zap.Any("panic", r),
		zap.String("stack", string(debug.Stack())))
}
