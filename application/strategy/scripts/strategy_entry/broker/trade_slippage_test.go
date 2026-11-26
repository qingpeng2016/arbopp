package broker

import "testing"

func TestLimitPriceWithSlippageBps(t *testing.T) {
	got := limitPriceWithSlippageBps(0.53, 500)
	want := 0.53 * 1.05
	if got < want-1e-12 || got > want+1e-12 {
		t.Fatalf("got %v want %v", got, want)
	}
	if limitPriceWithSlippageBps(0.53, 0) != 0.53 {
		t.Fatal("zero bps should keep price")
	}
	if limitPriceWithSlippageBps(0.99, 500) != 0.99 {
		t.Fatal("should cap at 0.99")
	}
}
