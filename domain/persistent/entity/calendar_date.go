package entity

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// CalendarDate MySQL DATE（YYYY-MM-DD 字符串），GORM 直写/直读，避开 time.Time + loc=Local。
type CalendarDate string

// CalendarDateFrom 从 time.Time 取 UTC 日历日。
func CalendarDateFrom(t time.Time) CalendarDate {
	if t.IsZero() {
		return ""
	}
	y, m, d := t.In(time.UTC).Date()
	return CalendarDate(fmt.Sprintf("%04d-%02d-%02d", y, int(m), d))
}

// String 返回 YYYY-MM-DD。
func (c CalendarDate) String() string {
	return string(c)
}

// TimeUTC 当天 00:00 UTC。
func (c CalendarDate) TimeUTC() time.Time {
	s := strings.TrimSpace(string(c))
	if len(s) < 10 {
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02", s[:10], time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Value 写入 MySQL。
func (c CalendarDate) Value() (driver.Value, error) {
	s := strings.TrimSpace(string(c))
	if s == "" {
		return nil, fmt.Errorf("calendar date empty")
	}
	return s, nil
}

// Scan 从 MySQL 读 DATE。
func (c *CalendarDate) Scan(value interface{}) error {
	switch v := value.(type) {
	case nil:
		*c = ""
		return nil
	case []byte:
		*c = CalendarDate(strings.TrimSpace(string(v)))
		return nil
	case string:
		*c = CalendarDate(strings.TrimSpace(v))
		return nil
	case time.Time:
		*c = CalendarDateFrom(v)
		return nil
	default:
		return fmt.Errorf("calendar date: unsupported scan type %T", value)
	}
}