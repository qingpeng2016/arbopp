package predict

import (
	"context"
	"fmt"
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
		amounts, ok = marketSellByQuantity(bids, qtyWei, defaultSlippageBps)
	} else {
		side = 0
		var asks []depthLevel
		if req.BuyYes {
			asks = levelsFromBook(book.Asks)
		} else {
			asks = asksFromYesBids(book.Bids)
		}
		amounts, ok = marketBuyByQuantity(asks, qtyWei, defaultSlippageBps)
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
			SlippageBps:   strconv.FormatInt(defaultSlippageBps, 10),
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