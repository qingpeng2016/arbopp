package running

import (
	"testing"

	entrystat "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/stat"
)

func TestMarketIDsFromGroups(t *testing.T) {
	ids := marketIDsFromGroups([]entrystat.Group{
		{MapKey: "a", Markets: map[string]uint{"POLYMARKET": 1, "SXBET": 2}},
		{MapKey: "b", Markets: map[string]uint{"PREDICT": 3, "SXBET": 2}},
	})
	if len(ids) != 3 {
		t.Fatalf("want 3 ids, got %v", ids)
	}
}