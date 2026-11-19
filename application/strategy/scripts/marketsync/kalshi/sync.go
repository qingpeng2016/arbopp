package kalshi

import (
	"context"
	"fmt"
	"regexp"
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
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"go.uber.org/zap"
)

const (
	kalshiPageSize = 200
	kalshiMaxPages = 300
)

// Sync 拉取并写入 market + market_outcome。
func (s *Syncer) Sync(ctx context.Context, subCat cryptoPredictionEnum.SubMarketCategory, cfg SeriesConfig, outcomeTypes syncopt.OutcomeTypeSet) (leagueopt.SaveStats, error) {
	if outcomeTypes.Empty() {
		outcomeTypes = syncopt.DefaultOutcomeTypeSet()
	}
	if s.API == nil {
		return leagueopt.SaveStats{}, nil
	}

	seriesList := kalshiGameSeriesForSubCategory(subCat, cfg)
	if len(seriesList) == 0 {
		return leagueopt.SaveStats{}, nil
	}

	platform := string(cryptoPredictionEnum.PlatformKalshi)
	category := string(cryptoPredictionEnum.MarketCategorySports)
	subCategory := string(subCat)

	seen := leagueopt.NewSeen()
	leagueopt.Each(ctx, s.LeagueConcurrency, len(seriesList), func(i int) {
		series := seriesList[i]
		var leagueOutcomes int
		leagueFixtures := make(map[string]struct{})
		cursor := ""
		var fetchErr error
		for page := 0; page < kalshiMaxPages; page++ {
			time.Sleep(100 * time.Millisecond)
			resp, err := s.API.GetEvents(ctx, httprepo.KalshiListParams{
				Limit:             kalshiPageSize,
				Cursor:            cursor,
				Status:            "open",
				Category:          "Sports",
				SeriesTicker:      series,
				WithNestedMarkets: true,
			})
			if err != nil {
				fetchErr = fmt.Errorf("get events series=%s page=%d: %w", series, page, err)
				break
			}
			if resp == nil || len(resp.Events) == 0 {
				break
			}

			var foBatch batch.FixtureOutcome
			fixturesThisPage := make(map[string]*persistententity.Market)
			for _, ev := range resp.Events {
				if !kalshiEventIsMatchMarket(ev) {
					continue
				}
				compName := cfg.CompetitionBySeries[series]
				compEn := cfg.LeagueBySeries[series].NameEn
				eventTitle := strings.TrimSpace(ev.Title)
				if eventTitle == "" {
					eventTitle = strings.TrimSpace(ev.SubTitle)
				}
				fid := strings.TrimSpace(ev.EventTicker)
				if fid == "" {
					continue
				}
				for _, m := range ev.Markets {
					okType, outcomeKind := kalshiMarketForSync(m, outcomeTypes)
					if !okType {
						continue
					}
					oc := kalshiOutcomeFromMarket(platform, category, subCategory, fid, eventTitle, compName, m, outcomeKind)
					if oc == nil {
						continue
					}
					if fixturesThisPage[fid] == nil {
						fixturesThisPage[fid] = kalshiFixtureFromEvent(platform, category, subCategory, ev, compName, compEn)
					}
					seen.Keep(fid, oc.PlatformMarketOutcomeID)
					leagueFixtures[fid] = struct{}{}
					foBatch.AddOutcome(oc)
				}
			}
			for _, fx := range fixturesThisPage {
				foBatch.AddFixture(fx)
			}

			if len(foBatch.Outcomes) > 0 {
				if err := s.W.PersistFixtureOutcomes(ctx, &foBatch); err != nil {
					notification.SendErrorLog(ctx, "market_sync-kalshi-BatchUpsertFailed",
						zap.String("series", series),
						zap.Int("count", len(foBatch.Outcomes)), zap.Error(err))
					seen.Fail(err)
					return
				}
				n := len(foBatch.Outcomes)
				totalSaved := seen.AddSaved(n)
				leagueOutcomes += n
				logger.InfoZ(ctx, "market_sync-kalshi-BatchUpserted",
					zap.String("series", series),
					zap.String("sub_category", subCategory),
					zap.Int("batch_count", len(foBatch.Outcomes)),
					zap.Int("total_saved", totalSaved))
			}

			cursor = resp.Cursor
			if cursor == "" || len(resp.Events) < kalshiPageSize {
				break
			}
		}
		if outcomeTypes.Handicap {
			n, err := s.syncSpreadOutcomes(ctx, series, platform, category, subCategory, cfg, outcomeTypes, seen, leagueFixtures)
			leagueOutcomes += n
			if err != nil && fetchErr == nil {
				fetchErr = err
			}
		}
		if outcomeTypes.Total {
			n, err := s.syncTotalOutcomes(ctx, series, platform, category, subCategory, cfg, outcomeTypes, seen, leagueFixtures)
			leagueOutcomes += n
			if err != nil && fetchErr == nil {
				fetchErr = err
			}
		}
		notifyKalshiLeagueDone(ctx, series, cfg.LeagueBySeries[series], len(leagueFixtures), leagueOutcomes)
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
		logger.InfoZ(ctx, "market_sync-kalshi-stale-done",
			zap.String("sub_category", subCategory),
			zap.Int("fixtures", stats.Fixtures),
			zap.Int("outcomes", stats.Outcomes),
			zap.Int("fixtures_closed", fxClosed),
			zap.Int("outcomes_closed", ocClosed),
			zap.Int("outcome_upserts", seen.Saved()))
	}
	return stats, nil
}

func formatKalshiTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format("2006-01-02 15:04:05")
		}
	}
	return common.Truncate(raw, 255)
}

// buildKalshiMarketURL 官网为 /markets/{series}/{title-slug}/{event_ticker}/{market_ticker}（全小写）。
// 缺 title-slug 时 Kalshi 前端常会错跳到别的场次。
func buildKalshiMarketURL(ticker, eventTicker, eventTitle string) string {
	ticker = strings.TrimSpace(ticker)
	eventTicker = strings.TrimSpace(eventTicker)
	if ticker == "" {
		return "https://kalshi.com"
	}
	lticker := strings.ToLower(ticker)
	if eventTicker == "" {
		return "https://kalshi.com/markets/" + lticker
	}
	levent := strings.ToLower(eventTicker)
	series := strings.ToLower(seriesTickerFromEvent(eventTicker))
	if series == "" {
		return "https://kalshi.com/events/" + levent
	}
	if slug := kalshiWebPathSlug(eventTitle); slug != "" {
		return "https://kalshi.com/markets/" + series + "/" + slug + "/" + levent + "/" + lticker
	}
	return "https://kalshi.com/markets/" + series + "/" + levent + "/" + lticker
}

var reKalshiVs = regexp.MustCompile(`(?i)\s+vs\.?\s+`)

// kalshiWebPathSlug 如 "Levante vs Elche" → levante-vs-elche
func kalshiWebPathSlug(eventTitle string) string {
	s := strings.TrimSpace(eventTitle)
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	s = reKalshiVs.ReplaceAllString(s, "-vs-")
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if r == '-' && !lastHyphen && b.Len() > 0 {
			b.WriteRune('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func seriesTickerFromEvent(eventTicker string) string {
	eventTicker = strings.TrimSpace(eventTicker)
	if eventTicker == "" {
		return ""
	}
	if i := strings.Index(eventTicker, "-"); i > 0 {
		return eventTicker[:i]
	}
	return eventTicker
}

func parseKalshiFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
