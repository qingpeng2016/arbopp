package sxbet

import (
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestBuildSxbetEventURLFromMarket(t *testing.T) {
	m := httpentity.SxbetMarket{
		SportXeventID: "L20826652",
		SportID:       5,
		SportLabel:    "Soccer",
		LeagueLabel:   "Liga Profesional",
		Group1:        "Liga Profesional",
	}
	got := buildSxbetEventURLFromMarket(m)
	want := "https://sx.bet/soccer/liga-profesional/event/L20826652/1X2"
	if got != want {
		t.Fatalf("event url: got %q want %q", got, want)
	}
}

func TestBuildSxbetMarketURLFromMarket(t *testing.T) {
	m := httpentity.SxbetMarket{
		SportXeventID: "L20826652",
		SportID:       5,
		Group1:        "Liga Profesional",
		MarketHash:    "0xabc",
	}
	got := buildSxbetMarketURLFromMarket(m)
	want := "https://sx.bet/soccer/liga-profesional/event/L20826652/1X2"
	if got != want {
		t.Fatalf("market url: got %q want %q", got, want)
	}
}

func TestSlugify(t *testing.T) {
	if slugify("La Liga") != "la-liga" {
		t.Fatal(slugify("La Liga"))
	}
	if slugify("German Bundesliga") != "german-bundesliga" {
		t.Fatal(slugify("German Bundesliga"))
	}
}
