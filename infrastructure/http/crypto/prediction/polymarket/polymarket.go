package polymarket

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	polyenum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	repo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
	"go.uber.org/zap"
)

var clobBaseURL = "https://clob.polymarket.com"

const (
	eip712DomainName          = "Polymarket CTF Exchange"
	eip712DomainVersion       = "2"
	clobAuthDomainVersion     = "1"
	eip712ChainID             = 137
	usdcDecimals              = 6
	conditionalTokenDecimals  = 6
	polymarketOperatorAddress = "0x0000000000000000000000000000000000000000"
	ctfExchangeAddress        = "0xE111180000d2663C0091e4f400237545B87B996B"
	negRiskCTFExchangeAddress = "0xe2222d279d744050d28e00520010520000310F59"
	zeroBytes32               = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

type OrderConditionType string

const (
	// Gamma keyset 等大响应可慢；HTTP client 硬上限 60s（见 infrastructure/http/client.go）。
	polymarketSlowRequestLog   = 15 * time.Second
	polymarketSlowRequestAlert = 50 * time.Second
)

const OrderConditionTypeGTD OrderConditionType = "GTD"

type Polymarket struct {
	client       *http.Client
	baseURL      string
	orderBookURL string
	rpcURL       string
}

// NewPolymarket 仅测试/兜底；生产请用 NewPolymarketWithPlatform（platform 表）。
func NewPolymarket(client *http.Client) repo.PolymarketRepo {
	return NewPolymarketWithPlatform(client, "https://gamma-api.polymarket.com", clobBaseURL)
}

// NewPolymarketWithPlatform 从 platform 表注入 Gamma 与 CLOB 地址。
func NewPolymarketWithPlatform(client *http.Client, marketAPIBase, orderAPIBase string) repo.PolymarketRepo {
	if strings.TrimSpace(marketAPIBase) == "" {
		marketAPIBase = "https://gamma-api.polymarket.com"
	}
	if strings.TrimSpace(orderAPIBase) == "" {
		orderAPIBase = clobBaseURL
	}
	return &Polymarket{
		client:       client,
		baseURL:      strings.TrimRight(strings.TrimSpace(marketAPIBase), "/"),
		orderBookURL: strings.TrimRight(strings.TrimSpace(orderAPIBase), "/"),
		rpcURL:       "https://polygon-bor-rpc.publicnode.com",
	}
}

// PlaceOrder 单笔下单（CLOB Go HTTP，支持限价/市价）
func (p *Polymarket) PlaceOrder(ctx context.Context, req entity.PlaceOrderReq, polymarketAccount entity.PolymarketAccount) (*entity.PlaceOrderResp, error) {
	startTime := time.Now()
	reqJSON, _ := json.Marshal(req)
	logger.InfoZ(ctx, "polymarket-PlaceOrder-Req",
		zap.String("request", string(reqJSON)),
		zap.String("tokenID", req.TokenID),
		zap.String("side", string(req.Side)),
		zap.String("orderType", string(req.OrderType)),
		zap.Bool("isMarket", req.IsMarket),
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.String("endpoint", clobBaseURL+"/orders"))
	if req.OrderType == "" {
		notification.SendErrorLog(ctx, "polymarket-PlaceOrder-OrderTypeEmpty", zap.String("tokenID", req.TokenID))
		return nil, fmt.Errorf("order_type不能为空")
	}
	validOrderTypes := map[polyenum.PolymarketOrderType]bool{
		polyenum.PolymarketOrderTypeGTC: true, polyenum.PolymarketOrderTypeGTD: true,
		polyenum.PolymarketOrderTypeFOK: true, polyenum.PolymarketOrderTypeFAK: true,
	}
	if !validOrderTypes[req.OrderType] {
		return nil, fmt.Errorf("不支持的订单类型: %s", req.OrderType)
	}
	if req.IsMarket {
		if req.Side == polyenum.OrderSideBuy && req.MarketAmount == "" {
			return nil, fmt.Errorf("市价买单必须传 market_amount")
		}
		if req.Side == polyenum.OrderSideSell && req.MarketSize == "" {
			return nil, fmt.Errorf("市价卖单必须传 market_size")
		}
		if req.OrderType != polyenum.PolymarketOrderTypeFOK && req.OrderType != polyenum.PolymarketOrderTypeFAK {
			return nil, fmt.Errorf("市价单必须使用 FOK 或 FAK")
		}
		if req.PostOnly {
			return nil, fmt.Errorf("市价单不能 PostOnly")
		}
	} else {
		if req.LimitPrice == "" || req.LimitSize == "" {
			return nil, fmt.Errorf("限价单必须传 limit_price 和 limit_size")
		}
	}
	orderData := map[string]interface{}{
		"token_id":   req.TokenID,
		"side":       strings.ToUpper(string(req.Side)),
		"order_type": string(req.OrderType),
		"post_only":  req.PostOnly && !req.IsMarket,
	}
	if req.IsMarket {
		orderData["price"] = ""
		if req.MarketWorstPrice != "" {
			orderData["market_worst_price"] = req.MarketWorstPrice
		}
		if req.Side == polyenum.OrderSideBuy {
			orderData["amount"] = req.MarketAmount
			orderData["size"] = ""
		} else {
			orderData["size"] = req.MarketSize
			orderData["amount"] = ""
		}
	} else {
		orderData["price"] = req.LimitPrice
		orderData["size"] = req.LimitSize
		orderData["amount"] = ""
	}
	if builderCode := strings.TrimSpace(req.BuilderCode); builderCode != "" {
		orderData["builder"] = builderCode
	}

	// sig_type=3（POLY_1271 / V2 Deposit Wallet）走独立的 POST /order 单对象报文；
	// 老账号路径保持不变，避免新逻辑影响存量账号。
	if polymarketAccount.SignatureType == sigType1271 {
		return p.placeOrderV2(ctx, req, polymarketAccount, orderData, startTime, reqJSON)
	}

	signedOrder, feeRateBps, err := p.signOrder(ctx, orderData, polymarketAccount, req.NegRisk)
	if err != nil {
		return nil, fmt.Errorf("failed to sign order: %w", err)
	}
	requestBody := []map[string]interface{}{{"order": signedOrder, "orderType": string(req.OrderType)}}
	if req.PostOnly && !req.IsMarket {
		requestBody[0]["postOnly"] = true
	}
	if polymarketAccount.APIKey != "" {
		requestBody[0]["owner"] = polymarketAccount.APIKey
	} else if polymarketAccount.ProxyAddress != "" {
		requestBody[0]["owner"] = strings.ToLower(polymarketAccount.ProxyAddress)
	} else if polymarketAccount.EOAAddress != "" {
		requestBody[0]["owner"] = strings.ToLower(polymarketAccount.EOAAddress)
	}
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	bodyJSONStr := string(bodyJSON)
	api := clobBaseURL + "/orders"
	headers, err := p.buildAuthHeaders(ctx, "POST", "/orders", polymarketAccount, bodyJSONStr)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth headers: %w", err)
	}
	resp, err := p.client.Post(ctx, api, nil, headers, string(bodyJSON))
	p.checkTimeoutAndAlert(ctx, startTime, "PlaceOrder")
	var apiResp []struct {
		OrderID            string   `json:"orderID"`
		Status             string   `json:"status"`
		Message            string   `json:"message"`
		ErrorMsg           string   `json:"errorMsg,omitempty"`
		TransactionsHashes []string `json:"transactionsHashes,omitempty"`
	}
	if err != nil {
		statusCode, rawBody := 0, ""
		if resp != nil {
			statusCode = resp.StatusCode()
			rawBody = string(resp.Body())
		}
		notification.SendErrorLog(ctx, "polymarket-PlaceOrder-Resp",
			zap.String("tokenID", req.TokenID),
			zap.Error(err),
			zap.Int("statusCode", statusCode),
			zap.String("rawBody", rawBody))
		if resp != nil && len(resp.Body()) > 0 {
			_ = json.Unmarshal(resp.Body(), &apiResp)
			if len(apiResp) > 0 && apiResp[0].ErrorMsg != "" {
				return nil, fmt.Errorf("post %s error, status code: %d, errorMsg: %s", api, resp.StatusCode(), apiResp[0].ErrorMsg)
			}
		}
		return nil, err
	}
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		notification.SendErrorLog(ctx, "polymarket-PlaceOrder-Resp",
			zap.String("tokenID", req.TokenID),
			zap.String("rawBody", string(resp.Body())),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(apiResp) == 0 {
		notification.SendErrorLog(ctx, "polymarket-PlaceOrder-Resp",
			zap.String("tokenID", req.TokenID),
			zap.String("rawBody", string(resp.Body())),
			zap.String("error", "empty response from batch order API"))
		return nil, fmt.Errorf("empty response from batch order API")
	}
	orderResp := apiResp[0]
	respOut := &entity.PlaceOrderResp{
		OrderID:            orderResp.OrderID,
		Status:             polyenum.OrderStatus(strings.ToLower(orderResp.Status)),
		Message:            orderResp.Message,
		ErrorMsg:           orderResp.ErrorMsg,
		FeeRateBps:         feeRateBps,
		TransactionsHashes: orderResp.TransactionsHashes,
	}
	respJSON, _ := json.Marshal(respOut)
	logger.InfoZ(ctx, "polymarket-PlaceOrder-Resp",
		zap.String("request", string(reqJSON)),
		zap.String("tokenID", req.TokenID),
		zap.String("orderID", respOut.OrderID),
		zap.String("status", string(respOut.Status)),
		zap.String("message", respOut.Message),
		zap.String("response", string(respJSON)))
	return respOut, nil
}

// placeOrderV2 处理 sig_type=3（POLY_1271 / V2 Deposit Wallet）新账号下单。
// 与老账号路径的区别：POST /order（单对象报文，不是 /orders 数组），响应为单对象，
// 并返回链上结算 tx hash 供 activity 兜底匹配。
func (p *Polymarket) placeOrderV2(
	ctx context.Context,
	req entity.PlaceOrderReq,
	polymarketAccount entity.PolymarketAccount,
	orderData map[string]interface{},
	startTime time.Time,
	reqJSON []byte,
) (*entity.PlaceOrderResp, error) {
	signedOrder, feeRateBps, err := p.signOrderPoly1271(ctx, orderData, polymarketAccount, req.NegRisk)
	if err != nil {
		return nil, fmt.Errorf("failed to sign V2 order: %w", err)
	}

	single := map[string]interface{}{
		"order":     signedOrder,
		"orderType": string(req.OrderType),
	}
	if req.PostOnly && !req.IsMarket {
		single["postOnly"] = true
	}
	if polymarketAccount.APIKey != "" {
		single["owner"] = polymarketAccount.APIKey
	} else if polymarketAccount.ProxyAddress != "" {
		single["owner"] = strings.ToLower(polymarketAccount.ProxyAddress)
	} else if polymarketAccount.EOAAddress != "" {
		single["owner"] = strings.ToLower(polymarketAccount.EOAAddress)
	}
	bodyJSON, err := json.Marshal(single)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal V2 single-order body: %w", err)
	}
	bodyJSONStr := string(bodyJSON)

	const v2Path = "/order"
	api := clobBaseURL + v2Path
	headers, err := p.buildAuthHeaders(ctx, "POST", v2Path, polymarketAccount, bodyJSONStr)
	if err != nil {
		return nil, fmt.Errorf("failed to build V2 auth headers: %w", err)
	}
	resp, err := p.client.Post(ctx, api, nil, headers, bodyJSONStr)
	p.checkTimeoutAndAlert(ctx, startTime, "PlaceOrderV2")

	// /order 返回单对象 {orderID, status, message, errorMsg, transactionsHashes}
	var apiResp struct {
		OrderID            string   `json:"orderID"`
		Status             string   `json:"status"`
		Message            string   `json:"message"`
		ErrorMsg           string   `json:"errorMsg,omitempty"`
		TransactionsHashes []string `json:"transactionsHashes,omitempty"`
	}
	if err != nil {
		statusCode, rawBody := 0, ""
		if resp != nil {
			statusCode = resp.StatusCode()
			rawBody = string(resp.Body())
		}
		notification.SendErrorLog(ctx, "polymarket-PlaceOrderV2-Resp",
			zap.String("tokenID", req.TokenID),
			zap.Error(err),
			zap.Int("statusCode", statusCode),
			zap.String("rawBody", rawBody))
		if resp != nil && len(resp.Body()) > 0 {
			_ = json.Unmarshal(resp.Body(), &apiResp)
			if apiResp.ErrorMsg != "" {
				return nil, fmt.Errorf("post %s error, status code: %d, errorMsg: %s", api, resp.StatusCode(), apiResp.ErrorMsg)
			}
			// CLOB 400 经常返 {"error":"..."} 而不是 errorMsg，把原始 body 带出来
			truncated := strings.TrimSpace(rawBody)
			if len(truncated) > 1024 {
				truncated = truncated[:1024] + "...(truncated)"
			}
			return nil, fmt.Errorf("post %s error, status code: %d, body: %s", api, resp.StatusCode(), truncated)
		}
		return nil, err
	}
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		notification.SendErrorLog(ctx, "polymarket-PlaceOrderV2-Resp",
			zap.String("tokenID", req.TokenID),
			zap.String("rawBody", string(resp.Body())),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse V2 response: %w", err)
	}
	respOut := &entity.PlaceOrderResp{
		OrderID:            apiResp.OrderID,
		Status:             polyenum.OrderStatus(strings.ToLower(apiResp.Status)),
		Message:            apiResp.Message,
		ErrorMsg:           apiResp.ErrorMsg,
		FeeRateBps:         feeRateBps,
		TransactionsHashes: apiResp.TransactionsHashes,
	}
	respJSON, _ := json.Marshal(respOut)
	logger.InfoZ(ctx, "polymarket-PlaceOrderV2-Resp",
		zap.String("request", string(reqJSON)),
		zap.String("tokenID", req.TokenID),
		zap.String("orderID", respOut.OrderID),
		zap.String("status", string(respOut.Status)),
		zap.String("message", respOut.Message),
		zap.String("response", string(respJSON)))
	return respOut, nil
}

