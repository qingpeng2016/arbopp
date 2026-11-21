package strategyentry

import (
	"context"
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	obcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/orderbook/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/broker"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/onextwo"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	entryws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws"
	kalshiws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/kalshi"
	polyws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/polymarket"
	predictws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/predict"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

var errBusy = errors.New("strategy_entry event queue full")

// WS 落库折中：不写每笔 price_change，同 token 限速，Redis 队列有上限（见 wsbookqueue）。
const wsBookPersistMinInterval = 1 * time.Second

type event struct {
	Gen      int
	Platform string
	Kind     string
	ID       string
	Bids     []book.Level
	Asks     []book.Level
	NoBids   []book.Level
	Side     string
	Price    float64
	Size     float64
	Delta    float64
	Seq      int64
	Tick     float64
	MinSize  float64
	TS       time.Time
	Changes  []book.PolyChange
	RecvAt   time.Time // WS 回调收到消息的本地时间
}

type connState struct {
	gen    int
	cancel context.CancelFunc
}

type watch struct {
	*watchgroup.Watch
	entry    *Entry
	ev       chan event
	mu       sync.Mutex
	conn     map[string]*connState
	yesno    *yesno.Runner
	onextwo  *onextwo.Runner
	negRisk  []bool
	wsHealth       *wsHealth
	wsPersistMu    sync.Mutex
	wsPersistLast  map[string]time.Time
}

// newWatch 为一组 map_key 建内存盘口，并记下 token、ticker 对应哪条结果。
func newWatch(ctx context.Context, e *Entry, g group) *watch {
	marketsByRow := map[uint]persistententity.Market{}
	for id, m := range g.MarketsByRowID {
		marketsByRow[id] = m
	}
	wg := &watchgroup.Watch{
		Ctx:            ctx,
		MapKey:         g.MapKey,
		Outcomes:       append([]persistententity.MarketOutcome(nil), g.Outcomes...),
		MarketsByRowID: marketsByRow,
		ByToken:        map[string]int{},
		ByTicker:       map[string]int{},
		ByPredict:      map[string]int{},
		BySxbet:        map[string]int{},
		Books:          book.NewStore(),
	}
	w := &watch{
		Watch:   wg,
		entry:   e,
		ev:      make(chan event, 2048),
		conn:    map[string]*connState{},
		negRisk: make([]bool, len(g.Outcomes)),
	}
	for i := range w.Outcomes {
		oc := &w.Outcomes[i]
		if strings.EqualFold(oc.Platform, appcommon.PlatformPolymarket) {
			if id := strings.TrimSpace(oc.YesTokenID); id != "" {
				w.ByToken[id] = i
			}
			if id := strings.TrimSpace(oc.NoTokenID); id != "" {
				w.ByToken[id] = i
			}
		}
		if strings.EqualFold(oc.Platform, appcommon.PlatformKalshi) {
			if id := strings.TrimSpace(oc.PlatformMarketOutcomeID); id != "" {
				w.ByTicker[id] = i
			}
		}
		if strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
			if id := strings.TrimSpace(oc.PlatformMarketOutcomeID); id != "" {
				w.ByPredict[id] = i
			}
		}
		if strings.EqualFold(oc.Platform, appcommon.PlatformSxbet) {
			if id := strings.TrimSpace(oc.PlatformMarketOutcomeID); id != "" {
				w.BySxbet[id] = i
			}
		}
		w.negRisk[i] = broker.ReadNegRisk(oc.Config)
	}
	w.Ops = watchgroup.Ops{
		StrategyParams: func() (float64, float64, float64) {
			if w.entry == nil {
				return 0.95, 100, 1
			}
			p := w.entry.Params()
			return p.MaxCost, p.OrderAmount, p.LevelFactor
		},
		OneX2Limits: func() (float64, float64) {
			if w.entry == nil {
				return 0.95, 1.90
			}
			p := w.entry.Params()
			return p.OneX2YesMax, p.OneX2NoMax
		},
		TradeFee: func(platform, category, sub string) float64 {
			if w.entry == nil {
				return 0
			}
			return w.entry.Params().Fee(platform, category, sub)
		},
		CanTrade: func(platform string) bool {
			if w.entry == nil {
				return false
			}
			return w.entry.canTrade(platform)
		},
		SendOnOutcome:    w.sendOnOutcome,
		FlattenOnOutcome: w.flattenOnOutcome,
		OppRepo: func() persistentrepo.ArbBinaryOpportunityRepo {
			if w.entry == nil {
				return nil
			}
			return w.entry.oppRepo
		},
		OrderRepo: func() persistentrepo.ArbBinaryOrderRepo {
			if w.entry == nil {
				return nil
			}
			return w.entry.orderRepo
		},
		BookRepo: func() persistentrepo.OrderBookRepo {
			if w.entry == nil {
				return nil
			}
			return w.entry.bookRepo
		},
		BooksJSONForPair: w.booksJSONForOutcomePair,
		BookRowsForPair:  w.bookRowsForPairIndices,
	}
	w.yesno = yesno.Entry(wg)
	w.onextwo = onextwo.Entry(wg)
	return w
}

