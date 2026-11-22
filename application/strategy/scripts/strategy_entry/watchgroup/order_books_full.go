package watchgroup

import (
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

type fullOutcomeBooks struct {
	Platform                string      `json:"platform"`
	PlatformMarketOutcomeID string      `json:"platform_market_outcome_id"`
	Books                   []bookSlice `json:"books"`
}

// FullOrderBooksJSON 单腿全量盘口（不截断档位）；buyYes 表示本条套利腿买的是 YES 还是 NO。
func (w *Watch) FullOrderBooksJSON(outcomeIndex int, buyYes bool) string {
	if w == nil {
		return "{}\n"
	}
	ob := w.collectOutcomeBooks(outcomeIndex, 0, buyYes)
	if ob.Platform == "" && len(ob.Books) == 0 {
		return prettyJSON(fullOutcomeBooks{})
	}
	out := fullOutcomeBooks{
		Platform:                ob.Platform,
		PlatformMarketOutcomeID: ob.PlatformMarketOutcomeID,
		Books:                   ob.Books,
	}
	return prettyJSON(out)
}

func (w *Watch) collectOutcomeBooks(outcomeIndex, depth int, buyYes bool) outcomeBooks {
	var ob outcomeBooks
	if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
		return ob
	}
	oc := &w.Outcomes[outcomeIndex]
	ob = outcomeBooks{
		Platform:                strings.ToUpper(strings.TrimSpace(oc.Platform)),
		PlatformMarketOutcomeID: strings.TrimSpace(oc.PlatformMarketOutcomeID),
	}
	limit := depth
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
			Bids:    sliceLevels(b.Bids, limit),
			Asks:    sliceLevels(b.Asks, limit),
		})
	}
	switch strings.ToUpper(strings.TrimSpace(oc.Platform)) {
	case appcommon.PlatformPolymarket:
		if buyYes {
			appendBook(appcommon.PlatformPolymarket, oc.YesTokenID, "YES_TOKEN")
		} else {
			appendBook(appcommon.PlatformPolymarket, oc.NoTokenID, "NO_TOKEN")
		}
	case appcommon.PlatformKalshi:
		mid := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		noKey := strings.TrimSpace(oc.NoTokenID)
		if noKey == "" {
			noKey = mid + ":no"
		}
		if buyYes {
			appendBook(appcommon.PlatformKalshi, mid, "YES_BOOK")
		} else {
			appendBook(appcommon.PlatformKalshi, noKey, "NO_TOKEN")
		}
	case appcommon.PlatformSxbet:
		mid := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		if buyYes {
			appendBook(oc.Platform, mid, "YES_BOOK")
		} else {
			appendBook(oc.Platform, mid+":not", "NO_BOOK")
		}
	case appcommon.PlatformPredict:
		mid := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		noKey := strings.TrimSpace(oc.NoTokenID)
		if noKey == "" {
			noKey = mid + ":no"
		}
		if buyYes {
			appendBook(oc.Platform, mid, "YES_BOOK")
		} else {
			appendBook(oc.Platform, noKey, "NO_TOKEN")
		}
	}
	return ob
}

func sliceLevels(ls []book.Level, n int) []book.Level {
	if n <= 0 || len(ls) <= n {
		return ls
	}
	return ls[:n]
}