// GetOrder 查询单个订单信息（CLOB Go HTTP）
func (p *Polymarket) GetOrderById(ctx context.Context, req entity.GetOrderReq, polymarketAccount entity.PolymarketAccount) (*entity.GetOrderResp, error) {
	startTime := time.Now()
	reqJSON, _ := json.Marshal(req)
	logger.InfoZ(ctx, "polymarket-GetOrderById-Req",
		zap.String("request", string(reqJSON)),
		zap.String("orderID", req.OrderID),
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.String("endpoint", clobBaseURL+"/data/order/"+req.OrderID))
	api := clobBaseURL + "/data/order/" + req.OrderID
	path := "/data/order/" + req.OrderID
	headers, err := p.buildAuthHeaders(ctx, "GET", path, polymarketAccount, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build auth headers: %w", err)
	}
	resp, err := p.client.Get(ctx, api, nil, headers)
	p.checkTimeoutAndAlert(ctx, startTime, "GetOrder")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetOrderById-Resp",
			zap.String("orderID", req.OrderID),
			zap.Error(err))
		return nil, err
	}
	// CLOB /data/order/{id} 直接返回订单对象（不是 {order:{...}} 包装）
	var order entity.OpenOrder
	if err = json.Unmarshal(resp.Body(), &order); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetOrderById-Resp",
			zap.String("orderID", req.OrderID),
			zap.String("rawBody", string(resp.Body())),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	out := &entity.GetOrderResp{Order: order}
	outJSON, _ := json.Marshal(out)
	logger.InfoZ(ctx, "polymarket-GetOrderById-Resp",
		zap.String("orderID", req.OrderID),
		zap.String("response", string(outJSON)))
	return out, nil
}

