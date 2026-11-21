package entity

// SxbetMarket SX Bet 单个二元盘口（GET /markets/active、/markets/find）。
type SxbetMarket struct {
	Status              string   `json:"status"`
	MarketHash          string   `json:"marketHash"`
	OutcomeOneName      string   `json:"outcomeOneName"`
	OutcomeTwoName      string   `json:"outcomeTwoName"`
	OutcomeVoidName     string   `json:"outcomeVoidName"`
	TeamOneName         string   `json:"teamOneName"`
	TeamTwoName         string   `json:"teamTwoName"`
	Type                int      `json:"type"`
	GameTime            int64    `json:"gameTime"`
	Line                *float64 `json:"line"`
	SportXeventID       string   `json:"sportXeventId"`
	LiveEnabled         bool     `json:"liveEnabled"`
	SportLabel          string   `json:"sportLabel"`
	SportID             int      `json:"sportId"`
	LeagueID            int      `json:"leagueId"`
	LeagueLabel         string   `json:"leagueLabel"`
	Group1              string   `json:"group1"`
	MainLine            *bool    `json:"mainLine"`
	IsQuarterLineMarket bool     `json:"isQuarterLineMarket"`
}

// SxbetActiveMarketsData GET /markets/active 的 data 节点。
type SxbetActiveMarketsData struct {
	Markets []SxbetMarket `json:"markets"`
	NextKey string        `json:"nextKey"`
}

// SxbetActiveMarketsResponse GET /markets/active。
type SxbetActiveMarketsResponse struct {
	Status string                 `json:"status"`
	Data   SxbetActiveMarketsData `json:"data"`
}

// SxbetLeague GET /leagues/active 单项。
type SxbetLeague struct {
	LeagueID   int    `json:"leagueId"`
	Label      string `json:"label"`
	SportID    int    `json:"sportId"`
	Popularity float64 `json:"popularity"`
}

// SxbetLeaguesResponse GET /leagues/active。
type SxbetLeaguesResponse struct {
	Status string        `json:"status"`
	Data   []SxbetLeague `json:"data"`
}

// SxbetSport GET /sports。
type SxbetSport struct {
	SportID int    `json:"sportId"`
	Label   string `json:"label"`
}

// SxbetSportsResponse GET /sports。
type SxbetSportsResponse struct {
	Status string       `json:"status"`
	Data   []SxbetSport `json:"data"`
}