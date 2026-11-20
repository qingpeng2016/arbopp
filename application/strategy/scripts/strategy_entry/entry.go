package strategyentry

import (
	"context"
	"fmt"
	"sort"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/persist"
	entrystat "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/stat"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Entry 常驻监控已核对场次。调度每次调用 Run 只负责把循环拉起来。
// 行情回调里直接比较卖一并发两笔 FAK。落库在别的协程。
type Entry struct {
	marketRepo         persistentrepo.MarketRepo
	outcomeRepo        persistentrepo.MarketOutcomeRepo
	bookRepo           persistentrepo.OrderBookRepo
	wsBookRepo         persistentrepo.OrderBookWSRepo
	thresholdRepo      persistentrepo.ConfigStrategyThresholdRepo
	feeRepo            persistentrepo.ConfigStrategyFeeRepo
	oppRepo            persistentrepo.ArbBinaryOpportunityRepo
	orderRepo          persistentrepo.ArbBinaryOrderRepo
	configStrategyRepo persistentrepo.ConfigStrategyRepo
	statRepo           persistentrepo.StatKVRepo
	platformRepo       persistentrepo.PlatformRepo
	accountRepo        persistentrepo.AccountRepo
	poly               httprepo.PolymarketRepo
	kalshi             httprepo.KalshiRepo
	predict            httprepo.PredictRepo

	mu      sync.Mutex
	run     bool               // 为真表示监控循环已启动
	stop    context.CancelFunc // 调用它才会停掉循环
	done    chan struct{}
	watches map[string]*running
	flush   *persist.Flush

	paramBox     atomic.Value
	credBox      atomic.Value
	wsStaggerBox atomic.Value
	// warnKalshiKey 没有 Kalshi 签名账户时只打一次日志。
	warnKalshiKey    atomic.Bool
	detectLatYesno   *DetectLatency
	detectLatOnextwo *DetectLatency
}

type running struct {
	cancel context.CancelFunc
	done   chan struct{}
	sig    string
	mapKey string
	ws     *watch
}

// NewEntry 装上仓库和两家交易所客户端。阈值、账号先放默认空值。
func NewEntry(
	marketRepo persistentrepo.MarketRepo,
	outcomeRepo persistentrepo.MarketOutcomeRepo,
	bookRepo persistentrepo.OrderBookRepo,
	wsBookRepo persistentrepo.OrderBookWSRepo,
	thresholdRepo persistentrepo.ConfigStrategyThresholdRepo,
	feeRepo persistentrepo.ConfigStrategyFeeRepo,
	oppRepo persistentrepo.ArbBinaryOpportunityRepo,
	orderRepo persistentrepo.ArbBinaryOrderRepo,
	configStrategyRepo persistentrepo.ConfigStrategyRepo,
	statRepo persistentrepo.StatKVRepo,
	platformRepo persistentrepo.PlatformRepo,
	accountRepo persistentrepo.AccountRepo,
	poly httprepo.PolymarketRepo,
	kalshi httprepo.KalshiRepo,
	predict httprepo.PredictRepo,
) *Entry {
	e := &Entry{
		marketRepo:         marketRepo,
		outcomeRepo:        outcomeRepo,
		bookRepo:           bookRepo,
		wsBookRepo:         wsBookRepo,
		thresholdRepo:      thresholdRepo,
		feeRepo:            feeRepo,
		oppRepo:            oppRepo,
		orderRepo:          orderRepo,
		configStrategyRepo: configStrategyRepo,
		statRepo:           statRepo,
		platformRepo:       platformRepo,
		accountRepo:        accountRepo,
		poly:               poly,
		kalshi:             kalshi,
		predict:            predict,
		watches:            map[string]*running{},
		detectLatYesno:     NewDetectLatency(),
		detectLatOnextwo:   NewDetectLatency(),
	}
	e.paramBox.Store(defaultParams())
	e.credBox.Store(creds{})
	e.wsStaggerBox.Store(marketmap.WsStaggerMs{MinMs: marketmap.DefaultWsStaggerMinMs, MaxMs: marketmap.DefaultWsStaggerMaxMs})
	return e
}

func (e *Entry) wsStaggerBounds() marketmap.WsStaggerMs {
	if e == nil {
		return marketmap.WsStaggerMs{MinMs: marketmap.DefaultWsStaggerMinMs, MaxMs: marketmap.DefaultWsStaggerMaxMs}
	}
	v := e.wsStaggerBox.Load()
	if v == nil {
		return marketmap.WsStaggerMs{MinMs: marketmap.DefaultWsStaggerMinMs, MaxMs: marketmap.DefaultWsStaggerMaxMs}
	}
	return v.(marketmap.WsStaggerMs)
}

func (e *Entry) predictAPIKey() string {
	c := e.credsNow()
	if c.predictKeyOK {
		return strings.TrimSpace(c.predictAPIKey)
	}
	return ""
}

// canTrade 这家交易所有可用账号才允许发单。
func (e *Entry) canTrade(platform string) bool {
	c := e.credsNow()
	if strings.EqualFold(platform, appcommon.PlatformKalshi) {
		return c.kalshiOK && e.kalshi != nil
	}
	if strings.EqualFold(platform, appcommon.PlatformPredict) {
		return c.predictOK && e.predict != nil
	}
	return c.polyOK && e.poly != nil
}

// Params 读当前阈值。比较卖一时直接拿这份，不再查库。
func (e *Entry) Params() Params {
	v := e.paramBox.Load()
	if v == nil {
		return defaultParams()
	}
	return v.(Params)
}

// Run 启动监控循环。已经在跑时直接返回。
func (e *Entry) Run(context.Context) {
	if e == nil {
		return
	}

	// 卡住只在程序启动的第一次启动
	e.mu.Lock()
	if e.run {
		e.mu.Unlock()
		return
	}

	// 进入循环后，程序不退出的情况下，动态的WS新增和删除，就由下面loop来处理了
	ctx, stop := context.WithCancel(context.Background())
	e.run = true
	e.stop = stop
	e.done = make(chan struct{})
	e.watches = map[string]*running{}
	e.mu.Unlock()
	go e.loop(ctx)
}

// 整个程序停止的时候，需要调用这个来停止全部WS是
func (e *Entry) Stop() {
	if e == nil {
		return
	}
	e.mu.Lock()
	stop := e.stop
	done := e.done
	e.run = false
	e.stop = nil
	e.mu.Unlock()
	if stop != nil {
		stop()
	}
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		notification.SendErrorLog(context.Background(), "strategy_entry-stop-timeout")
	}
}

