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

// Sync 拉取并写入 market（场次）+ market_outcome（盘口）。非足球与 soccer-games 均走 /markets/keyset。
func (s *Syncer) Sync(ctx context.Context, subCat cryptoPredictionEnum.SubMarketCategory, sportsCfg SportsConfig, outcomeTypes syncopt.OutcomeTypeSet) (leagueopt.SaveStats, error) {
	if outcomeTypes.Empty() {
		outcomeTypes = syncopt.DefaultOutcomeTypeSet()
	}
	if s.API == nil {
		return leagueopt.SaveStats{}, nil
	}
	useSoccerGames := sportsCfg.SyncAllSoccerGames && subCat == cryptoPredictionEnum.SubMarketCategoryFootball
	if useSoccerGames {
		return s.syncSoccerGamesEvents(ctx, subCat, sportsCfg, outcomeTypes)
	}
	leagueFetches, err := s.resolveLeagueFetches(ctx, subCat, sportsCfg)
	if err != nil {
		return leagueopt.SaveStats{}, err
	}
	if len(leagueFetches) == 0 {
		logger.InfoZ(ctx, "market_sync-polymarket-no-tags", zap.String("sub_category", string(subCat)))
		return leagueopt.SaveStats{}, nil
	}

	platform := string(cryptoPredictionEnum.PlatformPolymarket)
	category := string(cryptoPredictionEnum.MarketCategorySports)
	subCategory := string(subCat)
	closed := false
	closedPtr := &closed

	seen := leagueopt.NewSeen()
	nameBySport := sportNamesByID(sportsCfg, subCat)
	nameDescBySport := sportNameDescByID(sportsCfg, subCat)
	compBySport := sportsCfg.CompetitionBySportID
	marketTypes := soccerSportsMarketTypes(outcomeTypes)

	leagueopt.Each(ctx, s.LeagueConcurrency, len(leagueFetches), func(i int) {
		lf := leagueFetches[i]
		var leagueOutcomes int
		leagueFixtures := make(map[string]struct{})
		for _, tagID := range lf.tagIDs {
			for _, marketType := range marketTypes {
				asc := false
				afterCursor := ""
				for pageIdx := 0; pageIdx < polymarketMarketsKeysetPages; pageIdx++ {
					time.Sleep(80 * time.Millisecond)
					page, err := s.API.ListMarketsKeyset(ctx, httpentity.ListMarketsKeysetRequest{
						Limit:             polymarketMarketsKeysetLimit,
						Closed:            closedPtr,
						TagID:             tagID,
						SportsMarketTypes: []string{marketType},
						AfterCursor:       afterCursor,
						Order:             "id",
						Ascending:         &asc,
					})
					if err != nil {
						notification.SendErrorLog(ctx, "market_sync-polymarket-markets-stopped",
							zap.String("sport_id", lf.sportID), zap.Int("tag_id", tagID),
							zap.String("market_type", marketType), zap.Int("page", pageIdx), zap.Error(err))
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
						if sportsCfg.Active {
							if !polymarketSlugMatchesSportID(&m, lf.sportID) {
								continue
							}
						} else if !polymarketMatchesConfiguredLeague(&m, sportsCfg.CompetitionBySportID) {
							continue
						}
						ok, outcomeKind := polymarketMarketForSync(&m, outcomeTypes, subCat)
						if !ok {
							continue
						}
						yesTok, noTok := parsePolymarketClobTokenIds(m.ClobTokenIds)
						if yesTok == "" || noTok == "" {
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
						if !seen.Claim(dedupeKey) {
							continue
						}

						comp, sid := polymarketCompetitionName(&m, sportsCfg.CompetitionBySportID)
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
							notification.SendErrorLog(ctx, "market_sync-polymarket-BatchUpsertFailed",
								zap.Int("count", len(foBatch.Outcomes)), zap.Error(err))
							seen.Fail(err)
							return
						}
						n := len(foBatch.Outcomes)
						totalSaved := seen.AddSaved(n)
						leagueOutcomes += n
						logger.InfoZ(ctx, "market_sync-polymarket-BatchUpserted",
							zap.String("sub_category", subCategory),
							zap.String("sport_id", lf.sportID),
							zap.Int("tag_id", tagID),
							zap.String("market_type", marketType),
							zap.Int("batch_count", n),
							zap.Int("total_saved", totalSaved))
					}

					if page.NextCursor == "" || len(page.Markets) < polymarketMarketsKeysetLimit {
						break
					}
					if page.NextCursor == afterCursor {
						notification.SendErrorLog(ctx, "market_sync-polymarket-markets-cursor-stuck",
							zap.String("sport_id", lf.sportID), zap.Int("tag_id", tagID),
							zap.String("market_type", marketType))
						break
					}
					afterCursor = page.NextCursor
				}
			}
		}
		notifyPolymarketLeagueDone(ctx, lf.sportID,
			leagueDisplayName(lf.sportID, nameBySport, compBySport),
			leagueDisplayNameEn(lf.sportID, nameDescBySport, nil),
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
	return stats, nil
}

type polymarketLeagueFetch struct {
	sportID  string
	tagIDs   []int
	seriesID int // GET /sports series，用于欧冠等显式 tag_ids 联赛的 keyset 拉取
}

type polymarketLeagueEventSource struct {
	tagID    int
	seriesID int
}

func (lf polymarketLeagueFetch) eventSources() []polymarketLeagueEventSource {
	out := make([]polymarketLeagueEventSource, 0, len(lf.tagIDs))
	for _, id := range lf.tagIDs {
		out = append(out, polymarketLeagueEventSource{tagID: id})
	}
	return out
}

func (s *Syncer) resolveLeagueFetches(ctx context.Context, subCat cryptoPredictionEnum.SubMarketCategory, sportsCfg SportsConfig) ([]polymarketLeagueFetch, error) {
	sports, err := s.API.GetSportsMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("get sports metadata: %w", err)
	}
	if sportsCfg.Active {
		items := sportsCfg.BySubCategory[subCat]
		if len(items) == 0 {
			return nil, nil
		}
		metaBySport := make(map[string]httpentity.SportsMetadata, len(sports))
		for _, s := range sports {
			metaBySport[strings.ToLower(strings.TrimSpace(s.Sport))] = s
		}
		var out []polymarketLeagueFetch
		for _, it := range items {
			meta, ok := metaBySport[it.SportID]
			if !ok {
				notification.SendErrorLog(ctx, "market_sync-polymarket-sport-unknown",
					zap.String("sport_id", it.SportID), zap.String("name", it.Name))
				continue
			}
			tagIDs := tagIDsForPolymarketLeague(meta, it)
			if len(tagIDs) == 0 {
				notification.SendErrorLog(ctx, "market_sync-polymarket-no-tag-for-league",
					zap.String("sport_id", it.SportID), zap.String("name", it.Name))
				continue
			}
			out = append(out, polymarketLeagueFetch{sportID: it.SportID, tagIDs: tagIDs})
		}
		return out, nil
	}
	sportIDFilter := make(map[string]struct{})
	tagsInOtherSports := make(map[int]struct{})
	if subCat != "" {
		ids, err := cryptoPredictionEnum.ToPolymarketSportIDs(subCat)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			sportIDFilter[id] = struct{}{}
		}
		for _, s := range sports {
			if _, ok := sportIDFilter[s.Sport]; ok {
				continue
			}
			for _, part := range strings.Split(s.Tags, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				id, _ := strconv.Atoi(part)
				tagsInOtherSports[id] = struct{}{}
			}
		}
	}
	tagIDSet := make(map[int]struct{})
	for _, s := range sports {
		if len(sportIDFilter) > 0 {
			if _, ok := sportIDFilter[s.Sport]; !ok {
				continue
			}
		}
		for _, part := range strings.Split(s.Tags, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.Atoi(part)
			if err != nil || id == 1 {
				continue
			}
			if _, excluded := tagsInOtherSports[id]; excluded {
				continue
			}
			tagIDSet[id] = struct{}{}
		}
	}
	out := make([]polymarketLeagueFetch, 0, len(tagIDSet))
	for id := range tagIDSet {
		out = append(out, polymarketLeagueFetch{sportID: "", tagIDs: []int{id}})
	}
	return out, nil
}

// polymarketSharedTagIDs 全站/多运动共用 tag，不能单独用来拉某一联赛（会混盘且分页极深）。
var polymarketSharedTagIDs = map[int]struct{}{
	1:      {},
	100350: {},
	100639: {},
}

// polymarketLeagueGameTagFallback primaryTag 常只索引期货；大学篮球单场在 NCAA(100396)/CWBB-Games(102780)。
var polymarketLeagueGameTagFallback = map[string][]int{
	"cbb":  {100396},
	"cwbb": {102780, 100396},
}

// tagIDsForPolymarketLeague 优先 JSON tag_ids，否则 GET /sports 的 primaryTagId + 联赛专用 tag。
// 例如 ucl：primary 1234 多为赛季期货，单场盘在 100977；sea：单场在 101962，100618 多为期货。
func tagIDsForPolymarketLeague(meta httpentity.SportsMetadata, item polymarketSportItem) []int {
	if len(item.TagIDs) > 0 {
		return mergeUniqueIntIDs(append([]int(nil), item.TagIDs...), polymarketLeagueGameTagFallback[strings.ToLower(item.SportID)])
	}
	leagueIDs, _ := leagueTagIDsFromSportsMeta(meta)
	matchIDs := tagIDsFromSportsMeta(meta.Tags, item.SportID, item)
	return mergeUniqueIntIDs(leagueIDs, matchIDs)
}

// leagueTagIDsFromSportsMeta 取联赛专用 tag；跳过全站共用 tag、赛季 tag(82)、primary（多为期货）。
// 若去掉 primary 后为空（如 epl 只剩 306），再回退 primary。
func leagueTagIDsFromSportsMeta(meta httpentity.SportsMetadata) ([]int, map[int]struct{}) {
	primary := meta.PrimaryTagID
	seen := make(map[int]struct{})
	var candidates []int
	addCand := func(id int) {
		if id <= 0 || id == 1 || id == 82 {
			return
		}
		if _, shared := polymarketSharedTagIDs[id]; shared {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		candidates = append(candidates, id)
	}
	for _, part := range strings.Split(meta.Tags, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		addCand(id)
	}

	var out []int
	outSeen := make(map[int]struct{})
	appendID := func(id int) {
		if _, ok := outSeen[id]; ok {
			return
		}
		outSeen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range candidates {
		if id == primary {
			continue
		}
		appendID(id)
	}
	if len(out) == 0 && primary > 0 {
		if _, shared := polymarketSharedTagIDs[primary]; !shared && primary != 1 && primary != 82 {
			appendID(primary)
		}
	}
	return out, outSeen
}
