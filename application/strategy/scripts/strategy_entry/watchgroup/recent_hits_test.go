package watchgroup

import (
	"testing"
	"time"
)

func TestRecentHitsKeepsPriceChangesAndRetriesFailedInsert(t *testing.T) {
	hits := make(RecentHits)
	now := time.Now()
	key := HitKey{Kind: "BINARY_YES_NO", Indices: [3]int{1, 2}, Prices: [3]float64{.4, .5}, Quantity: 5}
	saves := 0
	save := func() bool {
		saves++
		return true
	}
	if hits.Save(key, now, func() bool { return false }) || !hits.Save(key, now, save) {
		t.Fatal("failed inserts must allow the next attempt")
	}
	if hits.Save(key, now.Add(time.Second), save) || saves != 1 {
		t.Fatal("identical opportunity was inserted twice")
	}
	changed := key
	changed.Prices[0] = .39
	if !hits.Save(changed, now.Add(time.Second), save) {
		t.Fatal("price change was suppressed")
	}
	later := now.Add(repeatWindow + time.Millisecond)
	hits.Prune(later)
	if !hits.Save(key, later, save) || saves != 3 {
		t.Fatal("expired signature should be eligible again")
	}
}
