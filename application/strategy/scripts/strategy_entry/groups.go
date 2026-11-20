package strategyentry

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// creds 下单身份。在对账协程里刷新，回调里只读。
type creds struct {
	poly          httpentity.PolymarketAccount
	polyOK        bool
	kalshi        httpentity.KalshiAccount
	kalshiOK      bool
	predict       httpentity.PredictAccount
	predictOK     bool
	predictAPIKey string
	predictKeyOK  bool
}

const (
	TaskName                     = "strategy_entry"
	defaultMaxGroups             = 50
	thresholdMaxCost             = "binary_yes_no_max_cost"
	thresholdOrderAmount         = "binary_yes_no_order_amount"
	thresholdLevelFactor         = "binary_yes_no_level_factor"
	threshold1X2YesMax           = "1x2_yes_triple_max_cost"
	threshold1X2NoMax            = "1x2_no_triple_max_cost"
	defaultMaxCost               = 0.95
	default1X2YesMax             = 0.95
	default1X2NoMax              = 1.90
	defaultOrderAmount           = 100.0
	defaultLevelFactor           = 2.0
	defaultPolySports            = 0.05
	defaultKalshiSports          = 0.07
	defaultPredictSports         = 0.02
	configKeyPreEventScanMinutes = "pre_event_scan_minutes"
	defaultPreEventScanMinutes   = 30
)

// Params 5.2 的可配置门槛。手续费按平台和运动放在 fees 里，比较时只读。
type Params struct {
	MaxCost     float64
	OrderAmount float64
	LevelFactor float64
	OneX2YesMax float64
	OneX2NoMax  float64
	fees        map[string]float64
}

// defaultParams 库里没有门槛时用的默认值。体育吃单系数按两家现行价目。
func defaultParams() Params {
	return Params{
		MaxCost:     defaultMaxCost,
		OrderAmount: defaultOrderAmount,
		LevelFactor: defaultLevelFactor,
		OneX2YesMax: default1X2YesMax,
		OneX2NoMax:  default1X2NoMax,
		fees: map[string]float64{
			feeKey(appcommon.PlatformPolymarket, "Sports", ""): defaultPolySports,
			feeKey(appcommon.PlatformKalshi, "Sports", ""):     defaultKalshiSports,
			feeKey(appcommon.PlatformPredict, "Sports", ""):    defaultPredictSports,
		},
	}
}

// Fee 取这场的吃单系数。先运动，再该平台体育默认，最后按平台兜底。
func (p Params) Fee(platform, category, sub string) float64 {
	if p.fees != nil {
		if v, ok := p.fees[feeKey(platform, category, sub)]; ok {
			return v
		}
		if v, ok := p.fees[feeKey(platform, category, "")]; ok {
			return v
		}
		if v, ok := p.fees[feeKey(platform, "", "")]; ok {
			return v
		}
	}
	if strings.EqualFold(platform, appcommon.PlatformKalshi) {
		return defaultKalshiSports
	}
	if strings.EqualFold(category, "Sports") || strings.TrimSpace(category) == "" {
		return defaultPolySports
	}
	return 0
}

func feeKey(platform, category, sub string) string {
	return strings.ToUpper(strings.TrimSpace(platform)) + "|" + strings.TrimSpace(category) + "|" + strings.TrimSpace(sub)
}

// LoadParams 读策略阈值与手续费（回测等非 WS 路径）。
func LoadParams(ctx context.Context, thresholdRepo persistentrepo.ConfigStrategyThresholdRepo, feeRepo persistentrepo.ConfigStrategyFeeRepo) Params {
	e := &Entry{thresholdRepo: thresholdRepo, feeRepo: feeRepo}
	return e.loadParams(ctx)
}

