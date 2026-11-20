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

// LoadFromStatKV 读 description JSON；按各 group.map_key 加载 ACTIVE 场次及其盘口（盘口无 status，随 market）。
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
	seenMK := map[string]struct{}{}
	var marketRowIDs []uint
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
			fixtures[m.ID] = m
			marketRowIDs = append(marketRowIDs, m.ID)
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

func parseGroups(description string) ([]entrystat.Group, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, nil
	}
	var payload struct {
		Groups []entrystat.Group `json:"groups"`
	}
	if err := json.Unmarshal([]byte(description), &payload); err != nil {
		return nil, err
	}
	return payload.Groups, nil
}