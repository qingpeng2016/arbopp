package book

import (
	"testing"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
)

func TestKalshiSeqGapPauses(t *testing.T) {
	s := NewStore()
	if !s.ApplyKalshiSnapshot("T", []Level{{Price: 0.40, Size: 10}}, []Level{{Price: 0.40, Size: 10}}, 1, timeZero()) {
		t.Fatal("snapshot")
	}
	got, ok := s.Get(appcommon.PlatformKalshi, "T")
	if !ok || !got.Live {
		t.Fatal("expected live book")
	}
	if len(got.Asks) != 1 || got.Asks[0].Price < 0.59 || got.Asks[0].Price > 0.61 {
		t.Fatalf("yes ask from no bid: %+v", got.Asks)
	}
	applied, gap := s.ApplyKalshiDelta("T", "yes", 0.41, 1, 2, timeZero())
	if !applied || gap {
		t.Fatal("seq 2 follows 1")
	}
	_, gap = s.ApplyKalshiDelta("T", "yes", 0.42, 1, 4, timeZero())
	if !gap {
		t.Fatal("expected gap")
	}
	got, _ = s.Get(appcommon.PlatformKalshi, "T")
	if got.Live {
		t.Fatal("gap must pause the book")
	}
	_, gap = s.ApplyKalshiDelta("T", "yes", 0.42, 1, 5, timeZero())
	if gap {
		t.Fatal("delta on a paused book is ignored")
	}
}

func TestPolyLevelIgnoredUntilBook(t *testing.T) {
	s := NewStore()
	if s.ApplyPolyLevel("yes", "SELL", 0.42, 0, timeZero()) {
		t.Fatal("level before websocket book must not trade")
	}
	if !s.ApplyPolyBook("yes", []Level{{Price: 0.4, Size: 5}}, []Level{{Price: 0.42, Size: 5}}, 0.01, 1, timeZero()) {
		t.Fatal("book")
	}
	if !s.ApplyPolyLevel("yes", "SELL", 0.42, 0, timeZero()) {
		t.Fatal("expected absolute size update")
	}
	got, _ := s.Get(appcommon.PlatformPolymarket, "yes")
	if len(got.Asks) != 0 {
		t.Fatalf("size 0 removes the level: %+v", got.Asks)
	}
}

func timeZero() time.Time { return time.Time{} }
