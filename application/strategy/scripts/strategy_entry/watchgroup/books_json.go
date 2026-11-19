package watchgroup

import (
	"encoding/json"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
)

type bookSlice struct {
	BookKey string       `json:"book_key"`
	Role    string       `json:"role"`
	Bids    []book.Level `json:"bids,omitempty"`
	Asks    []book.Level `json:"asks,omitempty"`
}

type outcomeBooks struct {
	SideNo                  int         `json:"side_no"`
	Platform                string      `json:"platform"`
	PlatformMarketOutcomeID string      `json:"platform_market_outcome_id"`
	Books                   []bookSlice `json:"books"`
}

// BooksJSONForOutcomePair 机会表 order_books_json：按 side_no 记录两平台盘口（Poly 含 YES/NO token 各一本）。
func (w *Watch) BooksJSONForOutcomePair(outcomeIndexA, outcomeIndexB int) string {
	if w == nil {
		return `{"outcomes":[]}`
	}
	var outcomes []outcomeBooks
	appendOutcome := func(sideNo, outcomeIndex int) {
		if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
			return
		}
		oc := &w.Outcomes[outcomeIndex]
		ob := outcomeBooks{
			SideNo:                  sideNo,
			Platform:                strings.ToUpper(strings.TrimSpace(oc.Platform)),
			PlatformMarketOutcomeID: strings.TrimSpace(oc.PlatformMarketOutcomeID),
		}
		appendBook := func(platform, bookKey, role string) {
			bookKey = strings.TrimSpace(bookKey)
			if bookKey == "" || w.Books == nil {
				return
			}
			b, ok := w.Books.Get(platform, bookKey)
			if !ok {
				return
			}
			ob.Books = append(ob.Books, bookSlice{
				BookKey: bookKey,
				Role:    role,
				Bids:    topLevels(b.Bids, 8),
				Asks:    topLevels(b.Asks, 8),
			})
		}
		switch strings.ToUpper(strings.TrimSpace(oc.Platform)) {
		case appcommon.PlatformPolymarket:
			appendBook(appcommon.PlatformPolymarket, oc.YesTokenID, "YES_TOKEN")
			appendBook(appcommon.PlatformPolymarket, oc.NoTokenID, "NO_TOKEN")
		case appcommon.PlatformKalshi:
			appendBook(appcommon.PlatformKalshi, oc.PlatformMarketOutcomeID, "YES_BOOK")
		case appcommon.PlatformPredict:
			mid := strings.TrimSpace(oc.PlatformMarketOutcomeID)
			appendBook(appcommon.PlatformPredict, mid, "YES_BOOK")
			if yes, ok := w.Books.Get(appcommon.PlatformPredict, mid); ok {
				noBids, noAsks := book.NoBookFromYesBook(yes.Bids, yes.Asks)
				noKey := strings.TrimSpace(oc.NoTokenID)
				if noKey == "" {
					noKey = mid + ":no"
				}
				ob.Books = append(ob.Books, bookSlice{
					BookKey: noKey,
					Role:    "NO_TOKEN",
					Bids:    topLevels(noBids, 8),
					Asks:    topLevels(noAsks, 8),
				})
			}
		}
		outcomes = append(outcomes, ob)
	}
	appendOutcome(1, outcomeIndexA)
	if outcomeIndexB >= 0 && outcomeIndexB != outcomeIndexA {
		appendOutcome(2, outcomeIndexB)
	}
	raw, err := json.Marshal(map[string]any{"outcomes": outcomes})
	if err != nil || len(raw) == 0 {
		return `{"outcomes":[]}`
	}
	return string(raw)
}

func topLevels(ls []book.Level, n int) []book.Level {
	if len(ls) > n {
		return ls[:n]
	}
	return ls
}