// loop 先对一次账，之后每 20 秒再对一次。行情不走这里。
func (e *Entry) loop(ctx context.Context) {
	defer close(e.done)
	e.flush = persist.Start(e.bookRepo, e.oppRepo)
	defer e.flush.Stop()
	e.reconcile(ctx)
	tick := time.NewTicker(20 * time.Second)
	hb := time.NewTicker(2 * time.Minute)
	defer tick.Stop()
	defer hb.Stop()
	for {
		select {
		case <-ctx.Done():
			// 等待停止信息
			e.stopWatches()
			return
		case <-tick.C:
			e.reconcile(ctx)
		case <-hb.C:
			e.notifyWSHeartbeat(ctx)
		}
	}
}

// reconcile 刷新阈值和账号。场次没变就接着盯，变了就换掉那一组。
func (e *Entry) reconcile(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	params := e.loadParams(ctx)
	e.paramBox.Store(params)
	e.wsStaggerBox.Store(e.loadWsStagger(ctx))
	e.refreshCreds(ctx)

	// 取比赛信息（同一个比赛多平台分为一组）
	groups, _, err := e.loadGroups(ctx)
	if err != nil {
		notification.SendErrorLog(ctx, "strategy_entry-LoadGroupsFailed", zap.Error(err))
		return
	}
	want := make(map[string]group, len(groups))
	for _, g := range groups {
		want[g.MapKey] = g
	}
	// 已有市场：进行可能会关闭1
	e.mu.Lock()
	removed := 0
	for k, rw := range e.watches {
		g, ok := want[k]
		// 如果比赛全局watches（也就是比赛没结束），且签名OK，就不管继续它跑。
		// 否则就delete(e.watches, k)
		if ok && signature(g) == rw.sig {
			continue
		}
		// 关闭ws
		rw.cancel()
		// 全局watches中删除
		delete(e.watches, k)
		removed++
	}

	// 新增比赛：进行可能的开启。
	added := 0
	for k, g := range want {
		if _, ok := e.watches[k]; ok {
			continue
		}
		// 新增不在全局watches中的比赛，进行开启监控
		wctx, cancel := context.WithCancel(ctx)
		rw := &running{cancel: cancel, done: make(chan struct{}), sig: signature(g), mapKey: k}
		e.watches[k] = rw
		go e.runWatch(wctx, rw, g)
		added++
	}

	// 消息通知
	running := len(e.watches)
	e.mu.Unlock()
	notifyWatchSummary(ctx, removed, added, running)
}

