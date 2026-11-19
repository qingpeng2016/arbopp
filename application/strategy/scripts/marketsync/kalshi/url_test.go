package kalshi

import "testing"

func TestBuildKalshiMarketURL_LevanteElche(t *testing.T) {
	got := buildKalshiMarketURL(
		"KXLALIGAGAME-26NOV23LEVELC-ELC",
		"KXLALIGAGAME-26NOV23LEVELC",
		"Levante vs Elche",
	)
	want := "https://kalshi.com/markets/kxlaligagame/levante-vs-elche/kxlaligagame-26nov23levelc/kxlaligagame-26nov23levelc-elc"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestKalshiWebPathSlug(t *testing.T) {
	if kalshiWebPathSlug("Bilbao vs. Espanyol") != "bilbao-vs-espanyol" {
		t.Fatal("bilbao slug")
	}
}
