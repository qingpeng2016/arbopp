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

func TestPredictFeeSharesFitBook(t *testing.T) {
	// Predict 卖一 10 @0.40、费率 2%；净 qty=10 放大后 >10 → 应拒绝。
	fee := 0.02
	px := 0.40
	bookSz := 10.0
	pred := ExecSide{
		Side:     Side{Fee: fee, YesPx: px, YesSz: bookSz},
		Platform: "PREDICT",
	}

	ok, need, have := pred.PredictFeeSharesFitBook(true, bookSz)
	if ok {
		t.Fatalf("expected reject: need=%v have=%v", need, have)
	}
	if !(need > have) {
		t.Fatalf("need=%v should exceed have=%v", need, have)
	}

	// 缩小净 qty 使放大后刚好 ≤ book
	ratio := PredictNetRatio(px, fee)
	netOK := bookSz * ratio
	ok2, need2, have2 := pred.PredictFeeSharesFitBook(true, netOK)
	if !ok2 {
		t.Fatalf("expected fit: need=%v have=%v", need2, have2)
	}

	// 非 Predict 直接通过
	poly := ExecSide{Platform: "POLYMARKET", Side: Side{YesPx: px, YesSz: bookSz}}
	if ok3, _, _ := poly.PredictFeeSharesFitBook(true, bookSz); !ok3 {
		t.Fatal("non-predict should pass")
	}
}
