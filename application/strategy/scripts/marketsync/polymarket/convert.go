package polymarket

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func parsePolymarketClobTokenIds(clobTokenIds string) (yesToken, noToken string) {
	clobTokenIds = strings.TrimSpace(clobTokenIds)
	if clobTokenIds == "" {
		return "", ""
	}
	if strings.HasPrefix(clobTokenIds, "[") {
		var tokens []string
		if err := json.Unmarshal([]byte(clobTokenIds), &tokens); err != nil || len(tokens) < 2 {
			return "", ""
		}
		return strings.TrimSpace(tokens[0]), strings.TrimSpace(tokens[1])
	}
	parts := strings.Split(clobTokenIds, ",")
	if len(parts) < 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func extractPolymarketTimes(m *httpentity.Market) (eventStart, eventEnd, marketStart, marketEnd string) {
	parsePolyDT := func(s string) time.Time {
		s = strings.TrimSpace(s)
		if s == "" {
			return time.Time{}
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05-07:00", "2006-01-02 15:04:05+00", "2006-01-02", "2006-01-02 15:04:05-07", "2006-01-02 15:04:05Z07:00"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
		return time.Time{}
	}
	formatStd := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	eventStart = formatStd(parsePolyDT(m.GameStartTime))
	if eventStart == "" {
		eventStart = formatStd(parsePolyDT(m.StartDate))
	}
	if eventStart == "" {
		eventStart = formatStd(parsePolyDT(m.StartDateIso))
	}
	if len(m.Events) > 0 {
		e := &m.Events[0]
		eventEnd = formatStd(parsePolyDT(e.FinishedTimestamp))
		marketEnd = formatStd(parsePolyDT(e.ClosedTime))
	}
	if eventEnd == "" && !m.EndDate.IsZero() {
		eventEnd = m.EndDate.UTC().Format("2006-01-02 15:04:05")
	}
	if eventEnd == "" {
		eventEnd = formatStd(parsePolyDT(m.EndDateIso))
	}
	if marketEnd == "" && eventEnd != "" {
		marketEnd = eventEnd
	}
	marketStart = formatStd(parsePolyDT(m.AcceptingOrdersTimestamp))
	if marketStart == "" && !m.CreatedAt.IsZero() {
		marketStart = m.CreatedAt.UTC().Format("2006-01-02 15:04:05")
	}
	return eventStart, eventEnd, marketStart, marketEnd
}

func extractPolymarketLeague(m *httpentity.Market) string {
	for _, e := range m.Events {
		for _, s := range e.Series {
			if s.Title != "" {
				return s.Title
			}
			if s.Slug != "" {
				return s.Slug
			}
		}
	}
	return ""
}

func buildPolymarketMarketURL(slug string) string {
	if slug == "" {
		return ""
	}
	return common.Truncate("https://polymarket.com/market/"+slug, 255)
}
