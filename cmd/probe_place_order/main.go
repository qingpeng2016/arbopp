// 实盘下单探针：只测 broker.SendOnOutcome（不跑策略筛选）。
// 默认 mode=1x2_yes：平+主+客 各买 YES 一腿，尽量凑满 3 个平台（含 SX）。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/probe_place_order -map_key='v1|阿甲|arg-boc-riv1-2026-11-25'
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/probe_place_order -map_key='...' -live
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/probe_place_order -map_key='...' -mode=group -group=draw
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	strategyentry "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/broker"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/watchgroup"
	predictws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/predict"
	sxbetws "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/ws/sxbet"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	apphttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

func main() {
	mapKey := flag.String("map_key", "", "场次 map_key（必填）")
	mode := flag.String("mode", "1x2_yes", "1x2_yes=平+主+客三腿买YES（默认）；group=同一 OutcomeGroup 多平台")
	groupFilter := flag.String("group", "", "mode=group 时 OutcomeGroup 子串过滤")
	live := flag.Bool("live", false, "真正发单；默认只打印检查结果（dry-run）")
	listOnly := flag.Bool("list", false, "只列出该 map_key 下 moneyline YES 分组后退出")
	flag.Parse()

	mk := strings.TrimSpace(*mapKey)
	if mk == "" {
		fmt.Fprintln(os.Stderr, "usage: -map_key=v1|联赛|fixture [-mode=1x2_yes|group] [-live] [-list]")
		os.Exit(2)
	}

	_ = conf.NewCfg()
	db := boot.NewDBClient()
	catalog, _ := boot.NewPlatformCatalog(mysql.NewPlatformImpl(db))
	accountRepo := mysql.NewAccountImpl(db)
	outcomeRepo := mysql.NewMarketOutcomeImpl(db)
	thresholdRepo := mysql.NewConfigStrategyThresholdImpl(db)
	feeRepo := mysql.NewConfigStrategyFeeImpl(db)

	httpClient := apphttp.NewHttpClient()
	poly := boot.NewPolymarketRepoFromPlatform(catalog, httpClient)
	kalshi := boot.NewKalshiRepoFromPlatform(catalog)
	predict := boot.NewPredictRepoFromPlatform(catalog, accountRepo)
	sxbet := boot.NewSxbetRepoFromPlatform(catalog, accountRepo)

	ctx := context.Background()
	params := strategyentry.LoadParams(ctx, thresholdRepo, feeRepo)
	creds := loadCreds(ctx, accountRepo)

	outcomes, err := loadMoneylineYES(ctx, outcomeRepo, mk)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load outcomes: %v\n", err)
		os.Exit(1)
	}
	if len(outcomes) == 0 {
		fmt.Fprintf(os.Stderr, "map_key=%q 无 moneyline outcome\n", mk)
		os.Exit(1)
	}

	groups := groupOutcomes(mk, outcomes)
	printGroups(mk, groups)
	if *listOnly {
		return
	}

	var picked outcomeGroup
	var ok bool
	switch strings.ToLower(strings.TrimSpace(*mode)) {
	case "group":
		picked, ok = pickGroup(groups, strings.TrimSpace(*groupFilter))
		if !ok {
			fmt.Fprintln(os.Stderr, "未选中可用 OutcomeGroup（需要≥2 个平台）；用 -group= 指定")
			os.Exit(1)
		}
		fmt.Printf("\n选用 group=%s platforms=%d\n", picked.key, len(picked.legs))
	default: // 1x2_yes
		picked, ok = pickOneX2YesLegs(groups)
		if !ok {
			fmt.Fprintln(os.Stderr, "无法凑齐 1X2 三腿（draw/tie + 两队）；用 -list 看分组")
			os.Exit(1)
		}
		fmt.Printf("\n选用 1x2_yes 三腿（仅测下单，不做策略过滤）:\n")
		for _, l := range picked.legs {
			fmt.Printf("  %s role=%s outcome_id=%d\n", l.platform, l.role, l.oc.ID)
		}
	}

	store := book.NewStore()
	wgWatch := &watchgroup.Watch{
		Ctx: mkCtx(ctx), MapKey: mk, Outcomes: picked.outcomes(), Books: store,
		Ops: watchgroup.Ops{
			StrategyParams: func() (float64, float64, float64) {
				return params.MaxCost, params.MinAmount, params.MaxAmount
			},
			TradeFee: func(platform, category, sub string) float64 {
				return params.Fee(platform, category, sub)
			},
			CanTrade: func(platform string) bool {
				return canTrade(platform, creds, poly, kalshi, predict, sxbet)
			},
		},
	}

	if err := fetchBooksConcurrent(ctx, wgWatch, poly, kalshi, predict, sxbet, creds); err != nil {
		fmt.Fprintf(os.Stderr, "fetch books: %v\n", err)
		os.Exit(1)
	}
	printPositions(ctx, poly, creds, picked)

	view, at := wgWatch.StrategySnapshot()
	_ = at
	platforms := make([]string, 0, len(picked.legs))
	asks := make([]float64, 0, len(picked.legs))
	sizes := make([]float64, 0, len(picked.legs))
	sides := make([]yesno.ExecSide, 0, len(picked.legs))
	for _, leg := range picked.legs {
		es, ok := yesno.QuoteExecSide(wgWatch, view, leg.idx)
		if !ok {
			fmt.Fprintf(os.Stderr, "no quote platform=%s outcome_id=%d\n", leg.platform, leg.oc.ID)
			os.Exit(1)
		}
		px, sz, topOK := es.TopAskSz(true)
		if !topOK || !(px > 0) || !(sz > 0) {
			fmt.Fprintf(os.Stderr, "YES ask empty platform=%s outcome_id=%d\n", leg.platform, leg.oc.ID)
			os.Exit(1)
		}
		platforms = append(platforms, leg.platform)
		asks = append(asks, px)
		sizes = append(sizes, sz)
		sides = append(sides, es)
		name := ""
		if leg.oc.OutcomeName != nil {
			name = *leg.oc.OutcomeName
		}
		fmt.Printf("  book %s id=%d ask=%.4f sz=%.4f name=%s\n", leg.platform, leg.oc.ID, px, sz, trunc(name, 80))
	}

	chk := yesno.CheckOrderSize(yesno.OrderSizeCheckInput{
		Platforms: platforms, BestAsks: asks, AskSizes: sizes,
		MinAmount: params.MinAmount, MaxAmount: params.MaxAmount,
	})
	snap, _ := json.MarshalIndent(chk.Snapshot, "  ", "  ")
	fmt.Printf("\nCheckOrderSize ok=%v qty=%.6f reason=%s\n  %s\n", chk.OK, chk.Qty, chk.FailReason, snap)
	if !chk.OK {
		os.Exit(1)
	}
	if params.MinAmount <= 0 || params.MaxAmount <= 0 {
		fmt.Fprintln(os.Stderr, "min_amount/max_amount 未配置，拒绝发单")
		os.Exit(1)
	}

	fmt.Printf("\ndry_run=%v live=%v net_qty=%.6f min_amount=%.4f max_amount=%.4f\n", !*live, *live, chk.Qty, params.MinAmount, params.MaxAmount)
	if !*live {
		fmt.Println("未发单（加 -live 才会调用 broker.SendOnOutcome）")
		return
	}

	for _, leg := range picked.legs {
		if why := tradeBlockReason(leg.platform, creds, poly, kalshi, predict, sxbet); why != "" {
			fmt.Fprintf(os.Stderr, "platform %s 不能发单：%s\n", leg.platform, why)
			os.Exit(1)
		}
	}

	type placed struct {
		platform string
		px, net  float64
		orderQty float64
		res      watchgroup.SendResult
		err      error
	}
	out := make([]placed, len(picked.legs))
	var wg sync.WaitGroup
	for i := range picked.legs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			leg := picked.legs[i]
			es := sides[i]
			px, _, _ := es.TopAskSz(true)
			orderQty := es.OrderShares(true, chk.Qty)
			neg := broker.ReadNegRisk(leg.oc.Config)
			res, err := broker.SendOnOutcome(
				poly, kalshi, predict, sxbet,
				broker.PolyCreds{Account: creds.poly, OK: creds.polyOK},
				broker.KalshiCreds{Account: creds.kalshi, OK: creds.kalshiOK},
				broker.PredictCreds{Account: creds.predict, OK: creds.predictOK},
				broker.SxbetCreds{Account: creds.sxbet, OK: creds.sxbetOK},
				leg.oc, true, px, orderQty, neg,
			)
			if err == nil {
				res.Fill = es.FillToNet(true, res.Fill)
			}
			out[i] = placed{
				platform: leg.platform, px: px, net: chk.Qty, orderQty: orderQty,
				res: res, err: err,
			}
		}(i)
	}
	wg.Wait()

	fmt.Println("\n=== place results ===")
	for _, r := range out {
		errS := ""
		if r.err != nil {
			errS = r.err.Error()
		}
		fmt.Printf("%s px=%.4f net_qty=%.4f order_qty=%.4f fill=%.4f client=%s platform_oid=%s err=%s\n",
			r.platform, r.px, r.net, r.orderQty, r.res.Fill, r.res.ClientOrderID, r.res.PlatformOrderID, errS)
	}
}

