package sxbet

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	repo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
)

const (
	oddsScale   = 1e20
	usdcScale   = 1e6
	orderExpiry = 3600 // seconds
)

// GetMetadataObv3 GET /metadata/obv3。
func (c *Client) GetMetadataObv3(ctx context.Context) (*entity.SxbetMetadataObv3, error) {
	var resp entity.SxbetMetadataResponse
	if err := c.doGET(ctx, "/metadata/obv3", nil, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") {
		return nil, fmt.Errorf("sxbet metadata/obv3: status failure")
	}
	return &resp.Data, nil
}

// PlaceIOCOrder 限价 IOC（≈ FAK）：以 best_ask 为限价，立刻成交能成交的部分。
func (c *Client) PlaceIOCOrder(ctx context.Context, acc entity.SxbetAccount, req entity.SxbetPlaceOrderReq) (*repo.SxbetPlaceOrderResult, error) {
	if c == nil {
		return nil, fmt.Errorf("sxbet client is nil")
	}
	marketHash := strings.TrimSpace(req.MarketHash)
	if marketHash == "" || !(req.Price > 0 && req.Price < 1) || !(req.Quantity > 0) {
		return nil, fmt.Errorf("sxbet order: invalid market/price/qty")
	}
	apiKey := strings.TrimSpace(acc.APIKey)
	if apiKey == "" {
		apiKey = c.apiKey
	}
	if apiKey == "" || strings.TrimSpace(acc.PrivateKey) == "" {
		return nil, fmt.Errorf("sxbet order: missing api_key or private_key")
	}
	meta, err := c.GetMetadataObv3(ctx)
	if err != nil {
		return nil, err
	}
	priv, addr, err := parsePrivateKey(acc.PrivateKey)
	if err != nil {
		return nil, err
	}
	if eoa := strings.TrimSpace(acc.EOAAddress); eoa != "" && !strings.EqualFold(eoa, addr.Hex()) {
		return nil, fmt.Errorf("sxbet order: eoa_address mismatch key")
	}
	salt, err := randomSaltHex()
	if err != nil {
		return nil, err
	}
	totalBet := formatBaseUnits(req.Quantity)
	pctOdds := formatOdds(req.Price)
	expiry := time.Now().Unix() + orderExpiry
	item := entity.SxbetCreateOrderItem{
		MarketHash:               marketHash,
		Maker:                    addr.Hex(),
		TotalBetSize:             totalBet,
		PercentageOdds:           pctOdds,
		Salt:                     salt,
		Expiry:                   expiry,
		BaseToken:                strings.TrimSpace(meta.ActiveAsset.BaseToken),
		IsMakerBettingOutcomeOne: req.BuyYes,
		TimeInForce:              "IOC",
		ClientOrderID:            strings.TrimSpace(req.ClientOrderID),
	}
	sig, err := signSxbetOrder(meta, item, priv)
	if err != nil {
		return nil, err
	}
	item.OrderSignature = sig
	body := entity.SxbetCreateOrdersRequest{
		Orders:         []entity.SxbetCreateOrderItem{item},
		WaitForOutcome: true,
		MaxWaitTime:    12000,
	}
	var resp entity.SxbetCreateOrdersResponse
	if err := c.doPOSTJSON(ctx, apiKey, "/orders-v3", body, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") || len(resp.Data.Orders) == 0 {
		return nil, fmt.Errorf("sxbet place order failed: status=%s", resp.Status)
	}
	one := resp.Data.Orders[0]
	out := &repo.SxbetPlaceOrderResult{
		OrderID: one.OrderID,
		Status:  one.Status,
		State:   "",
	}
	if one.Status == "FAILED" {
		out.RawError = one.FailureReason
		return out, fmt.Errorf("sxbet order FAILED: %s", one.FailureReason)
	}
	if one.Outcome != nil {
		out.State = one.Outcome.State
		out.FillQty = parseBaseUnits(one.Outcome.FillAmount)
	}
	return out, nil
}

// GetOrderByID GET /orders-v3/{orderId}。
func (c *Client) GetOrderByID(ctx context.Context, acc entity.SxbetAccount, orderID string) (*repo.SxbetOrderFill, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, fmt.Errorf("sxbet order id required")
	}
	apiKey := strings.TrimSpace(acc.APIKey)
	if apiKey == "" {
		apiKey = c.apiKey
	}
	var resp entity.SxbetGetOrderResponse
	if err := c.doGETJSON(ctx, apiKey, "/orders-v3/"+orderID, nil, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") {
		return nil, fmt.Errorf("sxbet get order failure")
	}
	o := resp.Data
	fill := parseBaseUnits(o.TotalBetSize) - parseBaseUnits(o.RemainingSize)
	if fill < 0 {
		fill = 0
	}
	return &repo.SxbetOrderFill{
		OrderID: strings.TrimSpace(o.OrderID),
		Status:  strings.TrimSpace(o.Status),
		FillQty: fill,
	}, nil
}

// GetFillsByOrderID GET /fills-v3?orderId=；汇总 fillAmount 与 ceRefundFeeAmount。
func (c *Client) GetFillsByOrderID(ctx context.Context, acc entity.SxbetAccount, orderID string) (*repo.SxbetOrderFill, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, fmt.Errorf("sxbet order id required")
	}
	apiKey := strings.TrimSpace(acc.APIKey)
	if apiKey == "" {
		apiKey = c.apiKey
	}
	q := url.Values{}
	q.Set("orderId", orderID)
	q.Set("perPage", "100")
	var resp entity.SxbetGetFillsResponse
	if err := c.doGETJSON(ctx, apiKey, "/fills-v3", q, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Status, "failure") {
		return nil, fmt.Errorf("sxbet get fills failure")
	}
	out := &repo.SxbetOrderFill{OrderID: orderID}
	for _, f := range resp.Data.Fills {
		out.FillQty += parseBaseUnits(f.FillAmount)
		out.FeeUSDT += parseBaseUnits(f.CeRefundFeeAmount)
		if st := strings.TrimSpace(f.Status); st != "" {
			out.Status = st
		}
	}
	return out, nil
}