// GetTrades 查询成交记录（CLOB Go HTTP）
func (p *Polymarket) GetTrades(ctx context.Context, req entity.GetTradesReq, polymarketAccount entity.PolymarketAccount) (*entity.GetTradesResp, error) {
	startTime := time.Now()
	reqJSON, _ := json.Marshal(req)
	logger.InfoZ(ctx, "polymarket-GetTrades-Req",
		zap.String("request", string(reqJSON)),
		zap.String("tradeID", req.TradeID),
		zap.String("market", req.Market),
		zap.String("tokenID", req.TokenID),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset),
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.String("endpoint", clobBaseURL+"/data/trades"))
	api := clobBaseURL + "/data/trades"
	params := make(map[string][]string)
	if req.TradeID != "" {
		params["id"] = []string{req.TradeID}
	}
	if req.Market != "" {
		params["market"] = []string{req.Market}
	}
	if req.TokenID != "" {
		params["asset_id"] = []string{req.TokenID}
	}
	if req.Limit != 0 {
		params["limit"] = []string{strconv.Itoa(req.Limit)}
	}
	if req.Offset != 0 {
		params["offset"] = []string{strconv.Itoa(req.Offset)}
	}
	if !req.Before.IsZero() {
		params["before"] = []string{strconv.FormatInt(req.Before.Unix(), 10)}
	}
	if !req.After.IsZero() {
		params["after"] = []string{strconv.FormatInt(req.After.Unix(), 10)}
	}
	headers, err := p.buildAuthHeaders(ctx, "GET", "/data/trades", polymarketAccount, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build auth headers: %w", err)
	}
	resp, err := p.client.Get(ctx, api, params, headers)
	p.checkTimeoutAndAlert(ctx, startTime, "GetTrades")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetTrades-Resp",
			zap.String("market", req.Market),
			zap.String("tokenID", req.TokenID),
			zap.Error(err))
		return nil, err
	}
	var apiResp struct {
		Data []struct {
			ID              string `json:"id"`
			TakerOrderID    string `json:"taker_order_id"`
			Market          string `json:"market"`
			AssetID         string `json:"asset_id"`
			Side            string `json:"side"`
			Size            string `json:"size"`
			Price           string `json:"price"`
			FeeRateBps      string `json:"fee_rate_bps"`
			Status          string `json:"status"`
			MatchTime       string `json:"match_time"`
			LastUpdate      string `json:"last_update"`
			Outcome         string `json:"outcome"`
			MakerAddress    string `json:"maker_address"`
			Owner           string `json:"owner"`
			TransactionHash string `json:"transaction_hash"`
			BucketIndex     int    `json:"bucket_index"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetTrades-Resp",
			zap.String("market", req.Market),
			zap.String("rawBody", string(resp.Body())),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if apiResp.Data == nil {
		apiResp.Data = []struct {
			ID              string `json:"id"`
			TakerOrderID    string `json:"taker_order_id"`
			Market          string `json:"market"`
			AssetID         string `json:"asset_id"`
			Side            string `json:"side"`
			Size            string `json:"size"`
			Price           string `json:"price"`
			FeeRateBps      string `json:"fee_rate_bps"`
			Status          string `json:"status"`
			MatchTime       string `json:"match_time"`
			LastUpdate      string `json:"last_update"`
			Outcome         string `json:"outcome"`
			MakerAddress    string `json:"maker_address"`
			Owner           string `json:"owner"`
			TransactionHash string `json:"transaction_hash"`
			BucketIndex     int    `json:"bucket_index"`
		}{}
	}
	parseUnixTime := func(s string) time.Time {
		sec, _ := strconv.ParseInt(s, 10, 64)
		return time.Unix(sec, 0)
	}
	trades := make([]entity.Trade, 0, len(apiResp.Data))
	for _, t := range apiResp.Data {
		trades = append(trades, entity.Trade{
			ID: t.ID, TakerOrderID: t.TakerOrderID, Market: t.Market, AssetID: t.AssetID,
			Side: t.Side, Size: t.Size, Price: t.Price, FeeRateBps: t.FeeRateBps, Status: t.Status,
			MatchTime: parseUnixTime(t.MatchTime), LastUpdate: parseUnixTime(t.LastUpdate),
			Outcome: t.Outcome, MakerAddress: t.MakerAddress, Owner: t.Owner,
			TransactionHash: t.TransactionHash, BucketIndex: t.BucketIndex,
		})
	}
	out := &entity.GetTradesResp{Trades: trades, Total: len(trades)}
	outJSON, _ := json.Marshal(out)
	outPreview := string(outJSON)
	if len(outPreview) > 2000 {
		outPreview = outPreview[:2000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-GetTrades-Resp",
		zap.String("market", req.Market),
		zap.String("tokenID", req.TokenID),
		zap.Int("tradesCount", len(trades)),
		zap.Int("total", out.Total),
		zap.String("response", outPreview))
	return out, nil
}

// checkTimeoutAndAlert 慢请求：>15s 写 warn 日志；≥50s 发 Lark（低于 client 60s 硬超时）。
func (p *Polymarket) checkTimeoutAndAlert(ctx context.Context, startTime time.Time, methodName string) {
	elapsed := time.Since(startTime)
	if elapsed < polymarketSlowRequestLog {
		return
	}
	if elapsed < polymarketSlowRequestAlert {
		notification.SendErrorLog(ctx, "polymarket-client-SlowRequest",
			zap.String("method", methodName),
			zap.Duration("elapsed", elapsed),
			zap.Float64("elapsedSeconds", elapsed.Seconds()))
		return
	}
	notification.SendErrorLog(ctx, "polymarket-client-RequestTimeout",
		zap.String("method", methodName),
		zap.Duration("elapsed", elapsed),
		zap.Float64("elapsedSeconds", elapsed.Seconds()),
		zap.String("message", fmt.Sprintf("%s请求耗时超过%.0f秒", methodName, polymarketSlowRequestAlert.Seconds())))
}

// GetMarketByMarketId 获取单个市场信息
func (p *Polymarket) GetMarketByMarketId(ctx context.Context, req entity.GetMarketRequest) (*entity.Market, error) {
	// 记录发起时间
	startTime := time.Now()

	// 详细打印输入
	reqJSON, _ := json.Marshal(req)
	logger.InfoZ(ctx, "polymarket-GetMarket-Req",
		zap.String("request", string(reqJSON)),
		zap.String("marketID", req.MarketID),
		zap.String("endpoint", fmt.Sprintf("%s/markets/%s", p.baseURL, req.MarketID)))

	// 构建请求参数
	params := make(map[string][]string)

	// 发送请求
	api := fmt.Sprintf("%s/markets/%s", p.baseURL, req.MarketID)

	resp, err := p.client.Get(ctx, api, params, nil)
	// 检查超时并报警
	p.checkTimeoutAndAlert(ctx, startTime, "GetMarket")

	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetMarket-RequestFailed",
			zap.String("marketID", req.MarketID),
			zap.String("endpoint", api),
			zap.Error(err))
		return nil, err
	}

	// 解析响应
	var market entity.Market
	if err = json.Unmarshal(resp.Body(), &market); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetMarket-ParseError",
			zap.String("marketID", req.MarketID),
			zap.String("response", string(resp.Body())),
			zap.Error(err))
		return nil, err
	}

	// 详细打印输出
	marketJSON, _ := json.Marshal(market)
	logger.InfoZ(ctx, "polymarket-GetMarket-Resp",
		zap.String("marketID", req.MarketID),
		zap.String("slug", market.Slug),
		zap.String("question", market.Question),
		zap.String("market", string(marketJSON)))

	return &market, nil
}

// GetMarketsByMarketAddressList
func (p *Polymarket) GetMarketsByMarketAddressList(ctx context.Context, req entity.GetMarketsByConditionIdsRequest) ([]entity.Market, error) {
	// 记录发起时间
	startTime := time.Now()

	// 详细打印输入
	reqJSON, _ := json.Marshal(req)
	logger.InfoZ(ctx, "polymarket-GetMarketsByMarketAddressList-Req",
		zap.String("request", string(reqJSON)),
		zap.Int("conditionIdsCount", len(req.ConditionIDs)),
		zap.Strings("conditionIds", req.ConditionIDs),
		zap.String("endpoint", fmt.Sprintf("%s/markets", p.baseURL)))

	if len(req.ConditionIDs) == 0 {
		return nil, fmt.Errorf("conditionIds不能为空")
	}

	// 构建请求参数 - 多个condition_ids参数
	params := make(map[string][]string)
	for _, conditionID := range req.ConditionIDs {
		params["condition_ids"] = append(params["condition_ids"], conditionID)
	}

	// 发送请求
	api := p.baseURL + "/markets"

	resp, err := p.client.Get(ctx, api, params, nil)
	// 检查超时并报警
	p.checkTimeoutAndAlert(ctx, startTime, "GetMarketsByMarketAddressList")

	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetMarketsByMarketAddressList-RequestFailed",
			zap.Int("conditionIdsCount", len(req.ConditionIDs)),
			zap.Strings("conditionIds", req.ConditionIDs),
			zap.String("endpoint", api),
			zap.Error(err))
		return nil, err
	}

	// 解析响应 - API返回数组
	var markets []entity.Market
	if err = json.Unmarshal(resp.Body(), &markets); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetMarketsByMarketAddressList-ParseError",
			zap.Int("conditionIdsCount", len(req.ConditionIDs)),
			zap.Strings("conditionIds", req.ConditionIDs),
			zap.String("response", string(resp.Body())),
			zap.Error(err))
		return nil, err
	}

	// 详细打印输出
	marketsJSON, _ := json.Marshal(markets)
	marketsPreview := string(marketsJSON)
	if len(marketsPreview) > 1000 {
		marketsPreview = marketsPreview[:1000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-GetMarketsByMarketAddressList-Resp",
		zap.Int("conditionIdsCount", len(req.ConditionIDs)),
		zap.Strings("conditionIds", req.ConditionIDs),
		zap.Int("marketsCount", len(markets)),
		zap.String("markets", marketsPreview))

	return markets, nil
}

// GetSportsMetadata 获取运动元数据（GET /sports，参考 https://docs.polymarket.com/api-reference/sports/get-sports-metadata-information）
func (p *Polymarket) GetSportsMetadata(ctx context.Context) ([]entity.SportsMetadata, error) {
	startTime := time.Now()
	api := p.baseURL + "/sports"
	resp, err := p.client.Get(ctx, api, nil, nil)
	p.checkTimeoutAndAlert(ctx, startTime, "GetSportsMetadata")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetSportsMetadata-RequestFailed",
			zap.String("endpoint", api), zap.Error(err))
		return nil, err
	}
	var list []entity.SportsMetadata
	if err = json.Unmarshal(resp.Body(), &list); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetSportsMetadata-ParseError",
			zap.String("response", string(resp.Body())), zap.Error(err))
		return nil, err
	}
	if list == nil {
		list = []entity.SportsMetadata{}
	}
	logger.InfoZ(ctx, "polymarket-GetSportsMetadata-Resp",
		zap.Int("sportsCount", len(list)))
	return list, nil
}

// ListMarkets 列出市场（支持 closed、limit、offset、tag_id 等，参考 https://docs.polymarket.com/api-reference/markets/list-markets）
func (p *Polymarket) ListMarkets(ctx context.Context, req entity.ListMarketsRequest) ([]entity.Market, error) {
	startTime := time.Now()

	params := make(map[string][]string)
	if req.Limit < 1 {
		req.Limit = 100
	}
	params["limit"] = []string{strconv.Itoa(req.Limit)}
	params["offset"] = []string{strconv.Itoa(req.Offset)}
	if req.Closed != nil {
		params["closed"] = []string{strconv.FormatBool(*req.Closed)}
	}
	if req.Order != "" {
		params["order"] = []string{req.Order}
		params["ascending"] = []string{strconv.FormatBool(req.Ascending)}
	}
	if req.TagID > 0 {
		params["tag_id"] = []string{strconv.Itoa(req.TagID)}
	}

	api := p.baseURL + "/markets"
	resp, err := p.client.Get(ctx, api, params, nil)
	p.checkTimeoutAndAlert(ctx, startTime, "ListMarkets")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListMarkets-RequestFailed",
			zap.Int("limit", req.Limit), zap.Int("offset", req.Offset), zap.String("endpoint", api), zap.Error(err))
		return nil, err
	}

	var markets []entity.Market
	if err = json.Unmarshal(resp.Body(), &markets); err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListMarkets-ParseError",
			zap.String("response", string(resp.Body())), zap.Error(err))
		return nil, err
	}

	logger.InfoZ(ctx, "polymarket-ListMarkets-Resp",
		zap.Int("limit", req.Limit), zap.Int("offset", req.Offset), zap.Int("marketsCount", len(markets)))

	return markets, nil
}

// ListMarketsKeyset 游标分页（GET /markets/keyset）。offset 会 422，必须用 after_cursor。
func (p *Polymarket) ListMarketsKeyset(ctx context.Context, req entity.ListMarketsKeysetRequest) (*entity.PolymarketMarketsKeyset, error) {
	startTime := time.Now()
	limit := req.Limit
	if limit < 1 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	params := map[string][]string{
		"limit": []string{strconv.Itoa(limit)},
	}
	order := strings.TrimSpace(req.Order)
	if order == "" {
		order = "id"
	}
	params["order"] = []string{order}
	asc := false
	if req.Ascending != nil {
		asc = *req.Ascending
	}
	params["ascending"] = []string{strconv.FormatBool(asc)}
	if req.Closed != nil {
		params["closed"] = []string{strconv.FormatBool(*req.Closed)}
	}
	if req.TagID > 0 {
		params["tag_id"] = []string{strconv.Itoa(req.TagID)}
	}
	if len(req.SportsMarketTypes) > 0 {
		params["sports_market_types"] = append([]string(nil), req.SportsMarketTypes...)
	}
	if c := strings.TrimSpace(req.AfterCursor); c != "" {
		params["after_cursor"] = []string{c}
	}
	api := p.baseURL + "/markets/keyset"
	resp, err := p.client.Get(ctx, api, params, nil)
	p.checkTimeoutAndAlert(ctx, startTime, "ListMarketsKeyset")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListMarketsKeyset-RequestFailed",
			zap.Int("limit", limit), zap.Int("tag_id", req.TagID),
			zap.Strings("sports_market_types", req.SportsMarketTypes), zap.Error(err))
		return nil, err
	}
	var page entity.PolymarketMarketsKeyset
	if err = json.Unmarshal(resp.Body(), &page); err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListMarketsKeyset-ParseError",
			zap.String("response", string(resp.Body())), zap.Error(err))
		return nil, err
	}
	if page.Markets == nil {
		page.Markets = []entity.Market{}
	}
	logger.InfoZ(ctx, "polymarket-ListMarketsKeyset-Resp",
		zap.Int("limit", limit), zap.Int("tag_id", req.TagID),
		zap.Int("marketsCount", len(page.Markets)),
		zap.Bool("has_next", page.NextCursor != ""))
	return &page, nil
}

// ListEventsPagination 分页拉取事件（GET /events/pagination，limit>50 需 gzip）
func (p *Polymarket) ListEventsPagination(ctx context.Context, req entity.ListEventsPaginationRequest) (*entity.PolymarketEventsPagination, error) {
	startTime := time.Now()
	limit := req.Limit
	if limit < 1 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	params := map[string][]string{
		"limit":  []string{strconv.Itoa(limit)},
		"offset": []string{strconv.Itoa(req.Offset)},
	}
	if req.Closed != nil {
		params["closed"] = []string{strconv.FormatBool(*req.Closed)}
	}
	if req.TagID > 0 {
		params["tag_id"] = []string{strconv.Itoa(req.TagID)}
	}
	if strings.TrimSpace(req.TagSlug) != "" {
		params["tag_slug"] = []string{strings.TrimSpace(req.TagSlug)}
	}
	headers := map[string]string{"Accept-Encoding": "gzip"}
	api := p.baseURL + "/events/pagination"
	resp, err := p.client.Get(ctx, api, params, headers)
	p.checkTimeoutAndAlert(ctx, startTime, "ListEventsPagination")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListEventsPagination-RequestFailed",
			zap.Int("limit", limit), zap.Int("offset", req.Offset), zap.Int("tag_id", req.TagID), zap.Error(err))
		return nil, err
	}
	var page entity.PolymarketEventsPagination
	if err = json.Unmarshal(resp.Body(), &page); err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListEventsPagination-ParseError",
			zap.String("response", string(resp.Body())), zap.Error(err))
		return nil, err
	}
	if page.Data == nil {
		page.Data = []entity.PolymarketEvent{}
	}
	logger.InfoZ(ctx, "polymarket-ListEventsPagination-Resp",
		zap.Int("limit", limit), zap.Int("offset", req.Offset), zap.Int("tag_id", req.TagID),
		zap.Int("eventsCount", len(page.Data)),
		zap.Int("totalResults", page.Pagination.TotalResults),
		zap.Bool("hasMore", page.Pagination.HasMore))
	return &page, nil
}

// ListEventsKeyset 游标分页（GET /events/keyset，深分页用 after_cursor，禁止 offset）
func (p *Polymarket) ListEventsKeyset(ctx context.Context, req entity.ListEventsKeysetRequest) (*entity.PolymarketEventsKeyset, error) {
	startTime := time.Now()
	limit := req.Limit
	if limit < 1 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	params := map[string][]string{
		"limit": []string{strconv.Itoa(limit)},
	}
	order := strings.TrimSpace(req.Order)
	if order == "" {
		order = "id"
	}
	params["order"] = []string{order}
	asc := true
	if req.Ascending != nil {
		asc = *req.Ascending
	}
	params["ascending"] = []string{strconv.FormatBool(asc)}
	if req.Closed != nil {
		params["closed"] = []string{strconv.FormatBool(*req.Closed)}
	}
	if req.TagID > 0 {
		params["tag_id"] = []string{strconv.Itoa(req.TagID)}
	}
	if req.SeriesID > 0 {
		params["series_id"] = []string{strconv.Itoa(req.SeriesID)}
	}
	if c := strings.TrimSpace(req.AfterCursor); c != "" {
		params["after_cursor"] = []string{c}
	}
	api := p.baseURL + "/events/keyset"
	resp, err := p.client.Get(ctx, api, params, nil)
	p.checkTimeoutAndAlert(ctx, startTime, "ListEventsKeyset")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListEventsKeyset-RequestFailed",
			zap.Int("limit", limit), zap.Int("tag_id", req.TagID), zap.Int("series_id", req.SeriesID),
			zap.Bool("has_cursor", req.AfterCursor != ""), zap.Error(err))
		return nil, err
	}
	var page entity.PolymarketEventsKeyset
	if err = json.Unmarshal(resp.Body(), &page); err != nil {
		notification.SendErrorLog(ctx, "polymarket-ListEventsKeyset-ParseError",
			zap.String("response", string(resp.Body())), zap.Error(err))
		return nil, err
	}
	if page.Events == nil {
		page.Events = []entity.PolymarketEvent{}
	}
	logger.InfoZ(ctx, "polymarket-ListEventsKeyset-Resp",
		zap.Int("limit", limit), zap.Int("tag_id", req.TagID), zap.Int("series_id", req.SeriesID),
		zap.Int("eventsCount", len(page.Events)),
		zap.Bool("has_next", page.NextCursor != ""))
	return &page, nil
}

// GetOrderBook 获取订单簿
func (p *Polymarket) GetOrderBook(ctx context.Context, req entity.GetOrderBookRequest) (*entity.OrderBookResponse, error) {
	// 获取配置中的订单簿API地址
	orderBookBaseURL := p.orderBookURL
	if orderBookBaseURL == "" {
		orderBookBaseURL = clobBaseURL
	}

	// 详细打印输入
	reqJSON, _ := json.Marshal(req)
	logger.InfoZ(ctx, "polymarket-GetOrderBook-Req",
		zap.String("request", string(reqJSON)),
		zap.String("tokenID", req.TokenID),
		zap.String("endpoint", orderBookBaseURL+"/book"))

	// 构建请求参数
	params := make(map[string][]string)
	params["token_id"] = []string{req.TokenID}

	// 发送请求
	api := orderBookBaseURL + "/book"

	// 记录发起时间
	startTime := time.Now()

	resp, err := p.client.Get(ctx, api, params, nil)
	// 检查超时并报警
	p.checkTimeoutAndAlert(ctx, startTime, "GetOrderBook")

	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetOrderBook-RequestFailed",
			zap.String("tokenID", req.TokenID),
			zap.String("endpoint", api),
			zap.Error(err))
		return nil, err
	}

	// 解析响应
	var orderBook entity.OrderBookResponse
	if err = json.Unmarshal(resp.Body(), &orderBook); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetOrderBook-ParseError",
			zap.String("tokenID", req.TokenID),
			zap.String("response", string(resp.Body())),
			zap.Error(err))
		return nil, err
	}

	// 详细打印输出
	orderBookJSON, _ := json.Marshal(orderBook)
	orderBookPreview := string(orderBookJSON)
	if len(orderBookPreview) > 2000 {
		orderBookPreview = orderBookPreview[:2000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-GetOrderBook-Resp",
		zap.String("tokenID", req.TokenID),
		zap.Int("bidsCount", len(orderBook.Bids)),
		zap.Int("asksCount", len(orderBook.Asks)),
		zap.String("orderBook", orderBookPreview))

	return &orderBook, nil
}

// BatchGetOrderBooks 批量获取订单簿摘要
func (p *Polymarket) BatchGetOrderBooks(ctx context.Context, req entity.BatchGetOrderBooksRequest) (*entity.BatchGetOrderBooksResponse, error) {
	// 获取配置中的订单簿API地址
	orderBookBaseURL := p.orderBookURL
	if orderBookBaseURL == "" {
		orderBookBaseURL = clobBaseURL
	}

	// 验证请求
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("items list is empty")
	}

	// 最大数组长度为500
	if len(req.Items) > 500 {
		return nil, fmt.Errorf("too many items, maximum 500 items per batch")
	}

	// 详细打印输入
	reqJSON, _ := json.Marshal(req)
	reqPreview := string(reqJSON)
	if len(reqPreview) > 2000 {
		reqPreview = reqPreview[:2000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-BatchGetOrderBooks-Req",
		zap.String("request", reqPreview),
		zap.Int("itemsCount", len(req.Items)),
		zap.String("endpoint", orderBookBaseURL+"/books"))

	// 构建请求体（直接使用数组格式）
	var requestBody []map[string]interface{}
	for _, item := range req.Items {
		itemMap := map[string]interface{}{
			"token_id": item.TokenID,
		}
		// 如果指定了 side，添加到请求中
		if item.Side != "" {
			itemMap["side"] = string(item.Side)
		}
		requestBody = append(requestBody, itemMap)
	}

	// 序列化请求体
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-BatchGetOrderBooks-MarshalError",
			zap.Int("itemsCount", len(req.Items)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 记录发起时间
	startTime := time.Now()

	// 发送POST请求
	// HTTP client 的 Post 方法支持直接传递 JSON 字符串作为 contentType 参数
	api := orderBookBaseURL + "/books"
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	resp, err := p.client.Post(ctx, api, nil, headers, string(requestBodyJSON))
	// 检查超时并报警
	p.checkTimeoutAndAlert(ctx, startTime, "BatchGetOrderBooks")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-BatchGetOrderBooks-RequestFailed",
			zap.Int("itemsCount", len(req.Items)),
			zap.String("endpoint", api),
			zap.Error(err))
		return nil, err
	}

	// 解析响应（返回的是数组）
	var orderBooks []entity.OrderBookResponse
	if err = json.Unmarshal(resp.Body(), &orderBooks); err != nil {
		notification.SendErrorLog(ctx, "polymarket-BatchGetOrderBooks-ParseError",
			zap.Int("itemsCount", len(req.Items)),
			zap.String("response", string(resp.Body())),
			zap.Error(err))
		return nil, err
	}

	response := &entity.BatchGetOrderBooksResponse{
		OrderBooks: orderBooks,
	}

	// 详细打印输出
	responseJSON, _ := json.Marshal(response)
	responsePreview := string(responseJSON)
	if len(responsePreview) > 2000 {
		responsePreview = responsePreview[:2000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-BatchGetOrderBooks-Resp",
		zap.Int("itemsCount", len(req.Items)),
		zap.Int("orderBooksCount", len(orderBooks)),
		zap.String("response", responsePreview))

	return response, nil
}

// GetAllOrders 获取用户所有订单
func (p *Polymarket) GetAllOrders(ctx context.Context, polymarketAccount entity.PolymarketAccount, market *string) (*entity.GetActiveOrdersResponse, error) {
	marketStr := ""
	if market != nil && *market != "" {
		marketStr = *market
	}
	startTime := time.Now()
	logger.InfoZ(ctx, "polymarket-GetAllOrders-Req",
		zap.String("market", marketStr),
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.String("endpoint", clobBaseURL+"/data/orders"))
	api := clobBaseURL + "/data/orders"
	params := make(map[string][]string)
	if marketStr != "" {
		params["market"] = []string{marketStr}
	}
	headers, err := p.buildAuthHeaders(ctx, "GET", "/data/orders", polymarketAccount, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build auth headers: %w", err)
	}
	resp, err := p.client.Get(ctx, api, params, headers)
	p.checkTimeoutAndAlert(ctx, startTime, "GetAllOrders")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetAllOrders-Resp",
			zap.String("market", marketStr),
			zap.String("proxyAddress", polymarketAccount.ProxyAddress),
			zap.Error(err))
		return nil, err
	}
	// API 返回 { "orders": [...], "total": N }，不是裸数组
	var apiResp struct {
		Orders []struct {
			ID            string `json:"id"`
			TokenID       string `json:"token_id"`
			AssetID       string `json:"asset_id"`
			Market        string `json:"market"`
			Side          string `json:"side"`
			Price         string `json:"price"`
			Size          string `json:"size"`
			OriginalSize  string `json:"original_size"`
			SizeMatched   string `json:"size_matched"`
			RemainingSize string `json:"remaining_size"`
			Status        string `json:"status"`
			OrderType     string `json:"order_type"`
			CreatedAt     int64  `json:"created_at"`
			Expiration    *int64 `json:"expiration,omitempty"`
			Outcome       string `json:"outcome"`
			Owner         string `json:"owner"`
		} `json:"orders"`
		Total int `json:"total,omitempty"`
	}
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetAllOrders-Resp",
			zap.String("market", marketStr),
			zap.String("proxyAddress", polymarketAccount.ProxyAddress),
			zap.String("rawBody", string(resp.Body())),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	orders := make([]entity.Order, 0, len(apiResp.Orders))
	for _, o := range apiResp.Orders {
		orders = append(orders, entity.Order{
			ID: o.ID, TokenID: o.TokenID, AssetID: o.AssetID, Market: o.Market,
			Price: o.Price, Size: o.RemainingSize, OriginalSize: o.OriginalSize,
			SizeMatched: o.SizeMatched,
			Side:        polyenum.OrderSide(strings.ToUpper(o.Side)),
			Type:        polyenum.OrderTypeLimit,
			Status:      polyenum.OrderStatus(strings.ToLower(o.Status)),
			Outcome:     o.Outcome, CreatedAt: time.Unix(o.CreatedAt, 0),
		})
	}
	total := apiResp.Total
	if total <= 0 {
		total = len(orders)
	}
	out := &entity.GetActiveOrdersResponse{Orders: orders, Total: total}
	outJSON, _ := json.Marshal(out)
	outPreview := string(outJSON)
	if len(outPreview) > 2000 {
		outPreview = outPreview[:2000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-GetAllOrders-Resp",
		zap.String("market", marketStr),
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.Int("ordersCount", len(orders)),
		zap.Int("total", total),
		zap.String("response", outPreview))
	return out, nil
}

// GetPositions 获取Magic账户的持仓
func (p *Polymarket) GetPositions(ctx context.Context, proxyAddress string) (*entity.GetMagicAccountPositionsResponse, error) {
	// 详细打印输入
	logger.InfoZ(ctx, "polymarket-GetPositions-Req",
		zap.String("proxyAddress", proxyAddress))

	// 直接通过Polymarket官方API获取NFT持仓
	positions, err := p.getPositionsFromPolymarketAPI(ctx, proxyAddress)
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetMagicAccountPositions-FailedToGetPositions", zap.Error(err))
		return &entity.GetMagicAccountPositionsResponse{
			Positions: []entity.Position{},
			Total:     0,
			Success:   false,
			ErrorMsg:  fmt.Sprintf("获取持仓失败: %s", err.Error()),
		}, nil
	}

	// 构建响应
	positionsResp := &entity.GetMagicAccountPositionsResponse{
		Positions: positions,
		Total:     len(positions),
		Success:   true,
		Message:   fmt.Sprintf("成功获取 %d 个持仓", len(positions)),
	}

	// 详细打印输出
	responseJSON, _ := json.Marshal(positionsResp)
	responsePreview := string(responseJSON)
	if len(responsePreview) > 2000 {
		responsePreview = responsePreview[:2000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-GetPositions-Resp",
		zap.String("proxyAddress", proxyAddress),
		zap.Int("positionsCount", len(positions)),
		zap.Int("total", positionsResp.Total),
		zap.Bool("success", positionsResp.Success),
		zap.String("message", positionsResp.Message),
		zap.String("response", responsePreview))

	return positionsResp, nil
}

// GetAccountInfo 获取账户信息
func (p *Polymarket) GetAccountInfo(ctx context.Context, polymarketAccount entity.PolymarketAccount) (*entity.MagicAccountInfo, error) {
	// 详细打印输入（不打印敏感信息）
	accountInfo := map[string]interface{}{
		"proxyAddress":    polymarketAccount.ProxyAddress,
		"eoaAddress":      polymarketAccount.EOAAddress,
		"signatureType":   polymarketAccount.SignatureType,
		"useMagicAccount": polymarketAccount.UseMagicAccount,
	}
	accountJSON, _ := json.Marshal(accountInfo)
	logger.InfoZ(ctx, "polymarket-GetMagicAccountInfo-Req",
		zap.String("accountInfo", string(accountJSON)),
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.String("eoaAddress", polymarketAccount.EOAAddress))

	// 获取Polymarket配置（仅用于获取API基础URL）
	orderBookBaseURL := p.orderBookURL
	if orderBookBaseURL == "" {
		orderBookBaseURL = clobBaseURL
	}

	// 获取抵押币余额（pUSD，使用传入的地址参数）
	balance, err := p.getUSDCBalance(ctx, polymarketAccount.ProxyAddress)

	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetMagicAccountInfo-FailedToGetBalance",
			zap.String("proxyAddress", polymarketAccount.ProxyAddress),
			zap.Error(err))
		// 即使获取余额失败，也返回基本信息
		balance = "0"
	}

	walletInfo := &entity.MagicAccountInfo{
		Success:         true,
		EOAAddress:      polymarketAccount.EOAAddress,
		ProxyAddress:    polymarketAccount.ProxyAddress,
		SignatureType:   polymarketAccount.SignatureType,
		UseMagicAccount: polymarketAccount.UseMagicAccount,
		Host:            orderBookBaseURL,
		ChainID:         137, // Polygon主网
		ClientStatus:    "connected",
		ClientError:     "",
		USDCBalance:     balance, // 添加USDC余额字段
	}

	// 详细打印输出
	walletInfoJSON, _ := json.Marshal(walletInfo)
	logger.InfoZ(ctx, "polymarket-GetMagicAccountInfo-Resp",
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.String("usdcBalance", balance),
		zap.String("clientStatus", walletInfo.ClientStatus),
		zap.Bool("success", walletInfo.Success),
		zap.String("walletInfo", string(walletInfoJSON)))

	return walletInfo, nil
}

// BatchCancelOrder 批量取消订单（CLOB Go HTTP）
func (p *Polymarket) BatchCancelOrder(ctx context.Context, req entity.BatchCancelOrderRequest, polymarketAccount entity.PolymarketAccount) (*entity.BatchCancelOrderResponse, error) {
	if len(req.OrderIDs) == 0 {
		return nil, fmt.Errorf("order IDs list is empty")
	}
	return p.cancelOrders(ctx, req.OrderIDs, polymarketAccount)
}

const clobAuthMessage = "This message attests that I control the given wallet"

// l1AuthDomain 是 L1 认证（创建/派生 API Key）用的 domain：
// 名字是 ClobAuthDomain、没有 verifyingContract，与下单 domain 是两套，不能混用。
func l1AuthDomain() map[string]interface{} {
	return map[string]interface{}{
		"name":    "ClobAuthDomain",
		"version": clobAuthDomainVersion,
		"chainId": big.NewInt(eip712ChainID),
	}
}

// signL1AuthMessage 使用私钥对 L1 认证消息做 EIP-712 签名（ClobAuthDomain + ClobAuth）
func (p *Polymarket) signL1AuthMessage(privateKeyHex, address, timestamp string, nonce int64) (string, error) {
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}
	domain := l1AuthDomain()
	types := map[string][]map[string]string{
		"EIP712Domain": {
			{"name": "name", "type": "string"}, {"name": "version", "type": "string"},
			{"name": "chainId", "type": "uint256"},
		},
		"ClobAuth": {
			{"name": "address", "type": "address"},
			{"name": "timestamp", "type": "string"},
			{"name": "nonce", "type": "uint256"},
			{"name": "message", "type": "string"},
		},
	}
	message := map[string]interface{}{
		"address":   address,
		"timestamp": timestamp,
		"nonce":     nonce,
		"message":   clobAuthMessage,
	}
	hash, err := p.hashEIP712L1Auth(domain, types, "ClobAuth", message)
	if err != nil {
		return "", err
	}
	sig, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(sig), nil
}

// hashEIP712L1Auth EIP-712 hash for L1 auth（ClobAuthDomain 无 verifyingContract）
func (p *Polymarket) hashEIP712L1Auth(domain map[string]interface{}, types map[string][]map[string]string, primaryType string, message map[string]interface{}) (common.Hash, error) {
	domainSeparator, err := p.encodeData("EIP712Domain", domain, types)
	if err != nil {
		return common.Hash{}, err
	}
	messageHash, err := p.encodeData(primaryType, message, types)
	if err != nil {
		return common.Hash{}, err
	}
	prefix := []byte{0x19, 0x01}
	data := append(prefix, domainSeparator...)
	data = append(data, messageHash...)
	return crypto.Keccak256Hash(data), nil
}

// buildL1AuthHeaders 构建 L1 认证请求头（POLY_ADDRESS, POLY_SIGNATURE, POLY_TIMESTAMP, POLY_NONCE）
func (p *Polymarket) buildL1AuthHeaders(ctx context.Context, privateKeyHex, signerAddress string, nonce int64) (map[string]string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature, err := p.signL1AuthMessage(privateKeyHex, signerAddress, timestamp, nonce)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"POLY_ADDRESS":   signerAddress,
		"POLY_SIGNATURE": signature,
		"POLY_TIMESTAMP": timestamp,
		"POLY_NONCE":     strconv.FormatInt(nonce, 10),
	}
	return headers, nil
}

// CreateAPIKey 使用私钥创建 L2 API 凭证（L1 认证：POST /auth/api-key）
func (p *Polymarket) CreateAPIKey(ctx context.Context, privateKeyHex, signerAddress string) (apiKey, secret, passphrase string, err error) {
	api := clobBaseURL + "/auth/api-key"
	logger.InfoZ(ctx, "polymarket-CreateAPIKey-Req",
		zap.String("signerAddress", signerAddress),
		zap.String("endpoint", api))
	nonce := time.Now().UnixNano()
	headers, err := p.buildL1AuthHeaders(ctx, privateKeyHex, signerAddress, nonce)
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-CreateAPIKey-Resp",
			zap.String("signerAddress", signerAddress),
			zap.Error(err))
		return "", "", "", err
	}
	bodyStr := "{}"
	resp, err := p.client.Post(ctx, api, nil, headers, bodyStr)
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-CreateAPIKey-Resp",
			zap.String("signerAddress", signerAddress),
			zap.Error(err))
		return "", "", "", fmt.Errorf("create api key request failed: %w", err)
	}
	var result struct {
		APIKey     string `json:"apiKey"`
		Secret     string `json:"secret"`
		Passphrase string `json:"passphrase"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		notification.SendErrorLog(ctx, "polymarket-CreateAPIKey-Resp",
			zap.String("signerAddress", signerAddress),
			zap.String("rawBody", string(resp.Body())),
			zap.Error(err))
		return "", "", "", fmt.Errorf("parse create api key response: %w", err)
	}
	// 响应详细打印（脱敏：apiKey 只打前8位，secret/passphrase 只打长度）
	apiKeyMask := result.APIKey
	if len(apiKeyMask) > 8 {
		apiKeyMask = apiKeyMask[:8] + "***"
	}
	logger.InfoZ(ctx, "polymarket-CreateAPIKey-Resp",
		zap.String("signerAddress", signerAddress),
		zap.String("apiKeyMask", apiKeyMask),
		zap.Int("secretLen", len(result.Secret)),
		zap.Int("passphraseLen", len(result.Passphrase)))
	return result.APIKey, result.Secret, result.Passphrase, nil
}