// run 同时盯多平台行情，这组被换掉或进程停时一起退出。
func (w *watch) run() {
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		w.servePoly()
	}()
	go func() {
		defer wg.Done()
		w.serveKalshi()
	}()
	go func() {
		defer wg.Done()
		w.servePredict()
	}()
	go func() {
		defer wg.Done()
		w.serveSxbet()
	}()
	<-w.Ctx.Done()
	wg.Wait()
}

// servePoly 有 Polymarket token 时维持这条连接。
func (w *watch) servePoly() {
	defer w.reconnect(appcommon.PlatformPolymarket)
	assets := polyws.Assets(w.Outcomes)
	if len(assets) == 0 || w.entry.poly == nil {
		return
	}
	st := w.entry.wsStaggerBounds()
	entryws.StaggerConnect(w.Ctx, st.MinMs, st.MaxMs)
	polyws.Run(w.Ctx, assets, w.MapKey, polyws.Session{
		Begin: func() (context.Context, int) { return w.beginConn(appcommon.PlatformPolymarket) },
		Apply: w.applyPoly,
	})
}

// applyPoly 一条 Polymarket 推送调用一次。price_change 的多档就是 Changes。
func (w *watch) applyPoly(msg polyws.Message) {
	recv := time.Now()
	switch msg.Kind {
	case "pong":
		w.touchWS(appcommon.PlatformPolymarket)
	case "pause":
		w.write(event{Gen: msg.Gen, Platform: appcommon.PlatformPolymarket, Kind: "pause"})
	case "book":
		w.write(event{
			Gen: msg.Gen, Platform: appcommon.PlatformPolymarket, Kind: "poly_all_book", ID: msg.ID,
			Bids: msg.Bids, Asks: msg.Asks, Tick: msg.Tick, MinSize: msg.MinSize, TS: msg.TS,
			RecvAt: recv,
		})
	case "price_change":
		w.write(event{
			Gen: msg.Gen, Platform: appcommon.PlatformPolymarket, Kind: "poly_price_change",
			Changes: msg.Changes, TS: msg.TS, RecvAt: recv,
		})
	}
}

// servePredict 有 Predict market id 且配置了 api key 时维持 WS。
func (w *watch) servePredict() {
	defer w.reconnect(appcommon.PlatformPredict)
	ids := predictws.MarketIDs(w.Outcomes)
	key := w.entry.predictAPIKey()
	if len(ids) == 0 || key == "" {
		return
	}
	st := w.entry.wsStaggerBounds()
	entryws.StaggerConnect(w.Ctx, st.MinMs, st.MaxMs)
	predictws.Run(w.Ctx, key, w.Outcomes, w.MapKey, predictws.Session{
		Begin:    func() (context.Context, int) { return w.beginConn(appcommon.PlatformPredict) },
		Apply:    w.applyPredict,
		Snapshot: w.bootstrapPredictBooks,
	})
}

// bootstrapPredictBooks REST 拉本组全部 Predict 盘口，避免 WS 只推变动时库里缺行。
func (w *watch) bootstrapPredictBooks(gen int) {
	if w.entry == nil || w.entry.predict == nil {
		return
	}
	ids := predictws.MarketIDs(w.Outcomes)
	if len(ids) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(w.Ctx, 15*time.Second)
	defer cancel()
	books, err := w.entry.predict.GetOrderbooks(ctx, ids)
	if err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-predict-snapshot-failed",
			zap.String("map_key", w.MapKey),
			zap.Int("markets", len(ids)),
			zap.Error(err))
		return
	}
	for i := range books {
		id := strconv.FormatInt(books[i].MarketID, 10)
		if id == "0" {
			continue
		}
		ts := time.Now().UTC()
		if books[i].UpdateTimestampMs > 0 {
			ts = time.UnixMilli(books[i].UpdateTimestampMs).UTC()
		}
		w.write(event{
			Gen: gen, Platform: appcommon.PlatformPredict, Kind: "predict_all_book", ID: id,
			Bids: predictws.LevelsFrom(books[i].Bids), Asks: predictws.LevelsFrom(books[i].Asks), TS: ts,
		})
	}
}

// serveSxbet 有 SX marketHash 且配置 API Key 时维持 orderbook_v3 WS。
func (w *watch) serveSxbet() {
	defer w.reconnect(appcommon.PlatformSxbet)
	hashes := sxbetws.MarketHashes(w.Outcomes)
	if len(hashes) == 0 {
		if w.platformExpected(appcommon.PlatformSxbet) {
			notification.SendErrorLog(w.Ctx, "strategy_entry-sxbet-ws-skip",
				zap.String("map_key", w.MapKey),
				zap.String("reason", "group has SXBET market but outcomes lack marketHash"))
		}
		return
	}
	if w.entry == nil || w.entry.sxbet == nil || w.entry.sxbetAPIKey() == "" {
		notification.SendErrorLog(w.Ctx, "strategy_entry-sxbet-ws-skip",
			zap.String("map_key", w.MapKey),
			zap.String("reason", "missing sxbet client or api_key (platform.extra 或 account)"),
			zap.Int("hashes", len(hashes)))
		return
	}
	st := w.entry.wsStaggerBounds()
	entryws.StaggerConnect(w.Ctx, st.MinMs, st.MaxMs)
	sxbetws.Run(w.Ctx, w.entry.sxbet, w.Outcomes, w.MapKey, sxbetws.Session{
		Begin: func() (context.Context, int) { return w.beginConn(appcommon.PlatformSxbet) },
		Apply: w.applySxbet,
		Snapshot: func(gen int) {
			w.bootstrapSxbetBooks(gen)
		},
	})
}

