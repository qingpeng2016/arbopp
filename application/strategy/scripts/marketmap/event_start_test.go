package marketmap

import "testing"

func TestFixturePairStartOKRejectsDayApart(t *testing.T) {
	scope := "FIFA Friendlies"
	poly := scope + "|p:fif-tah-cok-2026-11-17"
	kOld := scope + "|k:kxintlfriendlygame-26nov16tahcok"
	kNew := scope + "|k:kxintlfriendlygame-26nov17tahcok"
	starts := map[string]string{
		poly:   "2026-11-17 05:30:00",
		kOld:   "2026-11-16 12:00:00",
		kNew:   "2026-11-17 05:00:00",
	}
	if fixturePairStartOK(kOld, poly, starts) {
		t.Fatal("16th kalshi should not pair with 17th poly kickoff")
	}
	if !fixturePairStartOK(kNew, poly, starts) {
		t.Fatal("17th kalshi should pair with poly")
	}
}

func TestLinkSplitsKalshiByEventStart(t *testing.T) {
	scope := "FIFA Friendlies"
	poly := scope + "|p:fif-tah-cok-2026-11-17"
	kOld := scope + "|k:kxintlfriendlygame-26nov16tahcok"
	kNew := scope + "|k:kxintlfriendlygame-26nov17tahcok"
	names := map[string]string{
		poly: "Tahiti vs. Cook Islands",
		kOld: "Tahiti vs Cook Islands",
		kNew: "Tahiti vs Cook Islands",
	}
	starts := map[string]string{
		poly: "2026-11-17 05:30:00",
		kOld: "2026-11-16 12:00:00",
		kNew: "2026-11-17 05:00:00",
	}
	linked := linkPlatformFixtures(map[string]struct{}{poly: {}, kOld: {}, kNew: {}}, names, starts)
	if linked[kOld] == linked[poly] {
		t.Fatalf("old kalshi merged into poly group: %#v", linked)
	}
	if linked[kNew] != linked[poly] {
		t.Fatalf("new kalshi should merge with poly: %#v", linked)
	}
}