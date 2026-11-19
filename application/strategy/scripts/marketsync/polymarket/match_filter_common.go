package polymarket

import (
	"strings"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func polySlugLower(m *httpentity.Market) string {
	if m == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(m.Slug))
}

func polyQuestionLower(m *httpentity.Market) string {
	if m == nil {
		return ""
	}
	return strings.ToLower(m.Question)
}

func polyQuestionDescLower(m *httpentity.Market) string {
	if m == nil {
		return ""
	}
	return strings.ToLower(m.Question + " " + m.Description)
}

func hasPolymarketScheduledKickoff(m *httpentity.Market) bool {
	if m == nil {
		return false
	}
	if strings.TrimSpace(m.GameStartTime) != "" {
		return true
	}
	for _, e := range m.Events {
		if strings.TrimSpace(e.StartTime) != "" {
			return true
		}
	}
	return false
}

func polySlugIsSpread(slug string) bool {
	return strings.Contains(slug, "-spread-")
}

func polySlugIsTotal(slug string) bool {
	return strings.Contains(slug, "-total-")
}

func polyQuestionLooksLikeHandicap(q string) bool {
	return strings.Contains(q, "spread:") || strings.HasPrefix(q, "spread ")
}

func polyQuestionLooksLikeTotal(q string) bool {
	return strings.Contains(q, "o/u") || strings.Contains(q, "over/under")
}

func polyQuestionHasVs(q string) bool {
	return strings.Contains(q, " vs. ") || strings.Contains(q, " vs ") || strings.Contains(q, " vs.")
}

func polyQuestionContainsAny(q string, keywords []string) bool {
	for _, kw := range keywords {
		if kw != "" && strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

func polySlugContainsAny(slug string, fragments []string) bool {
	for _, frag := range fragments {
		if frag != "" && strings.Contains(slug, frag) {
			return true
		}
	}
	return false
}