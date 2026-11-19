package fetchopt

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
)

func TestPlatformFetch_Normalized_polyCap(t *testing.T) {
	f := PlatformFetch{BatchSize: 999, BatchConcurrency: 0}.Normalized(appcommon.PlatformPolymarket)
	if f.BatchSize != 500 || f.BatchConcurrency != 1 {
		t.Fatalf("got batch=%d conc=%d", f.BatchSize, f.BatchConcurrency)
	}
}
