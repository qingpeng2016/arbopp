package kalshi

import (
	"testing"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestKalshiSpreadSeriesMapsOntoGame(t *testing.T) {
	spread, ok := kalshiSpreadSeries("KXAFCONGAME")
	if !ok || spread != "KXAFCONSPREAD" {
		t.Fatalf("spread series: %q %v", spread, ok)
	}
	if _, ok := kalshiSpreadSeries("KXATPMATCH"); ok {
		t.Fatal("non-GAME series has no spread sibling")
	}
	got := kalshiGameEventTicker("KXAFCONGAME", "KXAFCONSPREAD-26NOV14DZATOG")
	if got != "KXAFCONGAME-26NOV14DZATOG" {
		t.Fatalf("game ticker: %s", got)
	}
	if kalshiMatchTitleFromSpread("Algeria vs Togo: Spread") != "Algeria vs Togo" {
		t.Fatal("spread title should keep the match name")
	}
}

func TestKalshiSpreadMarketIsHandicap(t *testing.T) {
	m := httpentity.KalshiMarket{Title: "Algeria wins by more than 1.5 goals?"}
	ok, kind := kalshiMarketForSync(m, syncopt.OutcomeTypeSet{Moneyline: true, Handicap: true})
	if !ok || kind != "handicap" {
		t.Fatalf("got %v %s", ok, kind)
	}
	ev := httpentity.KalshiEvent{
		Category:        "Sports",
		ProductMetadata: &httpentity.KalshiProductMetadata{CompetitionScope: "Spread"},
	}
	if !kalshiEventIsFullGameSpread(ev) {
		t.Fatal("full game spread should sync")
	}
	ev.ProductMetadata.CompetitionScope = "1st Half Spread"
	if kalshiEventIsFullGameSpread(ev) {
		t.Fatal("first half spread stays out")
	}
	if kalshiEventIsMatchMarket(httpentity.KalshiEvent{
		Category:        "Sports",
		ProductMetadata: &httpentity.KalshiProductMetadata{CompetitionScope: "Spread"},
	}) {
		t.Fatal("spread event is not its own fixture")
	}
}

func TestKalshiTotalSeriesMapsOntoGame(t *testing.T) {
	total, ok := kalshiTotalSeries("KXAFCONGAME")
	if !ok || total != "KXAFCONTOTAL" {
		t.Fatalf("total series: %q %v", total, ok)
	}
	got := kalshiGameEventTicker("KXEPLGAME", "KXEPLTOTAL-26NOV21MCIFUL")
	if got != "KXEPLGAME-26NOV21MCIFUL" {
		t.Fatalf("game ticker: %s", got)
	}
	if kalshiMatchTitleFromTotal("Manchester City vs Fulham: Total Goals") != "Manchester City vs Fulham" {
		t.Fatal("total title should keep the match name")
	}
	m := httpentity.KalshiMarket{Title: "Will over 2.5 goals be scored?"}
	okSync, kind := kalshiMarketForSync(m, syncopt.OutcomeTypeSet{Moneyline: true, Handicap: true, Total: true})
	if !okSync || kind != "total" {
		t.Fatalf("got %v %s", okSync, kind)
	}
	ev := httpentity.KalshiEvent{
		Category:        "Sports",
		ProductMetadata: &httpentity.KalshiProductMetadata{CompetitionScope: "Point Total"},
	}
	if !kalshiEventIsFullGameTotal(ev) {
		t.Fatal("full game total should sync")
	}
	ev.ProductMetadata.CompetitionScope = "Team Total"
	if kalshiEventIsFullGameTotal(ev) {
		t.Fatal("team total stays out")
	}
}
