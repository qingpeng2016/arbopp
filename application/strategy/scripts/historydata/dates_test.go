package historydata

import (
	"testing"
	"time"
)

func TestUTCDatesFromMapKey(t *testing.T) {
	dates := UTCDatesForFixture("v1|国际足球|fif-chn-bah1-2026-11-17", "", "", 1, 1)
	if len(dates) != 3 {
		t.Fatalf("want 3 days, got %d", len(dates))
	}
	want0 := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	if !dates[0].Equal(want0) {
		t.Fatalf("first day %v want %v", dates[0], want0)
	}
}