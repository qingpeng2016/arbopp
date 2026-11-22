package main

// outcomeRec 与 data/market_outcome.jsonl 字段一致（本包自用，不引 domain）。
type outcomeRec struct {
	MapKey                  string `json:"map_key"`
	OutcomeGroupKey         string `json:"outcome_group_key"` // 可选；有则直接用，与线上一致
	ID                      uint   `json:"id"`
	Platform                string `json:"platform"`
	PlatformMarketOutcomeID string `json:"platform_market_outcome_id"`
	OutcomeName             string `json:"outcome_name"`
	OutcomeType             string `json:"outcome_type"`
	YesTokenID              string `json:"yes_token_id"`
	NoTokenID               string `json:"no_token_id"`
	Slug                    string `json:"slug"`
}

const (
	platPoly     = "POLYMARKET"
	platPredict  = "PREDICT"
	platSxbet    = "SXBET"
	platKalshi   = "KALSHI"
)