// notifyWSHeartbeat 每 5 分钟对比库内应对场次与各平台应连 WS，汇报实际心跳正常数及差异。
func (e *Entry) notifyWSHeartbeat(ctx context.Context) {
	if e == nil {
		return
	}
	platforms := []string{
		appcommon.PlatformPolymarket,
		appcommon.PlatformKalshi,
		appcommon.PlatformPredict,
	}
	shouldGroups, matched, loadErr := e.loadGroups(ctx)
	shouldN := len(shouldGroups)
	shouldWS := expectedWSCounts(e, shouldGroups)

	type tally struct {
		inRun, ok int
	}
	byPlat := map[string]tally{}
	for _, p := range platforms {
		byPlat[p] = tally{}
	}
	wantByKey := make(map[string]group, len(shouldGroups))
	for _, g := range shouldGroups {
		wantByKey[g.MapKey] = g
	}
	runningN := 0
	e.mu.Lock()
	for _, rw := range e.watches {
		runningN++
		for _, p := range platforms {
			exp := false
			healthy := false
			if rw.ws != nil {
				exp, healthy = rw.ws.wsLinkOK(p)
			} else if g, ok := wantByKey[rw.mapKey]; ok {
				exp = outcomesPlatformWSExpected(e, g.Outcomes, p)
			}
			if !exp {
				continue
			}
			t := byPlat[p]
			t.inRun++
			if healthy {
				t.ok++
			}
			byPlat[p] = t
		}
	}
	e.mu.Unlock()

	allOK := true
	fields := []notification.FieldPair{
		{Key: "应对场次(库)", Value: strconv.Itoa(shouldN)},
		{Key: "在跑场次", Value: strconv.Itoa(runningN)},
	}
	if loadErr != nil {
		fields = append(fields, notification.FieldPair{Key: "加载场次", Value: "失败"})
		allOK = false
	} else if shouldN != runningN {
		fields = append(fields, notification.FieldPair{
			Key:   "场次差异",
			Value: fmt.Sprintf("在跑%d 应对%d (差%d)", runningN, shouldN, runningN-shouldN),
		})
		allOK = false
	} else {
		fields = append(fields, notification.FieldPair{Key: "场次差异", Value: "无"})
	}
	if matched > shouldN {
		fields = append(fields, notification.FieldPair{
			Key:   "库内匹配",
			Value: fmt.Sprintf("共%d组(仅跑前%d)", matched, shouldN),
		})
	}

	sumShould, sumOK := 0, 0
	for _, p := range platforms {
		want := shouldWS[p]
		t := byPlat[p]
		sumShould += want
		sumOK += t.ok
		val := "未配置"
		if want > 0 || t.inRun > 0 {
			diff := want - t.ok
			if want != t.inRun {
				allOK = false
			}
			if diff != 0 {
				allOK = false
			}
			val = fmt.Sprintf("应%d 在连%d 正常%d", want, t.inRun, t.ok)
			if want != t.inRun || diff != 0 {
				val += fmt.Sprintf(" (缺%d)", want-t.ok)
			}
		}
		fields = append(fields, notification.FieldPair{Key: p + " WS", Value: val})
	}
	fields = append(fields, notification.FieldPair{
		Key:   "WS 合计",
		Value: fmt.Sprintf("应%d 正常%d", sumShould, sumOK),
	})
	if sumShould != sumOK {
		allOK = false
	}

	status := "心跳正常"
	if shouldN == 0 && runningN == 0 {
		status = "无监控场次"
		allOK = false
	} else if !allOK {
		status = "有差异或异常"
	}
	fields = append(fields, notification.FieldPair{Key: "状态", Value: status})

	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "strategy_entry WS 心跳", fields)
	e.flushMarketRunningCountStrategy(ctx, wantByKey)
	e.flushDetectLatencyStat(ctx)
}

