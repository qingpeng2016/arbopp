package yesno

import "testing"

func TestPersistableBinarySnap(t *testing.T) {
	if persistableBinarySnap(0.94, 0.95) {
		// ok
	} else {
		t.Fatal("expected persistable")
	}
	if persistableBinarySnap(1.076557, 0.95) {
		t.Fatal("no arb when perShare above 1")
	}
	if persistableBinarySnap(0.96, 0.95) {
		t.Fatal("no arb when perShare >= maxCost")
	}
}