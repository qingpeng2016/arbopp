package yesno

import "testing"

func TestFillsAligned(t *testing.T) {
	t.Run("within_1pct", func(t *testing.T) {
		matched, ok := FillsAligned([]float64{10, 9.99992}, 0.01)
		if !ok || matched != 9.99992 {
			t.Fatalf("got matched=%v ok=%v", matched, ok)
		}
	})
	t.Run("zero_fill", func(t *testing.T) {
		if _, ok := FillsAligned([]float64{0, 9.99992}, 0.01); ok {
			t.Fatal("expected fail when one fill is 0")
		}
	})
	t.Run("over_tol", func(t *testing.T) {
		if _, ok := FillsAligned([]float64{10, 9.8}, 0.01); ok {
			t.Fatal("expected fail when (max-min)/max > 1%")
		}
	})
	t.Run("strict_when_pct_zero", func(t *testing.T) {
		if _, ok := FillsAligned([]float64{10, 9.999}, 0); ok {
			t.Fatal("expected strict equality when pct=0")
		}
		if matched, ok := FillsAligned([]float64{10, 10}, 0); !ok || matched != 10 {
			t.Fatalf("got matched=%v ok=%v", matched, ok)
		}
	})
	t.Run("triple", func(t *testing.T) {
		matched, ok := FillsAligned([]float64{10, 9.95, 9.97}, 0.01)
		if !ok || matched != 9.95 {
			t.Fatalf("got matched=%v ok=%v", matched, ok)
		}
	})
}

func TestEstProfitMaxQtyMinusCost(t *testing.T) {
	// max_qty=9.99992, amount=0.4*9.99992, fee=0 → 9.99992-3.999968=5.999952
	got := EstProfitMaxQtyMinusCost(
		[]float64{0, 9.99992},
		[]float64{0, 0.4 * 9.99992},
		[]float64{0, 0},
	)
	want := 9.99992 - 0.4*9.99992
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
	got = EstProfitMaxQtyMinusCost(
		[]float64{10, 10},
		[]float64{5.2, 4.0},
		[]float64{0.01, 0.02},
	)
	want = 10 - 9.2 - 0.03
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}