// cancelOrders 批量取消订单（CLOB DELETE /orders）
func (p *Polymarket) cancelOrders(ctx context.Context, orderIDs []string, polymarketAccount entity.PolymarketAccount) (*entity.BatchCancelOrderResponse, error) {
	startTime := time.Now()
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("order IDs list is empty")
	}
	orderIDsJSON, _ := json.Marshal(orderIDs)
	logger.InfoZ(ctx, "polymarket-BatchCancelOrder-Req",
		zap.String("orderIDs", string(orderIDsJSON)),
		zap.Int("orderIDsCount", len(orderIDs)),
		zap.String("proxyAddress", polymarketAccount.ProxyAddress),
		zap.String("endpoint", clobBaseURL+"/orders"))
	api := clobBaseURL + "/orders"
	bodyJSON, err := json.Marshal(orderIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal order IDs: %w", err)
	}
	bodyStr := string(bodyJSON)
	headers, err := p.buildAuthHeaders(ctx, "DELETE", "/orders", polymarketAccount, bodyStr)
	if err != nil {
		return nil, fmt.Errorf("build auth headers: %w", err)
	}
	resp, err := p.client.Delete(ctx, api, headers, bodyStr)
	p.checkTimeoutAndAlert(ctx, startTime, "CancelOrders")
	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-BatchCancelOrder-Resp",
			zap.String("orderIDs", string(orderIDsJSON)),
			zap.Int("orderIDsCount", len(orderIDs)),
			zap.String("proxyAddress", polymarketAccount.ProxyAddress),
			zap.Error(err))
		return nil, err
	}
	var apiResp struct {
		Canceled    []string          `json:"canceled"`
		NotCanceled map[string]string `json:"not_canceled"`
	}
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		notification.SendErrorLog(ctx, "polymarket-BatchCancelOrder-Resp",
			zap.String("orderIDs", string(orderIDsJSON)),
			zap.String("rawBody", string(resp.Body())),
			zap.Error(err))
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if apiResp.Canceled == nil {
		apiResp.Canceled = []string{}
	}
	if apiResp.NotCanceled == nil {
		apiResp.NotCanceled = map[string]string{}
	}
	orders := make([]entity.CancelOrderResponse, 0)
	for _, id := range apiResp.Canceled {
		orders = append(orders, entity.CancelOrderResponse{OrderID: id, Status: "canceled", Message: ""})
	}
	for id, reason := range apiResp.NotCanceled {
		orders = append(orders, entity.CancelOrderResponse{OrderID: id, Status: "failed", Message: reason})
	}
	successful := len(apiResp.Canceled)
	failed := len(apiResp.NotCanceled)
	out := &entity.BatchCancelOrderResponse{
		Orders:     orders,
		Total:      len(orderIDs),
		Successful: successful,
		Failed:     failed,
		Success:    failed == 0,
		Message:    fmt.Sprintf("成功取消 %d 笔，失败 %d 笔", successful, failed),
	}
	outJSON, _ := json.Marshal(out)
	logger.InfoZ(ctx, "polymarket-BatchCancelOrder-Resp",
		zap.String("orderIDs", string(orderIDsJSON)),
		zap.Int("orderIDsCount", len(orderIDs)),
		zap.Int("successful", successful),
		zap.Int("failed", failed),
		zap.Bool("success", out.Success),
		zap.String("response", string(outJSON)))
	return out, nil
}

