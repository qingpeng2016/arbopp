package yesno

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
)

func TestNetCostPerSharePlatforms(t *testing.T) {
	const rate = 0.05
	poly := NetCostPerShare(appcommon.PlatformPolymarket, 0.48, rate)
	wantPoly := 0.48 + Fee(0.48, rate)
	if poly < wantPoly-1e-9 || poly > wantPoly+1e-9 {
		t.Fatalf("poly cost %v want %v", poly, wantPoly)
	}
	pred := NetCostPerShare(appcommon.PlatformPredict, 0.41, 0.02)
	ratio := PredictNetRatio(0.41, 0.02)
	if pred < 0.41/ratio-1e-9 || pred > 0.41/ratio+1e-9 {
		t.Fatalf("predict cost %v", pred)
	}
	sx := NetCostPerShare(appcommon.PlatformSxbet, 0.35, 0.02)
	denom := 1 - 0.02*(1-0.35)
	if sx < 0.35/denom-1e-9 || sx > 0.35/denom+1e-9 {
		t.Fatalf("sx cost %v", sx)
	}
}

func TestArbQtyFromPickTopSizes(t *testing.T) {
	p := FilledPick{TopSizes: [2]float64{300, 80}}
	if q := arbQtyFromPick(p); q != 80 {
		t.Fatalf("qty %v want 80", q)
	}
}