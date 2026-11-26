package broker

import "testing"

func TestLimitPriceWithAbsSlip(t *testing.T) {
	if got := limitPriceWithAbsSlip(0.2, 0.05); got < 0.25-1e-12 || got > 0.25+1e-12 {
		t.Fatalf("0.2+0.05 got %v", got)
	}
	if got := limitPriceWithAbsSlip(0.9, 0.05); got < 0.95-1e-12 || got > 0.95+1e-12 {
		t.Fatalf("0.9+0.05 got %v", got)
	}
	if limitPriceWithAbsSlip(0.53, 0) != 0.53 {
		t.Fatal("zero abs should keep price")
	}
	if limitPriceWithAbsSlip(0.97, 0.05) != 0.99 {
		t.Fatal("should cap at 0.99")
	}
}

func TestAbsoluteSlipToPredictBps(t *testing.T) {
	// 买 isMinOut：0.2→0.25 ⇒ s=0.05/0.25=0.2 ⇒ 2000 bps
	if got := AbsoluteSlipToPredictBps(0.2, 0.05, false, true); got != 2000 {
		t.Fatalf("minOut 0.2: got %d want 2000", got)
	}
	// 买 isMinOut：0.9→0.95 ⇒ s=0.05/0.95 ≈ 0.05263 ⇒ 526 bps
	if got := AbsoluteSlipToPredictBps(0.9, 0.05, false, true); got != 526 {
		t.Fatalf("minOut 0.9: got %d want 526", got)
	}
	// 买 !isMinOut：s=abs/price；0.2+0.05 → 2500 bps
	if got := AbsoluteSlipToPredictBps(0.2, 0.05, false, false); got != 2500 {
		t.Fatalf("!minOut 0.2: got %d want 2500", got)
	}
	if AbsoluteSlipToPredictBps(0.5, 0, false, true) != 0 {
		t.Fatal("zero abs → 0 bps")
	}
}
