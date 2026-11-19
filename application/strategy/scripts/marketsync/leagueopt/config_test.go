package leagueopt

import "testing"

func TestParseLeagueConcurrency(t *testing.T) {
	cfg := Parse(`{"polymarket":{"concurrency":3},"kalshi":{"concurrency":99}}`, "json")
	if cfg.Polymarket.Concurrency() != 3 {
		t.Fatalf("poly %d", cfg.Polymarket.Concurrency())
	}
	if cfg.Kalshi.Concurrency() != maxLeagueConcurrency {
		t.Fatalf("kalshi %d", cfg.Kalshi.Concurrency())
	}
	if (Platform{}).Concurrency() != defaultLeagueConcurrency {
		t.Fatal("default")
	}
}
