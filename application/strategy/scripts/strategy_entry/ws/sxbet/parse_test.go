package sxbet

import "testing"

func TestParseOrderbookV3Payload(t *testing.T) {
	raw := []byte(`{"marketHash":"0xabc","version":"1","outcomeOne":[{"percentageOdds":"50000000000000000000","size":"1000000"}],"outcomeTwo":[]}`)
	ob, err := parseOrderbookV3(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ob.MarketHash != "0xabc" || ob.Version != "1" {
		t.Fatalf("got %+v", ob)
	}
}