// loadParams 从阈值表读成本上限和数量，从手续费表读各运动系数。
func (e *Entry) loadParams(ctx context.Context) Params {
	p := defaultParams()
	p.MaxCost = e.threshold(ctx, thresholdMaxCost, p.MaxCost)
	p.OrderAmount = e.threshold(ctx, thresholdOrderAmount, p.OrderAmount)
	p.LevelFactor = e.threshold(ctx, thresholdLevelFactor, p.LevelFactor)
	p.OneX2YesMax = e.threshold(ctx, threshold1X2YesMax, p.OneX2YesMax)
	p.OneX2NoMax = e.threshold(ctx, threshold1X2NoMax, p.OneX2NoMax)
	p.fees = e.loadFees(ctx, p.fees)
	if p.OrderAmount <= 0 {
		p.OrderAmount = defaultOrderAmount
	}
	if p.LevelFactor <= 0 {
		p.LevelFactor = defaultLevelFactor
	}
	if p.MaxCost <= 0 {
		p.MaxCost = defaultMaxCost
	}
	if p.OneX2YesMax <= 0 {
		p.OneX2YesMax = default1X2YesMax
	}
	if p.OneX2NoMax <= 0 {
		p.OneX2NoMax = default1X2NoMax
	}
	return p
}

// loadFees 把启用中的手续费行盖到默认系数上。比较路径只读这份 map。
func (e *Entry) loadFees(ctx context.Context, base map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(base)+8)
	for k, v := range base {
		out[k] = v
	}
	if e.feeRepo == nil {
		return out
	}
	rows, err := e.feeRepo.List(ctx, nil, map[string]interface{}{"status = ?": 1}, "")
	if err != nil {
		return out
	}
	for _, row := range rows {
		coef := row.TakerFeeCoef.InexactFloat64()
		if coef < 0 {
			continue
		}
		out[feeKey(row.Platform, row.Category, row.SubCategory)] = coef
	}
	return out
}

// threshold 读一个启用中的阈值，读不到就用默认值。
func (e *Entry) threshold(ctx context.Context, key string, def float64) float64 {
	if e.thresholdRepo == nil {
		return def
	}
	row, err := e.thresholdRepo.FindOne(ctx, nil, map[string]interface{}{
		"config_key = ?": key,
		"status = ?":     1,
	}, "")
	if err != nil {
		return def
	}
	return nonNegative(row.ConfigValue, def)
}

// nonNegative 负数当无效，退回默认值。
func nonNegative(v decimal.Decimal, def float64) float64 {
	f := v.InexactFloat64()
	if f < 0 {
		return def
	}
	return f
}

type group struct {
	MapKey         string
	Volume         float64
	Outcomes       []persistententity.MarketOutcome
	MarketsByRowID map[uint]persistententity.Market
}

// LoadMaxGroups 读 config_strategy.strategy_entry.max_groups（与实盘一致）。
func LoadMaxGroups(ctx context.Context, configStrategyRepo persistentrepo.ConfigStrategyRepo) int {
	if configStrategyRepo == nil {
		return defaultMaxGroups
	}
	cfg, err := configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": marketmap.ConfigGroupStrategyEntry,
		"item_key = ?":   marketmap.ConfigKeyMaxGroups,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "strategy_entry-max-groups-read-failed", zap.Error(err))
		}
		return defaultMaxGroups
	}
	return marketmap.ParseMaxGroups(cfg.ItemValue, defaultMaxGroups)
}

// loadMaxGroups 读 config_strategy.strategy_entry.max_groups。
func (e *Entry) loadMaxGroups(ctx context.Context) int {
	return LoadMaxGroups(ctx, e.configStrategyRepo)
}

// LoadPreEventScanMinutes 读 config_strategy.strategy_entry.pre_event_scan_minutes。
func LoadPreEventScanMinutes(ctx context.Context, configStrategyRepo persistentrepo.ConfigStrategyRepo) int {
	preEventScanMinutes := defaultPreEventScanMinutes
	if configStrategyRepo == nil {
		return preEventScanMinutes
	}
	cfg, err := configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": marketmap.ConfigGroupStrategyEntry,
		"item_key = ?":   configKeyPreEventScanMinutes,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			notification.SendErrorLog(ctx, "strategy_entry-pre-event-scan-minutes-read-failed", zap.Error(err))
		}
		return preEventScanMinutes
	}
	n, err := strconv.Atoi(strings.TrimSpace(cfg.ItemValue))
	if err != nil {
		return preEventScanMinutes
	}
	return n
}

// WatchGroupCounts 与 strategy_entry.loadGroups 同源的场次计数（stat_collect / 心跳应对场次）。
type WatchGroupCounts struct {
	Matched   int // 合规且有 outcome，未截 max_groups
	Watchable int // 按成交额取前 max_groups，应对监控组数
	MaxGroups int // 当前 max_groups 配置
}

