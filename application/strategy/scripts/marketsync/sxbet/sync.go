package sxbet

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
	sxbetPageSize = 100
	sxbetMaxPages = 500
)

// Sync 按 league_id 分页拉 active markets，写入 market / market_outcome。
func (s *Syncer) Sync(ctx context.Context, subCat cryptoPredictionEnum.SubMarketCategory, cfg LeaguesConfig, outcomeTypes syncopt.OutcomeTypeSet) (leagueopt.SaveStats, error) {
	if outcomeTypes.Empty() {
		outcomeTypes = syncopt.DefaultOutcomeTypeSet()
	}
	if s.API == nil {
		return leagueopt.SaveStats{}, nil
	}
	leagues := sxbetLeaguesForSubCategory(subCat, cfg)
	if len(leagues) == 0 {
		logger.InfoZ(ctx, "market_sync-sxbet-no-leagues",
			zap.String("sub_category", string(subCat)),
			zap.String("hint", "market_category 未配置 league_id；检查 SXBET config_json"))
		return leagueopt.SaveStats{}, nil
	}

	platform := string(cryptoPredictionEnum.PlatformSxbet)
	category := string(cryptoPredictionEnum.MarketCategorySports)
	subCategory := string(subCat)

	seen := leagueopt.NewSeen()
	leagueopt.Each(ctx, s.LeagueConcurrency, len(leagues), func(i int) {
		leagueID := leagues[i]
		meta := cfg.LeagueByID[leagueID]
		compName := cfg.CompetitionByLeague[leagueID]
		compEn := meta.NameEn
		if compEn == "" {
			compEn = meta.NameDesc
		}
		var leagueOutcomes int
		leagueFixtures := make(map[string]struct{})
		paginationKey := ""
		var fetchErr error
		for page := 0; page < sxbetMaxPages; page++ {
			time.Sleep(60 * time.Millisecond)
			resp, err := s.API.GetActiveMarkets(ctx, httprepo.SxbetListActiveMarketsParams{
				LeagueID:      leagueID,
				PageSize:      sxbetPageSize,
				PaginationKey: paginationKey,
			})
			if err != nil {
				fetchErr = fmt.Errorf("get markets league=%d page=%d: %w", leagueID, page, err)
				break
			}
			if resp == nil {
				break
			}
			if len(resp.Data.Markets) == 0 && strings.TrimSpace(resp.Data.NextKey) == "" {
				break
			}

			var foBatch batch.FixtureOutcome
			fixturesThisPage := make(map[string]*persistententity.Market)
			for _, m := range resp.Data.Markets {
				okType, outcomeKind := sxbetMarketForSync(m, outcomeTypes)
				if !okType {
					continue
				}
				fid := strings.TrimSpace(m.SportXeventID)
				if fid == "" {
					continue
				}
				fixtureTitle := sxbetFixtureTitle(m)
				if compName == "" {
					compName = strings.TrimSpace(m.LeagueLabel)
				}
				oc := sxbetOutcomeFromMarket(platform, category, subCategory, fid, fixtureTitle, compName, m, outcomeKind)
				if oc == nil {
					continue
				}
				if fixturesThisPage[fid] == nil {
					fixturesThisPage[fid] = sxbetFixtureFromMarket(platform, category, subCategory, m, compName, compEn)
				}
				seen.Keep(fid, oc.PlatformMarketOutcomeID)
				leagueFixtures[fid] = struct{}{}
				foBatch.AddOutcome(oc)
			}
			for _, fx := range fixturesThisPage {
				foBatch.AddFixture(fx)
			}

			if len(foBatch.Outcomes) > 0 {
				if err := s.W.PersistFixtureOutcomes(ctx, &foBatch); err != nil {
					notification.SendErrorLog(ctx, "market_sync-sxbet-BatchUpsertFailed",
						zap.Int("league_id", leagueID),
						zap.Int("count", len(foBatch.Outcomes)), zap.Error(err))
					seen.Fail(err)
					return
				}
				n := len(foBatch.Outcomes)
				totalSaved := seen.AddSaved(n)
				leagueOutcomes += n
				logger.InfoZ(ctx, "market_sync-sxbet-BatchUpserted",
					zap.Int("league_id", leagueID),
					zap.String("sub_category", subCategory),
					zap.Int("batch_count", n),
					zap.Int("total_saved", totalSaved))
			}

			paginationKey = strings.TrimSpace(resp.Data.NextKey)
			if paginationKey == "" {
				break
			}
		}
		notifySxbetLeagueDone(ctx, leagueID, meta, len(leagueFixtures), leagueOutcomes)
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
		logger.InfoZ(ctx, "market_sync-sxbet-stale-done",
			zap.String("sub_category", subCategory),
			zap.Int("fixtures", stats.Fixtures),
			zap.Int("outcomes", stats.Outcomes),
			zap.Int("fixtures_closed", fxClosed),
			zap.Int("outcomes_closed", ocClosed),
			zap.Int("outcome_upserts", seen.Saved()))
	}
	return stats, nil
}