package orderfillsync

import (
	"testing"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/shopspring/decimal"
)

func TestCalcArbProfitAmount(t *testing.T) {
	orders := []*persistententity.ArbBinaryOrder{
		{Platform: "POLYMARKET", Price: decimal.RequireFromString("0.40"), FillQty: decimal.RequireFromString("10"), Status: "FILLED"},
		{Platform: "PREDICT", Price: decimal.RequireFromString("0.50"), FillQty: decimal.RequireFromString("10"), Status: "FILLED"},
	}
	// amount=0.4*10+0.5*10=9; min_qty=min(10,10)=10; profit=1
	got, ok := calcArbProfitAmount(orders)
	if !ok {
		t.Fatal("expected ok")
	}
	if got < 0.999 || got > 1.001 {
		t.Fatalf("profit=%v want 1", got)
	}
}

func TestCalcArbProfitAmountSamePlatformLegs(t *testing.T) {
	orders := []*persistententity.ArbBinaryOrder{
		{Platform: "POLYMARKET", Price: decimal.RequireFromString("0.13"), FillQty: decimal.RequireFromString("27.27"), Status: "FILLED"},
		{Platform: "POLYMARKET", Price: decimal.RequireFromString("0.71"), FillQty: decimal.RequireFromString("10"), Status: "PARTIAL"},
		{Platform: "PREDICT", Price: decimal.RequireFromString("0.09"), FillQty: decimal.RequireFromString("10"), Status: "FILLED"},
	}
	// poly qty=37.27, predict=10 → min=10
	// amount=0.13*27.27 + 0.71*10 + 0.09*10 = 3.5451+7.1+0.9=11.5451
	// profit=10-11.5451=-1.5451
	got, ok := calcArbProfitAmount(orders)
	if !ok {
		t.Fatal("expected ok")
	}
	want := 10 - (0.13*27.27 + 0.71*10 + 0.09*10)
	if got < want-1e-6 || got > want+1e-6 {
		t.Fatalf("profit=%v want %v", got, want)
	}
}

func TestAllOrdersTerminal(t *testing.T) {
	if allOrdersTerminal([]*persistententity.ArbBinaryOrder{
		{Status: "FILLED"}, {Status: "PENDING"},
	}) {
		t.Fatal("pending should block")
	}
	if !allOrdersTerminal([]*persistententity.ArbBinaryOrder{
		{Status: "FILLED"}, {Status: "NO_FILL"}, {Status: "partial"},
	}) {
		t.Fatal("all terminal")
	}
}
