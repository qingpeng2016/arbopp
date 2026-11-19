package polymarket

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"
)

const (
	polymarketMarketsKeysetLimit = 100
	// 单 tag × 单 sports_market_types 最多翻页次数（如 cfb tag 体量大）
	polymarketMarketsKeysetPages = 120
)

// syncSoccerGamesEvents 经 Gamma GET /markets/keyset 按 sports_market_types 拉胜平负、让分、大小球。
// 英超 tag 306 的 /events/keyset 第一页就是比赛；欧冠 tag 100977 的 events 流前几页是赛季期货且单页数 MB，同一套 events 分页会超时得到 0。markets/keyset 按盘口类型过滤后两边第一页都是单场。
func (s *Syncer) syncSoccerGamesEvents(ctx context.Context, subCat cryptoPredictionEnum.SubMarketCategory, sportsCfg SportsConfig, outcomeTypes syncopt.OutcomeTypeSet) (leagueopt.SaveStats, error) {
	if outcomeTypes.Empty() {
		outcomeTypes = syncopt.DefaultOutcomeTypeSet()
	}
	sports, err := s.API.GetSportsMetadata(ctx)
	if err != nil {
		return leagueopt.SaveStats{}, fmt.Errorf("get sports metadata: %w", err)
	}
	metaBySport := make(map[string]httpentity.SportsMetadata, len(sports))
	for _, s := range sports {
		sid := strings.ToLower(strings.TrimSpace(s.Sport))
		if sid == "" {
			continue
		}
		metaBySport[sid] = s
	}
	items := sportsCfg.BySubCategory[subCat]
	leagueFetches := resolvePolymarketSoccerLeagueFetches(ctx, items, metaBySport)
	if len(leagueFetches) == 0 {
		logger.InfoZ(ctx, "market_sync-polymarket-soccer-no-leagues")
		return leagueopt.SaveStats{}, nil
	}

	compBySport := mergePolymarketCompetitionNames(sportsCfg.CompetitionBySportID, metaBySport)
	nameBySport := sportNamesByID(sportsCfg, subCat)
	nameDescBySport := sportNameDescByID(sportsCfg, subCat)
	platform := string(cryptoPredictionEnum.PlatformPolymarket)
	category := string(cryptoPredictionEnum.MarketCategorySports)
	subCategory := string(subCat)
	closed := false
	closedPtr := &closed

	seen := leagueopt.NewSeen()
	marketTypes := soccerSportsMarketTypes(outcomeTypes)

	leagueopt.Each(ctx, s.LeagueConcurrency, len(leagueFetches), func(i int) {
		lf := leagueFetches[i]
		var leagueOutcomes int
		leagueFixtures := make(map[string]struct{})
		for _, src := range lf.eventSources() {
			for _, marketType := range marketTypes {
				asc := false
				afterCursor := ""
				for pageIdx := 0; pageIdx < polymarketMarketsKeysetPages; pageIdx++ {
					time.Sleep(80 * time.Millisecond)
					page, err := s.API.ListMarketsKeyset(ctx, httpentity.ListMarketsKeysetRequest{
						Limit:             polymarketMarketsKeysetLimit,
						Closed:            closedPtr,
						TagID:             src.tagID,
						SportsMarketTypes: []string{marketType},
						AfterCursor:       afterCursor,
						Order:             "id",
						Ascending:         &asc,
					})
					if err != nil {
						notification.SendErrorLog(ctx, "market_sync-polymarket-markets-stopped",
							zap.String("sport_id", lf.sportID), zap.Int("tag_id", src.tagID),
							zap.Int("page", pageIdx), zap.Error(err))
						break
					}
					if len(page.Markets) == 0 {
						break
					}
					var foBatch batch.FixtureOutcome
					fixturesThisPage := make(map[string]*persistententity.Market)
					for i := range page.Markets {
						m := page.Markets[i]
						if m.Closed {
							continue
						}
						if lf.sportID != "" && !polymarketMarketInLeagueScope(&m, lf.sportID) {
							continue
						}
						ok, outcomeKind := polymarketMarketForSync(&m, outcomeTypes, cryptoPredictionEnum.SubMarketCategoryFootball)
						if !ok {
							continue
						}
						dedupeKey := m.ConditionId
						if dedupeKey == "" {
							dedupeKey = m.ID
						}
						evStart, evEnd, _, _ := extractPolymarketTimes(&m)
						if common.SkipEventEndedUTC(evStart, evEnd) {
							continue
						}
						yesTok, noTok := parsePolymarketClobTokenIds(m.ClobTokenIds)
						if yesTok == "" || noTok == "" {
							continue
						}
						if !seen.Claim(dedupeKey) {
							continue
						}

						comp, sid := polymarketCompetitionName(&m, compBySport)
						if comp == "" && lf.sportID != "" {
							sid = strings.ToLower(strings.TrimSpace(lf.sportID))
							comp = compBySport[sid]
						}
						compPtr, compEnPtr := polymarketCompetitionPtrs(comp, sportsCfg.CompetitionEnBySportID[sid])
						fid := polymarketFixtureIDFromFlatMarket(&m)
						if fid == "" {
							continue
						}
						oc := polymarketOutcomeFromMarket(platform, category, subCategory, fid, nil, &m, compPtr, outcomeKind)
						if oc == nil {
							continue
						}
						if fixturesThisPage[fid] == nil {
							fixturesThisPage[fid] = polymarketFixtureFromFlatMarket(platform, category, subCategory, &m, compPtr, compEnPtr)
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
							notification.SendErrorLog(ctx, "market_sync-polymarket-markets-BatchUpsertFailed",
								zap.Int("count", len(foBatch.Outcomes)), zap.Error(err))
							seen.Fail(err)
							return
						}
						n := len(foBatch.Outcomes)
						totalSaved := seen.AddSaved(n)
						leagueOutcomes += n
						logger.InfoZ(ctx, "market_sync-polymarket-markets-BatchUpserted",
							zap.String("sport_id", lf.sportID),
							zap.Int("tag_id", src.tagID),
							zap.Int("batch_count", n),
							zap.Int("fixtures", len(fixturesThisPage)),
							zap.Int("total_saved", totalSaved))
					}
					if page.NextCursor == "" || len(page.Markets) < polymarketMarketsKeysetLimit {
						break
					}
					if page.NextCursor == afterCursor {
						notification.SendErrorLog(ctx, "market_sync-polymarket-markets-cursor-stuck",
							zap.String("sport_id", lf.sportID), zap.Int("tag_id", src.tagID))
						break
					}
					afterCursor = page.NextCursor
				}
			}
		}
		if leagueOutcomes == 0 {
			logger.InfoZ(ctx, "market_sync-polymarket-league-zero",
				zap.String("sport_id", lf.sportID),
				zap.Ints("tag_ids", lf.tagIDs),
				zap.Strings("sports_market_types", marketTypes))
		}
		notifyPolymarketLeagueDone(ctx, lf.sportID,
			leagueDisplayName(lf.sportID, nameBySport, compBySport),
			leagueDisplayNameEn(lf.sportID, nameDescBySport, metaBySport),
			lf.tagIDs, len(leagueFixtures), leagueOutcomes)
	})

	stats := seen.SaveStats()
	if err := seen.Err(); err != nil {
		return stats, err
	}
	if fxClosed, ocClosed, err := s.W.CloseStaleAfterSync(ctx, platform, category, subCategory, seen.Fixtures(), seen.Outcomes()); err != nil {
		return stats, err
	} else if fxClosed > 0 || ocClosed > 0 {
		logger.InfoZ(ctx, "market_sync-polymarket-closed",
			zap.String("sub_category", subCategory),
			zap.Int("fixtures_closed", fxClosed),
			zap.Int("outcomes_closed", ocClosed))
	}
	logger.InfoZ(ctx, "market_sync-polymarket-soccer-events-done",
		zap.Int("fixtures", stats.Fixtures),
		zap.Int("outcomes", stats.Outcomes),
		zap.Int("outcome_upserts", seen.Saved()))
	return stats, nil
}