// signOrder 签名订单（EIP-712）。返回签名后的订单与本次下单的费率（bps，V2 恒为 "0"）。
func (p *Polymarket) signOrder(ctx context.Context, orderData map[string]interface{}, polymarketAccount entity.PolymarketAccount, negRisk bool) (map[string]interface{}, string, error) {
	if polymarketAccount.SignatureType == sigType1271 {
		return p.signOrderPoly1271(ctx, orderData, polymarketAccount, negRisk)
	}
	salt, err := p.generateSalt()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate salt: %w", err)
	}
	orderTypeStr := orderData["order_type"].(string)
	orderType := polyenum.PolymarketOrderType(strings.ToUpper(orderTypeStr))
	sideStr := strings.ToUpper(orderData["side"].(string))
	feeRateBps := "0"
	priceStr, _ := orderData["price"].(string)
	sizeStr, _ := orderData["size"].(string)
	amountStr, _ := orderData["amount"].(string)
	isMarket := priceStr == ""
	var size, usdcAmountDecimal *big.Int
	if isMarket {
		if sideStr == "BUY" {
			if amountStr == "" {
				return nil, "", fmt.Errorf("市价买单必须提供 amount")
			}
			usdcAmountDecimal, err = p.parseDecimalToBigInt(amountStr, usdcDecimals)
			if err != nil {
				return nil, "", fmt.Errorf("parse market amount: %w", err)
			}
			usdcAmountDecimal = new(big.Int).Mul(new(big.Int).Div(usdcAmountDecimal, big.NewInt(10000)), big.NewInt(10000))
			if usdcAmountDecimal.Sign() <= 0 {
				return nil, "", fmt.Errorf("市价买单金额取整后无效或为负")
			}
			// 默认 0.99；best_ask > 0.99 或 < 0.01 时用实际 best_ask，否则高波动盘口会签错价
			maxPrice := new(big.Int).SetInt64(990000)
			if worstStr, ok := orderData["market_worst_price"].(string); ok && worstStr != "" {
				if parsed, err := p.parseDecimalToBigInt(worstStr, usdcDecimals); err == nil && parsed.Sign() > 0 && parsed.Cmp(big.NewInt(1000000)) < 0 {
					if parsed.Cmp(big.NewInt(990000)) > 0 || parsed.Cmp(big.NewInt(10000)) < 0 {
						maxPrice = parsed
					}
				}
			}
			multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(conditionalTokenDecimals), nil)
			size = new(big.Int).Div(new(big.Int).Mul(usdcAmountDecimal, multiplier), maxPrice)
			if size.Sign() <= 0 {
				return nil, "", fmt.Errorf("市价买单推导份额无效或为负")
			}
			size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(10000)), big.NewInt(10000))
			// 隐含价 = makerAmount/takerAmount 必须严格 < 1。worst_price 为 3 位小数 tick 且
			// 0.01 网格下 size 被压到 <= usdcAmount 时，改用 0.001 网格反算，保证可成交。
			if size.Cmp(usdcAmountDecimal) <= 0 &&
				new(big.Int).Rem(maxPrice, big.NewInt(1000)).Sign() == 0 &&
				new(big.Int).Rem(maxPrice, big.NewInt(10000)).Sign() != 0 {
				size = new(big.Int).Div(new(big.Int).Mul(usdcAmountDecimal, multiplier), maxPrice)
				size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(1000)), big.NewInt(1000))
				if size.Cmp(usdcAmountDecimal) <= 0 {
					size = new(big.Int).Mul(new(big.Int).Add(new(big.Int).Div(usdcAmountDecimal, big.NewInt(1000)), big.NewInt(1)), big.NewInt(1000))
				}
			}
		} else {
			if sizeStr == "" {
				return nil, "", fmt.Errorf("市价卖单必须提供 size")
			}
			size, err = p.parseDecimalToBigInt(sizeStr, conditionalTokenDecimals)
			if err != nil {
				return nil, "", fmt.Errorf("parse market size: %w", err)
			}
			size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(10000)), big.NewInt(10000))
			if size.Sign() <= 0 {
				return nil, "", fmt.Errorf("市价卖单份额取整后无效或为负")
			}
			// 默认 0.01；best_bid < 0.01 或 > 0.99 时用实际 best_bid
			minPrice := new(big.Int).SetInt64(10000)
			if worstStr, ok := orderData["market_worst_price"].(string); ok && worstStr != "" {
				if parsed, err := p.parseDecimalToBigInt(worstStr, usdcDecimals); err == nil && parsed.Sign() > 0 && parsed.Cmp(big.NewInt(1000000)) < 0 {
					if parsed.Cmp(big.NewInt(990000)) > 0 || parsed.Cmp(big.NewInt(10000)) < 0 {
						minPrice = parsed
					}
				}
			}
			divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(conditionalTokenDecimals), nil)
			usdcAmountDecimal = new(big.Int).Div(new(big.Int).Mul(size, minPrice), divisor)
			if usdcAmountDecimal.Sign() <= 0 {
				return nil, "", fmt.Errorf("市价卖单推导金额无效或为负")
			}
			usdcAmountDecimal = new(big.Int).Mul(new(big.Int).Div(usdcAmountDecimal, big.NewInt(10000)), big.NewInt(10000))
		}
	} else {
		size, err = p.parseDecimalToBigInt(sizeStr, conditionalTokenDecimals)
		if err != nil {
			return nil, "", fmt.Errorf("parse limit size: %w", err)
		}
		size = new(big.Int).Mul(new(big.Int).Div(size, big.NewInt(10000)), big.NewInt(10000))
		if size.Sign() <= 0 {
			return nil, "", fmt.Errorf("限价单 size 无效或为负")
		}
		priceDecimal, err := p.parseDecimalToBigInt(priceStr, usdcDecimals)
		if err != nil {
			return nil, "", fmt.Errorf("parse limit price: %w", err)
		}
		if priceDecimal.Sign() <= 0 {
			return nil, "", fmt.Errorf("限价单 price 无效或为负")
		}
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(conditionalTokenDecimals), nil)
		usdcAmountDecimal = new(big.Int).Div(new(big.Int).Mul(priceDecimal, size), divisor)
		usdcAmountDecimal = new(big.Int).Mul(new(big.Int).Div(usdcAmountDecimal, big.NewInt(10000)), big.NewInt(10000))
		if usdcAmountDecimal.Sign() <= 0 {
			return nil, "", fmt.Errorf("限价单金额取整后无效或为负")
		}
	}
	var makerAmount, takerAmount *big.Int
	sideNum := 0
	if sideStr == "SELL" {
		sideNum = 1
	}
	if sideNum == 0 {
		makerAmount = new(big.Int).Set(usdcAmountDecimal)
		takerAmount = new(big.Int).Set(size)
	} else {
		makerAmount = new(big.Int).Set(size)
		takerAmount = new(big.Int).Set(usdcAmountDecimal)
	}
	tokenID, ok := new(big.Int).SetString(orderData["token_id"].(string), 10)
	if !ok {
		return nil, "", fmt.Errorf("invalid token_id")
	}
	makerAddress := polymarketAccount.ProxyAddress
	if makerAddress == "" {
		makerAddress = polymarketAccount.EOAAddress
	}
	signerAddress := polymarketAccount.EOAAddress
	if signerAddress == "" {
		signerAddress = polymarketAccount.ProxyAddress
	}
	expiration := int64(0)
	if orderType == polyenum.PolymarketOrderTypeGTD {
		expiration = time.Now().Add(24 * time.Hour).Unix()
	}
	timestamp := time.Now().UnixMilli()
	builder := zeroBytes32
	if value, ok := orderData["builder"].(string); ok && strings.TrimSpace(value) != "" {
		builder = strings.TrimSpace(value)
	}
	orderForSigning := map[string]interface{}{
		"salt": salt, "maker": strings.ToLower(makerAddress), "signer": strings.ToLower(signerAddress),
		"tokenId": tokenID, "makerAmount": makerAmount, "takerAmount": takerAmount,
		"side": sideNum, "signatureType": polymarketAccount.SignatureType,
		"timestamp": big.NewInt(timestamp), "metadata": zeroBytes32, "builder": builder,
	}
	signature, err := p.signOrderWithEIP712(ctx, orderForSigning, negRisk, polymarketAccount.PrivateKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign order with EIP-712: %w", err)
	}
	saltAbs := new(big.Int).Abs(salt)
	maxInt64 := big.NewInt(9223372036854775807)
	if saltAbs.Cmp(maxInt64) > 0 {
		saltAbs.Mod(saltAbs, maxInt64)
	}
	saltInt64 := saltAbs.Int64()
	if saltInt64 == 0 {
		saltInt64 = 1
	}
	return map[string]interface{}{
		"salt": saltInt64, "maker": orderForSigning["maker"], "signer": orderForSigning["signer"],
		"tokenId":     tokenID.String(),
		"makerAmount": makerAmount.String(), "takerAmount": takerAmount.String(),
		"expiration": strconv.FormatInt(expiration, 10), "side": sideStr, "signatureType": polymarketAccount.SignatureType,
		"timestamp": strconv.FormatInt(timestamp, 10), "metadata": zeroBytes32, "builder": builder,
		"signature": signature,
	}, feeRateBps, nil
}