func (w *watch) bootstrapSxbetBooks(gen int) {
	if w.entry == nil || w.entry.sxbet == nil {
		return
	}
	for _, hash := range sxbetws.MarketHashes(w.Outcomes) {
		ctx, cancel := context.WithTimeout(w.Ctx, 15*time.Second)
		snap, err := w.entry.sxbet.GetOrderbookSnapshot(ctx, hash)
		cancel()
		if err != nil || snap == nil {
			continue
		}
		msg, ok := sxbetws.MessageFromSnapshot(gen, snap)
		if !ok {
			continue
		}
		w.applySxbet(msg)
	}
}

func (w *watch) applySxbet(msg sxbetws.Message) {
	recv := time.Now()
	switch msg.Kind {
	case "heartbeat":
		w.touchWS(appcommon.PlatformSxbet)
	case "pause":
		w.write(event{Gen: msg.Gen, Platform: appcommon.PlatformSxbet, Kind: "pause"})
	case "book":
		w.write(event{
			Gen: msg.Gen, Platform: appcommon.PlatformSxbet, Kind: "sxbet_all_book", ID: msg.ID,
			Bids: msg.Bids, Asks: msg.Asks, TS: msg.TS, RecvAt: recv,
		})
	}
}

func (w *watch) applyPredict(msg predictws.Message) {
	recv := time.Now()
	switch msg.Kind {
	case "heartbeat":
		w.touchWS(appcommon.PlatformPredict)
	case "pause":
		w.write(event{Gen: msg.Gen, Platform: appcommon.PlatformPredict, Kind: "pause"})
	case "book":
		w.write(event{
			Gen: msg.Gen, Platform: appcommon.PlatformPredict, Kind: "predict_all_book", ID: msg.ID,
			Bids: msg.Bids, Asks: msg.Asks, TS: msg.TS, RecvAt: recv,
		})
	}
}

// serveKalshi 有 Kalshi ticker 时维持这条连接。
func (w *watch) serveKalshi() {
	defer w.reconnect(appcommon.PlatformKalshi)
	tickers := kalshiws.Tickers(w.Outcomes)
	if len(tickers) == 0 || w.entry.kalshi == nil {
		return
	}
	st := w.entry.wsStaggerBounds()
	entryws.StaggerConnect(w.Ctx, st.MinMs, st.MaxMs)
	kalshiws.Run(w.Ctx, w.entry.kalshi, tickers, w.MapKey, func() httpentity.KalshiAccount {
		return w.entry.credsNow().kalshi
	}, func() bool {
		return w.entry.warnKalshiKey.CompareAndSwap(false, true)
	}, kalshiws.Session{
		Begin: func() (context.Context, int) { return w.beginConn(appcommon.PlatformKalshi) },
		Apply: w.applyKalshi,
	})
}

// applyKalshi 一条 Kalshi 推送调用一次。
func (w *watch) applyKalshi(msg kalshiws.Feed) {
	recv := time.Now()
	switch msg.Kind {
	case "pause":
		w.write(event{Gen: msg.Gen, Platform: appcommon.PlatformKalshi, Kind: "pause"})
	case "snap":
		w.write(event{
			Gen: msg.Gen, Platform: appcommon.PlatformKalshi, Kind: "kalshi_all_book", ID: msg.ID,
			Bids: msg.Bids, NoBids: msg.NoBids, Seq: msg.Seq, TS: msg.TS, RecvAt: recv,
		})
	case "delta":
		w.write(event{
			Gen: msg.Gen, Platform: appcommon.PlatformKalshi, Kind: "kalshi_price_change", ID: msg.ID,
			Side: msg.Side, Price: msg.Price, Delta: msg.Delta, Seq: msg.Seq, TS: msg.TS, RecvAt: recv,
		})
	}
}

