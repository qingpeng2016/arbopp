package pmdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePolymarketL2Parquet_localFile(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "var", "pmdata", "polymarket", "l2", "2026-10-16", "epl-eve-che-2026-10-17-eve.parquet")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("no local pmdata parquet:", err)
	}
	snaps, err := ParsePolymarketL2Parquet(data, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) == 0 {
		t.Fatal("expected snapshots")
	}
	hasBook := false
	for _, s := range snaps {
		if len(s.Bids) > 0 && len(s.Asks) > 0 && s.Bids[0][0] > 0 && s.Asks[0][0] > 0 {
			hasBook = true
			break
		}
	}
	if !hasBook {
		t.Fatalf("no usable bids/asks in %d snaps", len(snaps))
	}
}