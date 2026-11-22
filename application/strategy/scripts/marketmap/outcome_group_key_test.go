package marketmap

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestOutcomeGroupKeyUnifiesSxbetWithPolyViaMapKey(t *testing.T) {
	mk := "v1|英超|epl-mac-ful-2026-11-21"
	polyName := "Will Manchester City win"
	sxName := "Manchester City wins"
	polySlug := "epl-mac-ful-2026-11-21-mac"
	comp := "英超"
	mkPtr := mk
	poly := &persistententity.MarketOutcome{
		Platform: appcommon.PlatformPolymarket, Competition: &comp,
		MapKey: &mkPtr, OutcomeName: &polyName, Slug: &polySlug,
	}
	sxSlug := "l20940405"
	sx := &persistententity.MarketOutcome{
		Platform: appcommon.PlatformSxbet, Competition: &comp,
		MapKey: &mkPtr, OutcomeName: &sxName, Slug: &sxSlug,
	}
	kPoly := OutcomeGroupKey(poly)
	kSx := OutcomeGroupKey(sx)
	if kPoly == "" || kSx == "" {
		t.Fatalf("keys empty poly=%q sx=%q", kPoly, kSx)
	}
	if kPoly != kSx {
		t.Fatalf("want same group key\ngot poly=%q\n   sx=%q", kPoly, kSx)
	}
	want := BuildOutcomeGroupKeyFromMapKey(mk, "manchester_city")
	if kPoly != want {
		t.Fatalf("want %q got %q", want, kPoly)
	}
}

func TestCanonicalFixtureIDSxbetPayload(t *testing.T) {
	key := "Sports|Premier League|x:l20940405"
	if got := canonicalFixtureID(key); got != "l20940405" {
		t.Fatalf("canonicalFixtureID sx: got %q", got)
	}
}