package syncopt

// OutcomeTypeSet market_sync 本轮要同步的盘口大类（由 config sync_outcome_types 解析）。
type OutcomeTypeSet struct {
	Moneyline bool
	Handicap  bool
	Total     bool
}

func (s OutcomeTypeSet) Empty() bool {
	return !s.Moneyline && !s.Handicap && !s.Total
}

func DefaultOutcomeTypeSet() OutcomeTypeSet {
	return OutcomeTypeSet{Moneyline: true}
}