// write 把这一条写入内存盘口。price_change 的多档是消息体，写完检测一次。
func (w *watch) write(e event) {
	if e.Gen != w.currentGen(e.Platform) {
		return
	}
	var skipGrab bool
	switch e.Kind {
	case "pause":
		// 连接换代或断开，这家盘口先标成不可交易。
		w.markWSPause(e.Platform)
		w.Books.PausePlatform(e.Platform)
		return
	case "poly_all_book":
		// 全量整本替换。订阅后的第一条，以及之后成交触发的 book，都走这里。
		if !w.Books.ApplyPolyBook(e.ID, e.Bids, e.Asks, e.Tick, e.MinSize, e.TS) {
			return
		}
		w.syncPolyComplementNo(e.ID, e.TS)
	case "poly_price_change":
		// 一条 price_change 里的多档一起写入。
		w.Books.ApplyPolyLevels(e.Changes, e.TS)
		for yesTok := range w.polyYesTokensTouched(e.Changes) {
			w.syncPolyComplementNo(yesTok, e.TS)
		}
	case "kalshi_all_book":
		// 订阅成功后的全量。seq 作为后续增量的起点。
		if !w.Books.ApplyKalshiSnapshot(e.ID, e.Bids, e.NoBids, e.Seq, e.TS) {
			return
		}
	case "predict_all_book":
		if !w.Books.ApplyPredictBook(e.ID, e.Bids, e.Asks, e.TS) {
			return
		}
	case "sxbet_all_book":
		if !w.Books.ApplySxbetBook(e.ID, e.Bids, e.Asks, e.TS) {
			return
		}
	case "kalshi_price_change":
		// 订阅WS成功后，kalshi开始推送增量变化的订单簿信息
		ok, gap := w.Books.ApplyKalshiDelta(e.ID, e.Side, e.Price, e.Delta, e.Seq, e.TS)
		if gap {
			notification.SendErrorLog(w.Ctx, "strategy_entry-kalshi-seq-gap",
				zap.String("map_key", w.MapKey),
				zap.String("ticker", e.ID),
				zap.Int64("seq", e.Seq))
			w.reconnect(appcommon.PlatformKalshi)
			return
		}
		if !ok {
			return
		}
		if strings.EqualFold(strings.TrimSpace(e.Side), "no") {
			skipGrab = true
		}
	default:
		return
	}
	w.touchWS(e.Platform)

	w.enqueueBookPersistWS(e)

	var outcomeID uint
	var ok bool
	if skipGrab {
		return
	}
	if e.Kind == "poly_price_change" {
		for _, ch := range e.Changes {
			if !w.isPolyYesToken(ch.ID) {
				continue
			}
			outcomeID, ok = w.outcomeIDOf(e.Platform, ch.ID)
			if ok {
				break
			}
		}
	} else {
		outcomeID, ok = w.outcomeIDOf(e.Platform, e.ID)
		if e.Kind == "poly_all_book" && ok && !w.isPolyYesToken(e.ID) {
			ok = false
		}
	}
	if !ok {
		return
	}

	// 调用策略
	if w.yesno == nil && w.onextwo == nil {
		return
	}
	start := recvAtOrNow(e.RecvAt)
	if w.yesno != nil {
		w.yesno.OnBookUpdate(outcomeID)
		if w.entry != nil {
			w.entry.recordDetectLatencyYesno(w.MapKey, time.Since(start))
		}
	}
	if w.onextwo != nil {
		w.onextwo.OnBookUpdate(outcomeID)
		if w.entry != nil {
			w.entry.recordDetectLatencyOnextwo(w.MapKey, time.Since(start))
		}
	}
}

