package noderisk

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/noderoute"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// Guard 节点 relay 风控：断链、内网 RTT、镜像 BOOK 活性。热路径 Allow 无锁读。
type Guard struct {
	ctx        context.Context
	cancel     context.CancelFunc
	nodeCode   string
	prof       *noderoute.Profile
	threshold  persistentrepo.ConfigStrategyThresholdRepo
	cfg        atomic.Value // Config
	platforms  []string
	startedAt  time.Time

	mu          sync.RWMutex
	plat        map[string]*platState
	lastAlertAt map[string]time.Time
}

type platState struct {
	platform      string
	tripped       bool
	reason        string
	connUp        bool
	peerLinks     int // 入站+出站 TCP 计数，避免单连接断开误熔断
	lastBookRx    time.Time
	gotBook       bool
	lastRTT       time.Duration
	okStreak      int
	lastTripAt    time.Time
}

func NewGuard(parent context.Context, prof *noderoute.Profile, threshold persistentrepo.ConfigStrategyThresholdRepo, nodeCode string) *Guard {
	if prof == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	g := &Guard{
		ctx:         ctx,
		cancel:      cancel,
		nodeCode:    strings.TrimSpace(nodeCode),
		prof:        prof,
		threshold:   threshold,
		startedAt:   time.Now(),
		plat:        map[string]*platState{},
		lastAlertAt: map[string]time.Time{},
	}
	g.cfg.Store(LoadConfig(ctx, threshold))
	g.platforms = mirrorPlatforms(prof)
	for _, p := range g.platforms {
		g.plat[p] = &platState{platform: p}
	}
	return g
}

