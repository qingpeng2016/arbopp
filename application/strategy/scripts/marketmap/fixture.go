package marketmap

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	msynccommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// platformFixtureKeyFromMarket 从场次行取平台内 fixture 键；仅在同一 category+sub_category+competition_en 桶内跨平台对齐。
func platformFixtureKeyFromMarket(m *persistententity.Market) (competition, competitionEn, key string, ok bool) {
	if m == nil || m.Competition == nil || m.CompetitionEn == nil || m.Slug == nil {
		return "", "", "", false
	}
	competition = strings.TrimSpace(*m.Competition)
	competitionEn = strings.TrimSpace(*m.CompetitionEn)
	category := ""
	if m.Category != nil {
		category = strings.TrimSpace(*m.Category)
	}
	subCategory := ""
	if m.SubCategory != nil {
		subCategory = strings.TrimSpace(*m.SubCategory)
	}
	slug := strings.ToLower(strings.TrimSpace(*m.Slug))
	if competition == "" || competitionEn == "" || slug == "" {
		return "", "", "", false
	}
	scope := msynccommon.MapMatchScopeKey(category, subCategory, competitionEn)
	if scope == "" {
		return "", "", "", false
	}
	switch strings.ToUpper(strings.TrimSpace(m.Platform)) {
	case appcommon.PlatformKalshi:
		return competition, competitionEn, scope + "|k:" + slug, true
	case appcommon.PlatformPolymarket:
		base := polySlugFixtureBase(slug)
		if base == "" {
			base = slug
		}
		return competition, competitionEn, scope + "|p:" + base, true
	case appcommon.PlatformPredict:
		base := polySlugFixtureBase(slug)
		if base == "" {
			base = slug
		}
		return competition, competitionEn, scope + "|f:" + base, true
	case appcommon.PlatformSxbet:
		return competition, competitionEn, scope + "|x:" + slug, true
	default:
		return "", "", "", false
	}
}

func leagueBucketKey(category, subCategory, competitionEn string) string {
	return msynccommon.MapMatchScopeKey(category, subCategory, competitionEn)
}

// platformFixtureKey 平台内场次标识（Kalshi 用整场 event slug；Poly 用 slug 去掉 outcome 后缀）。
func platformFixtureKey(m *persistententity.MarketOutcome) (competition, key string, ok bool) {
	if m.Competition == nil {
		return "", "", false
	}
	competition = strings.TrimSpace(*m.Competition)
	if competition == "" || m.Slug == nil {
		return "", "", false
	}
	slug := strings.ToLower(strings.TrimSpace(*m.Slug))
	if slug == "" {
		return "", "", false
	}
	switch strings.ToUpper(strings.TrimSpace(m.Platform)) {
	case appcommon.PlatformKalshi:
		return competition, competition + "|k:" + slug, true
	case appcommon.PlatformPolymarket:
		base := polySlugFixtureBase(slug)
		if base == "" {
			return "", "", false
		}
		return competition, competition + "|p:" + base, true
	case appcommon.PlatformPredict:
		base := polySlugFixtureBase(slug)
		if base == "" {
			return "", "", false
		}
		return competition, competition + "|f:" + base, true
	case appcommon.PlatformSxbet:
		return competition, competition + "|x:" + slug, true
	default:
		return "", "", false
	}
}

type platformKind int

const (
	platformKalshi platformKind = iota
	platformPoly
	platformPredict
	platformSxbet

	// maxKalshiPolyAssignHours 唯一 Kalshi↔Poly 边若日期可解析且差超过此值，视为误配（如 bou-not 21 日 vs ful-bou 29 日）。
	maxKalshiPolyAssignHours = 72
	// maxCrossPlatformEventStartHours 两边都有 event_start_time 时，超过此差值不成边（区分改期/重复场次，如 11-16 vs 11-17）。
	maxCrossPlatformEventStartHours = 18
)

type fixtureEdge struct {
	a, b  string
	delta int
	dated bool
}

