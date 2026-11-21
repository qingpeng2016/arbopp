package sxbet

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func sxbetFixtureTitle(m httpentity.SxbetMarket) string {
	a := strings.TrimSpace(m.TeamOneName)
	b := strings.TrimSpace(m.TeamTwoName)
	if a != "" && b != "" {
		return a + " vs " + b
	}
	return strings.TrimSpace(m.LeagueLabel)
}

func formatSxbetUnix(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04:05")
}

func sxbetFixtureFromMarket(
	platform, category, subCategory string,
	m httpentity.SxbetMarket,
	competition, competitionEn string,
) *persistententity.Market {
	fid := strings.TrimSpace(m.SportXeventID)
	if fid == "" {
		return nil
	}
	title := sxbetFixtureTitle(m)
	var compPtr, compEnPtr *string
	if competition != "" {
		compPtr = common.StrPtr128(competition)
	}
	if competitionEn != "" {
		compEnPtr = common.StrPtr128(competitionEn)
	}
	evStart := formatSxbetUnix(m.GameTime)
	return &persistententity.Market{
		Platform:         platform,
		Name:             common.StrPtr(common.Truncate(title, 2000)),
		PlatformMarketID: fid,
		MarketURL:        buildSxbetEventURL(fid),
		EventStartTime:   common.StrPtr255NilIfEmpty(evStart),
		Status:           string(cryptoPredictionEnum.MarketStatusActive),
		Category:         common.StrPtr200(category),
		SubCategory:      common.StrPtr200(subCategory),
		SportCN:          common.StrPtr128(common.SportCNForSubCategory(subCategory)),
		Slug:             common.StrPtr255(strings.ToLower(fid)),
		Competition:      compPtr,
		CompetitionEn:    compEnPtr,
	}
}

func sxbetOutcomeTitle(m httpentity.SxbetMarket, kind cryptoPredictionEnum.OutcomeMarketType) string {
	base := sxbetFixtureTitle(m)
	typeLabel := marketTypeLabel(m)
	if typeLabel == "" {
		return base
	}
	line := ""
	if m.Line != nil {
		line = fmt.Sprintf(" (%.2g)", *m.Line)
	}
	switch kind {
	case cryptoPredictionEnum.OutcomeMarketTypeHandicap:
		return common.Truncate(base+" — "+typeLabel+line, 2000)
	case cryptoPredictionEnum.OutcomeMarketTypeTotal:
		return common.Truncate(base+" — "+typeLabel+line, 2000)
	default:
		return common.Truncate(base+" — "+typeLabel, 2000)
	}
}

func marketTypeLabel(m httpentity.SxbetMarket) string {
	switch m.Type {
	case 1:
		return "1X2"
	case 52, 226:
		return "Moneyline"
	case 3, 342, 201:
		return "Spread"
	case 2, 28, 166, 1536:
		return "Total"
	default:
		return "type " + strconv.Itoa(m.Type)
	}
}

func sxbetOutcomeFromMarket(
	platform, category, subCategory, fixtureID, fixtureTitle, competition string,
	m httpentity.SxbetMarket,
	outcomeKind cryptoPredictionEnum.OutcomeMarketType,
) *persistententity.MarketOutcome {
	if !strings.EqualFold(m.Status, "ACTIVE") {
		return nil
	}
	hash := strings.TrimSpace(m.MarketHash)
	if hash == "" || fixtureID == "" {
		return nil
	}
	evStart := formatSxbetUnix(m.GameTime)
	if common.SkipEventEndedUTC(evStart, "") {
		return nil
	}
	name := sxbetOutcomeTitle(m, outcomeKind)
	if outcomeKind == "" {
		outcomeKind = cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	url := buildSxbetMarketURL(hash, fixtureID)
	var compPtr *string
	if competition != "" {
		compPtr = common.StrPtr128(competition)
	}
	yesSide := strings.TrimSpace(m.OutcomeOneName)
	noSide := strings.TrimSpace(m.OutcomeTwoName)
	desc := yesSide + " / " + noSide
	return &persistententity.MarketOutcome{
		Platform:                platform,
		PlatformMarketID:        fixtureID,
		OutcomeType:             string(outcomeKind),
		OutcomeName:             common.StrPtr(name),
		MarketName:              common.StrPtrOptional(fixtureTitle),
		PlatformMarketOutcomeID: hash,
		MarketURL:               url,
		YesTokenID:              hash,
		NoTokenID:               hash,
		Desc:                    common.StrPtrOptional(desc),
		MarketStartTime:         common.StrPtr255(evStart),
		Category:                common.StrPtr200(category),
		SubCategory:             common.StrPtr200(subCategory),
		Slug:                    common.StrPtr255(strings.ToLower(fixtureID)),
		Competition:             compPtr,
	}
}