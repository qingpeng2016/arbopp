package leaguecfg

import (
	"testing"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

func TestPlatformCategoryRunsSortNo(t *testing.T) {
	rows := []persistententity.ConfigMarketCategory{
		{Platform: appcommon.PlatformPolymarket, SubCategory: "Basketball", SortNo: 20, Concurrency: 2},
		{Platform: appcommon.PlatformPolymarket, SubCategory: "Football", SortNo: 10, Concurrency: 5},
		{Platform: appcommon.PlatformKalshi, SubCategory: "Football", SortNo: 1, Concurrency: 3},
	}
	plan := PlatformCategoryRuns(rows, appcommon.PlatformPolymarket)
	if len(plan) != 2 {
		t.Fatalf("want 2, got %d", len(plan))
	}
	if plan[0].SubCategory != "Football" || plan[0].Concurrency != 5 {
		t.Fatalf("first: %+v", plan[0])
	}
	if plan[1].SubCategory != "Basketball" {
		t.Fatalf("second: %+v", plan[1])
	}
}

func TestFilterByActivePlatforms(t *testing.T) {
	rows := []persistententity.ConfigMarketCategory{
		{Platform: appcommon.PlatformPolymarket, SubCategory: "Football"},
		{Platform: appcommon.PlatformPredict, SubCategory: "Football"},
	}
	active := map[string]struct{}{appcommon.PlatformPolymarket: {}}
	filtered := FilterByActivePlatforms(rows, active)
	if len(filtered) != 1 || filtered[0].Platform != appcommon.PlatformPolymarket {
		t.Fatalf("filtered: %+v", filtered)
	}
}