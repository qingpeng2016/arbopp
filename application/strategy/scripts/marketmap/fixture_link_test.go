package marketmap

import "testing"

func TestPolyPredictFixtureSlugMatch_rejectsCbbCwbb(t *testing.T) {
	if polyPredictFixtureSlugMatch("cbb-alaam-nal-2026-11-24", "cbb-alaam-nal-2026-11-24") != true {
		t.Fatal("same slug")
	}
	if polyPredictFixtureSlugMatch("cwbb-gardwb-ncar-2026-11-24", "cbb-alaam-nal-2026-11-24") {
		t.Fatal("cwbb poly must not match cbb predict")
	}
}

func TestCrossPlatformFixturePairMatches_polyPredict_slugOnly(t *testing.T) {
	ok := crossPlatformFixturePairMatches(
		platformPoly, platformPredict,
		"cwbb-gardwb-ncar-2026-11-24", "cbb-alaam-nal-2026-11-24",
		"Gardner-Webb vs North Carolina", "Alabama A&M vs North Alabama",
	)
	if ok {
		t.Fatal("must not match on title alone")
	}
	ok = crossPlatformFixturePairMatches(
		platformPoly, platformPredict,
		"cbb-alaam-nal-2026-11-24", "cbb-alaam-nal-2026-11-24",
		"a", "b",
	)
	if !ok {
		t.Fatal("same fixture slug should match")
	}
}