// orderEIP712Domain 返回下单签名的 EIP-712 domain。
// CLOB V2 要求 version="2"；写成 "1" 会签出被交易所判为 invalid signature 的订单，
// 所以这里集中一处、由 orderEIP712DomainVersionIsV2 单测守着。
func orderEIP712Domain(negRisk bool) map[string]interface{} {
	verifyingContract := ctfExchangeAddress
	if negRisk {
		verifyingContract = negRiskCTFExchangeAddress
	}
	return map[string]interface{}{
		"name": eip712DomainName, "version": eip712DomainVersion, "chainId": eip712ChainID,
		"verifyingContract": verifyingContract,
	}
}

// orderEIP712DomainTypes 是 domain 的 EIP-712 字段定义，单测也用它算 domain separator。
func orderEIP712DomainTypes() map[string][]map[string]string {
	return map[string][]map[string]string{
		"EIP712Domain": {
			{"name": "name", "type": "string"}, {"name": "version", "type": "string"},
			{"name": "chainId", "type": "uint256"}, {"name": "verifyingContract", "type": "address"},
		},
	}
}

func (p *Polymarket) signOrderWithEIP712(ctx context.Context, orderData map[string]interface{}, negRisk bool, privateKey string) (string, error) {
	domain := orderEIP712Domain(negRisk)
	types := orderEIP712DomainTypes()
	types["Order"] = []map[string]string{
		{"name": "salt", "type": "uint256"}, {"name": "maker", "type": "address"},
		{"name": "signer", "type": "address"},
		{"name": "tokenId", "type": "uint256"}, {"name": "makerAmount", "type": "uint256"},
		{"name": "takerAmount", "type": "uint256"}, {"name": "side", "type": "uint8"},
		{"name": "signatureType", "type": "uint8"}, {"name": "timestamp", "type": "uint256"},
		{"name": "metadata", "type": "bytes32"}, {"name": "builder", "type": "bytes32"},
	}
	hash, err := p.hashEIP712(domain, types, "Order", orderData)
	if err != nil {
		return "", err
	}
	sig, err := signLocal(privateKey, hash.Bytes())
	if err != nil {
		return "", err
	}
	return hexutil.Encode(sig), nil
}

