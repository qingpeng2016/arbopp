package predict

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"
)

const (
	predictPageSize = 100
	predictMaxPages = 300
)

// Sync 按联赛 tag 拉 category+markets，写入 market / market_outcome。
func (s *Syncer) Sync(ctx context.Context, subCat cryptoPredictionEnum.SubMarketCategory, cfg SportsConfig, outcomeTypes syncopt.OutcomeTypeSet) (leagueopt.SaveStats, error) {
	if outcomeTypes.Empty() {
		outcomeTypes = syncopt.DefaultOutcomeTypeSet()
	}
	if s.API == nil {
		return leagueopt.SaveStats{}, nil
	}
	leagues := predictLeaguesForSubCategory(subCat, cfg)
	if len(leagues) == 0 {
		logger.InfoZ(ctx, "market_sync-predict-no-leagues",
			zap.String("sub_category", string(subCat)),
			zap.String("hint", "market_category 未解析出联赛或 sync 根 tag；检查 config_json"))
		return leagueopt.SaveStats{}, nil
	}
	platform := string(cryptoPredictionEnum.PlatformPredict)
	category := string(cryptoPredictionEnum.MarketCategorySports)
	subCategory := string(subCat)
	variants := marketVariantsForSub(subCat, cfg)
	// 非足球：仅用 tag 过滤即可。Predict 各运动的 category.marketVariant 不一致
	//（如 SPORTS_NBA、SPORTS_TEAM_MATCH），传 SPORTS_MATCH,SPORTS_PROPS 会导致 API 返回空列表。
	if subCat != cryptoPredictionEnum.SubMarketCategoryFootball {
		variants = nil
	}

	seen := leagueopt.NewSeen()
	leagueopt.Each(ctx, s.LeagueConcurrency, len(leagues), func(i int) {
		item := leagues[i]
		cfgCn, cfgEn := competitionForLeague(item, cfg)
		syncRoot := predictIsSyncRootPass(subCat, item, cfg)
		useConfigLeague := predictItemHasTagIDs(item) && !syncRoot
		if !predictItemHasTagIDs(item) {
			logger.InfoZ(ctx, "market_sync-predict-unscoped",
				zap.String("hint", "predict_sports 未配置 tag_ids，按 OPEN 全站拉取；competition 取自 API tags"))
		} else if syncRoot {
			logger.InfoZ(ctx, "market_sync-predict-sync-all-soccer",
				zap.Strings("tag_ids", item.TagIDs),
				zap.String("hint", "Soccer 根 tag 全量拉取；联赛名按 items.tag_ids 与 Poly/Kalshi 对齐"))
		}
		var leagueOutcomes int
		leagueFixtures := make(map[string]struct{})
		var byCompetition map[string]*predictLeagueDoneStats
		if syncRoot {
			byCompetition = make(map[string]*predictLeagueDoneStats)
		}
		subItems := cfg.BySubCategory[subCat]
		cursor := ""
		var fetchErr error
		for page := 0; page < predictMaxPages; page++ {
			time.Sleep(80 * time.Millisecond)
			resp, err := s.API.GetCategories(ctx, httprepo.PredictListCategoriesParams{
				First:          predictPageSize,
				After:          cursor,
				Status:         "OPEN",
				TagIDs:         item.TagIDs,
				MarketVariants: variants,
			})
			if err != nil {
				fetchErr = fmt.Errorf("get categories tag=%v page=%d: %w", item.TagIDs, page, err)
				break
			}
			if resp == nil || len(resp.Data) == 0 {
				break
			}

			var foBatch batch.FixtureOutcome
			fixturesThisPage := make(map[string]*persistententity.Market)
			for _, cat := range resp.Data {
				if !predictCategoryIsSportsFixture(cat) {
					continue
				}
				fid := predictMainFixtureID(cat.Slug)
				if fid == "" {
					continue
				}
				compCn, compEn := resolveCompetition(cat, cfgCn, cfgEn, useConfigLeague, cfg)
				fixtureTitle := strings.TrimSpace(cat.Title)
				for _, m := range cat.Markets {
					if !predictMarketTradable(m) {
						continue
					}
					okType, outcomeKind := predictMarketForSync(m, outcomeTypes)
					if !okType {
						continue
					}
					oc := predictOutcomeFromMarket(platform, category, subCategory, fid, fixtureTitle, compCn, cat, m, outcomeKind)
					if oc == nil {
						continue
					}
					if fixturesThisPage[fid] == nil {
						fixturesThisPage[fid] = predictFixtureFromCategory(platform, category, subCategory, cat, compCn, compEn, fid)
					} else if predictIsMainMatchCategory(cat.Slug) {
						fixturesThisPage[fid] = predictFixtureFromCategory(platform, category, subCategory, cat, compCn, compEn, fid)
					}
					seen.Keep(fid, oc.PlatformMarketOutcomeID)
					leagueFixtures[fid] = struct{}{}
					if syncRoot && strings.TrimSpace(compCn) != "" {
						st := byCompetition[compCn]
						if st == nil {
							st = &predictLeagueDoneStats{}
							byCompetition[compCn] = st
						}
						st.addFixture(fid, 1)
					}
					foBatch.AddOutcome(oc)
				}
			}
			for _, fx := range fixturesThisPage {
				foBatch.AddFixture(fx)
			}

			if len(foBatch.Outcomes) > 0 {
				if err := s.W.PersistFixtureOutcomes(ctx, &foBatch); err != nil {
					notification.SendErrorLog(ctx, "market_sync-predict-BatchUpsertFailed",
						zap.Strings("tag_ids", item.TagIDs),
						zap.Int("count", len(foBatch.Outcomes)), zap.Error(err))
					seen.Fail(err)
					return
				}
				n := len(foBatch.Outcomes)
				totalSaved := seen.AddSaved(n)
				leagueOutcomes += n
				logger.InfoZ(ctx, "market_sync-predict-BatchUpserted",
					zap.Strings("tag_ids", item.TagIDs),
					zap.String("sub_category", subCategory),
					zap.Int("batch_count", len(foBatch.Outcomes)),
					zap.Int("total_saved", totalSaved))
			}

			cursor = strings.TrimSpace(resp.Cursor)
			if cursor == "" || len(resp.Data) < predictPageSize {
				break
			}
		}
		if syncRoot {
			notifyPredictLeaguesAfterSyncRoot(ctx, subItems, byCompetition)
		} else {
			notifyPredictLeagueDone(ctx, item, len(leagueFixtures), leagueOutcomes)
		}
		if fetchErr != nil {
			seen.Fail(fetchErr)
		}
	})

	stats := seen.SaveStats()
	if err := seen.Err(); err != nil {
		return stats, err
	}
	if fxClosed, ocClosed, err := s.W.CloseStaleAfterSync(ctx, platform, category, subCategory, seen.Fixtures(), seen.Outcomes()); err != nil {
		return stats, err
	} else {
		logger.InfoZ(ctx, "market_sync-predict-stale-done",
			zap.String("sub_category", subCategory),
			zap.Int("fixtures", stats.Fixtures),
			zap.Int("outcomes", stats.Outcomes),
			zap.Int("fixtures_closed", fxClosed),
			zap.Int("outcomes_closed", ocClosed),
			zap.Int("outcome_upserts", seen.Saved()))
	}
	return stats, nil
}