type legRef struct {
	platform string
	idx      int
	role     string // draw / team_a / team_b / ""
	oc       *persistententity.MarketOutcome
}

type outcomeGroup struct {
	key  string
	legs []legRef
}

func (g outcomeGroup) outcomes() []persistententity.MarketOutcome {
	out := make([]persistententity.MarketOutcome, len(g.legs))
	for i, l := range g.legs {
		out[i] = *l.oc
	}
	return out
}

func loadMoneylineYES(ctx context.Context, repo persistentrepo.MarketOutcomeRepo, mapKey string) ([]persistententity.MarketOutcome, error) {
	_, list, err := repo.List(ctx, nil, map[string]interface{}{
		"map_key = ?":      mapKey,
		"outcome_type = ?": "moneyline",
	}, "platform asc, id asc", "")
	if err != nil {
		return nil, err
	}
	var out []persistententity.MarketOutcome
	for i := range list {
		oc := list[i]
		plat := strings.ToUpper(strings.TrimSpace(oc.Platform))
		switch plat {
		case appcommon.PlatformPolymarket, appcommon.PlatformPredict:
			if strings.TrimSpace(oc.YesTokenID) == "" && strings.TrimSpace(oc.PlatformMarketOutcomeID) == "" {
				continue
			}
		case appcommon.PlatformKalshi, appcommon.PlatformSxbet:
			if strings.TrimSpace(oc.PlatformMarketOutcomeID) == "" {
				continue
			}
		default:
			continue
		}
		out = append(out, oc)
	}
	return out, nil
}

