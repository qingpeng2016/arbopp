package repository

import (
	"context"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

type PolymarketRepo interface {
	// GetMarketByMarketId 通过市场ID，获取市场信息
	GetMarketByMarketId(ctx context.Context, req entity.GetMarketRequest) (*entity.Market, error)

	// GetMarketsByMarketAddressList 获取市场地址，获取市场信息
	GetMarketsByMarketAddressList(ctx context.Context, req entity.GetMarketsByConditionIdsRequest) ([]entity.Market, error)

	// ListMarkets 列出市场（支持 closed、limit、offset、tag_id 等过滤，参考 Gamma API）
	ListMarkets(ctx context.Context, req entity.ListMarketsRequest) ([]entity.Market, error)

	// ListMarketsKeyset 游标分页拉取市场（可按 sports_market_types 只要胜平负/让分/大小球）
	ListMarketsKeyset(ctx context.Context, req entity.ListMarketsKeysetRequest) (*entity.PolymarketMarketsKeyset, error)

	// ListEventsPagination 分页拉取事件（含嵌套 markets，offset 上限约 2000）
	ListEventsPagination(ctx context.Context, req entity.ListEventsPaginationRequest) (*entity.PolymarketEventsPagination, error)

	// ListEventsKeyset 游标分页拉取事件（深分页，对齐官网 sports/games）
	ListEventsKeyset(ctx context.Context, req entity.ListEventsKeysetRequest) (*entity.PolymarketEventsKeyset, error)

	// GetSportsMetadata 获取运动元数据（GET /sports），用于解析各运动的 tag IDs 以过滤运动 market
	GetSportsMetadata(ctx context.Context) ([]entity.SportsMetadata, error)

	// GetOrderBook 获取订单簿
	GetOrderBook(ctx context.Context, req entity.GetOrderBookRequest) (*entity.OrderBookResponse, error)

	// BatchGetOrderBooks 批量获取订单簿摘要
	BatchGetOrderBooks(ctx context.Context, req entity.BatchGetOrderBooksRequest) (*entity.BatchGetOrderBooksResponse, error)

	// PlaceOrder 单笔下单（支持限价/市价）
	PlaceOrder(ctx context.Context, req entity.PlaceOrderReq, polymarketAccount entity.PolymarketAccount) (*entity.PlaceOrderResp, error)

	// GetOrderById 查询单个订单
	GetOrderById(ctx context.Context, req entity.GetOrderReq, polymarketAccount entity.PolymarketAccount) (*entity.GetOrderResp, error)

	// GetAllOrders 获取用户所有的订单
	GetAllOrders(ctx context.Context, polymarketAccount entity.PolymarketAccount, market *string) (*entity.GetActiveOrdersResponse, error)

	// GetTrades 查询成交记录
	GetTrades(ctx context.Context, req entity.GetTradesReq, polymarketAccount entity.PolymarketAccount) (*entity.GetTradesResp, error)

	// GetPositions 获取持仓
	GetPositions(ctx context.Context, proxyAddress string) (*entity.GetMagicAccountPositionsResponse, error)

	// GetPositionsV2 查询生产 Data API v2 持仓（带 cursor 翻页）。
	GetPositionsV2(ctx context.Context, proxyAddress, conditionID string) (*entity.GetMagicAccountPositionsResponse, error)

	// GetTradeHistoryV3V2 查询 Data API v2 /activity（带 cursor 翻页及 condition 分批）。
	GetTradeHistoryV3V2(ctx context.Context, req entity.GetTradeHistoryRequest, polymarketAccount entity.PolymarketAccount) (*entity.GetTradeHistoryResponse, error)

	// GetAccountInfo 获取账户信息
	GetAccountInfo(ctx context.Context, polymarketAccount entity.PolymarketAccount) (*entity.MagicAccountInfo, error)

	// GetPolymarketFeeRate 查询 CLOB /fee-rate（base fee bps）。
	GetPolymarketFeeRate(ctx context.Context, tokenID string, polymarketAccount entity.PolymarketAccount) (string, error)

	// BatchCancelOrder 批量取消订单
	BatchCancelOrder(ctx context.Context, req entity.BatchCancelOrderRequest, polymarketAccount entity.PolymarketAccount) (*entity.BatchCancelOrderResponse, error)

	// GetEventBySlug 根据slug获取事件信息
	GetEventBySlug(ctx context.Context, req entity.GetEventBySlugRequest) (*entity.GetEventBySlugResponse, error)

	// CreateAPIKey 创建私钥
	CreateAPIKey(ctx context.Context, privateKeyHex, signerAddress string) (apiKey, secret, passphrase string, err error)

	// CreateOrDeriveAPIKey 用私钥换 CLOB L2 三件套（nonce=0：先 create，失败则 derive）。
	CreateOrDeriveAPIKey(ctx context.Context, privateKeyHex, signerAddress string) (apiKey, secret, passphrase string, err error)
}