func mergePolymarketCompetitionNames(configured map[string]string, metaBySport map[string]httpentity.SportsMetadata) map[string]string {
	out := make(map[string]string, len(configured)+len(metaBySport))
	for k, v := range configured {
		out[k] = v
	}
	for sid, meta := range metaBySport {
		if out[sid] != "" {
			continue
		}
		if n := strings.TrimSpace(meta.Name); n != "" {
			out[sid] = n
		}
	}
	return out
}

func enrichMarketFromPolymarketEvent(m *httpentity.Market, ev *httpentity.PolymarketEvent) {
	if m == nil || ev == nil {
		return
	}
	if strings.TrimSpace(m.GameStartTime) == "" {
		m.GameStartTime = strings.TrimSpace(ev.StartDate)
	}
	if len(m.Events) == 0 && (ev.StartDate != "" || ev.EndDate != "") {
		m.Events = []httpentity.MarketEventEmbed{{
			StartTime:         ev.StartDate,
			FinishedTimestamp: ev.EndDate,
		}}
	}
}

func polymarketEventInLeagueScope(ev *httpentity.PolymarketEvent, sportID string) bool {
	if ev == nil || sportID == "" {
		return sportID == ""
	}
	slug := strings.ToLower(strings.TrimSpace(ev.Slug))
	sportID = strings.ToLower(strings.TrimSpace(sportID))
	if slug == "" {
		return false
	}
	return slug == sportID || strings.HasPrefix(slug, sportID+"-")
}

