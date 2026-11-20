package entity

import (
	"testing"
	"time"
)

func TestCalendarDate_Value_stringNoTZ(t *testing.T) {
	c := CalendarDateFrom(time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC))
	v, err := c.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != "2026-10-17" {
		t.Fatalf("got %v", v)
	}
}

func TestCalendarDate_Scan(t *testing.T) {
	var c CalendarDate
	if err := (&c).Scan("2026-10-17"); err != nil {
		t.Fatal(err)
	}
	if c.String() != "2026-10-17" {
		t.Fatalf("got %s", c.String())
	}
}