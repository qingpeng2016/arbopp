package watchgroup

import (
	"encoding/json"
)

// BooksJSONForOutcomePair 调试/兼容：按 side_no 记录两平台盘口（每簿至多 8 档，单行 JSON）。
func (w *Watch) BooksJSONForOutcomePair(outcomeIndexA, outcomeIndexB int) string {
	if w == nil {
		return `{"outcomes":[]}`
	}
	var outcomes []outcomeBooks
	appendOutcome := func(sideNo, outcomeIndex int, buyYes bool) {
		ob := w.collectOutcomeBooks(outcomeIndex, 8, buyYes)
		if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
			return
		}
		ob.SideNo = sideNo
		outcomes = append(outcomes, ob)
	}
	appendOutcome(1, outcomeIndexA, true)
	if outcomeIndexB >= 0 && outcomeIndexB != outcomeIndexA {
		appendOutcome(2, outcomeIndexB, false)
	}
	raw, err := json.Marshal(map[string]any{"outcomes": outcomes})
	if err != nil || len(raw) == 0 {
		return `{"outcomes":[]}`
	}
	return string(raw)
}