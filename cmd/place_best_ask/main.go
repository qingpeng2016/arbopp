// 按交易所 + market_outcome.id 拉最新订单簿 best_ask，用 strategy_entry 同款 broker 下单。
// 支持 POLYMARKET（LIMIT+FAK）/ PREDICT（MARKET）/ SXBET（LIMIT+IOC）。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/place_best_ask -platform=POLYMARKET -outcome_id=123
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/place_best_ask -platform=PREDICT -outcome_id=456 -live
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/place_best_ask -platform=SXBET -outcome_id=789 -qty=10 -side=yes -live
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	strategyentry "github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/book"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/broker"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/strategy/yesno"
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
	platform := flag.String("platform", "", "交易所：POLYMARKET | PREDICT | SXBET（必填）")
	outcomeID := flag.Uint64("outcome_id", 0, "market_outcome 表 id（必填）")
	qtyFlag := flag.Float64("qty", 10, "下单份额（默认 10）")
	sideFlag := flag.String("side", "yes", "买 yes 或 no（默认 yes）")
	live := flag.Bool("live", false, "真正发单；默认 dry-run")
	flag.Parse()

	plat := normalizePlatform(*platform)
	if plat == "" || *outcomeID == 0 {
		fmt.Fprintln(os.Stderr, "usage: -platform=POLYMARKET|PREDICT|SXBET -outcome_id=<market_outcome.id> [-qty=10] [-side=yes|no] [-live]")
		os.Exit(2)
	}
	buyYes := !strings.EqualFold(strings.TrimSpace(*sideFlag), "no")
	qty := *qtyFlag
	if qty <= 0 {
		fmt.Fprintln(os.Stderr, "qty must be > 0")
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
	predict := boot.NewPredictRepoFromPlatform(catalog, accountRepo)
	sxbet := boot.NewSxbetRepoFromPlatform(catalog, accountRepo)

	ctx := context.Background()
	params := strategyentry.LoadParams(ctx, thresholdRepo, feeRepo)
	creds := loadCreds(ctx, accountRepo)

	oc, err := outcomeRepo.FindOne(ctx, nil, map[string]interface{}{"id = ?": *outcomeID}, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load market_outcome id=%d: %v\n", *outcomeID, err)
		os.Exit(1)
	}
	ocPlat := strings.ToUpper(strings.TrimSpace(oc.Platform))
	if ocPlat != plat {
		fmt.Fprintf(os.Stderr, "outcome id=%d platform=%q，与 -platform=%s 不一致\n", oc.ID, oc.Platform, plat)
		os.Exit(1)
	}

	bookCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	bestAsk, bestSz, err := fetchBestAsk(bookCtx, plat, &oc, buyYes, poly, predict, sxbet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "best_ask: %v\n", err)
		os.Exit(1)
	}

	slip := broker.PredictSlippage{
		Abs:            params.PredictSlippage,
		IsMinAmountOut: params.PredictIsMinAmountOut,
	}
	negRisk := broker.ReadNegRisk(oc.Config)
	limitPx := bestAsk + slip.Abs
	if limitPx >= 1 {
		limitPx = 0.99
	}
	predictBps := broker.AbsoluteSlipToPredictBps(bestAsk, slip.Abs, false, slip.IsMinAmountOut)

	// 与 strategy_entry 一致：-qty 是净兑付份额；SX→USDC stake，Predict→扣费前份额。
	fee := params.Fee(plat, deref(oc.Category), deref(oc.SubCategory))
	es := yesno.ExecSide{
		Side:     yesno.Side{YesPx: bestAsk, NoPx: bestAsk, Fee: fee},
		Platform: plat,
	}
	orderQty := es.OrderShares(buyYes, qty)

	fmt.Printf("outcome_id=%d platform=%s side=%s\n", oc.ID, plat, sideLabel(buyYes))
	fmt.Printf("  name=%s\n", trunc(deref(oc.OutcomeName), 80))
	fmt.Printf("  platform_market_outcome_id=%s\n", trunc(oc.PlatformMarketOutcomeID, 64))
	fmt.Printf("  best_ask=%.6f ask_size=%.6f\n", bestAsk, bestSz)
	fmt.Printf("  net_qty=%.4f → platform_qty=%.6f", qty, orderQty)
	if plat == appcommon.PlatformSxbet {
		fmt.Printf(" (SX totalBetSize=USDC stake ≈ net_qty×price)")
	}
	if plat == appcommon.PlatformPredict {
		fmt.Printf(" (Predict 扣费前份额)")
	}
	fmt.Println()
	fmt.Printf("  fee_coef=%.4f slippage_abs=%.4f is_min_out=%v → worst=%.6f (predict_bps=%d)\n",
		fee, slip.Abs, slip.IsMinAmountOut, limitPx, predictBps)
	fmt.Printf("  neg_risk=%v\n", negRisk)

	if reason := tradeBlockReason(plat, creds, poly, predict, sxbet); reason != "" {
		fmt.Fprintf(os.Stderr, "不可发单: %s\n", reason)
		os.Exit(1)
	}
	if !*live {
		fmt.Println("\n[dry-run] 未发单。加 -live 真正下单。")
		return
	}

	res, err := broker.SendOnOutcome(
		poly, nil, predict, sxbet,
		broker.PolyCreds{Account: creds.poly, OK: creds.polyOK},
		broker.KalshiCreds{},
		broker.PredictCreds{Account: creds.predict, OK: creds.predictOK},
		broker.SxbetCreds{Account: creds.sxbet, OK: creds.sxbetOK},
		&oc, buyYes, bestAsk, orderQty, negRisk, slip,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "place failed: %v\n", err)
		if res.PlatformOrderID != "" {
			fmt.Fprintf(os.Stderr, "  platform_oid=%s (仍可能有部分受理)\n", res.PlatformOrderID)
		}
		os.Exit(1)
	}
	fillNet := es.FillToNet(buyYes, res.Fill)
	fmt.Printf("\n下单结果 client=%s platform_oid=%s platform_fill=%.6f net_fill=%.6f fee=%.6f\n",
		res.ClientOrderID, res.PlatformOrderID, res.Fill, fillNet, res.Fee)
}

