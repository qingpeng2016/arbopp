package yesno

import "testing"

func TestCheckOrderSizeUsePlatformMinQty(t *testing.T) {
	// min_price=0.4 → min_qty=5/0.4=12.5, max_qty=20/0.4=50
	// platform_min_qty=20 → 落在 [12.5,50] → qty=20
	r := CheckOrderSize(OrderSizeCheckInput{
		Platforms: []string{"POLYMARKET", "SXBET"},
		BestAsks:  []float64{0.40, 0.50},
		AskSizes:  []float64{30, 20},
		MinAmount: 5, MaxAmount: 20,
	})
	if !r.OK || r.Qty != 20 || r.PlatformMinPrice != 0.40 || r.PlatformMinQty != 20 {
		t.Fatalf("got %+v", r)
	}
}

func TestCheckOrderSizeCapAtMaxQty(t *testing.T) {
	// min_price=0.5 → max_qty=5/0.5=10; platform_min_qty=100 → qty=10
	r := CheckOrderSize(OrderSizeCheckInput{
		BestAsks:  []float64{0.50, 0.60},
		AskSizes:  []float64{100, 80},
		MinAmount: 5, MaxAmount: 5,
	})
	if !r.OK || r.Qty != 10 {
		t.Fatalf("got %+v", r)
	}
}

func TestCheckOrderSizeRejectBelowMinQty(t *testing.T) {
	// min_price=0.5 → min_qty=5/0.5=10; platform_min_qty=3 → reject
	r := CheckOrderSize(OrderSizeCheckInput{
		BestAsks:  []float64{0.50, 0.40},
		AskSizes:  []float64{3, 50},
		MinAmount: 5, MaxAmount: 20,
	})
	if r.OK || r.FailReason != "platform_min_qty_lt_min_qty" {
		t.Fatalf("got %+v", r)
	}
	if r.Snapshot["platform_min_qty"] == nil || r.Snapshot["min_qty"] == nil {
		t.Fatalf("snapshot missing thresholds: %+v", r.Snapshot)
	}
}
