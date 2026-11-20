package predict

import (
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func predictFixtureFromCategory(
	platform, category, subCategory string,
	cat httpentity.PredictCategory,
	competition, competitionEn string,
	fixtureID string,
) *persistententity.Market {
	slug := strings.TrimSpace(fixtureID)
	if slug == "" {
		slug = predictMainFixtureID(cat.Slug)
	}
	if slug == "" {
		slug = strconv.FormatInt(cat.ID, 10)
	}
	title := strings.TrimSpace(cat.Title)
	if title == "" {
		title = slug
	}
	if !predictIsMainMatchCategory(cat.Slug) && title == slug {
		title = strings.TrimSpace(cat.Title)
	}
	var compPtr, compEnPtr *string
	if competition != "" {
		compPtr = common.StrPtr128(competition)
	}
	if competitionEn != "" {
		compEnPtr = common.StrPtr128(competitionEn)
	}
	url := buildPredictFixtureURL(slug)
	evStart := formatPredictTime(cat.StartsAt)
	evEnd := formatPredictTime(cat.EndsAt)
	return &persistententity.Market{
		Platform:         platform,
		Name:             common.StrPtr(common.Truncate(title, 2000)),
		PlatformMarketID: slug,
		MarketURL:        url,
		EventStartTime:   common.StrPtr255NilIfEmpty(evStart),
		EventEndTime:     common.StrPtr255NilIfEmpty(evEnd),
		Status:           string(cryptoPredictionEnum.MarketStatusActive),
		Category:         common.StrPtr200(category),
		SubCategory:      common.StrPtr200(subCategory),
		SportCN:          common.StrPtr128(common.SportCNForSubCategory(subCategory)),
		Slug:             common.StrPtr255(strings.ToLower(slug)),
		Competition:      compPtr,
		CompetitionEn:    compEnPtr,
	}
}

func predictOutcomeFromMarket(
	platform, category, subCategory, fixtureID, fixtureTitle, competition string,
	cat httpentity.PredictCategory,
	m httpentity.PredictMarket,
	outcomeKind cryptoPredictionEnum.OutcomeMarketType,
) *persistententity.MarketOutcome {
	if fixtureID == "" {
		return nil
	}
	mid := m.ID
	if mid <= 0 {
		return nil
	}
	midStr := strconv.FormatInt(mid, 10)
	yesTok, noTok := predictYesNoTokens(m.Outcomes)
	title := strings.TrimSpace(m.Question)
	if title == "" {
		title = strings.TrimSpace(m.Title)
	}
	if title == "" {
		title = midStr
	}
	evStart := formatPredictTime(cat.StartsAt)
	evEnd := formatPredictTime(cat.EndsAt)
	if common.SkipEventEndedUTC(evStart, evEnd) {
		return nil
	}
	url := buildPredictFixtureURL(fixtureID)
	slug := common.StrPtr255(strings.ToLower(fixtureID))
	var compPtr *string
	if competition != "" {
		compPtr = common.StrPtr128(competition)
	}
	if outcomeKind == "" {
		outcomeKind = cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	desc := strings.TrimSpace(m.Description)
	var descPtr *string
	if desc != "" {
		descPtr = common.StrPtr(common.Truncate(desc, 2000))
	}
	return &persistententity.MarketOutcome{
		Platform:                platform,
		PlatformMarketID:        fixtureID,
		OutcomeType:             string(outcomeKind),
		OutcomeName:             common.StrPtr(common.Truncate(title, 2000)),
		MarketName:              common.StrPtrOptional(fixtureTitle),
		Desc:                    descPtr,
		PlatformMarketOutcomeID: midStr,
		MarketAddress:           common.StrPtr255(strings.TrimSpace(m.ConditionID)),
		MarketURL:               url,
		YesTokenID:              yesTok,
		NoTokenID:               noTok,
		MarketStartTime:         common.StrPtr255(evStart),
		MarketEndTime:           common.StrPtr255(evEnd),
		Category:                common.StrPtr200(category),
		SubCategory:             common.StrPtr200(subCategory),
		Slug:                    slug,
		Competition:             compPtr,
	}
}

func predictYesNoTokens(outcomes []httpentity.PredictOutcome) (yes, no string) {
	for _, o := range outcomes {
		switch o.IndexSet {
		case 1:
			yes = strings.TrimSpace(o.OnChainID)
		case 2:
			no = strings.TrimSpace(o.OnChainID)
		}
	}
	if yes == "" && len(outcomes) > 0 {
		yes = strings.TrimSpace(outcomes[0].OnChainID)
	}
	if no == "" && len(outcomes) > 1 {
		no = strings.TrimSpace(outcomes[1].OnChainID)
	}
	if yes == "" {
		yes = "0"
	}
	if no == "" {
		no = "0"
	}
	return yes, no
}

func formatPredictTime(raw string) string {
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

func competitionForLeague(item predictSportItem, cfg SportsConfig) (cn, en string) {
	sid := strings.TrimSpace(item.SportID)
	if sid == "" && len(item.TagIDs) > 0 {
		sid = strings.TrimSpace(item.TagIDs[0])
	}
	if v := strings.TrimSpace(item.Name); v != "" {
		cn = v
	} else if sid != "" {
		cn = cfg.CompetitionBySportID[sid]
	}
	if v := strings.TrimSpace(item.NameEn); v != "" {
		en = v
	} else if d := strings.TrimSpace(item.NameDesc); d != "" {
		en = d
	} else if sid != "" {
		en = cfg.CompetitionEnByTag[sid]
	}
	return cn, en
}