func fetchBestAsk(
	ctx context.Context,
	plat string,
	oc *persistententity.MarketOutcome,
	buyYes bool,
	poly httprepo.PolymarketRepo,
	predict httprepo.PredictRepo,
	sxbet httprepo.SxbetRepo,
) (ask, sz float64, err error) {
	switch plat {
	case appcommon.PlatformPolymarket:
		return bestAskPoly(ctx, poly, oc, buyYes)
	case appcommon.PlatformPredict:
		return bestAskPredict(ctx, predict, oc, buyYes)
	case appcommon.PlatformSxbet:
		return bestAskSxbet(ctx, sxbet, oc, buyYes)
	default:
		return 0, 0, fmt.Errorf("unsupported platform %s", plat)
	}
}

func bestAskPoly(ctx context.Context, poly httprepo.PolymarketRepo, oc *persistententity.MarketOutcome, buyYes bool) (float64, float64, error) {
	if poly == nil {
		return 0, 0, fmt.Errorf("nil polymarket client")
	}
	token := strings.TrimSpace(oc.YesTokenID)
	if !buyYes {
		token = strings.TrimSpace(oc.NoTokenID)
	}
	if token == "" {
		return 0, 0, fmt.Errorf("empty token_id for side")
	}
	resp, err := poly.GetOrderBook(ctx, httpentity.GetOrderBookRequest{TokenID: token})
	if err != nil || resp == nil {
		if err == nil {
			err = fmt.Errorf("empty book")
		}
		return 0, 0, err
	}
	return minAskLevel(polyAskLevels(resp.Asks))
}

