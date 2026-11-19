package kalshi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leagueopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"go.uber.org/zap"
)

// kalshiSpreadSeries 让分是独立 series：KXAFCONGAME → KXAFCONSPREAD。页面上和胜平负放在同一场。
func kalshiSpreadSeries(gameSeries string) (string, bool) {
	return kalshiSiblingSeries(gameSeries, "SPREAD")
}

// kalshiTotalSeries 大小球是独立 series：KXAFCONGAME → KXAFCONTOTAL。
func kalshiTotalSeries(gameSeries string) (string, bool) {
	return kalshiSiblingSeries(gameSeries, "TOTAL")
}

func kalshiSiblingSeries(gameSeries, suffix string) (string, bool) {
	s := strings.ToUpper(strings.TrimSpace(gameSeries))
	suffix = strings.ToUpper(strings.TrimSpace(suffix))
	if !strings.HasSuffix(s, "GAME") || suffix == "" {
		return "", false
	}
	return strings.TrimSuffix(s, "GAME") + suffix, true
}

func kalshiGameEventTicker(gameSeries, propEventTicker string) string {
	gameSeries = strings.ToUpper(strings.TrimSpace(gameSeries))
	if !strings.HasSuffix(gameSeries, "GAME") {
		return ""
	}
	base := strings.TrimSuffix(gameSeries, "GAME")
	et := strings.ToUpper(strings.TrimSpace(propEventTicker))
	if !strings.HasPrefix(et, base) {
		return ""
	}
	rest := strings.TrimPrefix(et, base)
	i := strings.Index(rest, "-")
	if i <= 0 {
		return ""
	}
	kind := rest[:i]
	if kind != "SPREAD" && kind != "TOTAL" {
		return ""
	}
	tail := rest[i+1:]
	if tail == "" {
		return ""
	}
	return gameSeries + "-" + tail
}

func kalshiMatchTitleFromSpread(title string) string {
	return kalshiTrimTitleSuffix(title, []string{": Spread", " Spread"})
}

func kalshiMatchTitleFromTotal(title string) string {
	return kalshiTrimTitleSuffix(title, []string{": Total Goals", " Total Goals"})
}

func kalshiTrimTitleSuffix(title string, suffixes []string) string {
	title = strings.TrimSpace(title)
	for _, suf := range suffixes {
		if len(title) >= len(suf) && strings.EqualFold(title[len(title)-len(suf):], suf) {
			return strings.TrimSpace(title[:len(title)-len(suf)])
		}
	}
	return title
}

// kalshiEventIsFullGameSpread 全场让分。上半场让分的 scope 是 "1st Half Spread"，不收。
func kalshiEventIsFullGameSpread(ev httpentity.KalshiEvent) bool {
	return kalshiEventScopeIs(ev, "spread")
}

// kalshiEventIsFullGameTotal 全场大小球。scope 是 "Point Total"。球队总分、上半场不收。
func kalshiEventIsFullGameTotal(ev httpentity.KalshiEvent) bool {
	return kalshiEventScopeIs(ev, "point total")
}

func kalshiEventScopeIs(ev httpentity.KalshiEvent, want string) bool {
	if cat := strings.TrimSpace(ev.Category); cat != "" && !strings.EqualFold(cat, "Sports") {
		return false
	}
	if ev.ProductMetadata == nil {
		return true
	}
	scope := strings.ToLower(strings.TrimSpace(ev.ProductMetadata.CompetitionScope))
	if scope == "" {
		return true
	}
	return scope == want
}

type kalshiSiblingSpec struct {
	seriesSuffix string
	accept       func(httpentity.KalshiEvent) bool
	titleFrom    func(string) string
}

func (s *Syncer) syncSpreadOutcomes(
	ctx context.Context,
	gameSeries, platform, category, subCategory string,
	cfg SeriesConfig,
	outcomeTypes syncopt.OutcomeTypeSet,
	seen *leagueopt.Seen,
	leagueFixtures map[string]struct{},
) (int, error) {
	if !outcomeTypes.Handicap {
		return 0, nil
	}
	return s.syncSiblingOutcomes(ctx, gameSeries, platform, category, subCategory, cfg, outcomeTypes, seen, leagueFixtures, kalshiSiblingSpec{
		seriesSuffix: "SPREAD",
		accept:       kalshiEventIsFullGameSpread,
		titleFrom:    kalshiMatchTitleFromSpread,
	})
}

