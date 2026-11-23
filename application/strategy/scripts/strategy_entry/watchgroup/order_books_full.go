package watchgroup

import (
	"strings"
	"time"

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
	QuoteVersion            int         `json:"quote_version,omitempty"`
	QuoteAtMs               int64       `json:"quote_at_ms,omitempty"`
	NetCost                 float64     `json:"net_cost,omitempty"`
	NetQty                  float64     `json:"net_qty,omitempty"`
	FeeRate                 *float64    `json:"fee_rate,omitempty"`
}

// LegPersistSnap 落库用：全量盘口 JSON 与顶卖一同源。
type LegPersistSnap struct {
	OrderBooks string
	TopAsk     float64
	TopSz      float64
	OK         bool
}

// MinLegTopAskQty 各腿卖一挂单量的最小值（套利可成交份数上限；无有效深度则 0）。
func MinLegTopAskQty(legs ...LegPersistSnap) float64 {
	var min float64
	var have bool
	for _, leg := range legs {
		if !leg.OK || leg.TopSz <= 0 {
			continue
		}
		if !have || leg.TopSz < min {
			min = leg.TopSz
			have = true
		}
	}
	if !have {
		return 0
	}
	return min
}

// LegSnapshot 单腿盘口快照（JSON + 卖一价同一次读取）。
func (w *Watch) LegSnapshot(outcomeIndex int, buyYes bool) LegPersistSnap {
	if w == nil {
		return LegPersistSnap{OrderBooks: "{}\n"}
	}
	ob := w.collectOutcomeBooks(outcomeIndex, 0, buyYes)
	topPx, topSz, ok := topAskFromOutcomeBooks(ob)
	json := prettyJSON(fullOutcomeBooks{
		Platform:                ob.Platform,
		PlatformMarketOutcomeID: ob.PlatformMarketOutcomeID,
		Books:                   ob.Books,
	})
	if ob.Platform == "" && len(ob.Books) == 0 {
		json = prettyJSON(fullOutcomeBooks{})
	}
	return LegPersistSnap{OrderBooks: json, TopAsk: topPx, TopSz: topSz, OK: ok}
}

// FullOrderBooksJSON 单腿全量盘口（不截断档位）；buyYes 表示本条套利腿买的是 YES 还是 NO。
func (w *Watch) FullOrderBooksJSON(outcomeIndex int, buyYes bool) string {
	return w.LegSnapshot(outcomeIndex, buyYes).OrderBooks
}

func topAskFromOutcomeBooks(ob outcomeBooks) (price, size float64, ok bool) {
	for _, bk := range ob.Books {
		if len(bk.Asks) == 0 {
			continue
		}
		p, s := bk.Asks[0].Price, bk.Asks[0].Size
		if book.ValidPrice(p) && s > 0 {
			return p, s, true
		}
	}
	return 0, 0, false
}

// LegQuote captures the selected side once for pricing and opportunity evidence.
type LegQuote struct {
	platform, market, key, role string
	book                        book.Book
}

func (q LegQuote) Top() (price, size float64, ok bool) {
	if !q.book.Live || len(q.book.Asks) == 0 {
		return 0, 0, false
	}
	ask := q.book.Asks[0]
	return ask.Price, ask.Size, book.ValidPrice(ask.Price) && ask.Size > 0
}

func (q LegQuote) UpdatedAt() time.Time { return q.book.ExchangeTS }

func (q LegQuote) Persist(cost, qty, feeRate float64) LegPersistSnap {
	px, size, ok := q.Top()
	if !ok {
		return LegPersistSnap{}
	}
	return LegPersistSnap{
		TopAsk: px, TopSz: size, OK: true,
		OrderBooks: prettyJSON(fullOutcomeBooks{
			Platform: q.platform, PlatformMarketOutcomeID: q.market,
			Books:        []bookSlice{{BookKey: q.key, Role: q.role, Bids: q.book.Bids, Asks: q.book.Asks}},
			QuoteVersion: 2, QuoteAtMs: q.book.ExchangeTS.UnixMilli(),
			NetCost: cost, NetQty: qty, FeeRate: &feeRate,
		}),
	}
}

func (w *Watch) Quote(view book.Snapshot, outcomeIndex int, buyYes bool) LegQuote {
	return w.readQuote(view.Get, outcomeIndex, buyYes)
}

func (w *Watch) collectOutcomeBooks(outcomeIndex, depth int, buyYes bool) outcomeBooks {
	if w.Books == nil {
		return outcomeBooks{}
	}
	q := w.readQuote(w.Books.Get, outcomeIndex, buyYes)
	ob := outcomeBooks{Platform: q.platform, PlatformMarketOutcomeID: q.market}
	if q.book.ID != "" {
		ob.Books = []bookSlice{{BookKey: q.key, Role: q.role,
			Bids: sliceLevels(q.book.Bids, depth), Asks: sliceLevels(q.book.Asks, depth)}}
	}
	return ob
}

func (w *Watch) readQuote(get func(string, string) (book.Book, bool), index int, buyYes bool) LegQuote {
	oc := w.OutcomeAt(index)
	if oc == nil {
		return LegQuote{}
	}
	q := LegQuote{platform: strings.ToUpper(strings.TrimSpace(oc.Platform)),
		market: strings.TrimSpace(oc.PlatformMarketOutcomeID)}
	switch q.platform {
	case appcommon.PlatformPolymarket:
		q.key, q.role = strings.TrimSpace(oc.YesTokenID), "YES_TOKEN"
		if !buyYes {
			q.key, q.role = strings.TrimSpace(oc.NoTokenID), "NO_TOKEN"
		}
	case appcommon.PlatformPredict, appcommon.PlatformKalshi, appcommon.PlatformSxbet:
		q.key, q.role = q.market, "YES_BOOK"
		if !buyYes {
			q.key, q.role = strings.TrimSpace(oc.NoTokenID), "NO_TOKEN"
			if q.key == "" {
				q.key = q.market + ":no"
			}
			if q.platform == appcommon.PlatformSxbet {
				q.key, q.role = q.market+":not", "NO_BOOK"
			}
		}
	default:
		return q
	}
	if q.key == "" {
		return q
	}
	var found bool
	q.book, found = get(q.platform, q.key)
	// Historical snapshots for Predict and Kalshi can contain only the YES book.
	if !found && !buyYes && q.market != "" &&
		(q.platform == appcommon.PlatformPredict || q.platform == appcommon.PlatformKalshi) {
		if yes, ok := get(q.platform, q.market); ok {
			yes.Bids, yes.Asks = book.NoBookFromYesBook(yes.Bids, yes.Asks)
			q.book = yes
		}
	}
	return q
}

func sliceLevels(ls []book.Level, n int) []book.Level {
	if n <= 0 || len(ls) <= n {
		return ls
	}
	return ls[:n]
}