func (p *Polymarket) hashEIP712(domain map[string]interface{}, types map[string][]map[string]string, primaryType string, message map[string]interface{}) (common.Hash, error) {
	domainSeparator, err := p.encodeData("EIP712Domain", domain, types)
	if err != nil {
		return common.Hash{}, err
	}
	messageHash, err := p.encodeData(primaryType, message, types)
	if err != nil {
		return common.Hash{}, err
	}
	prefix := []byte{0x19, 0x01}
	data := append(prefix, domainSeparator...)
	data = append(data, messageHash...)
	return crypto.Keccak256Hash(data), nil
}

func (p *Polymarket) encodeData(primaryType string, data map[string]interface{}, types map[string][]map[string]string) ([]byte, error) {
	typeHash := p.encodeType(primaryType, types)
	typeHashBytes := crypto.Keccak256([]byte(typeHash))
	var encodedValues []byte
	for _, field := range types[primaryType] {
		fieldName := field["name"]
		fieldType := field["type"]
		value := data[fieldName]
		encodedValue, err := p.encodeValue(fieldType, value)
		if err != nil {
			return nil, err
		}
		encodedValues = append(encodedValues, encodedValue...)
	}
	result := append(typeHashBytes, encodedValues...)
	return crypto.Keccak256(result), nil
}

func (p *Polymarket) encodeType(primaryType string, types map[string][]map[string]string) string {
	var parts []string
	parts = append(parts, primaryType+"(")
	for i, field := range types[primaryType] {
		if i > 0 {
			parts = append(parts, ",")
		}
		parts = append(parts, field["type"]+" "+field["name"])
	}
	parts = append(parts, ")")
	return strings.Join(parts, "")
}

func (p *Polymarket) encodeValue(dataType string, value interface{}) ([]byte, error) {
	switch dataType {
	case "uint256":
		bigInt, err := p.toBigInt(value)
		if err != nil {
			return nil, err
		}
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil
	case "uint8":
		var uint8Value uint8
		switch v := value.(type) {
		case int:
			if v < 0 || v > 255 {
				return nil, fmt.Errorf("uint8 value out of range: %d", v)
			}
			uint8Value = uint8(v)
		case int64:
			if v < 0 || v > 255 {
				return nil, fmt.Errorf("uint8 value out of range: %d", v)
			}
			uint8Value = uint8(v)
		default:
			return nil, fmt.Errorf("uint8 must be int or int64, got %T", value)
		}
		return common.LeftPadBytes([]byte{uint8Value}, 32), nil
	case "address":
		addrStr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("address must be string")
		}
		return common.LeftPadBytes(common.HexToAddress(addrStr).Bytes(), 32), nil
	case "string":
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("string must be string")
		}
		return crypto.Keccak256Hash([]byte(str)).Bytes(), nil
	case "bytes32":
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("bytes32 must be a hex string")
		}
		decoded, err := hexutil.Decode(str)
		if err != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("bytes32 must decode to exactly 32 bytes")
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", dataType)
	}
}

func (p *Polymarket) toBigInt(value interface{}) (*big.Int, error) {
	switch v := value.(type) {
	case *big.Int:
		return v, nil
	case string:
		bigInt, ok := new(big.Int).SetString(v, 10)
		if !ok {
			return nil, fmt.Errorf("invalid big.Int string: %s", v)
		}
		return bigInt, nil
	case int64:
		return big.NewInt(v), nil
	case int:
		return big.NewInt(int64(v)), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to *big.Int", value)
	}
}

func (p *Polymarket) getFeeRate(ctx context.Context, tokenID string, polymarketAccount entity.PolymarketAccount) (string, error) {
	api := clobBaseURL + "/fee-rate"
	params := map[string][]string{"token_id": {tokenID}}
	headers, err := p.buildAuthHeaders(ctx, "GET", "/fee-rate", polymarketAccount, "")
	if err != nil {
		return "0", err
	}
	resp, err := p.client.Get(ctx, api, params, headers)
	if err != nil {
		return "0", fmt.Errorf("fee-rate request: %w", err)
	}
	// CLOB /fee-rate 返回 {"base_fee": <bps>}，不是 fee_rate_bps；解析失败或非 200 必须上抛，
	// 否则调用方会把「查询失败」误当成 0 费率。
	var feeRateResp struct {
		BaseFee int64 `json:"base_fee"`
	}
	if err := json.Unmarshal(resp.Body(), &feeRateResp); err != nil {
		return "", fmt.Errorf("fee-rate parse: %w", err)
	}
	return strconv.FormatInt(feeRateResp.BaseFee, 10), nil
}

// GetPolymarketFeeRate exposes the CLOB base fee rate through the repository contract.
func (p *Polymarket) GetPolymarketFeeRate(ctx context.Context, tokenID string, account entity.PolymarketAccount) (string, error) {
	return p.getFeeRate(ctx, tokenID, account)
}

func (p *Polymarket) parseDecimalToBigInt(valueStr string, decimals int) (*big.Int, error) {
	valueFloat, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decimal: %w", err)
	}
	bigFloat := new(big.Float).SetFloat64(valueFloat)
	multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	bigFloat.Mul(bigFloat, multiplier)
	bigInt, _ := bigFloat.Int(nil)
	return bigInt, nil
}

func (p *Polymarket) generateSalt() (*big.Int, error) {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		randomBytes[i] = byte(timestamp >> (i * 8))
	}
	salt := new(big.Int).SetBytes(randomBytes)
	if salt.Sign() < 0 {
		salt.Neg(salt)
	}
	maxInt64 := big.NewInt(9223372036854775807)
	if salt.Cmp(maxInt64) > 0 {
		salt.Mod(salt, maxInt64)
	}
	if salt.Sign() == 0 {
		salt.SetInt64(1)
	}
	return salt, nil
}