func recvAtOrNow(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

func (w *watch) polyNoTokenForYes(yesToken string) string {
	yesToken = strings.TrimSpace(yesToken)
	idx, ok := w.ByToken[yesToken]
	if !ok || idx < 0 || idx >= len(w.Outcomes) {
		return ""
	}
	oc := &w.Outcomes[idx]
	if !strings.EqualFold(oc.Platform, appcommon.PlatformPolymarket) {
		return ""
	}
	if strings.TrimSpace(oc.YesTokenID) != yesToken {
		return ""
	}
	return strings.TrimSpace(oc.NoTokenID)
}

func (w *watch) isPolyYesToken(token string) bool {
	token = strings.TrimSpace(token)
	idx, ok := w.ByToken[token]
	if !ok || idx < 0 || idx >= len(w.Outcomes) {
		return false
	}
	oc := &w.Outcomes[idx]
	return strings.EqualFold(oc.Platform, appcommon.PlatformPolymarket) &&
		strings.TrimSpace(oc.YesTokenID) == token
}

func (w *watch) syncPolyComplementNo(yesToken string, ts time.Time) {
	noTok := w.polyNoTokenForYes(yesToken)
	if noTok == "" {
		return
	}
	w.Books.SyncPolyNoTokenFromYes(yesToken, noTok, ts)
}

func (w *watch) polyYesTokensTouched(changes []book.PolyChange) map[string]struct{} {
	out := make(map[string]struct{}, len(changes))
	for _, ch := range changes {
		if !w.isPolyYesToken(ch.ID) {
			continue
		}
		out[strings.TrimSpace(ch.ID)] = struct{}{}
	}
	return out
}

// enqueueBookPersistWS 将 WS 盘口推入 Redis，由 ws_book_flush 批量写 market_order_book_ws。
func eventSnapshotMs(e event) int64 {
	if !e.TS.IsZero() {
		return e.TS.UnixMilli()
	}
	if !e.RecvAt.IsZero() {
		return e.RecvAt.UnixMilli()
	}
	return time.Now().UnixMilli()
}

func snapshotMsForWSRow(row *persistententity.OrderBook, e event) int64 {
	if row != nil && row.SnapshotAtMs > 0 {
		return row.SnapshotAtMs
	}
	return eventSnapshotMs(e)
}

func (w *watch) enqueueBookPersistWS(e event) {
	if w.entry == nil || w.entry.wsBookQueue == nil {
		return
	}
	rows := w.orderBookRowsFromEvent(e)
	if len(rows) == 0 {
		return
	}
	q := w.entry.wsBookQueue
	for _, row := range rows {
		if row == nil {
			continue
		}
		dup := cloneOrderBookRow(row)
		w.enrichBookRow(dup)
		if dup.SnapshotAtMs <= 0 {
			dup.SnapshotAtMs = eventSnapshotMs(e)
		}
		wsRow := persistententity.OrderBookWSCopy(dup)
		if wsRow == nil {
			continue
		}
		if !w.allowWSPersistRow(dup.Platform, dup.TokenID) {
			continue
		}
		wsRow.SnapshotAtMs = snapshotMsForWSRow(dup, e)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := q.Push(ctx, wsRow)
		cancel()
		if err != nil {
			notification.SendErrorLog(ctx, "strategy_entry-ws-book-queue-push",
				zap.Error(err),
				zap.String("platform", dup.Platform),
				zap.String("outcome_id", dup.PlatformMarketOutcomeID),
				zap.String("token_id", dup.TokenID))
		}
	}
}

func (w *watch) allowWSPersistRow(platform, tokenID string) bool {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return true
	}
	key := strings.ToUpper(strings.TrimSpace(platform)) + "|" + tokenID
	now := time.Now()
	w.wsPersistMu.Lock()
	defer w.wsPersistMu.Unlock()
	if w.wsPersistLast == nil {
		w.wsPersistLast = make(map[string]time.Time)
	}
	if t, ok := w.wsPersistLast[key]; ok && now.Sub(t) < wsBookPersistMinInterval {
		return false
	}
	w.wsPersistLast[key] = now
	return true
}

func cloneOrderBookRow(src *persistententity.OrderBook) *persistententity.OrderBook {
	if src == nil {
		return nil
	}
	dup := *src
	if src.MapKey != nil {
		v := *src.MapKey
		dup.MapKey = &v
	}
	if src.Category != nil {
		v := *src.Category
		dup.Category = &v
	}
	if src.SubCategory != nil {
		v := *src.SubCategory
		dup.SubCategory = &v
	}
	if src.Competition != nil {
		v := *src.Competition
		dup.Competition = &v
	}
	if src.CompetitionEn != nil {
		v := *src.CompetitionEn
		dup.CompetitionEn = &v
	}
	if src.MarketName != nil {
		v := *src.MarketName
		dup.MarketName = &v
	}
	if src.OutcomeName != nil {
		v := *src.OutcomeName
		dup.OutcomeName = &v
	}
	if src.TokenName != nil {
		v := *src.TokenName
		dup.TokenName = &v
	}
	if src.SelectionID != nil {
		v := *src.SelectionID
		dup.SelectionID = &v
	}
	return &dup
}

func (w *watch) enrichBookRow(row *persistententity.OrderBook) {
	if w == nil || row == nil {
		return
	}
	oc := w.outcomeForBookRow(row)
	if oc == nil {
		return
	}
	var fx *persistententity.Market
	if m, ok := w.MarketsByRowID[oc.MarketRowID]; ok {
		fx = &m
	}
	obcommon.EnrichOrderBookFromOutcome(row, oc, fx, row.TokenID)
}

func (w *watch) outcomeForBookRow(row *persistententity.OrderBook) *persistententity.MarketOutcome {
	if w == nil || row == nil {
		return nil
	}
	wantPlat := strings.ToUpper(strings.TrimSpace(row.Platform))
	wantOutcome := strings.TrimSpace(row.PlatformMarketOutcomeID)
	for i := range w.Outcomes {
		oc := &w.Outcomes[i]
		if strings.ToUpper(strings.TrimSpace(oc.Platform)) != wantPlat {
			continue
		}
		if strings.TrimSpace(oc.PlatformMarketOutcomeID) == wantOutcome {
			return oc
		}
	}
	return nil
}

// orderBookRowsFromEvent 按平台与事件类型取出待落库的盘口行。
func (w *watch) orderBookRowsFromEvent(e event) []*persistententity.OrderBook {
	switch e.Kind {
	case "poly_all_book":
		return w.bookRowsForPolyTokenAllSides(e.ID)
	case "poly_price_change":
		var rows []*persistententity.OrderBook
		seenIdx := make(map[int]struct{}, len(e.Changes))
		for _, ch := range e.Changes {
			token := strings.TrimSpace(ch.ID)
			if token == "" {
				continue
			}
			idx, ok := w.ByToken[token]
			if !ok {
				continue
			}
			if _, dup := seenIdx[idx]; dup {
				continue
			}
			seenIdx[idx] = struct{}{}
			rows = append(rows, w.bookRowsForPolyOutcome(idx)...)
		}
		return rows
	case "kalshi_all_book", "kalshi_price_change":
		return w.bookRowsForKalshiTicker(e.ID)
	case "predict_all_book":
		return w.bookRowsForPredictMarket(e)
	case "sxbet_all_book":
		return w.bookRowsForSxbetMarket(e)
	default:
		return nil
	}
}

