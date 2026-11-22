package watchgroup

import (
	"bytes"
	"encoding/json"
	"strings"

	obcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/common"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// YesLegInfo 机会表 yes_info 字段（键顺序固定）。
type YesLegInfo struct {
	YesMarketID    string `json:"yes_market_id"`
	YesMarketName  string `json:"yes_market_name"`
	YesOutcomeID   uint   `json:"yes_outcome_id"`
	YesOutcomeName string `json:"yes_outcome_name"`
	YesTokenID     string `json:"yes_token_id"`
	YesTokenName   string `json:"yes_token_name"`
}

// DrawLegInfo 1X2 平局腿 draw_info 字段（键顺序固定）；二元机会为空 JSON。
type DrawLegInfo struct {
	DrawMarketID    string `json:"draw_market_id"`
	DrawMarketName  string `json:"draw_market_name"`
	DrawOutcomeID   uint   `json:"draw_outcome_id"`
	DrawOutcomeName string `json:"draw_outcome_name"`
	DrawTokenID     string `json:"draw_token_id"`
	DrawTokenName   string `json:"draw_token_name"`
}

// NoLegInfo 机会表 no_info 字段（键顺序固定）。
type NoLegInfo struct {
	NoMarketID    string `json:"no_market_id"`
	NoMarketName  string `json:"no_market_name"`
	NoOutcomeID   uint   `json:"no_outcome_id"`
	NoOutcomeName string `json:"no_outcome_name"`
	NoTokenID     string `json:"no_token_id"`
	NoTokenName   string `json:"no_token_name"`
}

// YesLegInfoJSON 由内存 outcome 与成交方向生成缩进 JSON。
func (w *Watch) YesLegInfoJSON(outcomeIndex int, buyYes bool) string {
	oc := w.OutcomeAt(outcomeIndex)
	if oc == nil {
		return prettyJSON(YesLegInfo{})
	}
	info := YesLegInfo{
		YesMarketID:    strings.TrimSpace(oc.PlatformMarketID),
		YesMarketName:  marketNameForOutcome(w, oc),
		YesOutcomeID:   oc.ID,
		YesOutcomeName: derefStr(oc.OutcomeName),
		YesTokenID:     legTokenID(oc, buyYes),
		YesTokenName:   derefStr(obcommon.TokenNameForOutcome(oc, legTokenID(oc, buyYes))),
	}
	return prettyJSON(info)
}

// DrawLegInfoJSON 由内存 outcome 与成交方向生成缩进 JSON。
func (w *Watch) DrawLegInfoJSON(outcomeIndex int, buyYes bool) string {
	oc := w.OutcomeAt(outcomeIndex)
	if oc == nil {
		return prettyJSON(DrawLegInfo{})
	}
	info := DrawLegInfo{
		DrawMarketID:    strings.TrimSpace(oc.PlatformMarketID),
		DrawMarketName:  marketNameForOutcome(w, oc),
		DrawOutcomeID:   oc.ID,
		DrawOutcomeName: derefStr(oc.OutcomeName),
		DrawTokenID:     legTokenID(oc, buyYes),
		DrawTokenName:   derefStr(obcommon.TokenNameForOutcome(oc, legTokenID(oc, buyYes))),
	}
	return prettyJSON(info)
}

// NoLegInfoJSON 由内存 outcome 与成交方向生成缩进 JSON。
func (w *Watch) NoLegInfoJSON(outcomeIndex int, buyYes bool) string {
	oc := w.OutcomeAt(outcomeIndex)
	if oc == nil {
		return prettyJSON(NoLegInfo{})
	}
	info := NoLegInfo{
		NoMarketID:    strings.TrimSpace(oc.PlatformMarketID),
		NoMarketName:  marketNameForOutcome(w, oc),
		NoOutcomeID:   oc.ID,
		NoOutcomeName: derefStr(oc.OutcomeName),
		NoTokenID:     legTokenID(oc, buyYes),
		NoTokenName:   derefStr(obcommon.TokenNameForOutcome(oc, legTokenID(oc, buyYes))),
	}
	return prettyJSON(info)
}

func marketNameForOutcome(w *Watch, oc *persistententity.MarketOutcome) string {
	if oc == nil {
		return ""
	}
	if oc.MarketName != nil && strings.TrimSpace(*oc.MarketName) != "" {
		return strings.TrimSpace(*oc.MarketName)
	}
	if w != nil && w.MarketsByRowID != nil {
		if fx, ok := w.MarketsByRowID[oc.MarketRowID]; ok && fx.Name != nil {
			return strings.TrimSpace(*fx.Name)
		}
	}
	return ""
}

func legTokenID(oc *persistententity.MarketOutcome, buyYes bool) string {
	if oc == nil {
		return ""
	}
	if buyYes {
		return strings.TrimSpace(oc.YesTokenID)
	}
	no := strings.TrimSpace(oc.NoTokenID)
	if no != "" {
		return no
	}
	// Predict 等 YES 簿上 NO 腿：合成 token 与订单逻辑一致
	return strings.TrimSpace(oc.PlatformMarketOutcomeID) + ":no"
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func prettyJSON(v any) string {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil || len(raw) == 0 {
		return "{}\n"
	}
	return string(raw) + "\n"
}

// PrettyJSONObject 任意对象缩进 JSON（末尾换行）。
func PrettyJSONObject(v any) string {
	return prettyJSON(v)
}

// MarshalOrdered is used in tests to validate stable key order in structs.
func MarshalOrdered(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}