func (p *Polymarket) buildAuthHeaders(ctx context.Context, method, path string, polymarketAccount entity.PolymarketAccount, body string) (map[string]string, error) {
	headers := make(map[string]string)
	if polymarketAccount.APIKey != "" && polymarketAccount.APISecret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		message := timestamp + strings.ToUpper(method) + path
		if body != "" {
			message += body
		}
		signature, err := p.hmacSHA256Sign(ctx, message, polymarketAccount.APISecret)
		if err != nil {
			return nil, fmt.Errorf("failed to sign auth message: %w", err)
		}
		headers["POLY_API_KEY"] = polymarketAccount.APIKey
		headers["POLY_PASSPHRASE"] = polymarketAccount.APIPassphrase
		headers["POLY_TIMESTAMP"] = timestamp
		headers["POLY_SIGNATURE"] = signature
		if polymarketAccount.EOAAddress != "" {
			headers["POLY_ADDRESS"] = polymarketAccount.EOAAddress
		} else if polymarketAccount.ProxyAddress != "" {
			headers["POLY_ADDRESS"] = polymarketAccount.ProxyAddress
		}
	}
	return headers, nil
}

func (p *Polymarket) hmacSHA256Sign(ctx context.Context, message string, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	var secretBytes []byte
	decoded, err := base64.URLEncoding.DecodeString(secret)
	if err == nil && len(decoded) > 0 {
		secretBytes = decoded
	} else {
		decoded, err = base64.StdEncoding.DecodeString(secret)
		if err == nil && len(decoded) > 0 {
			secretBytes = decoded
		} else {
			decoded, err := hex.DecodeString(strings.TrimPrefix(secret, "0x"))
			if err == nil && len(decoded) > 0 {
				secretBytes = decoded
			} else {
				secretBytes = []byte(secret)
			}
		}
	}
	h := hmac.New(sha256.New, secretBytes)
	h.Write([]byte(message))
	return base64.URLEncoding.EncodeToString(h.Sum(nil)), nil
}

// getUSDCBalance 获取指定地址的抵押币余额（Polymarket 结算币 pUSD，6 位小数）。
func (p *Polymarket) getUSDCBalance(ctx context.Context, address string) (string, error) {
	// Polygon主网RPC端点
	rpcURL := p.rpcURL
	if rpcURL == "" {
		rpcURL = "https://polygon-bor-rpc.publicnode.com"
	}

	// pUSD 合约地址 (Polygon主网)，符号 pUSD、6 位小数，替代旧抵押币 USDC.e(0x2791Bca1...)
	usdcAddress := "0xc011a7e12a19f7b1f670d46f03b03f3342e82dfb"

	// 连接到Polygon网络
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return "0", fmt.Errorf("failed to connect to Polygon network: %w", err)
	}
	defer client.Close()

	// ERC20标准balanceOf方法的ABI
	erc20ABI := `[{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"type": "function"
	}]`

	// 解析ABI
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return "0", fmt.Errorf("failed to parse ERC20 ABI: %w", err)
	}

	// 编码balanceOf调用数据
	data, err := parsedABI.Pack("balanceOf", common.HexToAddress(address))
	if err != nil {
		return "0", fmt.Errorf("failed to pack balanceOf call: %w", err)
	}

	// 构建调用消息
	msg := ethereum.CallMsg{
		To:   &common.Address{},
		Data: data,
	}
	copy(msg.To[:], common.HexToAddress(usdcAddress).Bytes())

	// 调用合约
	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return "0", fmt.Errorf("failed to call USDC contract: %w", err)
	}

	// 解析返回值
	var balance *big.Int
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return "0", fmt.Errorf("failed to unpack balance result: %w", err)
	}

	// pUSD 有6位小数，所以需要除以10^6来获得实际余额
	divisor := big.NewInt(1000000) // 10^6
	balanceFloat := new(big.Float).SetInt(balance)
	divisorFloat := new(big.Float).SetInt(divisor)
	resultFloat := new(big.Float).Quo(balanceFloat, divisorFloat)

	// 转换为字符串，保留6位小数
	return resultFloat.Text('f', 6), nil
}

// parseFloat 辅助函数：将字符串转换为float64
func (p *Polymarket) parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// getPositionsFromPolymarketAPI 通过Polymarket官方API获取持仓
func (p *Polymarket) getPositionsFromPolymarketAPI(ctx context.Context, address string) ([]entity.Position, error) {
	// 记录发起时间
	startTime := time.Now()

	// 使用Polymarket官方API
	apiURL := fmt.Sprintf("https://data-api.polymarket.com/positions?user=%s", address)
	resp, err := p.client.Get(ctx, apiURL, nil, nil)
	// 检查超时并报警
	p.checkTimeoutAndAlert(ctx, startTime, "getPositionsFromPolymarketAPI")

	if err != nil {
		return nil, fmt.Errorf("failed to call Polymarket positions API: %w", err)
	}

	// 解析Polymarket API响应
	var polymarketPositions []struct {
		ProxyWallet        string  `json:"proxyWallet"`
		Asset              string  `json:"asset"`
		ConditionID        string  `json:"conditionId"`
		Size               float64 `json:"size"`
		AvgPrice           float64 `json:"avgPrice"`
		InitialValue       float64 `json:"initialValue"`
		CurrentValue       float64 `json:"currentValue"`
		CashPnl            float64 `json:"cashPnl"`
		PercentPnl         float64 `json:"percentPnl"`
		TotalBought        float64 `json:"totalBought"`
		RealizedPnl        float64 `json:"realizedPnl"`
		PercentRealizedPnl float64 `json:"percentRealizedPnl"`
		CurPrice           float64 `json:"curPrice"`
		Redeemable         bool    `json:"redeemable"`
		Mergeable          bool    `json:"mergeable"`
		Title              string  `json:"title"`
		Slug               string  `json:"slug"`
		Icon               string  `json:"icon"`
		EventSlug          string  `json:"eventSlug"`
		Outcome            string  `json:"outcome"`
		OutcomeIndex       int     `json:"outcomeIndex"`
		OppositeOutcome    string  `json:"oppositeOutcome"`
		OppositeAsset      string  `json:"oppositeAsset"`
		EndDate            string  `json:"endDate"`
		NegativeRisk       bool    `json:"negativeRisk"`
	}

	if err := json.Unmarshal(resp.Body(), &polymarketPositions); err != nil {
		return nil, fmt.Errorf("failed to parse Polymarket API response: %w", err)
	}

	// 转换为内部Position格式，映射所有字段
	positions := []entity.Position{}
	for _, pos := range polymarketPositions {
		positions = append(positions, entity.Position{
			// 基础信息
			TokenID:      pos.Asset,
			AssetID:      pos.Asset,
			Market:       pos.ConditionID,
			Outcome:      pos.Outcome,
			Balance:      fmt.Sprintf("%.6f", pos.Size),
			BalanceFloat: pos.Size,
			MarketName:   pos.Title,
			OutcomeName:  pos.Outcome,
			CurrentPrice: fmt.Sprintf("%.6f", pos.CurPrice),
			ValueUSD:     fmt.Sprintf("%.6f", pos.CurrentValue),
			PnlUSD:       fmt.Sprintf("%.6f", pos.CashPnl),
			LastUpdated:  pos.EndDate,

			// Polymarket特有字段
			ProxyWallet:        pos.ProxyWallet,
			ConditionID:        pos.ConditionID,
			AvgPrice:           pos.AvgPrice,
			InitialValue:       pos.InitialValue,
			CurrentValue:       pos.CurrentValue,
			CashPnl:            pos.CashPnl,
			PercentPnl:         pos.PercentPnl,
			TotalBought:        pos.TotalBought,
			RealizedPnl:        pos.RealizedPnl,
			PercentRealizedPnl: pos.PercentRealizedPnl,
			Redeemable:         pos.Redeemable,
			Mergeable:          pos.Mergeable,
			Slug:               pos.Slug,
			Icon:               pos.Icon,
			EventSlug:          pos.EventSlug,
			OutcomeIndex:       pos.OutcomeIndex,
			OppositeOutcome:    pos.OppositeOutcome,
			OppositeAsset:      pos.OppositeAsset,
			EndDate:            pos.EndDate,
			NegativeRisk:       pos.NegativeRisk,
		})
	}

	return positions, nil
}

// GetEventBySlug 根据slug获取事件信息
func (p *Polymarket) GetEventBySlug(ctx context.Context, req entity.GetEventBySlugRequest) (*entity.GetEventBySlugResponse, error) {
	// 记录发起时间
	startTime := time.Now()

	// 详细打印输入
	reqJSON, _ := json.Marshal(req)
	logger.InfoZ(ctx, "polymarket-GetEventBySlug-Req",
		zap.String("request", string(reqJSON)),
		zap.String("slugName", req.SlugName),
		zap.String("endpoint", fmt.Sprintf("%s/events/slug/%s", p.baseURL, req.SlugName)))

	// 验证请求
	if req.SlugName == "" {
		return nil, fmt.Errorf("slug name is required")
	}

	// 发送请求
	api := fmt.Sprintf("%s/events/slug/%s", p.baseURL, req.SlugName)

	resp, err := p.client.Get(ctx, api, nil, nil)
	// 检查超时并报警
	p.checkTimeoutAndAlert(ctx, startTime, "GetEventBySlug")

	if err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetEventBySlug-RequestFailed",
			zap.String("slugName", req.SlugName),
			zap.String("endpoint", api),
			zap.Error(err))
		return nil, err
	}

	// 解析响应
	var event entity.Event
	if err = json.Unmarshal(resp.Body(), &event); err != nil {
		notification.SendErrorLog(ctx, "polymarket-GetEventBySlug-ParseError",
			zap.String("slugName", req.SlugName),
			zap.String("response", string(resp.Body())),
			zap.Error(err))
		return nil, err
	}

	// 详细打印输出
	eventJSON, _ := json.Marshal(event)
	eventPreview := string(eventJSON)
	if len(eventPreview) > 2000 {
		eventPreview = eventPreview[:2000] + "...(truncated)"
	}
	logger.InfoZ(ctx, "polymarket-GetEventBySlug-Resp",
		zap.String("slugName", req.SlugName),
		zap.String("eventID", event.ID),
		zap.String("title", event.Title),
		zap.Int("marketsCount", len(event.Markets)),
		zap.Int("tagsCount", len(event.Tags)),
		zap.String("event", eventPreview))

	response := &entity.GetEventBySlugResponse{
		Event: &event,
	}

	return response, nil
}
