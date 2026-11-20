package pmdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePredictL2Parquet_localFile(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "var", "pmdata", "predict_fun", "l2", "2026-10-16", "2905316.parquet")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("no local predict parquet:", err)
	}
	snaps, err := ParsePredictL2Parquet(data, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) == 0 {
		t.Fatal("expected snapshots")
	}
	for _, s := range snaps {
		if len(s.Bids) == 0 && len(s.Asks) == 0 {
			t.Fatalf("empty book at ts %d", s.TsMs)
		}
	}
}