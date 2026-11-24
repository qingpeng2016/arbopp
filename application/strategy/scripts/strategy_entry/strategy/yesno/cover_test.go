package yesno

import (
	"testing"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

func TestCoverUnderThreshold(t *testing.T) {
	yes := []book.Level{{Price: 0.40, Size: 10}}
	no := []book.Level{{Price: 0.40, Size: 10}}
	q, ok := Cover(yes, no, 0, 0, 1, 10, 0.95)
	if !ok || q.Qty < 9.9 || q.PerShare > 0.81 {
		t.Fatalf("quote %+v ok=%v", q, ok)
	}
}

func TestCoverRejectsWhenOverMaxCost(t *testing.T) {
	yes := []book.Level{{Price: 0.50, Size: 10}}
	no := []book.Level{{Price: 0.50, Size: 10}}
	_, ok := Cover(yes, no, 0, 0, 1, 10, 0.95)
	if ok {
		t.Fatal("expected no quote when per share >= maxCost")
	}
}

func TestFee(t *testing.T) {
	if Fee(0, 0.05) != 0 || Fee(0.5, 0) != 0 || Fee(1, 0.05) != 0 {
		t.Fatal("invalid price or coef should yield zero fee")
	}
	want := 0.05 * 0.4 * 0.6
	if got := Fee(0.4, 0.05); got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("Fee(0.4,0.05)=%v want %v", got, want)
	}
}

func TestTopHitUnderMaxCost(t *testing.T) {
	qty, per, ok := TopHit(0.38, 500, 0.44, 500, 0.05, 0.07, 5, 100, 0.95)
	if !ok || qty <= 0 || per >= 0.95 {
		t.Fatalf("qty=%v per=%v ok=%v", qty, per, ok)
	}
}