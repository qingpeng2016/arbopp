package predict

import (
	"math/big"
	"testing"
)

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
