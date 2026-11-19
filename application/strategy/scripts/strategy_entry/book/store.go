package book

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	"go.uber.org/zap"
)

// Book 一个可交易标的的内存盘口。
// Polymarket 的 ID 是 token，Bids/Asks 是该 token 自己的买卖盘。
// Kalshi 的 ID 是 ticker，Bids 是 YES 买盘，NoBids 是 NO 买盘，Asks 由 NO 买盘换算成 YES 卖盘。
// Live 为真才允许用它做检测：WebSocket 全量已经写入，且序号没有断开。
type Book struct {
	Platform   string
	ID         string
	Bids       []Level
	Asks       []Level
	NoBids     []Level
	Seq        int64
	Live       bool
	MinSize    float64
	Tick       float64
	ExchangeTS time.Time
}

// Store 两路 WebSocket 都会写。公开方法自带锁。
type Store struct {
	mu    sync.Mutex
	books map[string]*Book
}

func NewStore() *Store {
	return &Store{books: map[string]*Book{}}
}

func key(platform, id string) string {
	return strings.ToUpper(strings.TrimSpace(platform)) + "|" + strings.TrimSpace(id)
}

func (s *Store) ensure(platform, id string) *Book {
	k := key(platform, id)
	b := s.books[k]
	if b == nil {
		b = &Book{Platform: strings.ToUpper(platform), ID: id}
		s.books[k] = b
	}
	return b
}

func (s *Store) Get(platform, id string) (Book, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.books[key(platform, id)]
	if b == nil {
		return Book{}, false
	}
	cp := *b
	cp.Bids = Clone(b.Bids)
	cp.Asks = Clone(b.Asks)
	cp.NoBids = Clone(b.NoBids)
	return cp, true
}

func (s *Store) PausePlatform(platform string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := strings.ToUpper(platform)
	for _, b := range s.books {
		if b.Platform == p {
			b.Live = false
		}
	}
}

// noteReject 价格无效，或对不上最小跳动时记一次错误。同一原因、同一价格不重复报警。
func noteReject(platform, id, reason string, price, tick float64) {
	key := reason + "|" + platform + "|" + id + "|" + strconv.FormatFloat(price, 'g', -1, 64)
	if _, loaded := badPriceOnce.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	fields := []zap.Field{
		zap.String("reason", reason),
		zap.String("platform", platform),
		zap.String("id", id),
		zap.Float64("price", price),
		zap.Float64("tick", tick),
		zap.String("价格", strconv.FormatFloat(price, 'f', -1, 64)),
		zap.String("tick_size", strconv.FormatFloat(tick, 'f', -1, 64)),
	}
	if reason == "tick" && tick > 0 {
		n := math.Round(price / tick)
		nearest := n * tick
		drift := price - nearest
		fields = append(fields,
			zap.String("最近合法价", strconv.FormatFloat(nearest, 'f', -1, 64)),
			zap.String("偏离", fmt.Sprintf("%.10f (%.2f%%tick)", drift, math.Abs(drift/tick)*100)),
		)
	}
	notification.SendErrorLog(context.Background(), "strategy_entry-bad-price", fields...)
}

func noteBadLevels(platform, id string, levels []Level, tick float64) {
	for _, lv := range levels {
		if !ValidPrice(lv.Price) {
			noteReject(platform, id, "price", lv.Price, tick)
			continue
		}
		if !PriceOnTick(lv.Price, tick) {
			noteReject(platform, id, "tick", lv.Price, tick)
		}
	}
}

var badPriceOnce sync.Map

