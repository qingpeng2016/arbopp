package predict

import "testing"

func TestBuildPredictFixtureURL(t *testing.T) {
	got := buildPredictFixtureURL("nhl-nyi-det-2026-11-21")
	want := "https://predict.fun/market/nhl-nyi-det-2026-11-21"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}