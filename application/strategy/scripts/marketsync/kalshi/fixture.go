package kalshi

import (
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func kalshiFixtureFromEvent(
	platform, category, subCategory string,
	ev httpentity.KalshiEvent,
	competition, competitionEn string,
) *persistententity.Market {
	eventTicker := strings.TrimSpace(ev.EventTicker)
	if eventTicker == "" {
		return nil
	}
	title := strings.TrimSpace(ev.Title)
	if title == "" {
		title = strings.TrimSpace(ev.SubTitle)
	}
	var compPtr, compEnPtr *string
	if competition != "" {
		compPtr = common.StrPtr128(competition)
	}
	if competitionEn != "" {
		compEnPtr = common.StrPtr128(competitionEn)
	}
	url := "https://kalshi.com/events/" + strings.ToLower(eventTicker)
	evStart, evEnd := kalshiFixtureTimes(eventTicker, ev.Markets)
	return &persistententity.Market{
		Platform:         platform,
		Name:             common.StrPtr(common.Truncate(title, 2000)),
		PlatformMarketID: eventTicker,
		MarketURL:        url,
		EventStartTime:   common.StrPtr255NilIfEmpty(evStart),
		EventEndTime:     common.StrPtr255NilIfEmpty(evEnd),
		Status:           string(cryptoPredictionEnum.MarketStatusActive),
		Category:         common.StrPtr200(category),
		SubCategory:      common.StrPtr200(subCategory),
		SportCN:          common.StrPtr128(common.SportCNForSubCategory(subCategory)),
		Slug:             common.StrPtr255(strings.ToLower(eventTicker)),
		Competition:      compPtr,
		CompetitionEn:    compEnPtr,
	}
}

func parseKalshiTimeUTC(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func kalshiOutcomeFromMarket(
	platform, category, subCategory, fixtureID, eventTitle, competition string,
	m httpentity.KalshiMarket,
	outcomeKind cryptoPredictionEnum.OutcomeMarketType,
) *persistententity.MarketOutcome {
	if strings.EqualFold(m.Status, "closed") || strings.EqualFold(m.Status, "settled") {
		return nil
	}
	ticker := strings.TrimSpace(m.Ticker)
	if ticker == "" || fixtureID == "" {
		return nil
	}
	evStart := formatKalshiTime(m.OpenTime)
	evEnd := formatKalshiTime(m.CloseTime)
	if common.SkipEventEndedUTC(evStart, evEnd) {
		return nil
	}
	title := strings.TrimSpace(m.Title)
	if sub := strings.TrimSpace(m.Subtitle); sub != "" {
		title = title + " — " + sub
	}
	name := common.Truncate(title, 2000)
	volTotal := parseKalshiFloat(m.VolumeFp)
	vol24 := parseKalshiFloat(m.Volume24hFp)
	eventTicker := strings.TrimSpace(m.EventTicker)
	if eventTicker == "" {
		eventTicker = fixtureID
	}
	url := common.Truncate(buildKalshiMarketURL(ticker, eventTicker, eventTitle), 255)
	slug := common.StrPtr255(strings.ToLower(eventTicker))
	var compPtr *string
	if competition != "" {
		compPtr = common.StrPtr128(competition)
	}
	if outcomeKind == "" {
		outcomeKind = cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	return &persistententity.MarketOutcome{
		Platform:                platform,
		PlatformMarketID:        fixtureID,
		OutcomeType:             string(outcomeKind),
		Name:                    common.StrPtr(name),
		PlatformMarketOutcomeID: ticker,
		MarketURL:               url,
		YesTokenID:              ticker,
		NoTokenID:               ticker,
		MarketStartTime:         common.StrPtr255(evStart),
		MarketEndTime:           common.StrPtr255(evEnd),
		Category:                common.StrPtr200(category),
		SubCategory:             common.StrPtr200(subCategory),
		VolumeTotal:             common.FloatPtr(volTotal),
		Volume24hr:              common.FloatPtr(vol24),
		Slug:                    slug,
		Competition:             compPtr,
	}
}
