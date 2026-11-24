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

// SxbetPlaceOrderResult 发单结果（IOC + waitForOutcome）。
type SxbetPlaceOrderResult struct {
	OrderID  string
	Status   string
	FillQty  float64 // 策略份额口径（base units / 1e6）
	State    string
	RawError string
}

// SxbetRepo SX Bet REST（公开市场数据无需 API Key；下单/盘口快照等另需 x-sx-api-key）。
type SxbetRepo interface {
	APIKey() string
	GetActiveMarkets(ctx context.Context, params SxbetListActiveMarketsParams) (*entity.SxbetActiveMarketsResponse, error)
	GetActiveLeagues(ctx context.Context) ([]entity.SxbetLeague, error)
	GetSports(ctx context.Context) ([]entity.SxbetSport, error)
	GetRealtimeToken(ctx context.Context) (string, error)
	GetOrderbookSnapshot(ctx context.Context, marketHash string) (*entity.SxbetOrderbookV3, error)
	GetMetadataObv3(ctx context.Context) (*entity.SxbetMetadataObv3, error)
	// PlaceIOCOrder LIMIT+IOC（当场能成交多少是多少），waitForOutcome 取成交量。
	PlaceIOCOrder(ctx context.Context, acc entity.SxbetAccount, req entity.SxbetPlaceOrderReq) (*SxbetPlaceOrderResult, error)
}