func (e *Entry) flushMarketRunningCountStrategy(ctx context.Context, wantByKey map[string]group) {
	if e == nil {
		return
	}
	entrystat.FlushMarketRunningCountStrategy(ctx, entrystat.MarketRunningOpts{
		StatRepo:            e.statRepo,
		PreEventScanMinutes:   LoadPreEventScanMinutes(ctx, e.configStrategyRepo),
		EndEventScanMinutes:   LoadEndEventScanMinutes(ctx, e.configStrategyRepo),
		Groups:                e.runningMarketGroupsForStat(wantByKey),
	})
}

func (e *Entry) runningMarketGroupsForStat(wantByKey map[string]group) []entrystat.Group {
	e.mu.Lock()
	keys := make([]string, 0, len(e.watches))
	for k := range e.watches {
		keys = append(keys, k)
	}
	e.mu.Unlock()
	sort.Strings(keys)
	out := make([]entrystat.Group, 0, len(keys))
	for _, k := range keys {
		g, ok := wantByKey[k]
		if !ok {
			out = append(out, entrystat.Group{MapKey: k})
			continue
		}
		out = append(out, entrystat.Group{MapKey: k, MarketRowIDs: marketRowIDsFromGroup(g)})
	}
	return out
}

func marketRowIDsFromGroup(g group) []uint {
	if len(g.MarketsByRowID) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(g.MarketsByRowID))
	for id := range g.MarketsByRowID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func notifyWatchSummary(ctx context.Context, removed, added, running int) {
	if removed == 0 && added == 0 {
		return
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, "strategy_entry 场次监控", []notification.FieldPair{
		{Key: "关闭WS", Value: strconv.Itoa(removed)},
		{Key: "新增场次", Value: strconv.Itoa(added)},
		{Key: "当前在跑", Value: strconv.Itoa(running)},
	})
}

// stopWatches 取消全部组协程，并等到它们退出。
func (e *Entry) stopWatches() {
	e.mu.Lock()
	watches := e.watches
	e.watches = map[string]*running{}
	e.mu.Unlock()
	for _, rw := range watches {
		rw.cancel()
	}
	for _, rw := range watches {
		<-rw.done
	}
}

// runWatch 盯一组 map_key，直到这组被换掉或进程停。
func (e *Entry) runWatch(ctx context.Context, rw *running, g group) {
	defer close(rw.done)
	defer func() {
		if r := recover(); r != nil {
			notification.SendErrorLog(ctx, "strategy_entry-group-panic",
				zap.Any("panic", r),
				zap.String("map_key", g.MapKey))
		}
	}()

	w := newWatch(ctx, e, g)
	e.mu.Lock()
	if cur, ok := e.watches[g.MapKey]; ok {
		cur.ws = w
	}
	e.mu.Unlock()

	w.run()

	e.mu.Lock()
	if cur, ok := e.watches[g.MapKey]; ok && cur.ws == w {
		cur.ws = nil
	}
	e.mu.Unlock()
}
