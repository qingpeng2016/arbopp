package persist

import (
	"context"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Flush 把盘口和命中异步写入 MySQL，不挡判定协程。
type Flush struct {
	books persistentrepo.OrderBookRepo
	opps  persistentrepo.ArbBinaryOpportunityRepo

	mu     sync.Mutex
	latest map[string]*persistententity.OrderBook
	oppCh  chan *persistententity.ArbBinaryOpportunity
	stop   chan struct{}
	done   chan struct{}
}

func Start(books persistentrepo.OrderBookRepo, opps persistentrepo.ArbBinaryOpportunityRepo) *Flush {
	f := &Flush{
		books:  books,
		opps:   opps,
		latest: map[string]*persistententity.OrderBook{},
		oppCh:  make(chan *persistententity.ArbBinaryOpportunity, 64),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go f.loop()
	return f
}

func (f *Flush) Book(row *persistententity.OrderBook) {
	if f == nil || row == nil {
		return
	}
	key := row.Platform + "|" + row.PlatformMarketOutcomeID + "|" + row.TokenID
	f.mu.Lock()
	f.latest[key] = row
	f.mu.Unlock()
}

func (f *Flush) Opp(row *persistententity.ArbBinaryOpportunity) {
	if f == nil || row == nil {
		return
	}
	select {
	case f.oppCh <- row:
	default:
		notification.SendErrorLog(context.Background(), "strategy_entry-opp-dropped",
			zap.String("map_key", deref(row.MapKey)))
	}
}

func (f *Flush) Stop() {
	if f == nil {
		return
	}
	close(f.stop)
	<-f.done
}

func (f *Flush) loop() {
	defer close(f.done)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-f.stop:
			f.flushBooks()
			f.drainOpps()
			return
		case <-tick.C:
			f.flushBooks()
		case row := <-f.oppCh:
			f.writeOpp(row)
		}
	}
}

func (f *Flush) flushBooks() {
	if f.books == nil {
		f.mu.Lock()
		f.latest = map[string]*persistententity.OrderBook{}
		f.mu.Unlock()
		return
	}
	f.mu.Lock()
	if len(f.latest) == 0 {
		f.mu.Unlock()
		return
	}
	snapMs := time.Now().UnixMilli()
	snap := snapMs / 1000
	rows := make([]*persistententity.OrderBook, 0, len(f.latest))
	for _, row := range f.latest {
		dup := *row
		dup.SnapshotSecond = snap
		dup.SnapshotAtMs = snapMs
		rows = append(rows, &dup)
	}
	f.latest = map[string]*persistententity.OrderBook{}
	f.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.books.BatchCreate(ctx, nil, rows); err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-BookCreateFailed",
			zap.Error(err), zap.Int("rows", len(rows)))
	}
}

func (f *Flush) writeOpp(row *persistententity.ArbBinaryOpportunity) {
	if f.opps == nil || row == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.opps.BatchCreate(ctx, nil, []*persistententity.ArbBinaryOpportunity{row}); err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-OppCreateFailed", zap.Error(err))
	}
}

func (f *Flush) drainOpps() {
	for {
		select {
		case row := <-f.oppCh:
			f.writeOpp(row)
		default:
			return
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