// WatchGroupDeps loadGroups 所需仓储。
type WatchGroupDeps struct {
	MarketRepo         persistentrepo.MarketRepo
	OutcomeRepo        persistentrepo.MarketOutcomeRepo
	ConfigStrategyRepo persistentrepo.ConfigStrategyRepo
	PlatformRepo       persistentrepo.PlatformRepo
}

// mapKeyWithinPreEventScan preEventScanMinutes>0 时：同 map_key 最早 event_start_time <= now+N 分钟（UTC）才纳入；0 不滤。
func mapKeyWithinPreEventScan(rows []persistententity.Market, now time.Time, preEventScanMinutes int) bool {
	if preEventScanMinutes <= 0 {
		return true
	}
	var start time.Time
	hasStart := false
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"}
	for i := range rows {
		if rows[i].EventStartTime == nil {
			continue
		}
		raw := strings.TrimSpace(*rows[i].EventStartTime)
		if raw == "" {
			continue
		}
		var t time.Time
		parsed := false
		for _, layout := range layouts {
			if pt, err := time.Parse(layout, raw); err == nil {
				t, parsed = pt.UTC(), true
				break
			}
		}
		if !parsed {
			continue
		}
		if !hasStart || t.Before(start) {
			start, hasStart = t, true
		}
	}
	if !hasStart {
		return false
	}
	return !start.After(now.UTC().Add(time.Duration(preEventScanMinutes) * time.Minute))
}

func runnableKeysFromFixtures(ctx context.Context, fixtures map[uint]persistententity.Market, alertBadKeys bool, now time.Time, preEventScanMinutes int) map[string]struct{} {
	rowsByKey := map[string][]persistententity.Market{}
	for _, fx := range fixtures {
		if fx.MapKey == nil || fx.MapVerified != 1 {
			continue
		}
		k := strings.TrimSpace(*fx.MapKey)
		if k == "" {
			continue
		}
		rowsByKey[k] = append(rowsByKey[k], fx)
	}
	runnableKeys := map[string]struct{}{}
	for k, rows := range rowsByKey {
		if dup, plat, ids := marketmap.MapKeyDuplicatePlatform(rows); dup {
			if alertBadKeys {
				alertMapKeyBad(ctx, "dup", k, plat, ids)
			}
			continue
		}
		if !marketmap.MapKeyHasTwoPlatforms(rows) {
			continue
		}
		if !mapKeyWithinPreEventScan(rows, now, preEventScanMinutes) {
			continue
		}
		runnableKeys[k] = struct{}{}
	}
	return runnableKeys
}

// loadWsStagger 读 config_strategy.strategy_entry.ws_connect_stagger_ms（毫秒）。
func (e *Entry) loadWsStagger(ctx context.Context) marketmap.WsStaggerMs {
	def := marketmap.WsStaggerMs{MinMs: marketmap.DefaultWsStaggerMinMs, MaxMs: marketmap.DefaultWsStaggerMaxMs}
	if e.configStrategyRepo == nil {
		return def
	}
	cfg, err := e.configStrategyRepo.FindOne(ctx, nil, map[string]interface{}{
		"group_name = ?": marketmap.ConfigGroupStrategyEntry,
		"item_key = ?":   marketmap.ConfigKeyWsConnectStaggerMs,
		"is_enabled = ?": 1,
	}, "")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			notification.SendErrorLog(ctx, "strategy_entry-ws-stagger-read-failed", zap.Error(err))
		}
		return def
	}
	return marketmap.ParseWsStaggerMs(cfg.ItemValue)
}

// loadGroups 取出已核对场次，按成交额留下最多 max_groups 组（config_strategy）。
func (e *Entry) loadGroups(ctx context.Context) ([]group, int, error) {
	d := WatchGroupDeps{
		MarketRepo:         e.marketRepo,
		OutcomeRepo:        e.outcomeRepo,
		ConfigStrategyRepo: e.configStrategyRepo,
		PlatformRepo:       e.platformRepo,
	}
	return loadWatchGroups(ctx, d, true)
}