func (c *Client) doGETJSON(ctx context.Context, apiKey, path string, query url.Values, out any) error {
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if k := strings.TrimSpace(apiKey); k != "" {
		req.Header.Set(apiKeyHeader, k)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sxbet GET %s: http %d: %s", path, resp.StatusCode, truncateErrBody(b))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("sxbet decode %s: %w (body %s)", path, err, truncateErrBody(b))
	}
	return nil
}

func (c *Client) doPOSTJSON(ctx context.Context, apiKey, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if k := strings.TrimSpace(apiKey); k != "" {
		req.Header.Set(apiKeyHeader, k)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sxbet POST %s: http %d: %s", path, resp.StatusCode, truncateErrBody(b))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("sxbet decode %s: %w (body %s)", path, err, truncateErrBody(b))
	}
	return nil
}

func signSxbetOrder(meta *entity.SxbetMetadataObv3, item entity.SxbetCreateOrderItem, key *ecdsa.PrivateKey) (string, error) {
	if meta == nil || key == nil {
		return "", fmt.Errorf("sxbet sign: nil meta/key")
	}
	domain := apitypes.TypedDataDomain{
		Name:              meta.Domain.Name,
		Version:           meta.Domain.Version,
		ChainId:           ethmath.NewHexOrDecimal256(meta.Domain.ChainID),
		VerifyingContract: common.HexToAddress(meta.Domain.VerifyingContract).Hex(),
	}
	if domain.Name == "" {
		domain.Name = "OBv3 Escrow"
	}
	if domain.Version == "" {
		domain.Version = "1"
	}
	typed := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Order": {
				{Name: "marketHash", Type: "bytes32"},
				{Name: "baseToken", Type: "address"},
				{Name: "totalBetSize", Type: "uint256"},
				{Name: "percentageOdds", Type: "uint256"},
				{Name: "salt", Type: "uint256"},
				{Name: "expiry", Type: "uint256"},
				{Name: "maker", Type: "address"},
				{Name: "isMakerBettingOutcomeOne", Type: "bool"},
			},
		},
		PrimaryType: "Order",
		Domain:      domain,
		Message: apitypes.TypedDataMessage{
			"marketHash":               padBytes32(item.MarketHash),
			"baseToken":                common.HexToAddress(item.BaseToken).Hex(),
			"totalBetSize":             item.TotalBetSize,
			"percentageOdds":           item.PercentageOdds,
			"salt":                     saltToUint(item.Salt),
			"expiry":                   fmt.Sprintf("%d", item.Expiry),
			"maker":                    common.HexToAddress(item.Maker).Hex(),
			"isMakerBettingOutcomeOne": item.IsMakerBettingOutcomeOne,
		},
	}
	domainSep, err := typed.HashStruct("EIP712Domain", typed.Domain.Map())
	if err != nil {
		return "", err
	}
	msgHash, err := typed.HashStruct(typed.PrimaryType, typed.Message)
	if err != nil {
		return "", err
	}
	raw := crypto.Keccak256([]byte{0x19, 0x01}, domainSep, msgHash)
	sig, err := crypto.Sign(raw, key)
	if err != nil {
		return "", err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x" + hex.EncodeToString(sig), nil
}

func parsePrivateKey(hexKey string) (*ecdsa.PrivateKey, common.Address, error) {
	hexKey = strings.TrimPrefix(strings.TrimSpace(hexKey), "0x")
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("sxbet private key: %w", err)
	}
	key, err := crypto.ToECDSA(b)
	if err != nil {
		return nil, common.Address{}, err
	}
	return key, crypto.PubkeyToAddress(key.PublicKey), nil
}

func randomSaltHex() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(b), nil
}

func padBytes32(h string) string {
	h = strings.TrimPrefix(strings.TrimSpace(h), "0x")
	if len(h) > 64 {
		h = h[len(h)-64:]
	}
	for len(h) < 64 {
		h = "0" + h
	}
	return "0x" + h
}

func saltToUint(salt string) string {
	salt = strings.TrimPrefix(strings.TrimSpace(salt), "0x")
	if salt == "" {
		return "0"
	}
	n := new(big.Int)
	if _, ok := n.SetString(salt, 16); !ok {
		return "0"
	}
	return n.String()
}

func formatOdds(price float64) string {
	// taker 限价 → 1e20；向下对齐到 0.125% 阶梯（oddsLadderStepSize=125 → step=125e15）
	const step = 125e15
	raw := price * oddsScale
	if raw < step {
		raw = step
	}
	aligned := math.Floor(raw/step) * step
	i := new(big.Int)
	new(big.Float).SetFloat64(aligned).Int(i)
	if i.Sign() <= 0 {
		return "0"
	}
	return i.String()
}

func formatBaseUnits(qty float64) string {
	v := new(big.Float).SetFloat64(qty * usdcScale)
	i, _ := v.Int(nil)
	if i == nil || i.Sign() <= 0 {
		return "0"
	}
	return i.String()
}

func parseBaseUnits(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		f, err := parseFloat(s)
		if err != nil {
			return 0
		}
		return f / usdcScale
	}
	f := new(big.Float).SetInt(n)
	out, _ := new(big.Float).Quo(f, big.NewFloat(usdcScale)).Float64()
	if math.IsNaN(out) || math.IsInf(out, 0) {
		return 0
	}
	return out
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
