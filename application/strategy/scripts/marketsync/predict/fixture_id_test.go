package predict

import "testing"

func TestPredictMainFixtureID(t *testing.T) {
	const main = "epl-bre-eve-2026-11-23"
	if got := predictMainFixtureID(main); got != main {
		t.Fatalf("main: got %q", got)
	}
	if got := predictMainFixtureID(main + "-more-markets"); got != main {
		t.Fatalf("more-markets: got %q", got)
	}
}