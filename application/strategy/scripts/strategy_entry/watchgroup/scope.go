package watchgroup

// FixtureScope 套利机会落库用的场次分类/联赛。
type FixtureScope struct {
	Category      *string
	SubCategory   *string
	Competition   *string
	CompetitionEn *string
}

// ScopeForOutcomeIDs 从 YES/NO 腿 outcome 关联父场次取字段（两腿不一致时优先 YES 腿）。
func (w *Watch) ScopeForOutcomeIDs(yesMoID, noMoID uint) FixtureScope {
	if w == nil {
		return FixtureScope{}
	}
	for _, id := range []uint{yesMoID, noMoID} {
		if s := w.scopeForOutcomeID(id); scopeNonEmpty(s) {
			return s
		}
	}
	return FixtureScope{}
}

func (w *Watch) scopeForOutcomeID(outcomeID uint) FixtureScope {
	idx, ok := w.OutcomeIndex(outcomeID)
	if !ok {
		return FixtureScope{}
	}
	oc := w.OutcomeAt(idx)
	if oc == nil {
		return FixtureScope{}
	}
	var scope FixtureScope
	scope.Category = oc.Category
	scope.SubCategory = oc.SubCategory
	scope.Competition = oc.Competition
	if fx, ok := w.MarketsByRowID[oc.MarketRowID]; ok {
		if scope.Category == nil {
			scope.Category = fx.Category
		}
		if scope.SubCategory == nil {
			scope.SubCategory = fx.SubCategory
		}
		if scope.Competition == nil {
			scope.Competition = fx.Competition
		}
		scope.CompetitionEn = fx.CompetitionEn
	}
	return scope
}

func scopeNonEmpty(s FixtureScope) bool {
	return s.Category != nil || s.SubCategory != nil || s.Competition != nil || s.CompetitionEn != nil
}
