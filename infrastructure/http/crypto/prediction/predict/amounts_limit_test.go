package predict

import (
	"math/big"
	"testing"
)

func TestMarketAmountsAtLimitBuy(t *testing.T) {
	got, ok := marketAmountsAtLimit(10, 0.45, false, 0)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.takerAmount == nil || got.makerAmount == nil {
		t.Fatal("nil amounts")
	}
	// taker = shares; maker ≈ price * shares
	wantTaker := floatSharesToWei(10)
	wantTaker = retainSignificantDigits(wantTaker, 5)
	if got.takerAmount.Cmp(wantTaker) != 0 {
		t.Fatalf("taker=%s want %s", got.takerAmount, wantTaker)
	}
	prec := big.NewInt(int64(weiPrecision))
	wantMaker := new(big.Int).Mul(etherFloatToWei(0.45), wantTaker)
	wantMaker.Div(wantMaker, prec)
	if got.makerAmount.Cmp(wantMaker) != 0 {
		t.Fatalf("maker=%s want %s", got.makerAmount, wantMaker)
	}
}

func TestMarketAmountsAtLimitRejects(t *testing.T) {
	if _, ok := marketAmountsAtLimit(10, 0, false, 0); ok {
		t.Fatal("zero limit should fail")
	}
	if _, ok := marketAmountsAtLimit(0, 0.4, false, 0); ok {
		t.Fatal("zero qty should fail")
	}
}

func TestFilterAsksAtOrBelow(t *testing.T) {
	asks := []depthLevel{
		{priceWei: etherFloatToWei(0.40), qtyWei: big.NewInt(1e18)},
		{priceWei: etherFloatToWei(0.45), qtyWei: big.NewInt(1e18)},
		{priceWei: etherFloatToWei(0.50), qtyWei: big.NewInt(1e18)},
	}
	got := filterAsksAtOrBelow(asks, 0.45)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if filterAsksAtOrBelow(asks, 0) == nil || len(filterAsksAtOrBelow(asks, 0)) != 3 {
		t.Fatalf("zero limit should keep all")
	}
}

func TestFilterBidsAtOrAbove(t *testing.T) {
	bids := []depthLevel{
		{priceWei: etherFloatToWei(0.55), qtyWei: big.NewInt(1e18)},
		{priceWei: etherFloatToWei(0.40), qtyWei: big.NewInt(1e18)},
	}
	got := filterBidsAtOrAbove(bids, 0.50)
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
}