func groupOutcomes(mapKey string, outcomes []persistententity.MarketOutcome) []outcomeGroup {
	byKey := map[string][]legRef{}
	// 稳定 index：按传入顺序放入 group 时重建 watch Outcomes，故先按 group 聚合再展开。
	type keyed struct {
		oc  persistententity.MarketOutcome
		key string
	}
	var rows []keyed
	for i := range outcomes {
		oc := outcomes[i]
		key := marketmap.OutcomeGroupKeyForWatch(&oc, mapKey)
		if key == "" {
			continue
		}
		rows = append(rows, keyed{oc: oc, key: key})
	}
	// 每个 group 内 outcomes 连续存放，idx 相对 group.outcomes()
	order := make([]string, 0)
	seen := map[string]bool{}
	for _, r := range rows {
		if !seen[r.key] {
			order = append(order, r.key)
			seen[r.key] = true
		}
		byKey[r.key] = append(byKey[r.key], legRef{
			platform: strings.ToUpper(strings.TrimSpace(r.oc.Platform)),
			oc:       ptr(r.oc),
		})
	}
	var out []outcomeGroup
	for _, key := range order {
		legs := byKey[key]
		// 同平台只留一条（优先有 YesToken）
		uniq := map[string]legRef{}
		for _, l := range legs {
			prev, ok := uniq[l.platform]
			if !ok || (strings.TrimSpace(l.oc.YesTokenID) != "" && strings.TrimSpace(prev.oc.YesTokenID) == "") {
				uniq[l.platform] = l
			}
		}
		plats := make([]string, 0, len(uniq))
		for p := range uniq {
			plats = append(plats, p)
		}
		sort.Strings(plats)
		g := outcomeGroup{key: key}
		for i, p := range plats {
			l := uniq[p]
			l.idx = i
			g.legs = append(g.legs, l)
		}
		out = append(out, g)
	}
	return out
}

func ptr(oc persistententity.MarketOutcome) *persistententity.MarketOutcome {
	o := oc
	return &o
}

