package polymarket

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// polymarketEventSlugBase 主盘 event 与「More Markets」子 event 共用同一 slug 前缀（如 spread 在 *-more-markets 下）。
func polymarketParentMarketName(ev *httpentity.PolymarketEvent, m *httpentity.Market) string {
	if ev != nil {
		title := common.Truncate(strings.TrimSpace(ev.Title), 2000)
		if base := polymarketEventSlugBase(ev); base != "" {
			title = common.Truncate(strings.TrimSpace(strings.TrimSuffix(ev.Title, " - More Markets")), 2000)
		}
		return title
	}
	if m == nil {
		return ""
	}
	return common.Truncate(polymarketFixtureDisplayName(m), 2000)
}

func polymarketEventSlugBase(ev *httpentity.PolymarketEvent) string {
	if ev == nil {
		return ""
	}
	slug := strings.ToLower(strings.TrimSpace(ev.Slug))
	return strings.TrimSuffix(slug, "-more-markets")
}

func polymarketFixtureMarketID(ev *httpentity.PolymarketEvent) string {
	if ev == nil {
		return ""
	}
	if base := polymarketEventSlugBase(ev); base != "" {
		return "slug:" + base
	}
	if id := strings.TrimSpace(ev.ID); id != "" {
		return id
	}
	return ""
}

func polymarketFixtureFromEvent(
	platform, category, subCategory string,
	ev *httpentity.PolymarketEvent,
	comp, compEn *string,
	league string,
) *persistententity.Market {
	fid := polymarketFixtureMarketID(ev)
	if fid == "" {
		return nil
	}
	title := common.Truncate(strings.TrimSpace(ev.Title), 2000)
	if base := polymarketEventSlugBase(ev); base != "" {
		title = common.Truncate(strings.TrimSpace(strings.TrimSuffix(ev.Title, " - More Markets")), 2000)
	}
	slug := common.StrPtr255(polymarketEventSlugBase(ev))
	evStart := common.StrPtr255NilIfEmpty(ev.StartDate)
	evEnd := common.StrPtr255NilIfEmpty(ev.EndDate)
	url := buildPolymarketEventURL(ev.Slug)
	return &persistententity.Market{
		Platform:         platform,
		Name:             common.StrPtr(title),
		PlatformMarketID: fid,
		MarketURL:        url,
		EventStartTime:   evStart,
		EventEndTime:     evEnd,
		Status:           string(cryptoPredictionEnum.MarketStatusActive),
		Category:         common.StrPtr200(category),
		SubCategory:      common.StrPtr200(subCategory),
		SportCN:          common.StrPtr128(common.SportCNForSubCategory(subCategory)),
		Slug:             slug,
		League:           common.StrPtr128(league),
		Competition:      comp,
		CompetitionEn:    compEn,
	}
}

func buildPolymarketEventURL(eventSlug string) string {
	slug := strings.TrimSpace(eventSlug)
	if slug == "" {
		return "https://polymarket.com"
	}
	return "https://polymarket.com/event/" + slug
}

func polymarketOutcomeFromMarket(
	platform, category, subCategory, fixtureID string,
	ev *httpentity.PolymarketEvent,
	m *httpentity.Market,
	comp *string,
	outcomeKind cryptoPredictionEnum.OutcomeMarketType,
) *persistententity.MarketOutcome {
	if m == nil || fixtureID == "" {
		return nil
	}
	yesTok, noTok := parsePolymarketClobTokenIds(m.ClobTokenIds)
	if yesTok == "" || noTok == "" {
		return nil
	}
	_, _, mktStart, mktEnd := extractPolymarketTimes(m)

	name := common.Truncate(m.Question, 2000)
	desc := common.Truncate(m.Description, 2000)
	cfg := persistententity.BuildMarketPlatformConfig(platform, m.NegRisk)
	var cfgPtr *string
	if cfg != "" {
		cfgPtr = &cfg
	}
	volTotal := m.VolumeNum
	vol24 := m.Volume24hrClob
	if vol24 == 0 {
		vol24 = m.Volume24hr
	}
	addr := m.ConditionId
	oid := strings.TrimSpace(m.ID)
	if oid == "" {
		return nil
	}
	if outcomeKind == "" {
		outcomeKind = cryptoPredictionEnum.OutcomeMarketTypeMoneyline
	}
	mktName := polymarketParentMarketName(ev, m)
	return &persistententity.MarketOutcome{
		Platform:                platform,
		PlatformMarketID:        fixtureID,
		OutcomeType:             string(outcomeKind),
		OutcomeName:             common.StrPtr(name),
		MarketName:              common.StrPtrOptional(mktName),
		Desc:                    common.StrPtr(desc),
		PlatformMarketOutcomeID: oid,
		MarketAddress:           common.StrPtr255(addr),
		MarketURL:               buildPolymarketMarketURL(m.Slug),
		YesTokenID:              yesTok,
		NoTokenID:               noTok,
		MarketStartTime:         common.StrPtr255(mktStart),
		MarketEndTime:           common.StrPtr255(mktEnd),
		Category:                common.StrPtr200(category),
		SubCategory:             common.StrPtr200(subCategory),
		VolumeTotal:             common.FloatPtr(volTotal),
		Volume24hr:              common.FloatPtr(vol24),
		Slug:                    common.StrPtr255(m.Slug),
		Competition:             comp,
		Config:                  cfgPtr,
	}
}

