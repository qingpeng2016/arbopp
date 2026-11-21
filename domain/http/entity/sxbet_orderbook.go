package entity

// SxbetBookLevel orderbook_v3 单档（maker frame）。
type SxbetBookLevel struct {
	PercentageOdds string `json:"percentageOdds"`
	Size           string `json:"size"`
}

// SxbetOrderbookV3 REST 快照与 WS publication 共用结构。
type SxbetOrderbookV3 struct {
	MarketHash string           `json:"marketHash"`
	EventID    string           `json:"eventId"`
	Version    string           `json:"version"`
	OutcomeOne []SxbetBookLevel `json:"outcomeOne"`
	OutcomeTwo []SxbetBookLevel `json:"outcomeTwo"`
}

// SxbetRealtimeTokenResponse GET /user/realtime-token-v3/api-key。
type SxbetRealtimeTokenResponse struct {
	Status string `json:"status"`
	Token  string `json:"token"`
}

// SxbetOrderbookSnapshotResponse GET /orderbook-v3/snapshot/market。
type SxbetOrderbookSnapshotResponse struct {
	Status string           `json:"status"`
	Data   SxbetOrderbookV3 `json:"data"`
}