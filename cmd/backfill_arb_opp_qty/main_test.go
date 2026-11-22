package main

import (
	"testing"
)

func TestMinLegTopAskQtyFromStoredJSON(t *testing.T) {
	leg1 := `{
  "platform": "SXBET",
  "books": [{
    "asks": [{"price": 0.54, "size": 108.23478260869565}]
  }]
}`
	leg2 := `{
  "platform": "POLYMARKET",
  "books": [{
    "asks": [{"price": 0.36, "size": 28.12}]
  }]
}`
	got := minLegTopAskQty(leg1, leg2, "{}\n")
	if got != 28.12 {
		t.Fatalf("min=%v want 28.12", got)
	}
}

func TestMinLegTopAskQtyHorsens(t *testing.T) {
	sx := `{"books":[{"asks":[{"price":0.45625,"size":13.687500000000002}]}]}`
	poly := `{"books":[{"asks":[{"price":0.45,"size":19.2}]}]}`
	got := minLegTopAskQty(sx, poly, "{}")
	if got != 13.687500000000002 {
		t.Fatalf("min=%v", got)
	}
}