func mirrorPlatforms(prof *noderoute.Profile) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range prof.Edges {
		p := strings.ToUpper(strings.TrimSpace(e.MirrorPlatform))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// Start 周期评估 + 配置刷新。
func (g *Guard) Start() {
	if g == nil {
		return
	}
	go g.run()
}

func (g *Guard) Stop() {
	if g == nil || g.cancel == nil {
		return
	}
	g.cancel()
}

func (g *Guard) Config() Config {
	if g == nil {
		return defaultConfig()
	}
	v := g.cfg.Load()
	if v == nil {
		return defaultConfig()
	}
	return v.(Config)
}

func (g *Guard) Reload(ctx context.Context) {
	if g == nil || g.threshold == nil {
		return
	}
	g.cfg.Store(LoadConfig(ctx, g.threshold))
}

// Allow 镜像/对端平台是否允许参与策略（本机官方平台恒 true）。
func (g *Guard) Allow(platform string) bool {
	if g == nil || g.prof == nil {
		return true
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	if g.prof.IsLocalPlatform(platform) {
		return true
	}
	if !g.Config().Enabled {
		return true
	}
	if g.prof.EdgeForPlatform(platform) == nil {
		return true
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	st, ok := g.plat[platform]
	if !ok {
		return true
	}
	return !st.tripped
}

// Hub 回调 ------------------------------------------------------------------

func (g *Guard) OnPeerConnUp(remoteAddr string) {
	plat := g.prof.PlatformForPeerAddr(remoteAddr)
	if plat == "" {
		return
	}
	cfg := g.Config()
	g.mu.Lock()
	st := g.platState(plat)
	st.peerLinks++
	st.connUp = st.peerLinks > 0
	g.mu.Unlock()
	if cfg.Enabled {
		g.bumpOK(plat)
	}
}

func (g *Guard) OnPeerConnDown(remoteAddr string) {
	plat := g.prof.PlatformForPeerAddr(remoteAddr)
	if plat == "" {
		return
	}
	cfg := g.Config()
	g.mu.Lock()
	st := g.platState(plat)
	if st.peerLinks > 0 {
		st.peerLinks--
	}
	st.connUp = st.peerLinks > 0
	links := st.peerLinks
	g.mu.Unlock()
	if cfg.Enabled && links == 0 {
		g.trip(plat, "risk_peer_tcp_down", map[string]string{"peer": remoteAddr})
	}
}

func (g *Guard) OnPeerRTT(remoteAddr string, rtt time.Duration) {
	plat := g.prof.PlatformForPeerAddr(remoteAddr)
	if plat == "" {
		return
	}
	cfg := g.Config()
	g.mu.Lock()
	st := g.platState(plat)
	st.lastRTT = rtt
	tripped := st.tripped
	reason := st.reason
	g.mu.Unlock()

	if !cfg.Enabled {
		return
	}
	if rtt > cfg.MaxRTT {
		g.trip(plat, "risk_rtt_high", map[string]string{
			"rtt_ms": formatMs(rtt),
			"max_ms": formatMs(cfg.MaxRTT),
			"peer":   remoteAddr,
		})
		return
	}
	if tripped && strings.Contains(reason, "rtt") {
		g.bumpOK(plat)
	}
}

// RecordMirrorBook 对端推送的镜像簿到达（任意 TCP 方向）。
func (g *Guard) RecordMirrorBook(platform string) {
	if g == nil {
		return
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	now := time.Now()
	g.mu.Lock()
	st := g.platState(platform)
	st.lastBookRx = now
	st.gotBook = true
	g.mu.Unlock()
}

func (g *Guard) platState(platform string) *platState {
	st, ok := g.plat[platform]
	if !ok {
		st = &platState{platform: platform}
		g.plat[platform] = st
	}
	return st
}

func (g *Guard) run() {
	eval := time.NewTicker(time.Second)
	defer eval.Stop()
	var cfgReload <-chan time.Time
	var reloadTick *time.Ticker
	g.resetReloadTicker(&reloadTick, &cfgReload)
	defer func() {
		if reloadTick != nil {
			reloadTick.Stop()
		}
	}()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-eval.C:
			g.evaluate()
		case <-cfgReload:
			g.Reload(g.ctx)
			g.resetReloadTicker(&reloadTick, &cfgReload)
		}
	}
}

func (g *Guard) resetReloadTicker(t **time.Ticker, ch *<-chan time.Time) {
	if *t != nil {
		(*t).Stop()
	}
	*t = time.NewTicker(g.Config().CheckInterval)
	*ch = (*t).C
}

func (g *Guard) evaluate() {
	cfg := g.Config()
	if !cfg.Enabled {
		return
	}
	now := time.Now()
	grace := cfg.BookStale
	if grace < 2*cfg.BookPushInterval {
		grace = 2 * cfg.BookPushInterval
	}

	for _, plat := range g.platforms {
		g.mu.Lock()
		st := g.platState(plat)
		tripped := st.tripped
		gotBook := st.gotBook
		lastBook := st.lastBookRx
		connUp := st.connUp
		rtt := st.lastRTT
		g.mu.Unlock()

		if !connUp && tripped {
			continue
		}
		if !connUp && !tripped {
			// conn 由 OnPeerConnDown 即时处理
			continue
		}

		stale := false
		if gotBook {
			stale = now.Sub(lastBook) > cfg.BookStale
		} else if now.Sub(g.startedAt) > grace {
			stale = true
		}
		if stale && !tripped {
			g.trip(plat, "risk_book_push_stale", map[string]string{
				"stale_sec": strconv.FormatInt(int64(cfg.BookStale.Seconds()), 10),
			})
			continue
		}

		if tripped {
			ok := connUp && !stale && (rtt == 0 || rtt <= cfg.MaxRTT)
			if ok {
				g.bumpOK(plat)
			}
		}
	}
}

func (g *Guard) bumpOK(platform string) {
	cfg := g.Config()
	g.mu.Lock()
	st := g.platState(platform)
	if !st.tripped {
		st.okStreak = 0
		g.mu.Unlock()
		return
	}
	st.okStreak++
	need := cfg.RecoverOKSamples
	if need <= 0 {
		need = 3
	}
	if st.okStreak >= need && st.connUp {
		g.mu.Unlock()
		g.recover(platform)
		return
	}
	g.mu.Unlock()
}

func (g *Guard) trip(platform, reason string, extra map[string]string) {
	g.mu.Lock()
	st := g.platState(platform)
	if st.tripped && st.reason == reason {
		g.mu.Unlock()
		return
	}
	was := st.tripped
	st.tripped = true
	st.reason = reason
	st.okStreak = 0
	st.lastTripAt = time.Now()
	g.mu.Unlock()

	if was {
		return
	}
	g.alert(platform, false, reason, extra)
}

func (g *Guard) recover(platform string) {
	g.mu.Lock()
	st := g.platState(platform)
	if !st.tripped {
		g.mu.Unlock()
		return
	}
	prevReason := st.reason
	st.tripped = false
	st.reason = ""
	st.okStreak = 0
	g.mu.Unlock()
	g.alert(platform, true, prevReason, nil)
}

func (g *Guard) alert(platform string, recovered bool, reason string, extra map[string]string) {
	cfg := g.Config()
	g.mu.Lock()
	last := g.lastAlertAt[platform]
	if !recovered && !last.IsZero() && time.Since(last) < cfg.AlertRepeat {
		g.mu.Unlock()
		return
	}
	if !recovered {
		g.lastAlertAt[platform] = time.Now()
	}
	g.mu.Unlock()

	title := "relay风控触发"
	if recovered {
		title = "relay风控恢复"
	}
	fields := []notification.FieldPair{
		{Key: "node", Value: g.nodeCode},
		{Key: "platform", Value: platform},
		{Key: "reason", Value: reason},
	}
	for k, v := range extra {
		fields = append(fields, notification.FieldPair{Key: k, Value: v})
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(g.ctx, title, fields)
}

func formatMs(d time.Duration) string {
	return strconv.FormatInt(d.Milliseconds(), 10)
}

// HeartbeatLine 供 WS 心跳汇报：镜像平台 relay 状态（无 DB）。
func (g *Guard) HeartbeatLine(platform string) string {
	if g == nil || g.prof == nil {
		return ""
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	if g.prof.IsLocalPlatform(platform) || g.prof.EdgeForPlatform(platform) == nil {
		return ""
	}
	cfg := g.Config()
	g.mu.RLock()
	st, ok := g.plat[platform]
	g.mu.RUnlock()
	if !ok || st == nil {
		return "无状态"
	}
	if !cfg.Enabled {
		return "风控未启用"
	}
	if st.tripped {
		return "熔断(" + st.reason + ")"
	}
	if !st.gotBook {
		return "未收到镜像BOOK"
	}
	age := time.Since(st.lastBookRx)
	if age > cfg.BookStale {
		return fmt.Sprintf("BOOK陈旧 %ds", int(age.Seconds()))
	}
	rtt := "RTT未测"
	if st.lastRTT > 0 {
		rtt = formatMs(st.lastRTT) + "ms"
	}
	conn := "TCP断"
	if st.connUp || st.peerLinks > 0 {
		conn = "TCP连"
	}
	return fmt.Sprintf("%s 末BOOK %ds前 %s", conn, int(age.Seconds()), rtt)
}