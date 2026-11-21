package sxbet

import (
	"encoding/json"
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
		MarketURL:        buildSxbetEventURLFromMarket(m),
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

// sxbetOutcomeTitle 与 sx.bet 盘口页一致：Yes 侧 / No 侧 + 玩法标签（避免多场共用一个「1X2」看不出是哪条腿）。
func sxbetOutcomeTitle(m httpentity.SxbetMarket, kind cryptoPredictionEnum.OutcomeMarketType) string {
	yes := strings.TrimSpace(m.OutcomeOneName)
	no := strings.TrimSpace(m.OutcomeTwoName)
	typeLabel := marketTypeLabel(m)
	if typeLabel == "" {
		typeLabel = "market"
	}
	line := ""
	if m.Line != nil {
		line = fmt.Sprintf(" line=%.2g", *m.Line)
	}
	if yes != "" && no != "" {
		return common.Truncate(fmt.Sprintf("%s vs %s [%s%s]", yes, no, typeLabel, line), 2000)
	}
	base := sxbetFixtureTitle(m)
	if line == "" {
		return common.Truncate(base+" ["+typeLabel+"]", 2000)
	}
	return common.Truncate(base+" ["+typeLabel+line+"]", 2000)
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
	url := buildSxbetMarketURLFromMarket(m)
	var compPtr *string
	if competition != "" {
		compPtr = common.StrPtr128(competition)
	}
	yesSide := strings.TrimSpace(m.OutcomeOneName)
	noSide := strings.TrimSpace(m.OutcomeTwoName)
	desc := yesSide + " / " + noSide
	var cfgPtr *string
	if yesSide != "" || noSide != "" {
		b, _ := json.Marshal(map[string]string{
			"yes_label": yesSide,
			"no_label":  noSide,
		})
		if len(b) > 0 {
			s := string(b)
			cfgPtr = &s
		}
	}
	// 每个 marketHash 对应 SX 一张单簿（YES 视角）；勿与 Poly 双 token 混为两行。
	return &persistententity.MarketOutcome{
		Platform:                platform,
		PlatformMarketID:        fixtureID,
		OutcomeType:             string(outcomeKind),
		OutcomeName:             common.StrPtr(name),
		MarketName:              common.StrPtrOptional(fixtureTitle),
		PlatformMarketOutcomeID: hash,
		MarketURL:               url,
		YesTokenID:              hash,
		NoTokenID:               "",
		Desc:                    common.StrPtrOptional(desc),
		Config:                  cfgPtr,
		MarketStartTime:         common.StrPtr255(evStart),
		Category:                common.StrPtr200(category),
		SubCategory:             common.StrPtr200(subCategory),
		Slug:                    common.StrPtr255(strings.ToLower(fixtureID)),
		Competition:             compPtr,
	}
}