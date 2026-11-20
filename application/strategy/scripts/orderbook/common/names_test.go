package common

import (
	"testing"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestTokenNameForOutcomeYesNo(t *testing.T) {
	yes := "111"
	no := "222"
	oc := &persistententity.MarketOutcome{YesTokenID: yes, NoTokenID: no}
	if got := deref(TokenNameForOutcome(oc, yes)); got != "Yes" {
		t.Fatalf("yes token: got %q", got)
	}
	if got := deref(TokenNameForOutcome(oc, no)); got != "No" {
		t.Fatalf("no token: got %q", got)
	}
}

func TestTokenNameFromConfig(t *testing.T) {
	cfg := `{"yes_label":"Pakistan Women","no_label":"Ireland Women"}`
	oc := &persistententity.MarketOutcome{
		YesTokenID: "a",
		NoTokenID:  "b",
		Config:     &cfg,
	}
	if got := deref(TokenNameForOutcome(oc, "a")); got != "Pakistan Women" {
		t.Fatalf("yes label: got %q", got)
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}