package watchgroup

import "time"

const repeatWindow = 5 * time.Second

// RecentHits suppresses identical dry-run records while keeping later quote
// changes visible. The strategy runner serializes access with its order mutex.
type RecentHits map[HitKey]time.Time

type HitKey struct {
	Kind     string
	Indices  [3]int
	Prices   [3]float64
	Depths   [3]float64
	Cost     float64
	Quantity float64
}

func (h RecentHits) Prune(at time.Time) {
	for key, last := range h {
		if at.Sub(last) > repeatWindow {
			delete(h, key)
		}
	}
}

// Save skips a signature already recorded in the window. A failed insert
// remains eligible for the next scan.
func (h RecentHits) Save(key HitKey, at time.Time, insert func() bool) bool {
	if !h[key].IsZero() || !insert() {
		return false
	}
	h[key] = at
	return true
}
