package common

import (
	"encoding/json"
	"strings"

	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

type outcomeTokenLabels struct {
	YesLabel string `json:"yes_label"`
	NoLabel  string `json:"no_label"`
}

// EnrichOrderBookFromOutcome 写父场次名、子盘口名、token 名及联赛等冗余列。
func EnrichOrderBookFromOutcome(row *persistententity.OrderBook, oc *persistententity.MarketOutcome, fx *persistententity.Market, tokenID string) {
	if row == nil || oc == nil {
		return
	}
	row.OutcomeName = cloneStrPtr(oc.OutcomeName)
	if mn := parentMarketName(oc, fx); mn != nil {
		row.MarketName = mn
	}
	row.TokenName = TokenNameForOutcome(oc, tokenID)
	ApplyOrderBookFixtureMeta(row, oc, fx)
}

func parentMarketName(oc *persistententity.MarketOutcome, fx *persistententity.Market) *string {
	if oc != nil && oc.MarketName != nil && strings.TrimSpace(*oc.MarketName) != "" {
		return cloneStrPtr(oc.MarketName)
	}
	if fx != nil && fx.Name != nil && strings.TrimSpace(*fx.Name) != "" {
		return cloneStrPtr(fx.Name)
	}
	return nil
}

// TokenNameForOutcome 按 token 匹配 yes/no；config 可含 yes_label/no_label。
func TokenNameForOutcome(oc *persistententity.MarketOutcome, tokenID string) *string {
	if oc == nil {
		return nil
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil
	}
	yes := strings.TrimSpace(oc.YesTokenID)
	no := strings.TrimSpace(oc.NoTokenID)
	labels := tokenLabelsFromConfig(oc.Config)
	if tokenID == yes {
		if labels != nil && strings.TrimSpace(labels.YesLabel) != "" {
			return strPtr255(labels.YesLabel)
		}
		if yes != "" {
			return strPtr255("Yes")
		}
	}
	if tokenID == no {
		if labels != nil && strings.TrimSpace(labels.NoLabel) != "" {
			return strPtr255(labels.NoLabel)
		}
		if no != "" {
			return strPtr255("No")
		}
	}
	// Predict 合成 NO 簿：token 为 "{market_id}:no"
	if strings.HasSuffix(tokenID, ":no") && no != "" && tokenID == no+":no" {
		if labels != nil && strings.TrimSpace(labels.NoLabel) != "" {
			return strPtr255(labels.NoLabel)
		}
		return strPtr255("No")
	}
	return nil
}

func tokenLabelsFromConfig(cfg *string) *outcomeTokenLabels {
	if cfg == nil {
		return nil
	}
	raw := strings.TrimSpace(*cfg)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return nil
	}
	var labels outcomeTokenLabels
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return nil
	}
	if labels.YesLabel == "" && labels.NoLabel == "" {
		return nil
	}
	return &labels
}

func cloneStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func strPtr255(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if len(s) > 255 {
		s = s[:255]
	}
	return &s
}