func loadWatchGroups(ctx context.Context, d WatchGroupDeps, alertBadKeys bool) ([]group, int, error) {
	fixtures, err := marketmap.LoadVerifiedFixturesNew(ctx, d.MarketRepo, d.ConfigStrategyRepo, d.PlatformRepo)
	if err != nil {
		return nil, 0, err
	}
	if d.OutcomeRepo == nil || len(fixtures) == 0 {
		return nil, 0, nil
	}
	marketIDs := make([]uint, 0, len(fixtures))
	for id := range fixtures {
		marketIDs = append(marketIDs, id)
	}
	_, raw, err := d.OutcomeRepo.List(ctx, nil, map[string]interface{}{
		"market_row_id IN ?": marketIDs,
	}, "id asc", "")
	if err != nil {
		return nil, 0, err
	}

	preEventScanMinutes := LoadPreEventScanMinutes(ctx, d.ConfigStrategyRepo)
	runnableKeys := runnableKeysFromFixtures(ctx, fixtures, alertBadKeys, time.Now().UTC(), preEventScanMinutes)

	// 获取所有市场的outcomes盘口信息（多平台的）
	outcomes := marketmap.FilterOutcomesUnderVerifiedFixtures(raw, fixtures)
	byKey := map[string]*group{}

	for _, fx := range fixtures {
		if fx.MapKey == nil {
			continue
		}
		k := strings.TrimSpace(*fx.MapKey)
		if k == "" || fx.MapVerified != 1 {
			continue
		}
		if _, ok := runnableKeys[k]; !ok {
			continue
		}
		g := byKey[k]
		if g == nil {
			g = &group{MapKey: k}
			byKey[k] = g
		}
		if fx.VolumeTotal != nil {
			g.Volume += *fx.VolumeTotal
		}
	}

	for i := range outcomes {
		oc := outcomes[i]
		fx, ok := fixtures[oc.MarketRowID]
		if !ok || fx.MapKey == nil {
			continue
		}
		k := strings.TrimSpace(*fx.MapKey)
		if _, ok := runnableKeys[k]; !ok {
			continue
		}
		g := byKey[k]
		if g == nil {
			continue
		}
		g.Outcomes = append(g.Outcomes, oc)
		if g.MarketsByRowID == nil {
			g.MarketsByRowID = make(map[uint]persistententity.Market)
		}
		g.MarketsByRowID[oc.MarketRowID] = fx
	}
	list := make([]group, 0, len(byKey))
	for _, g := range byKey {
		if len(g.Outcomes) == 0 {
			continue
		}
		list = append(list, *g)
	}
	sortGroups(list)
	matched := len(list)
	limit := LoadMaxGroups(ctx, d.ConfigStrategyRepo)
	if len(list) > limit {
		list = list[:limit]
	}
	return list, matched, nil
}

// sortGroups 成交额高的排前面，相同则按 map_key。
func sortGroups(list []group) {
	for i := 1; i < len(list); i++ {
		j := i
		for j > 0 && groupLess(list[j], list[j-1]) {
			list[j], list[j-1] = list[j-1], list[j]
			j--
		}
	}
}

// groupLess 比较两组谁该排前面。
func groupLess(a, b group) bool {
	if a.Volume == b.Volume {
		return a.MapKey < b.MapKey
	}
	return a.Volume > b.Volume
}

// signature 把一组盘口拼成签名，用来判断要不要换掉这组连接。
func signature(g group) string {
	parts := make([]string, 0, len(g.Outcomes))
	for _, oc := range g.Outcomes {
		parts = append(parts, strings.Join([]string{
			oc.Platform,
			oc.PlatformMarketOutcomeID,
			oc.YesTokenID,
			oc.NoTokenID,
		}, "|"))
	}
	for i := 1; i < len(parts); i++ {
		j := i
		for j > 0 && parts[j] < parts[j-1] {
			parts[j], parts[j-1] = parts[j-1], parts[j]
			j--
		}
	}
	return strings.Join(parts, ",")
}

var mapKeyBadOnce sync.Map