func platformKindFromKey(k string) platformKind {
	switch {
	case strings.Contains(k, "|k:"):
		return platformKalshi
	case strings.Contains(k, "|p:"):
		return platformPoly
	case strings.Contains(k, "|f:"):
		return platformPredict
	case strings.Contains(k, "|x:"):
		return platformSxbet
	default:
		return -1
	}
}

// polySlugFixtureBase 场次键停在日期。market.slug 已是 epl-ast-sun-2026-11-21；盘口 slug 才在日期后多一段 outcome。
func polySlugFixtureBase(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return ""
	}
	for _, mark := range []string{"-spread-", "-total-"} {
		if i := strings.Index(slug, mark); i > 0 {
			slug = slug[:i]
			break
		}
	}
	slug = strings.TrimSuffix(slug, "-more-markets")
	parts := strings.Split(slug, "-")
	for i := 0; i+2 < len(parts); i++ {
		if isYMDParts(parts[i], parts[i+1], parts[i+2]) {
			return strings.Join(parts[:i+3], "-")
		}
	}
	if len(parts) >= 5 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return slug
}

func isYMDParts(y, m, d string) bool {
	if len(y) != 4 || len(m) != 2 || len(d) != 2 {
		return false
	}
	for _, s := range []string{y, m, d} {
		for _, c := range s {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// linkPlatformFixtures 同联赛下对齐场次。slug/队名匹配；若两边均有 event_start_time 则开赛时刻须接近，否则不成边。
// names、starts 的键与 keys 相同（starts 为 UTC 时间串，可空）。
func linkPlatformFixtures(keys map[string]struct{}, names map[string]string, starts map[string]string) map[string]string {
	parent := make(map[string]string, len(keys))
	var find func(string) string
	find = func(x string) string {
		if parent[x] == "" {
			parent[x] = x
		}
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}

	allKeys := make([]string, 0, len(keys))
	for k := range keys {
		allKeys = append(allKeys, k)
	}

	var edges []fixtureEdge
	for i := 0; i < len(allKeys); i++ {
		for j := i + 1; j < len(allKeys); j++ {
			ka, kb := allKeys[i], allKeys[j]
			kindA, kindB := platformKindFromKey(ka), platformKindFromKey(kb)
			if kindA < 0 || kindB < 0 || kindA == kindB {
				continue
			}
			slugA := platformFixtureSlugPayload(ka, kindA)
			slugB := platformFixtureSlugPayload(kb, kindB)
			if slugA == "" || slugB == "" {
				continue
			}
			if !crossPlatformFixturePairMatches(kindA, kindB, slugA, slugB, names[ka], names[kb]) {
				continue
			}
			if !fixturePairStartOK(ka, kb, starts) {
				continue
			}
			// Poly 与 Predict slug 一致时直接并入；避免三边图每点度数为 2 时无法 assign。
			if polyPredictSlugEqual(kindA, kindB, slugA, slugB) {
				union(ka, kb)
				continue
			}
			delta, dated := fixtureDateDeltaForPair(ka, kb, kindA, kindB, slugA, slugB, starts)
			edges = append(edges, fixtureEdge{a: ka, b: kb, delta: delta, dated: dated})
		}
	}

	used := make(map[string]bool)
	assign := func(a, b string) {
		if used[a] || used[b] {
			return
		}
		used[a] = true
		used[b] = true
		union(a, b)
	}
	edgeOther := func(e fixtureEdge, node string) string {
		if e.a == node {
			return e.b
		}
		return e.a
	}
	for {
		var rem []fixtureEdge
		for _, e := range edges {
			if used[e.a] || used[e.b] {
				continue
			}
			rem = append(rem, e)
		}
		if len(rem) == 0 {
			break
		}
		byNode := make(map[string][]fixtureEdge)
		for _, e := range rem {
			byNode[e.a] = append(byNode[e.a], e)
			byNode[e.b] = append(byNode[e.b], e)
		}
		byP := make(map[string][]fixtureEdge)
		byK := make(map[string][]fixtureEdge)
		for _, e := range rem {
			if !fixtureEdgeIsKalshiPoly(e) {
				continue
			}
			kSide, pSide := kalshiPolyNodes(e)
			byP[pSide] = append(byP[pSide], fixtureEdge{a: kSide, b: pSide, delta: e.delta, dated: e.dated})
			byK[kSide] = append(byK[kSide], fixtureEdge{a: kSide, b: pSide, delta: e.delta, dated: e.dated})
		}
		assigned := false
		for p, es := range byP {
			if len(es) != 1 {
				continue
			}
			k := es[0].a
			if len(byK[k]) > 1 {
				continue
			}
			if !kalshiPolyUniqueAssignOK(es[0]) {
				continue
			}
			assign(k, p)
			assigned = true
		}
		if assigned {
			continue
		}
		for k, es := range byK {
			if len(es) != 1 {
				continue
			}
			p := es[0].b
			if len(byP[p]) > 1 {
				continue
			}
			if !kalshiPolyUniqueAssignOK(es[0]) {
				continue
			}
			assign(k, p)
			assigned = true
		}
		if assigned {
			continue
		}
		for node, es := range byNode {
			if len(es) == 1 {
				assign(node, edgeOther(es[0], node))
				assigned = true
			}
		}
		if assigned {
			continue
		}
		for _, es := range byP {
			sort.Slice(es, func(i, j int) bool { return es[i].delta < es[j].delta })
		}
		for _, es := range byK {
			sort.Slice(es, func(i, j int) bool { return es[i].delta < es[j].delta })
		}
		progress := false
		seenK := make(map[string]bool)
		seenP := make(map[string]bool)
		for p, es := range byP {
			if len(es) < 2 || !es[0].dated || !es[1].dated || es[0].delta >= es[1].delta {
				continue
			}
			kes := byK[es[0].a]
			if len(kes) == 0 || kes[0].b != p {
				continue
			}
			if len(kes) > 1 && kes[0].dated && kes[1].dated && kes[0].delta >= kes[1].delta {
				continue
			}
			if seenK[es[0].a] || seenP[p] {
				continue
			}
			seenK[es[0].a] = true
			seenP[p] = true
			assign(es[0].a, p)
			progress = true
		}
		if !progress {
			break
		}
	}

	canonical := canonicalFixtureIDsByComponent(keys, find)
	return canonical
}

func canonicalFixtureIDsByComponent(keys map[string]struct{}, find func(string) string) map[string]string {
	byRoot := make(map[string][]string)
	for k := range keys {
		r := find(k)
		byRoot[r] = append(byRoot[r], k)
	}
	out := make(map[string]string, len(keys))
	for _, members := range byRoot {
		best := ""
		for _, k := range members {
			id := canonicalFixtureID(k)
			best = preferFixtureCanonical(best, id)
		}
		if best == "" && len(members) > 0 {
			best = canonicalFixtureID(members[0])
		}
		for _, k := range members {
			out[k] = best
		}
	}
	return out
}

func preferFixtureCanonical(cur, candidate string) string {
	if candidate == "" {
		return cur
	}
	if cur == "" {
		return candidate
	}
	rank := func(id string) int {
		if strings.Contains(id, "-") && !strings.HasPrefix(id, "kx") {
			return 3 // Poly / Predict 风格 slug
		}
		if strings.HasPrefix(id, "kxe") || strings.HasPrefix(id, "kx") {
			return 1
		}
		return 2
	}
	if rank(candidate) > rank(cur) {
		return candidate
	}
	return cur
}

func kalshiPolyUniqueAssignOK(e fixtureEdge) bool {
	if !e.dated {
		return true
	}
	if e.delta <= maxKalshiPolyAssignHours {
		return true
	}
	return false
}

func platformKeyPayload(platformKey, prefix string) string {
	// competition|p:base 或 competition|k:slug
	i := strings.Index(platformKey, "|"+prefix)
	if i < 0 {
		return ""
	}
	return strings.TrimPrefix(platformKey[i+1:], prefix)
}

func canonicalFixtureID(platformKey string) string {
	// 优先用 Poly / Predict 场次串做人类可读 map_key 中段
	if payload := platformKeyPayload(platformKey, "p:"); payload != "" {
		return payload
	}
	if payload := platformKeyPayload(platformKey, "f:"); payload != "" {
		return payload
	}
	if payload := platformKeyPayload(platformKey, "k:"); payload != "" {
		return payload
	}
	return platformKey
}

func platformFixtureSlugPayload(key string, kind platformKind) string {
	switch kind {
	case platformKalshi:
		return platformKeyPayload(key, "k:")
	case platformPoly:
		return platformKeyPayload(key, "p:")
	case platformPredict:
		return platformKeyPayload(key, "f:")
	case platformSxbet:
		return platformKeyPayload(key, "x:")
	default:
		return ""
	}
}

func parseMarketEventStart(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func eventStartDeltaHours(a, b time.Time) int {
	d := a.Sub(b)
	if d < 0 {
		d = -d
	}
	return int(d.Hours())
}

func fixturePairStartOK(ka, kb string, starts map[string]string) bool {
	if starts == nil {
		return true
	}
	sa, oka := parseMarketEventStart(starts[ka])
	sb, okb := parseMarketEventStart(starts[kb])
	if !oka || !okb {
		return true
	}
	dayA, dayB := sa.UTC().Format("2006-01-02"), sb.UTC().Format("2006-01-02")
	if dayA != dayB {
		return false
	}
	return eventStartDeltaHours(sa, sb) <= maxCrossPlatformEventStartHours
}

func fixtureDateDeltaForPair(ka, kb string, kindA, kindB platformKind, slugA, slugB string, starts map[string]string) (int, bool) {
	if starts != nil {
		if sa, oka := parseMarketEventStart(starts[ka]); oka {
			if sb, okb := parseMarketEventStart(starts[kb]); okb {
				return eventStartDeltaHours(sa, sb), true
			}
		}
	}
	return fixtureDateDeltaForKinds(kindA, kindB, slugA, slugB)
}

func crossPlatformFixturePairMatches(kindA, kindB platformKind, slugA, slugB, nameA, nameB string) bool {
	baseA := polySlugFixtureBase(slugA)
	baseB := polySlugFixtureBase(slugB)
	if baseA != "" && baseA == baseB {
		return true
	}
	if slugA == slugB && slugA != "" {
		return true
	}
	switch {
	case kindA == platformKalshi && kindB == platformPoly:
		return fixturePairMatches(slugA, slugB, nameA, nameB)
	case kindA == platformPoly && kindB == platformKalshi:
		return fixturePairMatches(slugB, slugA, nameB, nameA)
	case kindA == platformKalshi && kindB == platformPredict:
		return matchKalshiPolyFixture(slugA, slugB) || fixtureNamesMatch(nameA, nameB)
	case kindA == platformPredict && kindB == platformKalshi:
		return matchKalshiPolyFixture(slugB, slugA) || fixtureNamesMatch(nameA, nameB)
	case kindA == platformPoly && kindB == platformPredict, kindA == platformPredict && kindB == platformPoly:
		// 禁止仅凭队名模糊对齐（NCAA 男子 cbb 与女子 cwbb 标题相近时会误并到同一 map_key）。
		return polyPredictFixtureSlugMatch(slugA, slugB)
	case kindA == platformSxbet || kindB == platformSxbet:
		// SX 场次 slug 为 sportXeventId，与 Poly slug 不一致；用标题队名 + event_start_time 约束成边。
		return fixtureNamesMatch(nameA, nameB)
	default:
		return false
	}
}

// polyPredictFixtureSlugMatch Poly 与 Predict 须 slug 场次键一致，且联赛前缀一致（cbb≠cwbb）。
func polyPredictFixtureSlugMatch(slugA, slugB string) bool {
	baseA := polySlugFixtureBase(slugA)
	baseB := polySlugFixtureBase(slugB)
	if baseA == "" || baseB == "" || baseA != baseB {
		return false
	}
	return sportSlugPrefix(slugA) == sportSlugPrefix(slugB)
}

func sportSlugPrefix(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if i := strings.Index(slug, "-"); i > 0 {
		return slug[:i]
	}
	return slug
}

func fixtureDateDeltaForKinds(kindA, kindB platformKind, slugA, slugB string) (int, bool) {
	switch {
	case kindA == platformKalshi && kindB == platformPoly:
		return fixtureDateDeltaHours(slugA, slugB)
	case kindA == platformPoly && kindB == platformKalshi:
		return fixtureDateDeltaHours(slugB, slugA)
	case kindA == platformKalshi && kindB == platformPredict:
		return fixtureDateDeltaHours(slugA, slugB)
	case kindA == platformPredict && kindB == platformKalshi:
		return fixtureDateDeltaHours(slugB, slugA)
	case kindA == platformSxbet || kindB == platformSxbet:
		return math.MaxInt, false
	default:
		return fixtureDateDeltaPolyStyle(slugA, slugB)
	}
}

func fixtureDateDeltaPolyStyle(slugA, slugB string) (int, bool) {
	da, okA := dateFromPolySlug(slugA)
	db, okB := dateFromPolySlug(slugB)
	if !okA || !okB {
		return math.MaxInt, false
	}
	d := da.Sub(db)
	if d < 0 {
		d = -d
	}
	return int(d.Hours()), true
}

func fixtureEdgeIsKalshiPoly(e fixtureEdge) bool {
	ka, kb := platformKindFromKey(e.a), platformKindFromKey(e.b)
	return (ka == platformKalshi && kb == platformPoly) || (ka == platformPoly && kb == platformKalshi)
}

func kalshiPolyNodes(e fixtureEdge) (kalshiKey, polyKey string) {
	if platformKindFromKey(e.a) == platformKalshi {
		return e.a, e.b
	}
	return e.b, e.a
}

func polyPredictSlugEqual(kindA, kindB platformKind, slugA, slugB string) bool {
	var polySlug, predictSlug string
	switch {
	case kindA == platformPoly && kindB == platformPredict:
		polySlug, predictSlug = slugA, slugB
	case kindA == platformPredict && kindB == platformPoly:
		polySlug, predictSlug = slugB, slugA
	default:
		return false
	}
	baseP := polySlugFixtureBase(polySlug)
	baseF := polySlugFixtureBase(predictSlug)
	return baseP != "" && baseP == baseF
}

func fixturePairMatches(kalshiSlug, polyBase, kalshiName, polyName string) bool {
	if matchKalshiPolyFixture(kalshiSlug, polyBase) {
		return true
	}
	return fixtureNamesMatch(kalshiName, polyName)
}

func matchKalshiPolyFixture(kalshiSlug, polyBase string) bool {
	t1, t2, ok := polyTeamCodes(polyBase)
	if !ok {
		return false
	}
	blob, okBlob := kalshiTeamsBlobFromSlug(kalshiSlug)
	if !okBlob {
		return false
	}
	return kalshiBlobMatchesPair(blob, t1, t2) || kalshiBlobMatchesPair(blob, t2, t1)
}

// kalshiTeamsBlobFromSlug 取 Kalshi event slug 中日期后的队码粘连段（如 bounfo、fulbou）。
func kalshiTeamsBlobFromSlug(slug string) (string, bool) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	loc := reKalshiSlugDate.FindStringIndex(slug)
	if loc == nil {
		return "", false
	}
	blob := strings.Trim(slug[loc[1]:], "-")
	if blob == "" {
		return "", false
	}
	return blob, true
}

func kalshiBlobMatchesPair(blob, codeA, codeB string) bool {
	for _, a := range kalshiCodeVariants(codeA) {
		if a == "" || !strings.HasPrefix(blob, a) {
			continue
		}
		rest := blob[len(a):]
		for _, b := range kalshiCodeVariants(codeB) {
			if b != "" && rest == b {
				return true
			}
		}
	}
	return false
}

func kalshiCodeVariants(code string) []string {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return nil
	}
	if al, ok := polyKalshiBlobAliases[code]; ok {
		return al
	}
	return []string{code}
}

var reFixtureVs = regexp.MustCompile(`(?i)\s+vs\.?\s+`)

// fixtureNamesMatch 用标题里的队名对齐。两边各取「vs」两侧，每侧至少有一个完整词相同。
// Kalshi 常写成 Kashima vs Shimizu，Poly 写成 Kashima Antlers vs. Shimizu S-Pulse。
// FixtureNamesMatchForAudit 跨平台队名粗匹配（供离线审计工具使用）。
func FixtureNamesMatchForAudit(nameA, nameB string) bool {
	return fixtureNamesMatch(nameA, nameB)
}

func fixtureNamesMatch(kalshiName, polyName string) bool {
	k1, k2, okK := fixtureSides(kalshiName)
	p1, p2, okP := fixtureSides(polyName)
	if !okK || !okP {
		return false
	}
	return (sideTokensOverlap(k1, p1) && sideTokensOverlap(k2, p2)) ||
		(sideTokensOverlap(k1, p2) && sideTokensOverlap(k2, p1))
}

func fixtureSides(name string) ([]string, []string, bool) {
	parts := reFixtureVs.Split(foldFixtureName(name), -1)
	if len(parts) != 2 {
		return nil, nil, false
	}
	a := significantNameTokens(parts[0])
	b := significantNameTokens(parts[1])
	if len(a) == 0 || len(b) == 0 {
		return nil, nil, false
	}
	return a, b, true
}

func sideTokensOverlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

func significantNameTokens(side string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, raw := range strings.Fields(side) {
		tok := strings.TrimSpace(raw)
		_, stop := fixtureNameStopwords[tok]
		if len(tok) < 3 || stop {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}

var fixtureNameStopwords = map[string]struct{}{
	"the": {}, "and": {}, "club": {},
	"fc": {}, "sc": {}, "cf": {}, "ac": {}, "afc": {},
	"de": {}, "da": {}, "do": {}, "of": {},
}

func foldFixtureName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		r = unicode.ToLower(r)
		switch r {
		case 'ō', 'ó', 'ò', 'ô', 'ö', 'õ', 'ø':
			r = 'o'
		case 'ū', 'ú', 'ù', 'û', 'ü':
			r = 'u'
		case 'ā', 'á', 'à', 'â', 'ä', 'ã', 'å':
			r = 'a'
		case 'ē', 'é', 'è', 'ê', 'ë':
			r = 'e'
		case 'ī', 'í', 'ì', 'î', 'ï':
			r = 'i'
		case 'ñ':
			r = 'n'
		case 'ç':
			r = 'c'
		case '·', '・', '.', ',', '\'', '’', '-', '/':
			r = ' '
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func polyTeamCodes(polyBase string) (string, string, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(polyBase)), "-")
	for i := 0; i+2 < len(parts); i++ {
		if isYMDParts(parts[i], parts[i+1], parts[i+2]) && i >= 2 {
			return parts[i-2], parts[i-1], parts[i-2] != "" && parts[i-1] != ""
		}
	}
	if len(parts) >= 4 && parts[1] != "" && parts[2] != "" {
		return parts[1], parts[2], true
	}
	return "", "", false
}

func fixtureDateDeltaHours(kalshiSlug, polyBase string) (int, bool) {
	kd, okK := dateFromKalshiEventSlug(kalshiSlug)
	pd, okP := dateFromPolySlug(polyBase)
	if !okK || !okP {
		return math.MaxInt, false
	}
	d := kd.Sub(pd)
	if d < 0 {
		d = -d
	}
	return int(d.Hours()), true
}

// polyKalshiBlobAliases：Poly slug 队缩写 → Kalshi event slug 粘连缩写里可匹配的子串
var polyKalshiBlobAliases = map[string][]string{
	// 英超
	"ast": {"ast", "avl"}, "sun": {"sun"}, "cry": {"cry"}, "cov": {"cov"}, "ful": {"ful"},
	"mac": {"mac", "mci"}, "ars": {"ars"}, "new": {"new"}, "ips": {"ips", "totips"},
	"tot": {"tot", "totips"}, "bou": {"bou", "bounfo"}, "not": {"not", "nfo", "bounfo", "nfocfc"},
	"che": {"che", "cfc", "cfclee", "nfocfc"}, "lee": {"lee", "cfclee"}, "bri": {"bri", "hulbri"},
	"hul": {"hul", "hulbri"}, "liv": {"liv", "lfc", "lfcmun"}, "mun": {"mun", "lfcmun"},
	"eve": {"eve", "breeve"}, "bre": {"bre", "breeve"},
	// 意甲
	"tor": {"tor", "naptor"}, "nap": {"nap", "naptor"}, "com": {"com", "comcag"}, "cag": {"cag", "comcag"},
	"laz": {"laz", "lazlec"}, "lec": {"lec", "lazlec"}, "par": {"par", "parrom"}, "rom": {"rom", "parrom"},
	"mil": {"mil", "acmfro"}, "fro": {"fro", "acmfro"}, "ata": {"ata", "ataint"}, "int": {"int", "ataint"},
	"bol": {"bol", "bfc", "bfcudi", "venbfc"}, "udi": {"udi", "bfcudi"}, "sas": {"sas", "sasgen"},
	"gen": {"gen", "sasgen"}, "mon": {"mon", "monfio"}, "fio": {"fio", "monfio"},
	"juv": {"juv", "juvven"}, "ven": {"ven", "juvven", "venbfc"},
	// 德甲
	"bay": {"bay", "bmu", "bmukoe"}, "koe": {"koe", "bmukoe"}, "moe": {"moe", "bmg", "bmgbvb"},
	"dor": {"dor", "bvb", "bmgbvb"}, "b04": {"b04", "lev", "levpad"}, "scp": {"scp", "pad", "levpad"},
	"s04": {"s04", "sch", "schvfb"}, "stu": {"stu", "vfb", "schvfb"}, "ein": {"ein", "sge", "sgetsg"},
	"hof": {"hof", "tsg", "sgetsg"}, "elv": {"elv", "sve", "svescf"}, "fre": {"fre", "scf", "svescf"},
	"lei": {"lei", "rbl", "unirbl"}, "uni": {"uni", "unirbl"}, "aug": {"aug", "fca", "fcam05"},
	"mai": {"mai", "m05", "fcam05"}, "wer": {"wer", "svw", "svwhsv", "svwbmg"}, "hsv": {"hsv", "svwhsv"},
	// 西甲
	"bar": {"bar", "barvil"}, "vil": {"vil", "barvil"}, "elc": {"elc", "levelc"}, "lev": {"lev", "levelc", "levsan"},
	"bil": {"bil", "ath", "athesp"}, "esp": {"esp", "athesp"}, "ala": {"ala", "aladep"}, "dep": {"dep", "aladep"},
	"rea": {"rea", "rma", "rmarcc"}, "cel": {"cel", "rcc", "rmarcc"}, "ray": {"ray", "rvc", "rvcvcf"},
	"val": {"val", "vcf", "rvcvcf"}, "sev": {"sev", "sevrbb"}, "bet": {"bet", "rbb", "sevrbb"},
	"get": {"get", "getatm"}, "mad": {"mad", "atm", "getatm"}, "osa": {"osa", "osamcf"}, "mala": {"mala", "mcf", "osamcf"},
	"rrc": {"rrc", "san", "sanrso", "levsan"}, "rso": {"rso", "sanrso"},
}

