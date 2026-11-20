package common

import (
	"strings"
	"time"
)

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func StrPtr(s string) *string {
	s = Truncate(s, 2000)
	return &s
}

// StrPtrOptional 空串写 NULL；非空截断 2000。
func StrPtrOptional(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return StrPtr(s)
}

func StrPtr255(s string) *string {
	s = Truncate(s, 255)
	return &s
}

// StrPtr255NilIfEmpty 空串写 NULL；非空则截断 255（用于 event_start/end 跟 API 一致）。
func StrPtr255NilIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = Truncate(s, 255)
	return &s
}

func StrPtr200(s string) *string {
	s = Truncate(s, 200)
	return &s
}

func StrPtr128(s string) *string {
	s = Truncate(s, 128)
	return &s
}

func FloatPtr(v float64) *float64 {
	return &v
}

// SkipEventEndedUTC 赛事已结束则不写入。
func SkipEventEndedUTC(eventStartTime, eventEndTime string) bool {
	const layout = "2006-01-02 15:04:05"
	parseUTC := func(s string) (time.Time, bool) {
		s = strings.TrimSpace(s)
		if s == "" {
			return time.Time{}, false
		}
		t, err := time.ParseInLocation(layout, s, time.UTC)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	start, okStart := parseUTC(eventStartTime)
	end, okEnd := parseUTC(eventEndTime)
	if !okStart || !okEnd || !end.After(start) {
		return false
	}
	now := time.Now().UTC()
	return now.After(start) && now.After(end)
}