func alertMapKeyBad(ctx context.Context, kind, mapKey, platform string, marketIDs []uint) {
	key := kind + "|" + mapKey + "|" + platform
	if _, loaded := mapKeyBadOnce.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	ids := make([]string, 0, len(marketIDs))
	for _, id := range marketIDs {
		ids = append(ids, strconv.FormatUint(uint64(id), 10))
	}
	msg := "strategy_entry-map-key-dup-platform"
	if kind != "dup" {
		msg = "strategy_entry-map-key-bad"
	}
	notification.SendErrorLog(ctx, msg,
		zap.String("map_key", mapKey),
		zap.String("platform", platform),
		zap.String("reason", kind),
		zap.Strings("market_ids", ids))
}

// refreshCreds 每家取一条 ACTIVE 账号，供发单时直接使用。
func (e *Entry) refreshCreds(ctx context.Context) {
	c := creds{}
	if e.accountRepo == nil {
		e.credBox.Store(c)
		return
	}
	_, list, err := e.accountRepo.List(ctx, nil, map[string]interface{}{
		"status = ?": "ACTIVE",
	}, "id asc", "")
	if err != nil {
		return
	}
	for i := range list {
		a := list[i]
		switch strings.ToUpper(strings.TrimSpace(a.Platform)) {
		case appcommon.PlatformKalshi:
			if c.kalshiOK {
				continue
			}
			acc, ok := kalshiCred(a)
			if !ok {
				continue
			}
			c.kalshi = acc
			c.kalshiOK = true
		case appcommon.PlatformPolymarket:
			if c.polyOK {
				continue
			}
			acc, ok := polyCred(a)
			if !ok {
				continue
			}
			c.poly = acc
			c.polyOK = true
		case appcommon.PlatformPredict:
			if k := deref(a.APIKey); k != "" && !c.predictKeyOK {
				c.predictAPIKey = k
				c.predictKeyOK = true
			}
			if c.predictOK {
				continue
			}
			if acc, ok := predictCred(a); ok {
				c.predict = acc
				c.predictOK = true
				if !c.predictKeyOK && acc.APIKey != "" {
					c.predictAPIKey = acc.APIKey
					c.predictKeyOK = true
				}
			}
		}
	}
	e.credBox.Store(c)
}

// credsNow 读当前账号。发单路径只读这份，不查库。
func (e *Entry) credsNow() creds {
	v := e.credBox.Load()
	if v == nil {
		return creds{}
	}
	return v.(creds)
}

// polyCred 拼出 Polymarket 下单身份。密钥或地址缺了就不可用。
func polyCred(a persistententity.Account) (httpentity.PolymarketAccount, bool) {
	sig := 0
	if a.SignatureType != nil {
		sig = *a.SignatureType
	}
	acc := httpentity.PolymarketAccount{
		APIKey:        deref(a.APIKey),
		APISecret:     deref(a.APISecretEnc),
		APIPassphrase: deref(a.APIPassphraseEnc),
		EOAAddress:    deref(a.EoaAddress),
		ProxyAddress:  deref(a.ProxyAddress),
		PrivateKey:    deref(a.PrivateKeyEnc),
		SignatureType: sig,
	}
	if acc.APIKey == "" || acc.APISecret == "" || acc.APIPassphrase == "" || acc.PrivateKey == "" {
		return acc, false
	}
	if acc.ProxyAddress == "" && acc.EOAAddress == "" {
		return acc, false
	}
	return acc, true
}

// predictCred Predict 发单：api_key + 私钥；proxy_address 为 Predict 智能账户（可选）。
func predictCred(a persistententity.Account) (httpentity.PredictAccount, bool) {
	acc := httpentity.PredictAccount{
		APIKey:         deref(a.APIKey),
		PrivateKey:     deref(a.PrivateKeyEnc),
		PredictAccount: deref(a.ProxyAddress),
		EOAAddress:     deref(a.EoaAddress),
	}
	if acc.APIKey == "" || acc.PrivateKey == "" {
		return acc, false
	}
	return acc, true
}

// kalshiCred 拼出 Kalshi 签名身份。key 或私钥缺了就不可用。
func kalshiCred(a persistententity.Account) (httpentity.KalshiAccount, bool) {
	acc := httpentity.KalshiAccount{
		KeyID:      deref(a.APIKey),
		PrivateKey: deref(a.APISecretEnc),
		Passphrase: deref(a.APIPassphraseEnc),
	}
	if acc.KeyID == "" || acc.PrivateKey == "" {
		return acc, false
	}
	return acc, true
}

// deref 取出字符串并去掉空白。空指针当空串。
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