func (s *Syncer) syncTotalOutcomes(
	ctx context.Context,
	gameSeries, platform, category, subCategory string,
	cfg SeriesConfig,
	outcomeTypes syncopt.OutcomeTypeSet,
	seen *leagueopt.Seen,
	leagueFixtures map[string]struct{},
) (int, error) {
	if !outcomeTypes.Total {
		return 0, nil
	}
	return s.syncSiblingOutcomes(ctx, gameSeries, platform, category, subCategory, cfg, outcomeTypes, seen, leagueFixtures, kalshiSiblingSpec{
		seriesSuffix: "TOTAL",
		accept:       kalshiEventIsFullGameTotal,
		titleFrom:    kalshiMatchTitleFromTotal,
	})
}

// syncSiblingOutcomes 把 *SPREAD / *TOTAL 的盘口写到对应 *GAME 场次下，不另开一场。
func (s *Syncer) syncSiblingOutcomes(
	ctx context.Context,
	gameSeries, platform, category, subCategory string,
	cfg SeriesConfig,
	outcomeTypes syncopt.OutcomeTypeSet,
	seen *leagueopt.Seen,
	leagueFixtures map[string]struct{},
	spec kalshiSiblingSpec,
) (int, error) {
	if s == nil || s.API == nil {
		return 0, nil
	}
	propSeries, ok := kalshiSiblingSeries(gameSeries, spec.seriesSuffix)
	if !ok {
		return 0, nil
	}
	compName := cfg.CompetitionBySeries[gameSeries]
	compEn := cfg.LeagueBySeries[gameSeries].NameEn
	var saved int
	cursor := ""
	for page := 0; page < kalshiMaxPages; page++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := s.API.GetEvents(ctx, httprepo.KalshiListParams{
			Limit:             kalshiPageSize,
			Cursor:            cursor,
			Status:            "open",
			Category:          "Sports",
			SeriesTicker:      propSeries,
			WithNestedMarkets: true,
		})
		if err != nil {
			return saved, fmt.Errorf("get events series=%s page=%d: %w", propSeries, page, err)
		}
		if resp == nil || len(resp.Events) == 0 {
			break
		}
		var foBatch batch.FixtureOutcome
		for _, ev := range resp.Events {
			if spec.accept != nil && !spec.accept(ev) {
				continue
			}
			gameTicker := kalshiGameEventTicker(gameSeries, ev.EventTicker)
			if gameTicker == "" {
				continue
			}
			matchTitle := spec.titleFrom(ev.Title)
			if matchTitle == "" {
				matchTitle = spec.titleFrom(ev.SubTitle)
			}
			var n int
			for _, m := range ev.Markets {
				okType, outcomeKind := kalshiMarketForSync(m, outcomeTypes)
				if !okType {
					continue
				}
				oc := kalshiOutcomeFromMarket(platform, category, subCategory, gameTicker, matchTitle, compName, m, outcomeKind)
				if oc == nil {
					continue
				}
				seen.Keep(gameTicker, oc.PlatformMarketOutcomeID)
				leagueFixtures[gameTicker] = struct{}{}
				foBatch.AddOutcome(oc)
				n++
			}
			if n == 0 {
				continue
			}
			fxEv := ev
			fxEv.EventTicker = gameTicker
			fxEv.Title = matchTitle
			foBatch.AddFixture(kalshiFixtureFromEvent(platform, category, subCategory, fxEv, compName, compEn))
		}
		if len(foBatch.Outcomes) > 0 {
			if err := s.W.PersistFixtureOutcomes(ctx, &foBatch); err != nil {
				notification.SendErrorLog(ctx, "market_sync-kalshi-BatchUpsertFailed",
					zap.String("series", propSeries),
					zap.Int("count", len(foBatch.Outcomes)), zap.Error(err))
				return saved, err
			}
			n := len(foBatch.Outcomes)
			totalSaved := seen.AddSaved(n)
			saved += n
			logger.InfoZ(ctx, "market_sync-kalshi-BatchUpserted",
				zap.String("series", propSeries),
				zap.String("sub_category", subCategory),
				zap.Int("batch_count", n),
				zap.Int("total_saved", totalSaved))
		}
		cursor = resp.Cursor
		if cursor == "" || len(resp.Events) < kalshiPageSize {
			break
		}
	}
	return saved, nil
}
