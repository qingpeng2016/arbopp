package yesno

import (
	"testing"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

func TestTopFromStorePredictNOBook(t *testing.T) {
	store := book.NewStore()
	mid := "pm-1"
	ts := time.Now().UTC()
	store.ApplyPredictBook(mid, []book.Level{{Price: 0.38, Size: 10}}, []book.Level{{Price: 0.40, Size: 12}}, ts)
	store.SyncComplementNoFromYes(appcommon.PlatformPredict, mid, mid+":no", ts)
	oc := OutcomeFields(appcommon.PlatformPredict, mid, mid, "")
	s, ok := TopFromStore(store, oc, 0, 0)
	if !ok {
		t.Fatal("TopFromStore")
	}
	if s.YesPx != 0.40 || s.NoPx != 0.62 {
		t.Fatalf("yes=%v no=%v", s.YesPx, s.NoPx)
	}
}