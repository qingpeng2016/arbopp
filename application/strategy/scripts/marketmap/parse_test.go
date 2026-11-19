package marketmap

import (
	"strings"
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

const testCategorySports = "Sports"
const testSubFootball = "Football"

func testMatchScope(competitionEn string) string {
	return leagueBucketKey(testCategorySports, testSubFootball, competitionEn)
}

func testMarket(platform, comp, compEn, slug string) *persistententity.Market {
	cat, sub := testCategorySports, testSubFootball
	return &persistententity.Market{
		Platform:      platform,
		Category:      &cat,
		SubCategory:   &sub,
		Competition:   &comp,
		CompetitionEn: &compEn,
		Slug:          &slug,
	}
}

func TestOutcomeTeamFromName(t *testing.T) {
	team, ok := outcomeTeamFromName("Will Arsenal win on 2026-11-15?")
	if !ok || normalizeTeam(team) != "arsenal" {
		t.Fatalf("poly: got %q ok=%v", team, ok)
	}
	team, ok = outcomeTeamFromName("Arsenal vs Chelsea — Arsenal")
	if !ok || normalizeTeam(team) != "arsenal" {
		t.Fatalf("kalshi em dash: got %q ok=%v", team, ok)
	}
}

func TestKalshiSlugDate(t *testing.T) {
	tm, ok := dateFromKalshiEventSlug("kxeplgame-26nov21avlsun")
	if !ok || tm.Format("2006-01-02") != "2026-11-21" {
		t.Fatalf("kalshi slug date: %v ok=%v", tm, ok)
	}
}

func TestPolySlugFixtureBaseKeepsEventDate(t *testing.T) {
	if got := polySlugFixtureBase("epl-ast-sun-2026-11-21"); got != "epl-ast-sun-2026-11-21" {
		t.Fatalf("event slug chopped: %s", got)
	}
	if got := polySlugFixtureBase("epl-ast-sun-2026-11-21-ast"); got != "epl-ast-sun-2026-11-21" {
		t.Fatalf("outcome slug: %s", got)
	}
	scope := testMatchScope("Premier League")
	keys := map[string]struct{}{
		scope + "|k:kxeplgame-26nov21avlsun": {},
		scope + "|p:epl-ast-sun-2026-11-21":  {},
	}
	linked := linkPlatformFixtures(keys, nil, nil)
	kK, kP := scope+"|k:kxeplgame-26nov21avlsun", scope+"|p:epl-ast-sun-2026-11-21"
	if linked[kK] != linked[kP] {
		t.Fatalf("not linked: %#v", linked)
	}
}

func TestPlatformFixtureKeySeparatesCompetitionEn(t *testing.T) {
	comp := "英超"
	en := "Premier League"
	other := "EPL"
	slug := "epl-ast-sun-2026-11-21"
	poly := testMarket(appcommon.PlatformPolymarket, comp, en, slug)
	kalshi := testMarket(appcommon.PlatformKalshi, comp, other, slug)
	_, enA, keyA, okA := platformFixtureKeyFromMarket(poly)
	_, enB, keyB, okB := platformFixtureKeyFromMarket(kalshi)
	if !okA || !okB || enA != en || enB != other {
		t.Fatalf("keys ok=%v %v en=%s %s", okA, okB, enA, enB)
	}
	if leagueBucketKey(testCategorySports, testSubFootball, enA) == leagueBucketKey(testCategorySports, testSubFootball, enB) {
		t.Fatalf("different competition_en landed in one bucket: %s %s", keyA, keyB)
	}
	blank := ""
	missing := testMarket(appcommon.PlatformPolymarket, comp, blank, slug)
	if _, _, _, ok := platformFixtureKeyFromMarket(missing); ok {
		t.Fatal("empty competition_en should stay out of compare")
	}
}

func TestLinkMatchesWhenDatesDiffer(t *testing.T) {
	scope := testMatchScope("Premier League")
	keys := map[string]struct{}{
		scope + "|k:kxeplgame-26nov22avlsun": {},
		scope + "|p:epl-ast-sun-2026-11-21":  {},
	}
	linked := linkPlatformFixtures(keys, nil, nil)
	kK, kP := scope+"|k:kxeplgame-26nov22avlsun", scope+"|p:epl-ast-sun-2026-11-21"
	if linked[kK] != linked[kP] {
		t.Fatalf("date gap rejected unique teams: %#v", linked)
	}
}

func TestLinkSplitsRepeatFixturesByClosestDate(t *testing.T) {
	scope := testMatchScope("Premier League")
	novK := scope + "|k:kxeplgame-26nov21avlsun"
	decK := scope + "|k:kxeplgame-26dec21avlsun"
	novP := scope + "|p:epl-ast-sun-2026-11-22"
	decP := scope + "|p:epl-ast-sun-2026-12-20"
	linked := linkPlatformFixtures(map[string]struct{}{
		novK: {}, decK: {}, novP: {}, decP: {},
	}, nil, nil)
	if linked[novK] != linked[novP] || linked[decK] != linked[decP] {
		t.Fatalf("repeat fixtures crossed: %#v", linked)
	}
	if linked[novK] == linked[decK] {
		t.Fatalf("repeat fixtures collapsed: %#v", linked)
	}
}

func TestFixtureNamesMatchJ1ShortNames(t *testing.T) {
	pairs := [][2]string{
		{"Kashima vs Shimizu", "Kashima Antlers vs. Shimizu S-Pulse"},
		{"Tokyo V vs Nagoya", "Tōkyō Verdy vs. Nagoya Grampus"},
		{"Machida Z vs V-Varen", "FC Machida Zelvia vs. V-Varen Nagasaki"},
		{"Hiroshima vs Marinos", "Sanfrecce Hiroshima vs. Yokohama F·Marinos"},
		{"Kyoto Sanga vs Urawa", "Kyōto Sanga FC vs. Urawa Red Diamonds"},
		{"Cerezo vs Tokyo", "Cerezo Ōsaka vs. FC Tōkyō"},
		{"Kobe vs Mito H", "Vissel Kōbe vs. FC Mito Holly Hock"},
	}
	for _, p := range pairs {
		if !fixtureNamesMatch(p[0], p[1]) {
			t.Fatalf("names should match: %q vs %q", p[0], p[1])
		}
	}
	if fixtureNamesMatch("Kashima vs Shimizu", "Tōkyō Verdy vs. Nagoya Grampus") {
		t.Fatal("different fixtures matched by name")
	}
	if fixtureNamesMatch("Machida Z vs V-Varen", "V-Varen Nagasaki vs. Kawasaki Frontale") {
		t.Fatal("shared V-Varen must not link a different opponent")
	}
	scope := testMatchScope("J1")
	keys := map[string]struct{}{
		scope + "|k:kxjleaguegame-26nov20kasshi": {},
		scope + "|p:jap-kas-ssp-2026-11-20":      {},
	}
	names := map[string]string{
		scope + "|k:kxjleaguegame-26nov20kasshi": "Kashima vs Shimizu",
		scope + "|p:jap-kas-ssp-2026-11-20":      "Kashima Antlers vs. Shimizu S-Pulse",
	}
	linked := linkPlatformFixtures(keys, names, nil)
	if linked[scope+"|k:kxjleaguegame-26nov20kasshi"] != linked[scope+"|p:jap-kas-ssp-2026-11-20"] {
		t.Fatalf("j1 name link failed: %#v", linked)
	}
}

func TestKalshiBlobMustNotMatchSubstringBouInFulbou(t *testing.T) {
	if matchKalshiPolyFixture("kxeplgame-26nov29fulbou", "epl-bou-not-2026-11-21") {
		t.Fatal("ful-bou Kalshi must not match bou-not Poly slug")
	}
	if !matchKalshiPolyFixture("kxeplgame-26nov21bounfo", "epl-bou-not-2026-11-21") {
		t.Fatal("bounfo should still match bou-not")
	}
}

func TestLinkBouNotDoesNotAbsorbFulBouKalshi(t *testing.T) {
	scope := testMatchScope("Premier League")
	polyBouNot := scope + "|p:epl-bou-not-2026-11-21"
	predictBouNot := scope + "|f:epl-bou-not-2026-11-21"
	kalshiBouNot := scope + "|k:kxeplgame-26nov21bounfo"
	kalshiFulBou := scope + "|k:kxeplgame-26nov29fulbou"
	linked := linkPlatformFixtures(map[string]struct{}{
		polyBouNot: {}, predictBouNot: {}, kalshiBouNot: {}, kalshiFulBou: {},
	}, map[string]string{
		kalshiBouNot: "Bournemouth vs Nottingham Forest",
		kalshiFulBou: "Fulham vs Bournemouth",
		polyBouNot:   "AFC Bournemouth vs. Nottingham Forest FC",
	}, nil)
	if linked[kalshiFulBou] == linked[polyBouNot] {
		t.Fatalf("ful-bou kalshi merged into bou-not: %#v", linked)
	}
	if linked[kalshiBouNot] != linked[polyBouNot] || linked[polyBouNot] != linked[predictBouNot] {
		t.Fatalf("bou-not trio should stay together: %#v", linked)
	}
	if linked[polyBouNot] != "epl-bou-not-2026-11-21" {
		t.Fatalf("canonical should be poly slug, got %q", linked[polyBouNot])
	}
}

func TestMatchKalshiPolyFixture(t *testing.T) {
	cases := []struct{ k, p string }{
		{"kxeplgame-26nov21avlsun", "epl-ast-sun-2026-11-21"},
		{"kxeplgame-26nov21bounfo", "epl-bou-not-2026-11-21"},
		{"kxeplgame-26nov22lfcmun", "epl-liv-mun-2026-11-22"},
		{"kxbundesligagame-26nov21levpad", "bun-b04-scp-2026-11-21"},
		{"kxserieagame-26nov22bfcudi", "sea-bol-udi-2026-11-22"},
		{"kxlaligagame-26nov20athesp", "lal-bil-esp-2026-11-20"},
		{"kxeplgame-26nov22avlsun", "epl-ast-sun-2026-11-21"},
	}
	for _, c := range cases {
		if !matchKalshiPolyFixture(c.k, c.p) {
			t.Fatalf("fixture match failed %s vs %s", c.k, c.p)
		}
	}
}

func TestCanonicalOutcomeNormPolyCode(t *testing.T) {
	n := canonicalOutcomeNorm(appcommon.PlatformPolymarket, "Will Bayer 04 Leverkusen win", "bun-b04-scp-2026-11-21-b04")
	if n != "leverkusen" {
		t.Fatalf("poly code outcome: %q", n)
	}
	n = canonicalOutcomeNorm(appcommon.PlatformKalshi, "Leverkusen wins", "")
	if n != "leverkusen" {
		t.Fatalf("kalshi outcome: %q", n)
	}
}

func TestBuildMapKeyLength(t *testing.T) {
	key := buildMapKey("英超", "epl-ast-sun-2026-11-21", "arsenal")
	if key == "" || len(key) > 192 {
		t.Fatalf("key=%q len=%d", key, len(key))
	}
}

func TestLinkThreePlatformsWhenPolyPredictPrelinked(t *testing.T) {
	// 模拟线上：Poly+Predict 已同 slug；Kalshi 仍在图外。修复前 Kalshi 度数为 2 无法 assign。
	scope := testMatchScope("Premier League")
	polyBase := "epl-ast-sun-2026-11-21"
	kalshi := scope + "|k:kxeplgame-26nov21avlsun"
	poly := scope + "|p:" + polyBase
	predict := scope + "|f:" + polyBase
	names := map[string]string{
		kalshi: "Aston Villa vs Sunderland",
		poly:   "Aston Villa FC vs. Sunderland AFC",
	}
	linked := linkPlatformFixtures(map[string]struct{}{kalshi: {}, poly: {}, predict: {}}, names, nil)
	if linked[kalshi] != linked[poly] || linked[poly] != linked[predict] {
		t.Fatalf("avlsun 三平台未合并: %#v", linked)
	}
	if linked[poly] != polyBase {
		t.Fatalf("canonical want poly slug, got %q", linked[poly])
	}
}

func TestLinkPredictPolyKalshiTogether(t *testing.T) {
	scope := testMatchScope("Premier League")
	slug := "epl-bre-eve-2026-11-23"
	keys := map[string]struct{}{
		scope + "|p:" + slug:                           {},
		scope + "|f:" + slug:                           {},
		scope + "|k:kxeplgame-26nov23breeve":           {},
	}
	linked := linkPlatformFixtures(keys, nil, nil)
	pK, fK, kK := scope+"|p:"+slug, scope+"|f:"+slug, scope+"|k:kxeplgame-26nov23breeve"
	if linked[pK] != linked[fK] || linked[fK] != linked[kK] {
		t.Fatalf("three-platform link failed: %#v", linked)
	}
}

func TestGroupByCompetitionEnIgnoresCnName(t *testing.T) {
	en := "Premier League"
	cnA, cnB := "英超", "English Premier League"
	slug := "epl-ast-sun-2026-11-21"
	poly := testMarket(appcommon.PlatformPolymarket, cnA, en, slug)
	predict := testMarket(appcommon.PlatformPredict, cnB, en, slug)
	_, _, keyP, okP := platformFixtureKeyFromMarket(poly)
	_, _, keyF, okF := platformFixtureKeyFromMarket(predict)
	if !okP || !okF {
		t.Fatal("keys expected")
	}
	wantScope := testMatchScope(en)
	if wantScope != strings.Split(keyP, "|p:")[0] || wantScope != strings.Split(keyF, "|f:")[0] {
		t.Fatalf("scope mismatch: %s %s", keyP, keyF)
	}
}

func TestPlatformFixtureKeySeparatesSubCategory(t *testing.T) {
	comp, en, slug := "英超", "Premier League", "epl-ast-sun-2026-11-21"
	cat := testCategorySports
	subA, subB := testSubFootball, "Basketball"
	mA := &persistententity.Market{Platform: appcommon.PlatformPolymarket, Category: &cat, SubCategory: &subA, Competition: &comp, CompetitionEn: &en, Slug: &slug}
	mB := &persistententity.Market{Platform: appcommon.PlatformPolymarket, Category: &cat, SubCategory: &subB, Competition: &comp, CompetitionEn: &en, Slug: &slug}
	_, _, keyA, okA := platformFixtureKeyFromMarket(mA)
	_, _, keyB, okB := platformFixtureKeyFromMarket(mB)
	if !okA || !okB {
		t.Fatal("keys expected")
	}
	if strings.Split(keyA, "|p:")[0] == strings.Split(keyB, "|p:")[0] {
		t.Fatalf("same competition_en different sub_category must not share bucket: %s %s", keyA, keyB)
	}
}

func TestBuildFixtureMapKeyLength(t *testing.T) {
	key := buildFixtureMapKey("英超", "epl-ast-sun-2026-11-21")
	if key == "" || len(key) > 192 {
		t.Fatalf("key=%q len=%d", key, len(key))
	}
	if !strings.HasPrefix(key, "v1|") {
		t.Fatalf("key=%q", key)
	}
}
