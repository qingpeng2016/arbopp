package polymarket

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestPolymarketFixtureFromFlatMarket_spreadUsesEventTitle(t *testing.T) {
	m := &httpentity.Market{
		Question: "Spread: Togo (-5.5)",
		Slug:     "afcq-alg-tog-2026-11-14-spread-home-5pt5",
		Events: []httpentity.MarketEventEmbed{{
			Title: "Algeria vs. Togo - More Markets",
			Slug:  "afcq-alg-tog-2026-11-14-more-markets",
		}},
	}
	fx := polymarketFixtureFromFlatMarket(appcommon.PlatformPolymarket, "Sports", "Football", m, nil, nil)
	if fx == nil || fx.Name == nil || *fx.Name != "Algeria vs. Togo" {
		t.Fatalf("name=%v", fx)
	}
	if fx.MarketURL != "https://polymarket.com/event/afcq-alg-tog-2026-11-14" {
		t.Fatalf("url=%s", fx.MarketURL)
	}
}

func TestPolySlugFixtureBase_splitDate(t *testing.T) {
	cases := []struct{ slug, want string }{
		{"cbb-merc-presb-2026-11-18", "cbb-merc-presb-2026-11-18"},
		{"cbb-merc-presb-2026-11-18-merc", "cbb-merc-presb-2026-11-18"},
		{"nba-chi-bkn-2026-11-18-chi", "nba-chi-bkn-2026-11-18"},
		{"epl-mac-ful-2026-11-21-gal", "epl-mac-ful-2026-11-21"},
	}
	for _, tc := range cases {
		if got := polySlugFixtureBase(tc.slug); got != tc.want {
			t.Fatalf("slug=%q got=%q want=%q", tc.slug, got, tc.want)
		}
	}
}

func TestPolymarketFixtureMarketID_unifiesMoreMarkets(t *testing.T) {
	mainEv := &httpentity.PolymarketEvent{ID: "1337800", Slug: "epl-mac-ful-2026-11-21"}
	moreEv := &httpentity.PolymarketEvent{ID: "1338402", Slug: "epl-mac-ful-2026-11-21-more-markets"}
	fMain := polymarketFixtureMarketID(mainEv)
	fMore := polymarketFixtureMarketID(moreEv)
	want := "slug:epl-mac-ful-2026-11-21"
	if fMain != want || fMore != want {
		t.Fatalf("fixture ids mismatch: main=%q more=%q want %q", fMain, fMore, want)
	}
}