func printGroups(mapKey string, groups []outcomeGroup) {
	fmt.Printf("map_key=%s moneyline groups=%d\n", mapKey, len(groups))
	for _, g := range groups {
		plats := make([]string, 0, len(g.legs))
		for _, l := range g.legs {
			plats = append(plats, l.platform)
		}
		fmt.Printf("  [%d平台] %s → %s\n", len(g.legs), g.key, strings.Join(plats, ","))
	}
}

func pickGroup(groups []outcomeGroup, filter string) (outcomeGroup, bool) {
	filter = strings.ToLower(filter)
	var candidates []outcomeGroup
	for _, g := range groups {
		if len(g.legs) < 2 {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(g.key), filter) {
			continue
		}
		candidates = append(candidates, g)
	}
	if len(candidates) == 0 {
		return outcomeGroup{}, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		if len(candidates[i].legs) != len(candidates[j].legs) {
			return len(candidates[i].legs) > len(candidates[j].legs)
		}
		return candidates[i].key < candidates[j].key
	})
	return candidates[0], true
}

func outcomeNormSuffix(groupKey string) string {
	parts := strings.Split(groupKey, "|")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func normsCompatible(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// pickOneX2YesLegs 平+主+客各一腿买 YES；探针专用，不做 SameSettlementScope。
// 队名模糊合并（ca_boca_juniors ≈ boca_juniors），平台尽量互异并优先带上 SXBET。
func pickOneX2YesLegs(groups []outcomeGroup) (outcomeGroup, bool) {
	type bucket struct {
		norm string
		legs []legRef
	}
	var draw *bucket
	var teams []bucket
	for _, g := range groups {
		norm := outcomeNormSuffix(g.key)
		if norm == "" {
			continue
		}
		if norm == "draw" || norm == "tie" {
			if draw == nil {
				draw = &bucket{norm: "draw"}
			}
			draw.legs = append(draw.legs, g.legs...)
			continue
		}
		merged := false
		for i := range teams {
			if normsCompatible(teams[i].norm, norm) {
				teams[i].legs = append(teams[i].legs, g.legs...)
				if len(norm) > len(teams[i].norm) {
					teams[i].norm = norm
				}
				merged = true
				break
			}
		}
		if !merged {
			teams = append(teams, bucket{norm: norm, legs: append([]legRef{}, g.legs...)})
		}
	}
	if draw == nil || len(teams) < 2 {
		return outcomeGroup{}, false
	}
	sort.Slice(teams, func(i, j int) bool { return teams[i].norm < teams[j].norm })
	roles := []struct {
		role string
		legs []legRef
	}{
		{"draw", draw.legs},
		{"team_a", teams[0].legs},
		{"team_b", teams[1].legs},
	}

	usedPlat := map[string]bool{}
	pickFrom := func(legs []legRef, preferSX bool) (legRef, bool) {
		var cand []legRef
		for _, l := range legs {
			if usedPlat[l.platform] {
				continue
			}
			cand = append(cand, l)
		}
		if len(cand) == 0 {
			for _, l := range legs {
				cand = append(cand, l)
			}
		}
		if len(cand) == 0 {
			return legRef{}, false
		}
		sort.Slice(cand, func(i, j int) bool {
			if preferSX {
				iSX := strings.EqualFold(cand[i].platform, appcommon.PlatformSxbet)
				jSX := strings.EqualFold(cand[j].platform, appcommon.PlatformSxbet)
				if iSX != jSX {
					return iSX
				}
			}
			return cand[i].platform < cand[j].platform
		})
		return cand[0], true
	}

	out := outcomeGroup{key: "1x2_yes"}
	for i, r := range roles {
		preferSX := i == 0 // 平局腿优先 SX，验证 SX 下单
		l, ok := pickFrom(r.legs, preferSX)
		if !ok {
			return outcomeGroup{}, false
		}
		l.role = r.role
		l.idx = i
		usedPlat[l.platform] = true
		out.legs = append(out.legs, l)
	}
	return out, true
}

func fetchBooksConcurrent(
	ctx context.Context,
	w *watchgroup.Watch,
	poly httprepo.PolymarketRepo,
	kalshi httprepo.KalshiRepo,
	predict httprepo.PredictRepo,
	sxbet httprepo.SxbetRepo,
	creds activeCreds,
) error {
	type jobErr struct {
		platform string
		err      error
	}
	var (
		mu   sync.Mutex
		errs []jobErr
		wg   sync.WaitGroup
	)
	now := time.Now().UTC()
	for i := range w.Outcomes {
		i := i
		oc := &w.Outcomes[i]
		plat := strings.ToUpper(strings.TrimSpace(oc.Platform))
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			switch plat {
			case appcommon.PlatformPolymarket:
				err = fetchPolyBook(ctx, w, poly, oc, now)
			case appcommon.PlatformPredict:
				err = fetchPredictBook(ctx, w, predict, oc, now)
			case appcommon.PlatformSxbet:
				err = fetchSxbetBook(ctx, w, sxbet, oc, now)
			case appcommon.PlatformKalshi:
				err = fetchKalshiBook(ctx, w, kalshi, oc, now)
			default:
				err = fmt.Errorf("unsupported platform %s", plat)
			}
			if err != nil {
				mu.Lock()
				errs = append(errs, jobErr{platform: plat, err: err})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(errs) == 0 {
		return nil
	}
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, fmt.Sprintf("%s: %v", e.platform, e.err))
	}
	_ = creds
	return fmt.Errorf("book fetch failed: %s", strings.Join(parts, "; "))
}

func fetchPolyBook(ctx context.Context, w *watchgroup.Watch, poly httprepo.PolymarketRepo, oc *persistententity.MarketOutcome, now time.Time) error {
	if poly == nil {
		return fmt.Errorf("nil poly client")
	}
	token := strings.TrimSpace(oc.YesTokenID)
	if token == "" {
		return fmt.Errorf("empty yes_token_id")
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	resp, err := poly.GetOrderBook(cctx, httpentity.GetOrderBookRequest{TokenID: token})
	if err != nil || resp == nil {
		if err == nil {
			err = fmt.Errorf("empty book")
		}
		return err
	}
	bids := polyLevels(resp.Bids)
	asks := polyLevels(resp.Asks)
	tick := book.ParseNum(resp.TickSize)
	minSz := book.ParseNum(resp.MinOrderSize)
	ok := w.ApplyBookEvent(func() bool {
		return w.Books.ApplyPolyBook(token, bids, asks, tick, minSz, now)
	})
	if !ok {
		return fmt.Errorf("ApplyPolyBook rejected")
	}
	return nil
}

func polyLevels(rows []httpentity.OrderBookEntry) []book.Level {
	out := make([]book.Level, 0, len(rows))
	for _, r := range rows {
		px := book.ParsePrice(r.Price)
		sz := book.ParseNum(r.Size)
		if px > 0 && sz > 0 {
			out = append(out, book.Level{Price: px, Size: sz})
		}
	}
	return out
}

func fetchPredictBook(ctx context.Context, w *watchgroup.Watch, predict httprepo.PredictRepo, oc *persistententity.MarketOutcome, now time.Time) error {
	if predict == nil {
		return fmt.Errorf("nil predict client")
	}
	id := strings.TrimSpace(oc.PlatformMarketOutcomeID)
	if id == "" {
		return fmt.Errorf("empty platform_market_outcome_id")
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	books, err := predict.GetOrderbooks(cctx, []string{id})
	if err != nil {
		return err
	}
	for i := range books {
		mid := strconv.FormatInt(books[i].MarketID, 10)
		if mid == "0" || mid != id {
			continue
		}
		ts := now
		if books[i].UpdateTimestampMs > 0 {
			ts = time.UnixMilli(books[i].UpdateTimestampMs).UTC()
		}
		bids := predictws.LevelsFrom(books[i].Bids)
		asks := predictws.LevelsFrom(books[i].Asks)
		ok := w.ApplyBookEvent(func() bool {
			return w.Books.ApplyPredictBook(id, bids, asks, ts)
		})
		if !ok {
			return fmt.Errorf("ApplyPredictBook rejected")
		}
		return nil
	}
	return fmt.Errorf("market %s not in GetOrderbooks response", id)
}

func fetchSxbetBook(ctx context.Context, w *watchgroup.Watch, sxbet httprepo.SxbetRepo, oc *persistententity.MarketOutcome, now time.Time) error {
	if sxbet == nil {
		return fmt.Errorf("nil sxbet client")
	}
	hash := strings.TrimSpace(oc.PlatformMarketOutcomeID)
	if hash == "" {
		return fmt.Errorf("empty market hash")
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	snap, err := sxbet.GetOrderbookSnapshot(cctx, hash)
	if err != nil || snap == nil {
		if err == nil {
			err = fmt.Errorf("empty snapshot")
		}
		return err
	}
	msg, ok := sxbetws.MessageFromSnapshot(1, snap)
	if !ok {
		return fmt.Errorf("empty sxbet levels")
	}
	applied := w.ApplyBookEvent(func() bool {
		return w.Books.ApplySxbetBook(hash, msg.Bids, msg.Asks, now)
	})
	if !applied {
		return fmt.Errorf("ApplySxbetBook rejected")
	}
	return nil
}

func fetchKalshiBook(ctx context.Context, w *watchgroup.Watch, kalshi httprepo.KalshiRepo, oc *persistententity.MarketOutcome, now time.Time) error {
	if kalshi == nil {
		return fmt.Errorf("nil kalshi client")
	}
	ticker := strings.TrimSpace(oc.PlatformMarketOutcomeID)
	if ticker == "" {
		return fmt.Errorf("empty ticker")
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	ob, err := kalshi.GetOrderbook(cctx, ticker, 20)
	if err != nil || ob == nil {
		if err == nil {
			err = fmt.Errorf("empty orderbook")
		}
		return err
	}
	yesBids := kalshiLevels(ob.Yes)
	noBids := kalshiLevels(ob.No)
	ok := w.ApplyBookEvent(func() bool {
		return w.Books.ApplyKalshiSnapshot(ticker, yesBids, noBids, 1, now)
	})
	if !ok {
		return fmt.Errorf("ApplyKalshiSnapshot rejected")
	}
	return nil
}

func kalshiLevels(rows [][]string) []book.Level {
	out := make([]book.Level, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		px := book.ParsePrice(row[0])
		sz := book.ParseNum(row[1])
		if px > 0 && sz > 0 {
			out = append(out, book.Level{Price: px, Size: sz})
		}
	}
	return out
}

func printPositions(ctx context.Context, poly httprepo.PolymarketRepo, creds activeCreds, g outcomeGroup) {
	if poly == nil || !creds.polyOK {
		return
	}
	proxy := strings.TrimSpace(creds.poly.ProxyAddress)
	if proxy == "" {
		proxy = strings.TrimSpace(creds.poly.EOAAddress)
	}
	if proxy == "" {
		return
	}
	want := map[string]bool{}
	for _, l := range g.legs {
		if strings.EqualFold(l.platform, appcommon.PlatformPolymarket) {
			if t := strings.TrimSpace(l.oc.YesTokenID); t != "" {
				want[t] = true
			}
		}
	}
	if len(want) == 0 {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	pos, err := poly.GetPositions(cctx, proxy)
	if err != nil || pos == nil {
		fmt.Printf("poly YES 持仓查询失败: %v\n", err)
		return
	}
	fmt.Println("\nPoly YES 持仓（本组 token）:")
	hit := 0
	for _, p := range pos.Positions {
		asset := strings.TrimSpace(p.TokenID)
		if asset == "" {
			asset = strings.TrimSpace(p.AssetID)
		}
		if !want[asset] {
			continue
		}
		hit++
		fmt.Printf("  token=%s size=%s avg=%.4f\n", trunc(asset, 24), p.Balance, p.AvgPrice)
	}
	if hit == 0 {
		fmt.Println("  (无匹配持仓)")
	}
}

type activeCreds struct {
	poly      httpentity.PolymarketAccount
	polyOK    bool
	kalshi    httpentity.KalshiAccount
	kalshiOK  bool
	predict   httpentity.PredictAccount
	predictOK bool
	sxbet     httpentity.SxbetAccount
	sxbetOK   bool
}

func loadCreds(ctx context.Context, repo persistentrepo.AccountRepo) activeCreds {
	var c activeCreds
	if repo == nil {
		return c
	}
	_, list, err := repo.List(ctx, nil, map[string]interface{}{"status = ?": "ACTIVE"}, "id asc", "")
	if err != nil {
		return c
	}
	for i := range list {
		a := list[i]
		switch strings.ToUpper(strings.TrimSpace(a.Platform)) {
		case appcommon.PlatformPolymarket:
			if !c.polyOK {
				if acc, ok := polyAccount(a); ok {
					c.poly, c.polyOK = acc, true
				}
			}
		case appcommon.PlatformKalshi:
			if !c.kalshiOK {
				if acc, ok := kalshiAccount(a); ok {
					c.kalshi, c.kalshiOK = acc, true
				}
			}
		case appcommon.PlatformPredict:
			if !c.predictOK {
				if acc, ok := predictAccount(a); ok {
					c.predict, c.predictOK = acc, true
				}
			}
		case appcommon.PlatformSxbet:
			if !c.sxbetOK {
				if acc, ok := sxbetAccount(a); ok {
					c.sxbet, c.sxbetOK = acc, true
				}
			}
		}
	}
	return c
}

func canTrade(platform string, c activeCreds, poly httprepo.PolymarketRepo, kalshi httprepo.KalshiRepo, predict httprepo.PredictRepo, sxbet httprepo.SxbetRepo) bool {
	return tradeBlockReason(platform, c, poly, kalshi, predict, sxbet) == ""
}

func tradeBlockReason(platform string, c activeCreds, poly httprepo.PolymarketRepo, kalshi httprepo.KalshiRepo, predict httprepo.PredictRepo, sxbet httprepo.SxbetRepo) string {
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case appcommon.PlatformKalshi:
		if kalshi == nil {
			return "客户端未加载"
		}
		if !c.kalshiOK {
			return "account 缺 ACTIVE 发单凭证（需 api_key + api_secret_enc）"
		}
	case appcommon.PlatformPredict:
		if predict == nil {
			return "客户端未加载"
		}
		if !c.predictOK {
			return "account 缺 ACTIVE 发单凭证（需 api_key + private_key_enc）"
		}
	case appcommon.PlatformSxbet:
		if sxbet == nil {
			return "客户端未加载"
		}
		if !c.sxbetOK {
			return "account 缺 ACTIVE 发单凭证（需 api_key + private_key_enc + proxy_address；仅有 platform.extra api_key 只能拉盘口不能下单）"
		}
	default:
		if poly == nil {
			return "客户端未加载"
		}
		if !c.polyOK {
			return "account 缺 ACTIVE 发单凭证（需 api_key/secret/passphrase + private_key_enc + 地址）"
		}
	}
	return ""
}

func polyAccount(a persistententity.Account) (httpentity.PolymarketAccount, bool) {
	sig := 0
	if a.SignatureType != nil {
		sig = *a.SignatureType
	}
	acc := httpentity.PolymarketAccount{
		APIKey: deref(a.APIKey), APISecret: deref(a.APISecretEnc), APIPassphrase: deref(a.APIPassphraseEnc),
		EOAAddress: deref(a.EoaAddress), ProxyAddress: deref(a.ProxyAddress), PrivateKey: deref(a.PrivateKeyEnc),
		SignatureType: sig,
	}
	ok := acc.APIKey != "" && acc.APISecret != "" && acc.APIPassphrase != "" && acc.PrivateKey != "" &&
		(acc.ProxyAddress != "" || acc.EOAAddress != "")
	return acc, ok
}

func kalshiAccount(a persistententity.Account) (httpentity.KalshiAccount, bool) {
	acc := httpentity.KalshiAccount{
		KeyID: deref(a.APIKey), PrivateKey: deref(a.APISecretEnc), Passphrase: deref(a.APIPassphraseEnc),
	}
	return acc, acc.KeyID != "" && acc.PrivateKey != ""
}

func predictAccount(a persistententity.Account) (httpentity.PredictAccount, bool) {
	acc := httpentity.PredictAccount{
		APIKey: deref(a.APIKey), PrivateKey: deref(a.PrivateKeyEnc),
		PredictAccount: deref(a.ProxyAddress), EOAAddress: deref(a.EoaAddress),
	}
	return acc, acc.APIKey != "" && acc.PrivateKey != ""
}

func sxbetAccount(a persistententity.Account) (httpentity.SxbetAccount, bool) {
	acc := httpentity.SxbetAccount{
		APIKey: deref(a.APIKey), PrivateKey: deref(a.PrivateKeyEnc),
		EOAAddress: deref(a.EoaAddress), ProxyAddress: deref(a.ProxyAddress),
	}
	return acc, acc.APIKey != "" && acc.PrivateKey != "" && acc.ProxyAddress != ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func trunc(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func mkCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