func (w *watch) bookRowsForPolyTokenAllSides(token string) []*persistententity.OrderBook {
	idx, ok := w.ByToken[strings.TrimSpace(token)]
	if !ok {
		return nil
	}
	return w.bookRowsForPolyOutcome(idx)
}

func (w *watch) bookRowsForPolyOutcome(outcomeIndex int) []*persistententity.OrderBook {
	if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
		return nil
	}
	oc := &w.Outcomes[outcomeIndex]
	var rows []*persistententity.OrderBook
	for _, token := range []string{oc.YesTokenID, oc.NoTokenID} {
		rows = append(rows, w.bookRowsForPolyToken(token)...)
	}
	return rows
}

func (w *watch) bookRowsForPolyToken(token string) []*persistententity.OrderBook {
	idx, ok := w.ByToken[strings.TrimSpace(token)]
	if !ok {
		return nil
	}
	b, ok := w.Books.Get(appcommon.PlatformPolymarket, strings.TrimSpace(token))
	if !ok {
		return nil
	}
	oc := &w.Outcomes[idx]
	row := bookRow(oc, w.fixtureForOutcome(oc), b.Bids, b.Asks, strings.TrimSpace(token))
	if row == nil {
		return nil
	}
	return []*persistententity.OrderBook{row}
}

func (w *watch) bookRowsForSxbetMarket(e event) []*persistententity.OrderBook {
	marketHash := strings.TrimSpace(e.ID)
	idx, ok := w.BySxbet[marketHash]
	if !ok {
		return nil
	}
	b, ok := w.Books.Get(appcommon.PlatformSxbet, marketHash)
	if !ok {
		return nil
	}
	return w.sxbetPersistRows(&w.Outcomes[idx], b.Bids, b.Asks, eventSnapshotMs(e))
}

func (w *watch) sxbetPersistRows(oc *persistententity.MarketOutcome, yesBids, yesAsks []book.Level, updateTimestampMs int64) []*persistententity.OrderBook {
	if w == nil || oc == nil {
		return nil
	}
	var fx *persistententity.Market
	if m, ok := w.MarketsByRowID[oc.MarketRowID]; ok {
		fx = &m
	}
	snapMs := updateTimestampMs
	if snapMs <= 0 {
		snapMs = time.Now().UnixMilli()
	}
	rows := obcommon.PredictBookRowsWithFixture(oc, fx, yesBids, yesAsks, snapMs)
	if len(rows) == 0 {
		return nil
	}
	out := make([]*persistententity.OrderBook, 0, len(rows))
	for _, r := range rows {
		if r != nil {
			out = append(out, r)
		}
	}
	return out
}

func (w *watch) bookRowsForPredictMarket(e event) []*persistententity.OrderBook {
	marketID := strings.TrimSpace(e.ID)
	idx, ok := w.ByPredict[marketID]
	if !ok {
		return nil
	}
	b, ok := w.Books.Get(appcommon.PlatformPredict, marketID)
	if !ok {
		return nil
	}
	ms := int64(0)
	if !e.TS.IsZero() {
		ms = e.TS.UnixMilli()
	}
	return w.predictPersistRows(&w.Outcomes[idx], b.Bids, b.Asks, ms)
}

