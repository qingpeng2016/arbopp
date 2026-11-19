package historydata

import (
	"regexp"
	"strings"
	"time"
)

var mapKeyDateSuffix = regexp.MustCompile(`-(\d{4}-\d{2}-\d{2})$`)

// UTCDatesForFixture 生成需要下载的 UTC 日历日（含赛前/赛后缓冲）。
func UTCDatesForFixture(mapKey, eventStartUTC, eventEndUTC string, daysBefore, daysAfter int) []time.Time {
	anchor := parseFixtureUTCDate(mapKey, eventStartUTC, eventEndUTC)
	if anchor.IsZero() {
		return nil
	}
	start := anchor.AddDate(0, 0, -daysBefore)
	end := anchor.AddDate(0, 0, daysAfter)
	if end.Before(start) {
		start, end = end, start
	}
	var out []time.Time
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		out = append(out, d)
	}
	return out
}

// BaseSnapshotMsForMapKey 秒级序列起点：比赛日 12:00 UTC（ms）。
func BaseSnapshotMsForMapKey(mapKey string) int64 {
	d := parseFixtureUTCDate(mapKey, "", "")
	if d.IsZero() {
		d = time.Now().UTC().Truncate(24 * time.Hour)
	}
	start := time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
	return start.UnixMilli()
}

func parseFixtureUTCDate(mapKey, eventStartUTC, eventEndUTC string) time.Time {
	if t := parseEventTimeUTC(eventStartUTC); !t.IsZero() {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
	mapKey = strings.TrimSpace(mapKey)
	if m := mapKeyDateSuffix.FindStringSubmatch(mapKey); len(m) == 2 {
		if t, err := time.ParseInLocation("2006-01-02", m[1], time.UTC); err == nil {
			return t
		}
	}
	if t := parseEventTimeUTC(eventEndUTC); !t.IsZero() {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
	return time.Time{}
}

func parseEventTimeUTC(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}