// ApplyPolyBook WebSocket 全量。买卖盘整本换成这条消息，不和旧档合并。
func (s *Store) ApplyPolyBook(id string, bids, asks []Level, tick, minSize float64, ts time.Time) bool {
	var usedTick float64
	defer func() {
		noteBadLevels(appcommon.PlatformPolymarket, id, bids, usedTick)
		noteBadLevels(appcommon.PlatformPolymarket, id, asks, usedTick)
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.ensure(appcommon.PlatformPolymarket, id)
	if tick > 0 {
		b.Tick = tick
	}
	usedTick = b.Tick
	if minSize > 0 {
		b.MinSize = minSize
	}
	b.Bids = SortDesc(OnTick(bids, b.Tick))
	b.Asks = SortAsc(OnTick(asks, b.Tick))
	b.ExchangeTS = ts
	b.Live = true
	return true
}

// PolyChange 是 price_change 里的一档。
type PolyChange struct {
	ID    string
	Side  string
	Price float64
	Size  float64
}

// ApplyPolyLevel 绝对数量的一档增量。
func (s *Store) ApplyPolyLevel(id, side string, price, size float64, ts time.Time) (applied bool) {
	var reason string
	var tick float64
	defer func() {
		if reason != "" {
			noteReject(appcommon.PlatformPolymarket, id, reason, price, tick)
		}
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.books[key(appcommon.PlatformPolymarket, id)]
	if b == nil || !b.Live {
		return false
	}
	tick = b.Tick
	reason = applyPolyLevel(b, side, price, size, ts)
	return reason == ""
}

// SyncPolyNoTokenFromYes 用 YES token 整本互补刷新 NO token 簿（实盘仅订 YES WS 时调用）。
func (s *Store) SyncPolyNoTokenFromYes(yesTokenID, noTokenID string, ts time.Time) {
	yesTokenID = strings.TrimSpace(yesTokenID)
	noTokenID = strings.TrimSpace(noTokenID)
	if yesTokenID == "" || noTokenID == "" || yesTokenID == noTokenID {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	yesB := s.books[key(appcommon.PlatformPolymarket, yesTokenID)]
	if yesB == nil || !yesB.Live {
		return
	}
	noBids, noAsks := NoBookFromYesBook(yesB.Bids, yesB.Asks)
	noB := s.ensure(appcommon.PlatformPolymarket, noTokenID)
	noB.Tick = yesB.Tick
	noB.MinSize = yesB.MinSize
	noB.Bids = SortDesc(OnTick(noBids, noB.Tick))
	noB.Asks = SortAsc(OnTick(noAsks, noB.Tick))
	if !ts.IsZero() {
		noB.ExchangeTS = ts
	} else if !yesB.ExchangeTS.IsZero() {
		noB.ExchangeTS = yesB.ExchangeTS
	}
	noB.Live = yesB.Live
}

// ApplyPolyLevels 一次写入一条 price_change 的全部档。
func (s *Store) ApplyPolyLevels(changes []PolyChange, ts time.Time) {
	type reject struct {
		id, reason  string
		price, tick float64
	}
	var bad []reject
	defer func() {
		for _, r := range bad {
			noteReject(appcommon.PlatformPolymarket, r.id, r.reason, r.price, r.tick)
		}
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range changes {
		b := s.books[key(appcommon.PlatformPolymarket, ch.ID)]
		if b == nil || !b.Live {
			continue
		}
		if reason := applyPolyLevel(b, ch.Side, ch.Price, ch.Size, ts); reason != "" {
			bad = append(bad, reject{id: ch.ID, reason: reason, price: ch.Price, tick: b.Tick})
		}
	}
}

func applyPolyLevel(b *Book, side string, price, size float64, ts time.Time) string {
	if !ValidPrice(price) {
		return "price"
	}
	if !PriceOnTick(price, b.Tick) {
		return "tick"
	}
	switch strings.ToUpper(side) {
	case "BUY", "BID":
		b.Bids = Upsert(b.Bids, price, size, true)
	default:
		b.Asks = Upsert(b.Asks, price, size, false)
	}
	if !ts.IsZero() {
		b.ExchangeTS = ts
	}
	return ""
}

// ApplyPredictBook Predict WS 全量：盘口为 YES 视角，NO 卖一由 YES 买一换算（见 yesno.TopFromStore）。
func (s *Store) ApplyPredictBook(marketID string, bids, asks []Level, ts time.Time) bool {
	tick := 0.01
	defer func() {
		noteBadLevels(appcommon.PlatformPredict, marketID, bids, tick)
		noteBadLevels(appcommon.PlatformPredict, marketID, asks, tick)
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.ensure(appcommon.PlatformPredict, marketID)
	b.Tick = tick
	b.Bids = SortDesc(OnTick(bids, tick))
	b.Asks = SortAsc(OnTick(asks, tick))
	b.NoBids = nil
	b.ExchangeTS = ts
	b.Live = true
	return true
}

// ApplyKalshiYesBook 历史/回测 YES 视角 L2（与 Predict 单簿一致；NO 卖一=1−YES买一，见 yesno.TopFromStore）。
func (s *Store) ApplyKalshiYesBook(ticker string, bids, asks []Level, ts time.Time) bool {
	tick := 0.01
	defer func() {
		noteBadLevels(appcommon.PlatformKalshi, ticker, bids, tick)
		noteBadLevels(appcommon.PlatformKalshi, ticker, asks, tick)
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.ensure(appcommon.PlatformKalshi, ticker)
	b.Tick = tick
	b.Bids = SortDesc(OnTick(bids, tick))
	b.Asks = SortAsc(OnTick(asks, tick))
	b.NoBids = nil
	b.ExchangeTS = ts
	b.Live = true
	return true
}

// ApplyKalshiSnapshot WebSocket 全量。seq 作为后续增量的起点。
func (s *Store) ApplyKalshiSnapshot(ticker string, yesBids, noBids []Level, seq int64, ts time.Time) bool {
	defer func() {
		noteBadLevels(appcommon.PlatformKalshi, ticker, yesBids, 0)
		noteBadLevels(appcommon.PlatformKalshi, ticker, noBids, 0)
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.ensure(appcommon.PlatformKalshi, ticker)
	b.Bids = SortDesc(OnTick(yesBids, 0))
	b.NoBids = SortDesc(OnTick(noBids, 0))
	b.Asks = AsksFromOppositeBids(b.NoBids)
	b.Seq = seq
	b.ExchangeTS = ts
	b.Live = true
	return true
}

// ApplyKalshiDelta 序号必须连续。断开时 Live 置假，调用方重新订阅。
func (s *Store) ApplyKalshiDelta(ticker, side string, price, delta float64, seq int64, ts time.Time) (applied bool, gap bool) {
	var bad bool
	defer func() {
		if bad {
			noteReject(appcommon.PlatformKalshi, ticker, "price", price, 0)
		}
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.books[key(appcommon.PlatformKalshi, ticker)]
	if b == nil || !b.Live {
		return false, false
	}
	if b.Seq > 0 && seq > 0 && seq != b.Seq+1 {
		b.Live = false
		return false, true
	}
	if seq > 0 {
		b.Seq = seq
	}
	if !ValidPrice(price) {
		bad = true
		return false, false
	}
	if strings.EqualFold(side, "no") {
		b.NoBids = AddSize(b.NoBids, price, delta, true)
		b.Asks = AsksFromOppositeBids(b.NoBids)
	} else {
		b.Bids = AddSize(b.Bids, price, delta, true)
	}
	if !ts.IsZero() {
		b.ExchangeTS = ts
	}
	return true, false
}

// Ask 卖一。Kalshi 的 ID 是 ticker，返回的是 YES 卖一。
func (s *Store) Ask(platform, id string) (price, size float64, live bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.books[key(platform, id)]
	if b == nil || !b.Live || len(b.Asks) == 0 {
		return 0, 0, false
	}
	return b.Asks[0].Price, b.Asks[0].Size, true
}

// Bid 买一。Kalshi 返回 YES 买一。
func (s *Store) Bid(platform, id string) (price, size float64, live bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.books[key(platform, id)]
	if b == nil || !b.Live || len(b.Bids) == 0 {
		return 0, 0, false
	}
	return b.Bids[0].Price, b.Bids[0].Size, true
}
