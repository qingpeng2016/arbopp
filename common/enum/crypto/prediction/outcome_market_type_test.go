package prediction

import "testing"

func TestParseSyncOutcomeTypesGroupedBySport(t *testing.T) {
	raw := `[{"sport_cn":"足球","sub_category":"Football","items":["moneyline","handicap","total"]}]`
	cfg := ParseSyncOutcomeTypes(raw, "json")
	got := cfg.For(SubMarketCategoryFootball)
	if len(got) != 3 || got[0] != OutcomeMarketTypeMoneyline || got[2] != OutcomeMarketTypeTotal {
		t.Fatalf("football types: %#v", got)
	}
	if cfg.SportCN[SubMarketCategoryFootball] != "足球" {
		t.Fatalf("sport_cn: %q", cfg.SportCN[SubMarketCategoryFootball])
	}
	if cfg.For(SubMarketCategoryBasketball) != nil {
		t.Fatal("unlisted sport should wait for its own group")
	}
}

func TestParseSyncOutcomeTypesLegacyList(t *testing.T) {
	cfg := ParseSyncOutcomeTypes(`["moneyline","handicap"]`, "json")
	got := cfg.For(SubMarketCategoryFootball)
	if len(got) != 2 || got[1] != OutcomeMarketTypeHandicap {
		t.Fatalf("legacy: %#v", got)
	}
	if cfg.For(SubMarketCategoryBasketball) == nil {
		t.Fatal("legacy list applies to every sport")
	}
}
