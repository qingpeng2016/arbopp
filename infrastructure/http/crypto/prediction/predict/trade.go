package predict

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"

	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
)

// GetMarket GET /markets/{id}。
func (c *Client) GetMarket(ctx context.Context, marketID string) (*httpentity.PredictMarket, error) {
	marketID = strings.TrimSpace(marketID)
	if marketID == "" {
		return nil, fmt.Errorf("predict market id required")
	}
	var resp httpentity.PredictGetMarketResponse
	if err := c.doGET(ctx, "/markets/"+marketID, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("predict market %s: %s", marketID, apiErr(resp.Message, resp.Error))
	}
	m := resp.Data
	return &m, nil
}

// PlaceMarketOrder MARKET 买入 outcome token。
func (c *Client) PlaceMarketOrder(ctx context.Context, acc httpentity.PredictAccount, req repository.PredictMarketOrderReq) (*repository.PredictPlaceOrderResult, error) {
	if c == nil {
		return nil, fmt.Errorf("predict client is nil")
	}
	marketID := strings.TrimSpace(req.MarketID)
	tokenID := strings.TrimSpace(req.TokenID)
	if marketID == "" || tokenID == "" || req.Quantity <= 0 {
		return nil, fmt.Errorf("predict order: invalid market/token/qty")
	}
	jwt, err := c.ensureJWT(ctx, acc)
	if err != nil {
		return nil, err
	}
	mkt, err := c.GetMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}
	var bookResp httpentity.PredictGetMarketOrderbookResponse
	if err := c.doGET(ctx, "/markets/"+marketID+"/orderbook", nil, &bookResp); err != nil {
		return nil, err
	}
	if !bookResp.Success {
		return nil, fmt.Errorf("predict orderbook %s: %s", marketID, apiErr(bookResp.Message, bookResp.Error))
	}
	book := bookResp.Data
	qtyWei := floatSharesToWei(req.Quantity)
	// 有 PriceLimit 时按限价裁剪盘口，且不再加滑点，避免市价穿透 best_ask。
	slippage := int64(defaultSlippageBps)
	if req.PriceLimit > 0 && req.PriceLimit < 1 {
		slippage = 0
	}
	var amounts marketAmounts
	var ok bool
	var side int64
	if req.Sell {
		side = 1
		var bids []depthLevel
		if req.BuyYes {
			bids = bidsFromBook(book.Bids)
		} else {
			bids = bidsFromYesAsks(book.Asks)
		}
		bids = filterBidsAtOrAbove(bids, req.PriceLimit)
		amounts, ok = marketSellByQuantity(bids, qtyWei, slippage)
	} else {
		side = 0
		var asks []depthLevel
		if req.BuyYes {
			asks = levelsFromBook(book.Asks)
		} else {
			asks = asksFromYesBids(book.Bids)
		}
		asks = filterAsksAtOrBelow(asks, req.PriceLimit)
		amounts, ok = marketBuyByQuantity(asks, qtyWei, slippage)
	}
	if !ok {
		return nil, fmt.Errorf("predict order: insufficient book depth")
	}
	_, priv, addr, predictAcct, err := resolveSigner(acc)
	if err != nil {
		return nil, err
	}
	maker := addr.Hex()
	if predictAcct != "" {
		maker = predictAcct
	}
	feeBps := strconv.Itoa(mkt.FeeRateBps)
	signed, pricePerShare, expUnix, err := buildSignedMarketOrder(
		maker, tokenID, feeBps, amounts, side, predictAcct, mkt.IsNegRisk, mkt.IsYieldBearing, priv,
	)
	if err != nil {
		return nil, err
	}
	order := httpentity.PredictSignedOrder{
		Salt:          fmt.Sprint(signed["salt"]),
		Maker:         fmt.Sprint(signed["maker"]),
		Signer:        fmt.Sprint(signed["signer"]),
		Taker:         zeroAddress,
		TokenID:       tokenID,
		MakerAmount:   fmt.Sprint(signed["makerAmount"]),
		TakerAmount:   fmt.Sprint(signed["takerAmount"]),
		Expiration:    expUnix,
		Nonce:         "0",
		FeeRateBps:    feeBps,
		Side:          int(side),
		SignatureType: 0,
		Signature:     fmt.Sprint(signed["signature"]),
		Hash:          fmt.Sprint(signed["hash"]),
	}
	body := httpentity.PredictCreateOrderBody{
		Data: httpentity.PredictCreateOrderData{
			Order:         order,
			PricePerShare: pricePerShare,
			Strategy:      "MARKET",
			SlippageBps:   strconv.FormatInt(slippage, 10),
		},
	}
	apiKey := strings.TrimSpace(acc.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.apiKey)
	}
	var resp httpentity.PredictCreateOrderResponse
	if err := c.doPOSTWithKey(ctx, apiKey, "/orders", body, jwt, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("predict place order: %s", apiErr(resp.Message, resp.Error))
	}
	fill := weiToFloatShares(amounts.takerAmount)
	if req.Quantity < fill {
		fill = req.Quantity
	}
	return &repository.PredictPlaceOrderResult{
		OrderID:   resp.Data.OrderID,
		OrderHash: resp.Data.OrderHash,
		FillQty:   fill,
	}, nil
}

