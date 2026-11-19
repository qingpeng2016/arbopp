package kalshi

import "testing"

func TestParseSnapshotAndDelta(t *testing.T) {
	raw := []byte(`{"type":"orderbook_snapshot","seq":1,"msg":{"market_ticker":"KX","yes_dollars_fp":[["0.4000","10"]],"no_dollars_fp":[["0.4500","8"]]}}`)
	got, ok, err := Parse(raw)
	if err != nil || !ok || !got.Snapshot || got.Yes[0].Price != 0.4 || got.No[0].Size != 8 {
		t.Fatalf("snap: %+v %v %v", got, ok, err)
	}
	raw = []byte(`{"type":"orderbook_delta","seq":2,"msg":{"market_ticker":"KX","price_dollars":"0.4000","delta_fp":"-3","side":"yes"}}`)
	got, ok, err = Parse(raw)
	if err != nil || !ok || got.Snapshot || got.Delta != -3 || got.Seq != 2 {
		t.Fatalf("delta: %+v %v %v", got, ok, err)
	}
}