func (w *watch) predictPersistRows(oc *persistententity.MarketOutcome, yesBids, yesAsks []book.Level, updateTimestampMs int64) []*persistententity.OrderBook {
	if w == nil || oc == nil {
		return nil
	}
	var fx *persistententity.Market
	if m, ok := w.MarketsByRowID[oc.MarketRowID]; ok {
		fx = &m
	}
	snapMs := updateTimestampMs
	if snapMs <= 0 {
		snapMs = time.Now().UnixMilli()
	}
	rows := obcommon.PredictBookRowsWithFixture(oc, fx, yesBids, yesAsks, snapMs)
	if len(rows) == 0 {
		return nil
	}
	out := make([]*persistententity.OrderBook, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (w *watch) bookRowsForKalshiTicker(ticker string) []*persistententity.OrderBook {
	ticker = strings.TrimSpace(ticker)
	idx, ok := w.ByTicker[ticker]
	if !ok {
		return nil
	}
	b, ok := w.Books.Get(appcommon.PlatformKalshi, ticker)
	if !ok {
		return nil
	}
	oc := &w.Outcomes[idx]
	row := bookRow(oc, w.fixtureForOutcome(oc), b.Bids, b.Asks, ticker)
	if row == nil {
		return nil
	}
	return []*persistententity.OrderBook{row}
}

// bookRowsForHit 取出这次组合两边相关的盘口行，供协程直接写 DB。
func (w *watch) bookRowsForHit(aIndex, bIndex int) []*persistententity.OrderBook {
	var rows []*persistententity.OrderBook
	for _, outcomeIndex := range []int{aIndex, bIndex} {
		if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
			continue
		}
		oc := &w.Outcomes[outcomeIndex]
		if strings.EqualFold(oc.Platform, appcommon.PlatformPolymarket) {
			for _, token := range []string{oc.YesTokenID, oc.NoTokenID} {
				token = strings.TrimSpace(token)
				idx, ok := w.ByToken[token]
				if !ok {
					continue
				}
				bk, ok := w.Books.Get(appcommon.PlatformPolymarket, token)
				if !ok {
					continue
				}
				tokOc := &w.Outcomes[idx]
				rows = append(rows, bookRow(tokOc, w.fixtureForOutcome(tokOc), bk.Bids, bk.Asks, token))
			}
			continue
		}
		bookID := strings.TrimSpace(oc.PlatformMarketOutcomeID)
		var idx int
		var ok bool
		switch strings.ToUpper(strings.TrimSpace(oc.Platform)) {
		case appcommon.PlatformKalshi:
			idx, ok = w.ByTicker[bookID]
		case appcommon.PlatformPredict:
			idx, ok = w.ByPredict[bookID]
		case appcommon.PlatformSxbet:
			idx, ok = w.BySxbet[bookID]
		default:
			continue
		}
		if !ok {
			continue
		}
		bk, ok := w.Books.Get(oc.Platform, bookID)
		if !ok {
			continue
		}
		if strings.EqualFold(oc.Platform, appcommon.PlatformPredict) {
			rows = append(rows, w.predictPersistRows(&w.Outcomes[idx], bk.Bids, bk.Asks, 0)...)
			continue
		}
		if strings.EqualFold(oc.Platform, appcommon.PlatformSxbet) {
			rows = append(rows, w.sxbetPersistRows(&w.Outcomes[idx], bk.Bids, bk.Asks, 0)...)
			continue
		}
		hitOc := &w.Outcomes[idx]
		rows = append(rows, bookRow(hitOc, w.fixtureForOutcome(hitOc), bk.Bids, bk.Asks, bookID))
	}
	return rows
}

func (w *watch) bookRowsForPairIndices(a, b int) []*persistententity.OrderBook {
	return w.bookRowsForHit(a, b)
}

func (w *watch) sendOnOutcome(outcomeIndex int, buyYes bool, px, qty float64) (float64, error) {
	if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) || w.entry == nil {
		return 0, nil
	}
	oc := &w.Outcomes[outcomeIndex]
	c := w.entry.credsNow()
	return broker.SendOnOutcome(
		w.entry.poly, w.entry.kalshi, w.entry.predict,
		broker.PolyCreds{Account: c.poly, OK: c.polyOK},
		broker.KalshiCreds{Account: c.kalshi, OK: c.kalshiOK},
		broker.PredictCreds{Account: c.predict, OK: c.predictOK},
		oc, buyYes, px, qty,
		outcomeIndex < len(w.negRisk) && w.negRisk[outcomeIndex],
	)
}

func (w *watch) flattenOnOutcome(outcomeIndex int, boughtYes bool, mine, other float64) {
	if outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) || w.entry == nil {
		return
	}
	oc := &w.Outcomes[outcomeIndex]
	c := w.entry.credsNow()
	broker.FlattenOnOutcome(
		w.entry.poly, w.entry.kalshi, w.entry.predict,
		broker.PolyCreds{Account: c.poly, OK: c.polyOK},
		broker.KalshiCreds{Account: c.kalshi, OK: c.kalshiOK},
		broker.PredictCreds{Account: c.predict, OK: c.predictOK},
		oc, boughtYes, mine, other,
		outcomeIndex < len(w.negRisk) && w.negRisk[outcomeIndex],
	)
}

// savePoly 把这个 token 的盘口交给异步落库。
func (w *watch) savePoly(token string) {
	idx, ok := w.ByToken[token]
	if !ok || w.entry.flush == nil {
		return
	}
	b, ok := w.Books.Get(appcommon.PlatformPolymarket, token)
	if !ok {
		return
	}
	oc := &w.Outcomes[idx]
	w.entry.flush.Book(bookRow(oc, w.fixtureForOutcome(oc), b.Bids, b.Asks, token))
}

// saveKalshi 把这个 ticker 的盘口交给异步落库。
func (w *watch) saveKalshi(ticker string) {
	idx, ok := w.ByTicker[ticker]
	if !ok || w.entry.flush == nil {
		return
	}
	b, ok := w.Books.Get(appcommon.PlatformKalshi, ticker)
	if !ok {
		return
	}
	oc := &w.Outcomes[idx]
	w.entry.flush.Book(bookRow(oc, w.fixtureForOutcome(oc), b.Bids, b.Asks, ticker))
}

