package kalshi

import (
	"testing"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func TestDateFromEventTicker(t *testing.T) {
	tm, ok := dateFromEventTicker("KXINTLFRIENDLYGAME-26NOV17TAHCOK")
	if !ok || tm.UTC().Format("2006-01-02") != "2026-11-17" {
		t.Fatalf("date=%v ok=%v", tm, ok)
	}
}

func TestKalshiFixtureTimesUsesGameDayMoneyline(t *testing.T) {
	markets := []httpentity.KalshiMarket{
		{Status: "active", Title: "Tahiti wins", OpenTime: "2026-11-14T06:05:00Z", CloseTime: "2026-11-19T05:30:00Z"},
		{Status: "active", Title: "Tahiti wins", OpenTime: "2026-11-14T06:05:00Z", CloseTime: "2026-11-17T05:30:00Z"},
		{Status: "active", Title: "Cook Islands wins", OpenTime: "2026-11-14T06:05:00Z", CloseTime: "2026-11-17T07:45:00Z"},
		{Status: "settled", Title: "Old", OpenTime: "2026-11-01T00:00:00Z", CloseTime: "2026-11-01T01:00:00Z"},
	}
	start, end := kalshiFixtureTimes("kxintlfriendlygame-26nov17tahcok", markets)
	if start != "2026-11-17 05:30:00" {
		t.Fatalf("start=%q", start)
	}
	if end != "2026-11-17 07:45:00" {
		t.Fatalf("end=%q", end)
	}
}

func TestKalshiFixtureTimesSlugDayWhenCloseFarAfterGameDay(t *testing.T) {
	// 实盘：胜平负关盘在赛后数日，旧逻辑会落回 open 11-14 / close 11-19。
	markets := []httpentity.KalshiMarket{
		{Status: "active", Title: "Tahiti wins", OpenTime: "2026-11-14T06:05:00Z", CloseTime: "2026-11-19T05:30:00Z"},
		{Status: "active", Title: "Cook Islands wins", OpenTime: "2026-11-14T06:05:00Z", CloseTime: "2026-11-19T05:30:00Z"},
		{Status: "active", Title: "Tie is the result", OpenTime: "2026-11-14T06:05:00Z", CloseTime: "2026-11-19T05:30:00Z"},
	}
	start, end := kalshiFixtureTimes("KXINTLFRIENDLYGAME-26NOV17TAHCOK", markets)
	if start == "2026-11-14 06:05:00" || end == "2026-11-19 05:30:00" {
		t.Fatalf("still open/close trading window: start=%q end=%q", start, end)
	}
	if start != "2026-11-17 00:00:00" || end != "2026-11-17 02:30:00" {
		t.Fatalf("slug day default: start=%q end=%q", start, end)
	}
}

func TestKalshiFixtureTimesFallbackTradingWindow(t *testing.T) {
	markets := []httpentity.KalshiMarket{
		{Status: "active", OpenTime: "2026-11-17T12:00:00Z", CloseTime: "2026-11-17T14:00:00Z"},
		{Status: "active", OpenTime: "2026-11-16T10:00:00Z", CloseTime: "2026-11-17T15:00:00Z"},
	}
	start, end := kalshiFixtureTimes("KXUNKNOWN", markets)
	if start != "2026-11-16 10:00:00" {
		t.Fatalf("start=%q", start)
	}
	if end != "2026-11-17 15:00:00" {
		t.Fatalf("end=%q", end)
	}
}