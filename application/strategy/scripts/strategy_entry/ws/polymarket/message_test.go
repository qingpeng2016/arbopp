package polymarket

import "testing"

func TestParseBookAndLevel(t *testing.T) {
	raw := []byte(`{"event_type":"book","asset_id":"yes","bids":[{"price":"0.40","size":"5"}],"asks":[{"price":"0.42","size":"4"}],"timestamp":"1700000000000","tick_size":"0.01","min_order_size":"1"}`)
	got, err := Parse(raw)
	if err != nil || len(got) != 1 || got[0].Kind != "book" || got[0].Asks[0].Price != 0.42 {
		t.Fatalf("book: %+v %v", got, err)
	}
	raw = []byte(`{"event_type":"price_change","timestamp":"1700000000000","price_changes":[{"asset_id":"no","price":"0.33","size":"0","side":"SELL"}]}`)
	got, err = Parse(raw)
	if err != nil || len(got) != 1 || got[0].Kind != "price_change" || len(got[0].Changes) != 1 || got[0].Changes[0].ID != "no" || got[0].Changes[0].Size != 0 {
		t.Fatalf("level: %+v %v", got, err)
	}
}

func TestParsePong(t *testing.T) {
	got, err := Parse([]byte("PONG"))
	if err != nil || len(got) != 1 || got[0].Kind != "pong" {
		t.Fatalf("pong: %+v %v", got, err)
	}
}
