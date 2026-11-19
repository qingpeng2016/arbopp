package kalshi

import (
	"regexp"
	"strings"
	"time"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

const (
	kalshiSlugDayDefaultMatchDuration = 2*time.Hour + 30*time.Minute
	kalshiMaxCloseAfterGameDay        = 36 * time.Hour
)

var reKalshiEventSlugDate = regexp.MustCompile(`(?i)(\d{2})(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)(\d{2})`)

// dateFromEventTicker 从 event_ticker 解析场次日历日（如 kxeplgame-26nov21avlsun → 2026-11-21 UTC）。
func dateFromEventTicker(eventTicker string) (time.Time, bool) {
	eventTicker = strings.ToLower(strings.TrimSpace(eventTicker))
	m := reKalshiEventSlugDate.FindStringSubmatch(eventTicker)
	if len(m) < 4 {
		return time.Time{}, false
	}
	year := "20" + m[1]
	dateStr := kalshiFmtDateYMD(year, m[2], m[3])
	if dateStr == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func kalshiFmtDateYMD(year, monAbbr, day string) string {
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

func kalshiActiveMarkets(markets []httpentity.KalshiMarket) []httpentity.KalshiMarket {
	out := make([]httpentity.KalshiMarket, 0, len(markets))
	for _, m := range markets {
		st := strings.TrimSpace(m.Status)
		if strings.EqualFold(st, "closed") || strings.EqualFold(st, "settled") {
			continue
		}
		out = append(out, m)
	}
	return out
}

// kalshiMainMarketsForEventTime 胜平负/非 spread、大小；用于估场次时间。
func kalshiMainMarketsForEventTime(active []httpentity.KalshiMarket) []httpentity.KalshiMarket {
	var ml []httpentity.KalshiMarket
	for _, m := range active {
		if kalshiIsMoneylineMarket(m) {
			ml = append(ml, m)
		}
	}
	if len(ml) > 0 {
		return ml
	}
	var main []httpentity.KalshiMarket
	for _, m := range active {
		if kalshiIsHandicapMarket(m) || kalshiIsTotalMarket(m) {
			continue
		}
		main = append(main, m)
	}
	if len(main) > 0 {
		return main
	}
	return active
}

func kalshiFixtureTimes(eventTicker string, markets []httpentity.KalshiMarket) (start, end string) {
	active := kalshiActiveMarkets(markets)
	if len(active) == 0 {
		return "", ""
	}
	gameDay, hasGameDay := dateFromEventTicker(eventTicker)
	if hasGameDay {
		if s, e, ok := kalshiFixtureTimesOnGameDay(active, gameDay); ok {
			return s, e
		}
		return kalshiFixtureTimesSlugDayDefault(gameDay)
	}
	return kalshiFixtureTimesTradingWindow(active)
}

func kalshiSameUTCDay(a, b time.Time) bool {
	return a.UTC().Format("2006-01-02") == b.UTC().Format("2006-01-02")
}

// kalshiFixtureTimesOnGameDay 用 slug 比赛日 + 主盘 close_time；不用早于比赛日的 open_time。
func kalshiFixtureTimesOnGameDay(active []httpentity.KalshiMarket, gameDay time.Time) (start, end string, ok bool) {
	candidates := kalshiMainMarketsForEventTime(active)
	dayKey := gameDay.UTC().Format("2006-01-02")
	var bestStart, bestEnd time.Time
	for _, m := range candidates {
		t := parseKalshiTimeUTC(m.CloseTime)
		if t.IsZero() || !kalshiSameUTCDay(t, gameDay) {
			continue
		}
		if bestStart.IsZero() || t.Before(bestStart) {
			bestStart = t
		}
		if bestEnd.IsZero() || t.After(bestEnd) {
			bestEnd = t
		}
	}
	if !bestStart.IsZero() {
		if bestEnd.IsZero() {
			bestEnd = bestStart
		}
		if bestStart.Equal(bestEnd) {
			bestStart = bestStart.Add(-90 * time.Minute)
			if bestStart.Before(gameDay) {
				bestStart = gameDay.UTC()
			}
		}
		return bestStart.UTC().Format("2006-01-02 15:04:05"),
			bestEnd.UTC().Format("2006-01-02 15:04:05"),
			true
	}

	var nearestClose time.Time
	for _, m := range candidates {
		t := parseKalshiTimeUTC(m.CloseTime)
		if t.IsZero() || t.Before(gameDay) {
			continue
		}
		after := t.Sub(gameDay.UTC())
		if after > kalshiMaxCloseAfterGameDay {
			continue
		}
		if nearestClose.IsZero() || t.Before(nearestClose) {
			nearestClose = t
		}
	}
	if nearestClose.IsZero() {
		_ = dayKey
		return "", "", false
	}
	endT := nearestClose
	startT := endT.Add(-kalshiSlugDayDefaultMatchDuration)
	if startT.Before(gameDay) {
		startT = gameDay.UTC()
	}
	return startT.UTC().Format("2006-01-02 15:04:05"),
		endT.UTC().Format("2006-01-02 15:04:05"),
		true
}

func kalshiFixtureTimesSlugDayDefault(gameDay time.Time) (start, end string) {
	startT := gameDay.UTC()
	endT := startT.Add(kalshiSlugDayDefaultMatchDuration)
	return startT.Format("2006-01-02 15:04:05"),
		endT.Format("2006-01-02 15:04:05")
}

func kalshiFixtureTimesTradingWindow(active []httpentity.KalshiMarket) (start, end string) {
	var bestStart, bestEnd time.Time
	for _, m := range active {
		if t := parseKalshiTimeUTC(m.OpenTime); !t.IsZero() {
			if bestStart.IsZero() || t.Before(bestStart) {
				bestStart = t
			}
		}
		if t := parseKalshiTimeUTC(m.CloseTime); !t.IsZero() {
			if bestEnd.IsZero() || t.After(bestEnd) {
				bestEnd = t
			}
		}
	}
	if !bestStart.IsZero() {
		start = bestStart.UTC().Format("2006-01-02 15:04:05")
	}
	if !bestEnd.IsZero() {
		end = bestEnd.UTC().Format("2006-01-02 15:04:05")
	}
	return start, end
}