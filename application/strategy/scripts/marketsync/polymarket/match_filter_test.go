package polymarket

import (
	"testing"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestPolymarketMarketForSyncFootball(t *testing.T) {
	sub := cryptoPredictionEnum.SubMarketCategoryFootball
	win := &httpentity.Market{
		Slug:          "epl-arsenal-chelsea-2026-01-01",
		Question:      "Will Arsenal win on 2026-01-01?",
		GameStartTime: "2026-01-01T15:00:00Z",
	}
	draw := &httpentity.Market{
		Slug:          "epl-arsenal-chelsea-2026-01-01-draw",
		Question:      "Will Arsenal vs. Chelsea end in a draw?",
		GameStartTime: "2026-01-01T15:00:00Z",
	}
	spread := &httpentity.Market{
		Slug:          "epl-arsenal-chelsea-2026-01-01-spread-home-1pt5",
		Question:      "Spread: Arsenal (-1.5)",
		GameStartTime: "2026-01-01T15:00:00Z",
	}
	mlOnly := syncopt.OutcomeTypeSet{Moneyline: true}
	both := syncopt.OutcomeTypeSet{Moneyline: true, Handicap: true}

	if ok, _ := polymarketMarketForSync(win, mlOnly, sub); !ok {
		t.Fatal("expected win market")
	}
	if ok, _ := polymarketMarketForSync(draw, mlOnly, sub); !ok {
		t.Fatal("expected draw market")
	}
	if ok, _ := polymarketMarketForSync(spread, mlOnly, sub); ok {
		t.Fatal("moneyline-only should exclude spread")
	}
	if ok, _ := polymarketMarketForSync(spread, both, sub); !ok {
		t.Fatal("expected spread when handicap enabled")
	}
	total := &httpentity.Market{
		Slug:          "epl-arsenal-chelsea-2026-01-01-total-2pt5",
		Question:      "Arsenal vs. Chelsea: O/U 2.5",
		GameStartTime: "2026-01-01T15:00:00Z",
	}
	withTotal := syncopt.OutcomeTypeSet{Moneyline: true, Handicap: true, Total: true}
	if ok, kind := polymarketMarketForSync(total, withTotal, sub); !ok || kind != "total" {
		t.Fatalf("expected total, got %v %s", ok, kind)
	}
	if ok, _ := polymarketMarketForSync(total, both, sub); ok {
		t.Fatal("total stays out when the type is off")
	}
	if ok, _ := polymarketMarketForSync(draw, syncopt.OutcomeTypeSet{Moneyline: true}, cryptoPredictionEnum.SubMarketCategoryBasketball); ok {
		t.Fatal("basketball filter must exclude draw")
	}

	uclDesc := "In the upcoming UEFA Champions League game between Sporting CP and Manchester United FC, originally scheduled for Wednesday, November 25, 2026."
	uclWin := &httpentity.Market{
		Slug:          "ucl-spo-mnu-2026-11-25-spo",
		Question:      "Will Sporting CP win on 2026-11-25?",
		Description:   uclDesc,
		GameStartTime: "2026-11-25 20:00:00+00",
	}
	uclSpread := &httpentity.Market{
		Slug:          "ucl-spo-mnu-2026-11-25-spread-home-1pt5",
		Question:      "Spread: Sporting CP (-1.5)",
		Description:   uclDesc,
		GameStartTime: "2026-11-25 20:00:00+00",
	}
	futures := &httpentity.Market{
		Slug:          "will-sporting-cp-win-the-2026-27-uefa-champions-league",
		Question:      "Will Sporting CP win the 2026-27 UEFA Champions League?",
		GameStartTime: "2026-11-25T20:00:00Z",
	}
	if ok, _ := polymarketMarketForSync(uclWin, mlOnly, sub); !ok {
		t.Fatal("ucl match description must not trip champion filter")
	}
	if ok, _ := polymarketMarketForSync(uclSpread, both, sub); !ok {
		t.Fatal("ucl spread description must not trip champion filter")
	}
	if ok, _ := polymarketMarketForSync(futures, mlOnly, sub); ok {
		t.Fatal("champions league outright question must stay excluded")
	}
}

func TestPolymarketMarketForSyncUsSports(t *testing.T) {
	usML := syncopt.OutcomeTypeSet{Moneyline: true}
	nba := &httpentity.Market{
		Slug:          "nba-min-lac-2026-12-02",
		Question:      "Timberwolves vs. Clippers",
		GameStartTime: "2026-12-03T03:30:00Z",
	}
	nhl := &httpentity.Market{
		Slug:          "nhl-sea-sjs-2026-11-18",
		Question:      "Kraken vs. Sharks",
		GameStartTime: "2026-11-19T03:00:00Z",
	}
	ufc := &httpentity.Market{
		Slug:          "ufc-fight-night-123-main",
		Question:      "UFC Fight Night: Fighter A vs. Fighter B (Lightweight)",
		GameStartTime: "2026-11-20T02:00:00Z",
	}
	if ok, _ := polymarketMarketForSync(nba, usML, cryptoPredictionEnum.SubMarketCategoryBasketball); !ok {
		t.Fatal("nba vs. moneyline")
	}
	if ok, _ := polymarketMarketForSync(nhl, usML, cryptoPredictionEnum.SubMarketCategoryNHL); !ok {
		t.Fatal("nhl vs. moneyline")
	}
	if ok, _ := polymarketMarketForSync(ufc, usML, cryptoPredictionEnum.SubMarketCategoryMMA); !ok {
		t.Fatal("ufc vs. moneyline")
	}
	// 跨类 slug 黑名单只在 football 文件内；篮球 sync 不应误杀 nba-，足球 sync 要挡 nba-
	if ok, _ := polymarketMarketForSync(nba, usML, cryptoPredictionEnum.SubMarketCategoryFootball); ok {
		t.Fatal("football filter must reject nba slug")
	}
}