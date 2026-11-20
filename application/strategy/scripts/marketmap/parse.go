package marketmap

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

var (
	rePolyWinOn      = regexp.MustCompile(`(?i)^will\s+(.+?)\s+win\s+on\s+(\d{4}-\d{2}-\d{2})`)
	rePolyWin        = regexp.MustCompile(`(?i)^will\s+(.+?)\s+win\b`)
	reVs             = regexp.MustCompile(`(?i)^(.+?)\s+(?:vs\.?|v\.?|versus)\s+(.+?)(?:\s*(?:[—\-–]|$)|$)`)
	reBeat           = regexp.MustCompile(`(?i)^will\s+(.+?)\s+beat\s+(.+?)\??\s*$`)
	reKalshiWins     = regexp.MustCompile(`(?i)^(.+?)\s+wins?\s*$`)
	reKalshiTie      = regexp.MustCompile(`(?i)^tie\s+is\s+the\s+result\s*$`)
	reEndInDraw      = regexp.MustCompile(`(?i)end in a (?:draw|tie)\??\s*$`)
	reSlugISODate    = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	reKalshiSlugDate = regexp.MustCompile(`(?i)(\d{2})(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)(\d{2})`)
)

// parsedMatch 从一行解析：联赛、平台场次 key、胜出方（不读 event_start_time 等时间列）。
type parsedMatch struct {
	Competition      string
	PlatformFixture  string
	CanonicalFixture string
	OutcomeTeam      string
	OutcomeNorm      string
	ParseOK          bool
}

func parseMarketRow(m *persistententity.MarketOutcome, canonicalFixture map[string]string) parsedMatch {
	out := parsedMatch{ParseOK: false}
	if m == nil {
		return out
	}
	comp, pfKey, ok := platformFixtureKey(m)
	if !ok {
		return out
	}
	name := ""
	if m.OutcomeName != nil {
		name = strings.TrimSpace(*m.OutcomeName)
	}
	if name == "" {
		return out
	}
	team, ok := outcomeTeamFromName(name)
	if !ok || team == "" {
		return out
	}
	polySlug := ""
	if m.Slug != nil {
		polySlug = *m.Slug
	}
	outcomeNorm := canonicalOutcomeNorm(m.Platform, team, polySlug)
	if outcomeNorm == "" {
		return out
	}
	cf := canonicalFixture[pfKey]
	if cf == "" {
		cf = canonicalFixtureID(pfKey)
	}

	out.Competition = comp
	out.PlatformFixture = pfKey
	out.CanonicalFixture = cf
	out.OutcomeTeam = team
	out.OutcomeNorm = outcomeNorm
	out.ParseOK = true
	return out
}

func dateFromPolySlug(slug string) (time.Time, bool) {
	m := reSlugISODate.FindStringSubmatch(strings.ToLower(slug))
	if len(m) < 2 {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", m[1], time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func dateFromKalshiEventSlug(slug string) (time.Time, bool) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	m := reKalshiSlugDate.FindStringSubmatch(slug)
	if len(m) < 4 {
		return time.Time{}, false
	}
	year := "20" + m[1]
	dateStr := fmtDateYMD(year, m[2], m[3])
	if dateStr == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func fmtDateYMD(year, monAbbr, day string) string {
	months := map[string]string{
		"jan": "01", "feb": "02", "mar": "03", "apr": "04", "may": "05", "jun": "06",
		"jul": "07", "aug": "08", "sep": "09", "oct": "10", "nov": "11", "dec": "12",
	}
	mm, ok := months[strings.ToLower(monAbbr)]
	if !ok {
		return ""
	}
	if len(day) == 1 {
		day = "0" + day
	}
	return year + "-" + mm + "-" + day
}

func outcomeTeamFromName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	if reKalshiTie.MatchString(name) {
		return "draw", true
	}
	if reEndInDraw.MatchString(name) {
		return "draw", true
	}
	if m := reKalshiWins.FindStringSubmatch(name); len(m) >= 2 {
		return strings.TrimSpace(m[1]), true
	}
	if i := strings.Index(name, "—"); i >= 0 {
		sub := strings.TrimSpace(name[i+len("—"):])
		if sub != "" {
			return sub, true
		}
	}
	if i := strings.Index(name, " - "); i >= 0 {
		sub := strings.TrimSpace(name[i+3:])
		if sub != "" && len(sub) < 80 {
			return sub, true
		}
	}
	if m := rePolyWinOn.FindStringSubmatch(name); len(m) >= 2 {
		return strings.TrimSpace(m[1]), true
	}
	if m := rePolyWin.FindStringSubmatch(name); len(m) >= 2 {
		return strings.TrimSpace(m[1]), true
	}
	if m := reBeat.FindStringSubmatch(name); len(m) >= 2 {
		return strings.TrimSpace(m[1]), true
	}
	if m := reVs.FindStringSubmatch(name); len(m) >= 3 {
		return strings.TrimSpace(m[1]), true
	}
	return "", false
}

func normalizeTeam(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "´", "")
	s = strings.ReplaceAll(s, "`", "")
	for _, pre := range []string{"afc ", "fc "} {
		if strings.HasPrefix(s, pre) {
			s = strings.TrimSpace(s[len(pre):])
		}
	}
	// 勿删 united/city/forest/palace 等：Nottingham Forest、Man City、Crystal Palace 等 Kalshi 短标题会对不齐
	for _, suf := range []string{
		" football club", " fc", " cf", " sc", " afc",
		" town", " hotspur", " albion", " hove", " athletic",
		" county", " rovers", " wanderers", " & hove albion", " balompié", " balompie",
	} {
		if strings.HasSuffix(s, suf) {
			s = strings.TrimSpace(strings.TrimSuffix(s, suf))
		}
	}
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if r == ' ' || r == '-' {
			if !lastSpace && b.Len() > 0 {
				b.WriteRune('_')
				lastSpace = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