func bestAskPredict(ctx context.Context, predict httprepo.PredictRepo, oc *persistententity.MarketOutcome, buyYes bool) (float64, float64, error) {
	if predict == nil {
		return 0, 0, fmt.Errorf("nil predict client")
	}
	id := strings.TrimSpace(oc.PlatformMarketOutcomeID)
	if id == "" {
		return 0, 0, fmt.Errorf("empty platform_market_outcome_id")
	}
	books, err := predict.GetOrderbooks(ctx, []string{id})
	if err != nil {
		return 0, 0, err
	}
	for i := range books {
		mid := strconv.FormatInt(books[i].MarketID, 10)
		if mid == "0" || mid != id {
			continue
		}
		asks := predictws.LevelsFrom(books[i].Asks)
		bids := predictws.LevelsFrom(books[i].Bids)
		if buyYes {
			return minAskLevel(asks)
		}
		// NO 卖一：YES 簿互补（与内存簿 SyncComplementNoFromYes 一致）→ NO ask ≈ 1 - YES best bid
		bid, bidSz, ok := maxBidLevel(bids)
		if !ok {
			return 0, 0, fmt.Errorf("no yes bids to derive no ask")
		}
		return 1 - bid, bidSz, nil
	}
	return 0, 0, fmt.Errorf("market %s not in GetOrderbooks", id)
}

func bestAskSxbet(ctx context.Context, sxbet httprepo.SxbetRepo, oc *persistententity.MarketOutcome, buyYes bool) (float64, float64, error) {
	if sxbet == nil {
		return 0, 0, fmt.Errorf("nil sxbet client")
	}
	hash := strings.TrimSpace(oc.PlatformMarketOutcomeID)
	if hash == "" {
		return 0, 0, fmt.Errorf("empty market hash")
	}
	snap, err := sxbet.GetOrderbookSnapshot(ctx, hash)
	if err != nil || snap == nil {
		if err == nil {
			err = fmt.Errorf("empty snapshot")
		}
		return 0, 0, err
	}
	msg, ok := sxbetws.MessageFromSnapshot(1, snap)
	if !ok {
		return 0, 0, fmt.Errorf("empty sxbet levels")
	}
	if buyYes {
		return minAskLevel(msg.Asks)
	}
	return minAskLevel(msg.NotAsks)
}

func polyAskLevels(rows []httpentity.OrderBookEntry) []book.Level {
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

func minAskLevel(levels []book.Level) (float64, float64, error) {
	var bestPx, bestSz float64
	found := false
	for _, lv := range levels {
		if !book.ValidPrice(lv.Price) || lv.Size <= 0 {
			continue
		}
		if !found || lv.Price < bestPx {
			bestPx, bestSz, found = lv.Price, lv.Size, true
		}
	}
	if !found {
		return 0, 0, fmt.Errorf("empty asks")
	}
	return bestPx, bestSz, nil
}

func maxBidLevel(levels []book.Level) (float64, float64, bool) {
	var bestPx, bestSz float64
	found := false
	for _, lv := range levels {
		if !book.ValidPrice(lv.Price) || lv.Size <= 0 {
			continue
		}
		if !found || lv.Price > bestPx {
			bestPx, bestSz, found = lv.Price, lv.Size, true
		}
	}
	return bestPx, bestSz, found
}

func normalizePlatform(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "POLY", "POLYMARKET":
		return appcommon.PlatformPolymarket
	case "PREDICT", "PREDICT.FUN":
		return appcommon.PlatformPredict
	case "SX", "SXBET", "SX.BET":
		return appcommon.PlatformSxbet
	default:
		return ""
	}
}

func sideLabel(buyYes bool) string {
	if buyYes {
		return "YES"
	}
	return "NO"
}

type activeCreds struct {
	poly      httpentity.PolymarketAccount
	polyOK    bool
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

func tradeBlockReason(platform string, c activeCreds, poly httprepo.PolymarketRepo, predict httprepo.PredictRepo, sxbet httprepo.SxbetRepo) string {
	switch platform {
	case appcommon.PlatformPredict:
		if predict == nil {
			return "Predict 客户端未加载"
		}
		if !c.predictOK {
			return "account 缺 ACTIVE PREDICT 凭证（api_key + private_key_enc）"
		}
	case appcommon.PlatformSxbet:
		if sxbet == nil {
			return "SX 客户端未加载"
		}
		if !c.sxbetOK {
			return "account 缺 ACTIVE SXBET 凭证（api_key + private_key_enc + proxy_address）"
		}
	default:
		if poly == nil {
			return "Polymarket 客户端未加载"
		}
		if !c.polyOK {
			return "account 缺 ACTIVE POLYMARKET 凭证（api_key/secret/passphrase + private_key_enc + 地址）"
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
