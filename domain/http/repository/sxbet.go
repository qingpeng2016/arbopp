package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

// SxbetListActiveMarketsParams GET /markets/active 查询参数。
type SxbetListActiveMarketsParams struct {
	LeagueID      int
	EventID       string
	SportIDs      []int
	Types         []int
	OnlyMainLine  bool
	LiveOnly      bool
	PageSize      int
	PaginationKey string
	GameTimeAfter int64
}

// SxbetRepo SX Bet REST（公开市场数据无需 API Key；下单/盘口快照等另需 x-sx-api-key）。
type SxbetRepo interface {
	GetActiveMarkets(ctx context.Context, params SxbetListActiveMarketsParams) (*entity.SxbetActiveMarketsResponse, error)
	GetActiveLeagues(ctx context.Context) ([]entity.SxbetLeague, error)
	GetSports(ctx context.Context) ([]entity.SxbetSport, error)
}