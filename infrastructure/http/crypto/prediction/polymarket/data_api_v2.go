package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	"sort"
	"strconv"
	"strings"
	"time"
)

// dataAPIBaseURL is overridable by package tests so all Data API calls can use httptest.
var dataAPIBaseURL = "https://data-api.polymarket.com"

const (
	dataAPIDefaultLimit  = 1000
	dataAPIMaxConditions = 20
)

type dataAPIPage[T any] struct {
	Data       []T `json:"data"`
	Pagination *struct {
		HasMore    bool   `json:"has_more"`
		NextCursor string `json:"next_cursor"`
	} `json:"pagination"`
}

type positionV2 struct {
	ProxyWallet        string  `json:"proxy_wallet"`
	TokenID            string  `json:"token_id"`
	ConditionID        string  `json:"condition_id"`
	Size               float64 `json:"current_size"`
	AvgPrice           float64 `json:"avg_price"`
	EntryCost          float64 `json:"entry_cost_usdc"`
	CurrentValue       float64 `json:"current_value"`
	UnrealizedPnL      float64 `json:"unrealized_pnl"`
	PercentPnL         float64 `json:"percent_pnl"`
	TotalSize          float64 `json:"total_size"`
	RealizedPnL        float64 `json:"realized_pnl"`
	PercentRealizedPnL float64 `json:"percent_realized_pnl"`
	CurrentPrice       float64 `json:"current_price"`
	Redeemable         bool    `json:"redeemable"`
	Mergeable          bool    `json:"mergeable"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Icon               string  `json:"icon"`
	EventSlug          string  `json:"event_slug"`
	Outcome            string  `json:"outcome"`
	OutcomeIndex       int     `json:"outcome_index"`
	OppositeOutcome    string  `json:"opposite_outcome"`
	OppositeTokenID    string  `json:"opposite_token_id"`
	EndDate            string  `json:"end_date"`
	NegativeRisk       bool    `json:"negative_risk"`
}

type activityV2 struct {
	ProxyWallet     string  `json:"proxy_wallet"`
	Timestamp       int64   `json:"timestamp"`
	ConditionID     string  `json:"condition_id"`
	Type            string  `json:"type"`
	Size            float64 `json:"size"`
	USDCSize        float64 `json:"usdc_size"`
	TransactionHash string  `json:"transaction_hash"`
	Price           float64 `json:"price"`
	TokenID         string  `json:"token_id"`
	Side            string  `json:"side"`
	OutcomeIndex    int     `json:"outcome_index"`
	Outcome         string  `json:"outcome"`
}

// GetPositionsV2 reads all positions from the production Data API v2 endpoint.
func (p *Polymarket) GetPositionsV2(ctx context.Context, proxyAddress, conditionID string) (*entity.GetMagicAccountPositionsResponse, error) {
	if strings.TrimSpace(proxyAddress) == "" {
		return nil, fmt.Errorf("proxy address is required")
	}
	params := map[string][]string{"user": {proxyAddress}, "limit": {strconv.Itoa(dataAPIDefaultLimit)}}
	if conditionID != "" {
		params["condition"] = []string{conditionID}
	}
	items, err := getAllPages[positionV2](ctx, p, dataAPIBaseURL+"/v2/positions", params, nil, "GetPositionsV2")
	if err != nil {
		return nil, err
	}
	positions := make([]entity.Position, 0, len(items))
	for _, pos := range items {
		positions = append(positions, entity.Position{
			TokenID: pos.TokenID, AssetID: pos.TokenID, Market: pos.ConditionID,
			Outcome: pos.Outcome, Balance: strconv.FormatFloat(pos.Size, 'f', 6, 64),
			BalanceFloat: pos.Size, MarketName: pos.Title, OutcomeName: pos.Outcome,
			CurrentPrice: strconv.FormatFloat(pos.CurrentPrice, 'f', 6, 64),
			ValueUSD:     strconv.FormatFloat(pos.CurrentValue, 'f', 6, 64),
			PnlUSD:       strconv.FormatFloat(pos.UnrealizedPnL, 'f', 6, 64), LastUpdated: pos.EndDate,
			ProxyWallet: pos.ProxyWallet, ConditionID: pos.ConditionID, AvgPrice: pos.AvgPrice,
			InitialValue: pos.EntryCost, CurrentValue: pos.CurrentValue, CashPnl: pos.UnrealizedPnL,
			PercentPnl: pos.PercentPnL, TotalBought: pos.TotalSize, RealizedPnl: pos.RealizedPnL,
			PercentRealizedPnl: pos.PercentRealizedPnL, Redeemable: pos.Redeemable, Mergeable: pos.Mergeable,
			Slug: pos.Slug, Icon: pos.Icon, EventSlug: pos.EventSlug, OutcomeIndex: pos.OutcomeIndex,
			OppositeOutcome: pos.OppositeOutcome, OppositeAsset: pos.OppositeTokenID,
			EndDate: pos.EndDate, NegativeRisk: pos.NegativeRisk,
		})
	}
	return &entity.GetMagicAccountPositionsResponse{
		Positions: positions, Total: len(positions), Success: true,
		Message: fmt.Sprintf("successfully fetched %d positions", len(positions)),
	}, nil
}

// GetTradeHistoryV3V2 reads /v2/activity, following cursors and splitting condition filters
// into groups accepted by the upstream API. Condition IDs are passed through unchanged.
func (p *Polymarket) GetTradeHistoryV3V2(ctx context.Context, req entity.GetTradeHistoryRequest, account entity.PolymarketAccount) (*entity.GetTradeHistoryResponse, error) {
	user := account.ProxyAddress
	if user == "" {
		user = account.EOAAddress
	}
	if strings.TrimSpace(user) == "" {
		return nil, fmt.Errorf("proxy_address or eoa_address is required for activity history")
	}
	limit := dataAPIDefaultLimit
	if req.Limit != nil {
		if *req.Limit < 1 || *req.Limit > dataAPIDefaultLimit {
			return nil, fmt.Errorf("activity limit must be between 1 and %d", dataAPIDefaultLimit)
		}
		limit = *req.Limit
	}
	base := map[string][]string{
		"user": {user}, "limit": {strconv.Itoa(limit)},
		"excludeDepositsWithdrawals": {"false"}, "sortBy": {"TIMESTAMP"}, "sortDirection": {"DESC"},
	}
	if req.After != nil {
		base["start"] = []string{strconv.FormatInt(*req.After, 10)}
	}
	if req.Before != nil {
		base["end"] = []string{strconv.FormatInt(*req.Before, 10)}
	}
	if req.Type != nil && *req.Type != "" {
		base["type"] = []string{*req.Type}
	}
	conditions := req.Markets
	if len(conditions) == 0 && req.Market != nil && *req.Market != "" {
		conditions = []string{*req.Market}
	}
	conditions = uniqueNonEmpty(conditions)
	conditionBatches := [][]string{{}}
	if len(conditions) > 0 {
		conditionBatches = nil
		for i := 0; i < len(conditions); i += dataAPIMaxConditions {
			end := i + dataAPIMaxConditions
			if end > len(conditions) {
				end = len(conditions)
			}
			conditionBatches = append(conditionBatches, conditions[i:end])
		}
	}
	activities := make([]activityV2, 0)
	for _, conditions := range conditionBatches {
		params := cloneValues(base)
		if len(conditions) > 0 {
			params["condition"] = []string{strings.Join(conditions, ",")}
		}
		batch, err := getAllPages[activityV2](ctx, p, dataAPIBaseURL+"/v2/activity", params, map[string]string{"accept": "application/json, text/plain, */*"}, "GetTradeHistoryV3V2")
		if err != nil {
			return nil, err
		}
		activities = append(activities, batch...)
	}
	sort.SliceStable(activities, func(i, j int) bool { return activities[i].Timestamp > activities[j].Timestamp })
	trades := make([]entity.RawTradeRecord, 0, len(activities))
	for _, a := range activities {
		ts := ""
		if a.Timestamp > 0 {
			ts = strconv.FormatInt(a.Timestamp, 10)
		}
		id := a.TransactionHash
		if id == "" {
			id = fmt.Sprintf("%s-%s-%d", a.ConditionID, a.TokenID, a.Timestamp)
		}
		trades = append(trades, entity.RawTradeRecord{
			ID: id, Market: a.ConditionID, AssetId: a.TokenID, Side: a.Side,
			Size: strconv.FormatFloat(a.Size, 'f', -1, 64), UsdcSize: strconv.FormatFloat(a.USDCSize, 'f', 6, 64),
			FeeRateBps: "0", Price: strconv.FormatFloat(a.Price, 'f', -1, 64), Status: a.Type,
			MatchTime: ts, LastUpdate: ts, Outcome: a.Outcome, MakerAddress: a.ProxyWallet,
			Owner: a.ProxyWallet, TransactionHash: a.TransactionHash, BucketIndex: a.OutcomeIndex, Type: a.Type,
		})
	}
	return &entity.GetTradeHistoryResponse{Trades: trades, Total: len(trades), Success: true, Message: fmt.Sprintf("successfully fetched %d activity records", len(trades))}, nil
}

func getAllPages[T any](ctx context.Context, p *Polymarket, endpoint string, params map[string][]string, headers map[string]string, operation string) ([]T, error) {
	items := make([]T, 0)
	cursor := ""
	seen := map[string]struct{}{}
	for {
		pageParams := cloneValues(params)
		if cursor != "" {
			pageParams["cursor"] = []string{cursor}
		}
		started := time.Now()
		resp, err := p.client.Get(ctx, endpoint, pageParams, cloneHeaders(headers))
		p.checkTimeoutAndAlert(ctx, started, operation)
		if err != nil {
			return nil, fmt.Errorf("%s request failed: %w", operation, err)
		}
		var page dataAPIPage[T]
		if err := json.Unmarshal(resp.Body(), &page); err != nil {
			return nil, fmt.Errorf("%s response decode failed: %w", operation, err)
		}
		items = append(items, page.Data...)
		if page.Pagination == nil || !page.Pagination.HasMore || page.Pagination.NextCursor == "" {
			return items, nil
		}
		if _, ok := seen[page.Pagination.NextCursor]; ok {
			return nil, fmt.Errorf("%s pagination cursor repeated: %q", operation, page.Pagination.NextCursor)
		}
		seen[page.Pagination.NextCursor] = struct{}{}
		cursor = page.Pagination.NextCursor
	}
}

func cloneValues(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func cloneHeaders(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
