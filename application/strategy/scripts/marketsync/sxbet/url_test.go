package sxbet

import (
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestBuildSxbetEventURLFromMarket(t *testing.T) {
	m := httpentity.SxbetMarket{
		SportXeventID: "L20940405",
		SportID:       8,
		SportLabel:    "Football",
		LeagueLabel:   "NCAAF",
		Group1:        "NCAAF",
	}
	got := buildSxbetEventURLFromMarket(m)
	want := "https://sx.bet/sports/football/ncaaf/L20940405"
	if got != want {
		t.Fatalf("event url: got %q want %q", got, want)
	}
}

func TestBuildSxbetMarketURLFromMarket(t *testing.T) {
	m := httpentity.SxbetMarket{
		SportXeventID: "L20940405",
		SportID:       8,
		Group1:        "NCAAF",
		MarketHash:    "0xabc",
	}
	got := buildSxbetMarketURLFromMarket(m)
	want := "https://sx.bet/sports/football/ncaaf/L20940405/market/0xabc"
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