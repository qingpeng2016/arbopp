package predict

import "testing"

func TestParseOrderbook(t *testing.T) {
	raw := []byte(`{"type":"M","topic":"predictOrderbook/42","data":{"marketId":42,"updateTimestampMs":1736696400000,"asks":[[0.62,1500]],"bids":[[0.61,2000]]}}`)
	kind, msg, _, err := Parse(raw)
	if err != nil || kind != "book" || msg == nil || msg.ID != "42" {
		t.Fatalf("parse: kind=%q msg=%v err=%v", kind, msg, err)
	}
	if len(msg.Asks) != 1 || msg.Asks[0].Price != 0.62 {
		t.Fatalf("asks: %+v", msg.Asks)
	}
}

func TestParseHeartbeat(t *testing.T) {
	kind, _, hb, err := Parse([]byte(`{"type":"M","topic":"heartbeat","data":1736696400000}`))
	if err != nil || kind != "heartbeat" || hb != 1736696400000 {
		t.Fatalf("hb: kind=%q hb=%d err=%v", kind, hb, err)
	}
}