// polymarketFixtureIDFromFlatMarket ListMarkets 无 event 包裹时的合成场次 ID。
func polymarketFixtureIDFromFlatMarket(m *httpentity.Market) string {
	if m == nil {
		return ""
	}
	slug := strings.ToLower(strings.TrimSpace(m.Slug))
	if base := polySlugFixtureBase(slug); base != "" {
		return "slug:" + base
	}
	return strings.TrimSpace(m.ID)
}

func polySlugFixtureBase(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	for _, mark := range []string{"-spread-", "-total-"} {
		if i := strings.Index(slug, mark); i > 0 {
			return slug[:i]
		}
	}
	parts := strings.Split(slug, "-")
	if len(parts) < 4 {
		return ""
	}
	for i, p := range parts {
		if len(p) == 10 && p[4] == '-' && p[7] == '-' {
			if i+1 < len(parts)-1 {
				return strings.Join(parts[:len(parts)-1], "-")
			}
			return strings.Join(parts[:i+1], "-")
		}
	}
	// cbb/nba/cwbb 等常把日期拆成 YYYY-MM-DD 三段；不能误砍成 …-2026-11
	if base := polySlugBaseFromSplitDate(parts); base != "" {
		return base
	}
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return ""
}

func polySlugBaseFromSplitDate(parts []string) string {
	for i := 0; i+2 < len(parts); i++ {
		if !polySlugYearPart(parts[i]) || !polySlugMonthDay(parts[i+1], parts[i+2]) {
			continue
		}
		return strings.Join(parts[:i+3], "-")
	}
	return ""
}

func polySlugYearPart(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func polySlugMonthDay(month, day string) bool {
	if month == "" || day == "" || len(month) > 2 || len(day) > 2 {
		return false
	}
	for _, s := range []string{month, day} {
		for _, c := range s {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

func polymarketFixtureDisplayName(m *httpentity.Market) string {
	if m == nil {
		return ""
	}
	for _, e := range m.Events {
		title := strings.TrimSpace(e.Title)
		title = strings.TrimSpace(strings.TrimSuffix(title, " - More Markets"))
		if title != "" {
			return title
		}
	}
	q := strings.TrimSpace(m.Question)
	low := strings.ToLower(q)
	if strings.HasPrefix(low, "spread:") || strings.HasPrefix(low, "spread ") || strings.Contains(low, "o/u") || strings.Contains(low, "over/under") {
		return ""
	}
	return q
}

func polymarketFixtureFromFlatMarket(
	platform, category, subCategory string,
	m *httpentity.Market,
	comp, compEn *string,
) *persistententity.Market {
	fid := polymarketFixtureIDFromFlatMarket(m)
	if fid == "" {
		return nil
	}
	evStart, evEnd, _, _ := extractPolymarketTimes(m)
	baseSlug := polySlugFixtureBase(strings.ToLower(strings.TrimSpace(m.Slug)))
	var slugPtr *string
	if baseSlug != "" {
		slugPtr = common.StrPtr255(baseSlug)
	}
	name := polymarketFixtureDisplayName(m)
	urlSlug := strings.TrimSuffix(baseSlug, "-more-markets")
	return &persistententity.Market{
		Platform:         platform,
		Name:             common.StrPtr(common.Truncate(name, 2000)),
		PlatformMarketID: fid,
		MarketURL:        buildPolymarketEventURL(urlSlug),
		EventStartTime:   common.StrPtr255NilIfEmpty(evStart),
		EventEndTime:     common.StrPtr255NilIfEmpty(evEnd),
		Status:           string(cryptoPredictionEnum.MarketStatusActive),
		Category:         common.StrPtr200(category),
		SubCategory:      common.StrPtr200(subCategory),
		SportCN:          common.StrPtr128(common.SportCNForSubCategory(subCategory)),
		Slug:             slugPtr,
		League:           common.StrPtr128(extractPolymarketLeague(m)),
		Competition:      comp,
		CompetitionEn:    compEn,
	}
}