// GetOrderByHash GET /orders/{hash}（JWT）；amountFilled 为份额 wei。
func (c *Client) GetOrderByHash(ctx context.Context, acc httpentity.PredictAccount, orderHash string) (*repository.PredictOrderFill, error) {
	if c == nil {
		return nil, fmt.Errorf("predict client is nil")
	}
	orderHash = strings.TrimSpace(orderHash)
	if orderHash == "" {
		return nil, fmt.Errorf("predict order hash required")
	}
	jwt, err := c.ensureJWT(ctx, acc)
	if err != nil {
		return nil, err
	}
	apiKey := strings.TrimSpace(acc.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.apiKey)
	}
	var resp httpentity.PredictGetOrderResponse
	if err := c.doGETAuth(ctx, apiKey, jwt, "/orders/"+orderHash, nil, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("predict get order: %s", apiErr(resp.Message, resp.Error))
	}
	d := resp.Data
	out := &repository.PredictOrderFill{
		OrderID:      strings.TrimSpace(d.ID),
		OrderHash:    strings.TrimSpace(d.Order.Hash),
		Status:       strings.TrimSpace(d.Status),
		Amount:       weiStringToShares(d.Amount),
		AmountFilled: weiStringToShares(d.AmountFilled),
	}
	if out.OrderHash == "" {
		out.OrderHash = orderHash
	}
	// 手续费从 matches 汇总（可选；失败不挡查单）。
	if fee, err := c.sumMatchFees(ctx, apiKey, jwt, out.OrderHash); err == nil {
		out.FeeUSDT = fee
	}
	return out, nil
}

func (c *Client) sumMatchFees(ctx context.Context, apiKey, jwt, orderHash string) (float64, error) {
	q := url.Values{}
	q.Set("orderHashes", orderHash)
	q.Set("first", "50")
	var resp httpentity.PredictGetMatchesResponse
	if err := c.doGETAuth(ctx, apiKey, jwt, "/orders/matches", q, &resp); err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("predict matches: %s", apiErr(resp.Message, resp.Error))
	}
	var sum float64
	for _, m := range resp.Data {
		sum += matchFeeUSDT(m.TakerFee)
		if m.TakerFee == nil {
			sum += weiStringToShares(m.TakerFeeAmount) // 兼容旧字段，按 COLLATERAL 口径
		}
		for i := range m.Makers {
			sum += matchFeeUSDT(m.Makers[i].Fee)
		}
		for _, f := range m.MakerFeeAmounts {
			sum += weiStringToShares(f)
		}
	}
	return sum, nil
}

func matchFeeUSDT(f *httpentity.PredictMatchFee) float64 {
	if f == nil {
		return 0
	}
	// COLLATERAL=USDT wei；SHARES 时暂按份额 wei 记（无均价则无法换 USDT）
	if strings.EqualFold(f.Type, "SHARES") {
		return 0
	}
	return weiStringToShares(f.Amount)
}

func weiStringToShares(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return 0
	}
	return weiToFloatShares(n)
}