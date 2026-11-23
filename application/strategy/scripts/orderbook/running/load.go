package running

import (
	"context"
	"encoding/json"
	"strings"

	entrystat "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/stat"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Snapshot stat_kv map_key_audit / market_running_count_strategy。
type Snapshot struct {
	Fingerprint string
	Groups      []entrystat.Group
	Fixtures    map[uint]persistententity.Market
	Outcomes    []persistententity.MarketOutcome
}

// LoadFromStatKV 读 description JSON，加载 strategy_entry 在跑场次及其盘口。
// 优先用 groups[].markets 中的 market.id（与 entry 心跳一致，含 SX 已 CLOSE 仍在连 WS 的腿）；
// 无 markets 时回退为 map_key + status=ACTIVE。
func LoadFromStatKV(
	ctx context.Context,
	statRepo persistentrepo.StatKVRepo,
	marketRepo persistentrepo.MarketRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
) (Snapshot, error) {
	var out Snapshot
	if statRepo == nil || marketRepo == nil || outcomeRepo == nil {
		return out, nil
	}
	row, err := statRepo.FindOne(ctx, nil, entrystat.GroupMapKeyAudit(), entrystat.ItemKeyMarketRunningCountStrategy())
	if err != nil {
		return out, err
	}
	desc := strings.TrimSpace(row.Description)
	out.Fingerprint = desc
	groups, err := parseGroups(desc)
	if err != nil || len(groups) == 0 {
		return out, err
	}
	out.Groups = groups

	fixtures := map[uint]persistententity.Market{}
	seenID := map[uint]struct{}{}
	var marketRowIDs []uint
	addMarket := func(m persistententity.Market) {
		if _, ok := seenID[m.ID]; ok {
			return
		}
		seenID[m.ID] = struct{}{}
		fixtures[m.ID] = m
		marketRowIDs = append(marketRowIDs, m.ID)
	}

	fromStatMarkets := marketIDsFromGroups(groups)
	if len(fromStatMarkets) > 0 {
		_, markets, err := marketRepo.List(ctx, nil, map[string]interface{}{
			"id IN ?": fromStatMarkets,
		}, "id asc", "")
		if err != nil {
			return out, err
		}
		for _, m := range markets {
			addMarket(m)
		}
	}

	seenMK := map[string]struct{}{}
	for _, g := range groups {
		mk := strings.TrimSpace(g.MapKey)
		if mk == "" {
			continue
		}
		if _, dup := seenMK[mk]; dup {
			continue
		}
		seenMK[mk] = struct{}{}
		_, markets, err := marketRepo.List(ctx, nil, map[string]interface{}{
			"map_key = ?": mk,
			"status = ?":  "ACTIVE",
		}, "id asc", "")
		if err != nil {
			return out, err
		}
		for _, m := range markets {
			addMarket(m)
		}
	}
	if len(marketRowIDs) == 0 {
		return out, nil
	}
	_, outcomes, err := outcomeRepo.List(ctx, nil, map[string]interface{}{
		"market_row_id IN ?": marketRowIDs,
	}, "platform asc, id asc", "")
	if err != nil {
		return out, err
	}
	out.Fixtures = fixtures
	out.Outcomes = outcomes
	return out, nil
}

func marketIDsFromGroups(groups []entrystat.Group) []uint {
	seen := map[uint]struct{}{}
	var ids []uint
	for _, g := range groups {
		for _, id := range g.Markets {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func parseGroups(description string) ([]entrystat.Group, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, nil
	}
	var payload struct {
		Groups  []entrystat.Group `json:"groups"`
		MapKeys []string          `json:"map_keys"`
	}
	if err := json.Unmarshal([]byte(description), &payload); err != nil {
		return nil, err
	}
	if len(payload.Groups) > 0 {
		return payload.Groups, nil
	}
	out := make([]entrystat.Group, 0, len(payload.MapKeys))
	for _, mk := range payload.MapKeys {
		mk = strings.TrimSpace(mk)
		if mk == "" {
			continue
		}
		out = append(out, entrystat.Group{MapKey: mk})
	}
	return out, nil
}