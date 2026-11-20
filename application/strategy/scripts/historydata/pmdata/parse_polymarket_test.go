package pmdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePolymarketL2Parquet_localFile(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "var", "pmdata", "polymarket", "l2", "2026-10-17", "epl-eve-che-2026-10-17-eve.parquet")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("no local pmdata parquet:", err)
	}
	snaps, err := ParsePolymarketL2Parquet(data, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) < 10 {
		t.Fatalf("expected many snapshots, got %d", len(snaps))
	}
	deep := 0
	for _, s := range snaps {
		if len(s.Bids) > 1 {
			deep++
		}
	}
	if deep < 5 {
		t.Fatalf("expected multi-level book snapshots from book events, got %d", deep)
	}
}