package onextwo

const (
	ThresholdKeyYesTriple = "1x2_yes_triple_max_cost"
	ThresholdKeyNoTriple  = "1x2_no_triple_max_cost"
	OppKindYesTriple      = "ONE_X2_YES_TRIPLE"
	OppKindNoTriple       = "ONE_X2_NO_TRIPLE"
	StrategyTypeYesTriple = "ONE_X2_YES_TRIPLE"
	StrategyTypeNoTriple  = "ONE_X2_NO_TRIPLE"
)

// Limits 1X2 三路全覆盖阈值与下单量（与 yesno 共用 min_amount / max_amount）。
type Limits struct {
	YesTripleMax         float64
	NoTripleMax          float64
	MinAmount            float64
	MaxAmount            float64
	FillShareMismatchPct float64
}