// booksJSON 单 outcome 时只写 side 1；两边时写 side 1 + side 2。
func (w *watch) booksJSON(ocs ...*persistententity.MarketOutcome) string {
	if len(ocs) == 1 && ocs[0] != nil {
		return w.booksJSONForOutcomePair(w.outcomeIndexByPtr(ocs[0]), -1)
	}
	if len(ocs) >= 2 {
		return w.booksJSONForOutcomePair(w.outcomeIndexByPtr(ocs[0]), w.outcomeIndexByPtr(ocs[1]))
	}
	return `{"outcomes":[]}`
}

func (w *watch) outcomeIndexByPtr(oc *persistententity.MarketOutcome) int {
	if oc == nil {
		return -1
	}
	for i := range w.Outcomes {
		if w.Outcomes[i].ID == oc.ID {
			return i
		}
	}
	return -1
}

func (w *watch) booksJSONForOutcomePair(outcomeIndexA, outcomeIndexB int) string {
	return w.BooksJSONForOutcomePair(outcomeIndexA, outcomeIndexB)
}

// beginConn 换一代连接。旧连接上迟到的包会被丢掉。
func (w *watch) beginConn(platform string) (context.Context, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	st := w.conn[platform]
	if st == nil {
		st = &connState{}
		w.conn[platform] = st
	}
	if st.cancel != nil {
		st.cancel()
	}
	st.gen++
	ctx, cancel := context.WithCancel(w.Ctx)
	st.cancel = cancel
	w.markWSConn(platform)
	return ctx, st.gen
}

// currentGen 读这家当前是第几代连接。
func (w *watch) currentGen(platform string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	st := w.conn[platform]
	if st == nil {
		return 0
	}
	return st.gen
}

// reconnect 关掉这家当前连接，让外层重新订阅。
func (w *watch) reconnect(platform string) {
	w.mu.Lock()
	st := w.conn[platform]
	if st != nil && st.cancel != nil {
		st.cancel()
	}
	w.mu.Unlock()
}

// emit 把事件放进队列。队列满了就丢，不堵住读包。
func (w *watch) emit(e event) error {
	select {
	case <-w.Ctx.Done():
		return w.Ctx.Err()
	case w.ev <- e:
		return nil
	default:
		return errBusy
	}
}

// top 只留前 n 档。
func top(ls []book.Level, n int) []book.Level {
	if len(ls) > n {
		return ls[:n]
	}
	return ls
}

func (w *watch) outcomeIDOf(platform, id string) (uint, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0, false
	}
	var outcomeIndex int
	var ok bool
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case appcommon.PlatformPolymarket:
		outcomeIndex, ok = w.ByToken[id]
	case appcommon.PlatformKalshi:
		outcomeIndex, ok = w.ByTicker[id]
	case appcommon.PlatformPredict:
		outcomeIndex, ok = w.ByPredict[id]
	default:
		return 0, false
	}
	if !ok || outcomeIndex < 0 || outcomeIndex >= len(w.Outcomes) {
		return 0, false
	}
	return w.Outcomes[outcomeIndex].ID, true
}

// sleepCtx 等待一段时间。上下文取消时提前返回。
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// bookRow 把内存盘口收成一行待落库的订单簿。
func bookRow(m *persistententity.MarketOutcome, fx *persistententity.Market, bids, asks []book.Level, tokenID string) *persistententity.OrderBook {
	if m == nil {
		return nil
	}
	addr := ""
	if m.MarketAddress != nil {
		addr = strings.TrimSpace(*m.MarketAddress)
	}
	row := &persistententity.OrderBook{
		Platform:                strings.ToUpper(strings.TrimSpace(m.Platform)),
		MarketRowID:             m.MarketRowID,
		PlatformMarketID:        strings.TrimSpace(m.PlatformMarketID),
		MarketURL:               strings.TrimSpace(m.MarketURL),
		PlatformMarketOutcomeID: strings.TrimSpace(m.PlatformMarketOutcomeID),
		MarketAddress:           addr,
		TokenID:                 tokenID,
		OrderBookBids:           levelsJSON(bids),
		OrderBookAsks:           levelsJSON(asks),
	}
	if m.SelectionID != nil {
		row.SelectionID = m.SelectionID
	}
	obcommon.EnrichOrderBookFromOutcome(row, m, fx, tokenID)
	return row
}

func (w *watch) fixtureForOutcome(oc *persistententity.MarketOutcome) *persistententity.Market {
	if w == nil || oc == nil || oc.MarketRowID == 0 {
		return nil
	}
	if m, ok := w.MarketsByRowID[oc.MarketRowID]; ok {
		return &m
	}
	return nil
}

// levelsJSON 把档位收成价、量的 JSON。
func levelsJSON(ls []book.Level) string {
	rows := make([][]string, 0, len(ls))
	for _, lv := range ls {
		rows = append(rows, []string{formatNum(lv.Price), formatNum(lv.Size)})
	}
	b, err := json.Marshal(rows)
	if err != nil || len(b) == 0 {
		return "[]"
	}
	return string(b)
}

// formatNum 把浮点收成不带多余小数的字符串。
func formatNum(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