func polymarketMarketInLeagueScope(m *httpentity.Market, sportID string) bool {
	if m == nil || sportID == "" {
		return sportID == ""
	}
	if polymarketSlugMatchesSportID(m, sportID) {
		return true
	}
	slug := strings.ToLower(strings.TrimSpace(m.Slug))
	sportID = strings.ToLower(strings.TrimSpace(sportID))
	if slug == "" {
		return false
	}
	return strings.HasPrefix(slug, sportID+"-") || strings.Contains(slug, "-"+sportID+"-")
}

func soccerSportsMarketTypes(types syncopt.OutcomeTypeSet) []string {
	var out []string
	if types.Moneyline {
		out = append(out, "moneyline")
	}
	if types.Handicap {
		out = append(out, "spreads")
	}
	if types.Total {
		out = append(out, "totals")
	}
	if len(out) == 0 {
		out = []string{"moneyline"}
	}
	return out
}

func resolvePolymarketSoccerLeagueFetches(ctx context.Context, items []polymarketSportItem, metaBySport map[string]httpentity.SportsMetadata) []polymarketLeagueFetch {
	var out []polymarketLeagueFetch
	for _, it := range items {
		meta, ok := metaBySport[it.SportID]
		if !ok {
			notification.SendErrorLog(ctx, "market_sync-polymarket-sport-unknown",
				zap.String("sport_id", it.SportID), zap.String("name", it.Name))
			continue
		}
		tagIDs := allTagIDsForLeagueFetch(meta, it)
		if len(tagIDs) == 0 {
			notification.SendErrorLog(ctx, "market_sync-polymarket-no-tag-for-league",
				zap.String("sport_id", it.SportID), zap.String("name", it.Name))
			continue
		}
		seriesID, _ := strconv.Atoi(strings.TrimSpace(meta.Series))
		out = append(out, polymarketLeagueFetch{sportID: it.SportID, tagIDs: tagIDs, seriesID: seriesID})
	}
	logger.InfoZ(ctx, "market_sync-polymarket-soccer-games-mode",
		zap.Int("league_count", len(out)),
		zap.String("source", "events_keyset"))
	return out
}
