package pmdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePolymarketRawEvents_localFile(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "var", "pmdata", "polymarket", "l2", "2026-10-17", "epl-eve-che-2026-10-17-eve.parquet")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(path, err)
	}
	rows, err := ParsePolymarketRawEvents(data)
	if err != nil {
		t.Fatal(err)
	}
	book, pc := 0, 0
	for _, r := range rows {
		switch r.EventKind {
		case "book":
			book++
		case "price_change":
			pc++
		}
	}
	if book < 500 || pc < 10000 {
		t.Fatalf("expected many book+pc rows, got book=%d pc=%d total=%d", book, pc, len(rows))
	}
}