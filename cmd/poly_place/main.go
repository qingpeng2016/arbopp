// 只下 Polymarket：按 market 表 id 选盘口，固定买 YES 10 份（FAK）。
//
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/poly_place -market_id=12345
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/poly_place -market_id=12345 -outcome_id=678 -live
//	RUN_CONF=conf/local.conf.yaml go run ./cmd/poly_place -market_id=12345 -list
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
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/strategy_entry/broker"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	persistentrepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	apphttp "github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/http"
	"github.com/gph-tech/fgmm-strategy-bitfinex/infrastructure/mysql"
)

const fixedQty = 10.0

func main() {
	marketID := flag.Uint64("market_id", 0, "market 表 id（必填）")
	outcomeID := flag.Uint64("outcome_id", 0, "market_outcome 表 id；一个 market 多盘口时必填")
	listOnly := flag.Bool("list", false, "只列出该 market 下的 outcome 后退出")
	live := flag.Bool("live", false, "真正发单；默认 dry-run")
	priceFlag := flag.Float64("price", 0, "限价；默认吃卖一（按 10 份走盘口）")
	flag.Parse()

	if *marketID == 0 {
		fmt.Fprintln(os.Stderr, "usage: -market_id=<market.id> [-outcome_id=<market_outcome.id>] [-live] [-list] [-price=0.55]")
		os.Exit(2)
	}

	_ = conf.NewCfg()
	db := boot.NewDBClient()
	catalog, _ := boot.NewPlatformCatalog(mysql.NewPlatformImpl(db))
	accountRepo := mysql.NewAccountImpl(db)
	marketRepo := mysql.NewMarketImpl(db)
	outcomeRepo := mysql.NewMarketOutcomeImpl(db)
	poly := boot.NewPolymarketRepoFromPlatform(catalog, apphttp.NewHttpClient())

	ctx := context.Background()
	mkt, err := marketRepo.FindOne(ctx, nil, map[string]interface{}{"id = ?": *marketID}, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load market id=%d: %v\n", *marketID, err)
		os.Exit(1)
	}
	if !strings.EqualFold(mkt.Platform, appcommon.PlatformPolymarket) {
		fmt.Fprintf(os.Stderr, "market id=%d platform=%q，不是 POLYMARKET\n", mkt.ID, mkt.Platform)
		os.Exit(1)
	}

	_, outcomes, err := outcomeRepo.List(ctx, nil, map[string]interface{}{
		"market_row_id = ?": mkt.ID,
		"platform = ?":      appcommon.PlatformPolymarket,
	}, "id asc", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load outcomes: %v\n", err)
		os.Exit(1)
	}
	if len(outcomes) == 0 {
		fmt.Fprintf(os.Stderr, "market id=%d 无 POLYMARKET outcome\n", mkt.ID)
		os.Exit(1)
	}

	printMarket(mkt, outcomes)
	if *listOnly {
		return
	}

	oc, err := pickOutcome(outcomes, *outcomeID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	token := strings.TrimSpace(oc.YesTokenID)
	if token == "" {
		fmt.Fprintf(os.Stderr, "outcome id=%d 无 yes_token_id\n", oc.ID)
		os.Exit(1)
	}
	negRisk := broker.ReadNegRisk(oc.Config)
	if !negRisk {
		negRisk = broker.ReadNegRisk(mkt.Config)
	}

	bookCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	bookResp, err := poly.GetOrderBook(bookCtx, httpentity.GetOrderBookRequest{TokenID: token})
	if err != nil || bookResp == nil {
		if err == nil {
			err = fmt.Errorf("empty book")
		}
		fmt.Fprintf(os.Stderr, "GetOrderBook: %v\n", err)
		os.Exit(1)
	}

	price := *priceFlag
	if price <= 0 {
		var ok bool
		price, ok = bestAskForSize(bookResp.Asks, fixedQty)
		if !ok {
			fmt.Fprintln(os.Stderr, "卖盘不足以覆盖 10 份，或盘口为空；可用 -price= 手动指定")
			os.Exit(1)
		}
	}

	acc, ok := loadPolyAccount(ctx, accountRepo)
	if !ok {
		fmt.Fprintln(os.Stderr, "account 缺 ACTIVE POLYMARKET 发单凭证（api_key/secret/passphrase + private_key_enc + 地址）")
		os.Exit(1)
	}

	fmt.Printf("\n准备下单 POLYMARKET\n")
	fmt.Printf("  market_id=%d outcome_id=%d\n", mkt.ID, oc.ID)
	fmt.Printf("  name=%s\n", deref(oc.OutcomeName))
	fmt.Printf("  token=%s\n", token)
	fmt.Printf("  side=BUY yes qty=%.0f price=%.4f neg_risk=%v sig_type=%d\n",
		fixedQty, price, negRisk, acc.SignatureType)
	fmt.Printf("  proxy=%s eoa=%s\n", trunc(acc.ProxyAddress, 14), trunc(acc.EOAAddress, 14))

	if !*live {
		fmt.Println("\n[dry-run] 未发单。加 -live 真正下单。")
		return
	}

	cred := broker.PolyCreds{Account: acc, OK: true}
	res, err := broker.SendPoly(poly, cred, token, price, fixedQty, true, negRisk)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PlaceOrder failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n下单结果 client=%s platform_oid=%s fill=%.4f\n",
		res.ClientOrderID, res.PlatformOrderID, res.Fill)
}

func printMarket(mkt persistententity.Market, outcomes []persistententity.MarketOutcome) {
	fmt.Printf("market id=%d platform=%s name=%s map_key=%s\n",
		mkt.ID, mkt.Platform, trunc(deref(mkt.Name), 80), trunc(deref(mkt.MapKey), 60))
	fmt.Printf("outcomes (%d):\n", len(outcomes))
	for _, oc := range outcomes {
		fmt.Printf("  outcome_id=%-8d type=%-10s yes=%s name=%s\n",
			oc.ID, oc.OutcomeType, trunc(oc.YesTokenID, 18), trunc(deref(oc.OutcomeName), 60))
	}
}

func pickOutcome(outcomes []persistententity.MarketOutcome, outcomeID uint64) (persistententity.MarketOutcome, error) {
	if outcomeID > 0 {
		for _, oc := range outcomes {
			if uint64(oc.ID) == outcomeID {
				return oc, nil
			}
		}
		return persistententity.MarketOutcome{}, fmt.Errorf("outcome_id=%d 不属于该 market；用 -list 查看", outcomeID)
	}
	if len(outcomes) == 1 {
		return outcomes[0], nil
	}
	var lines []string
	for _, oc := range outcomes {
		lines = append(lines, strconv.FormatUint(uint64(oc.ID), 10))
	}
	return persistententity.MarketOutcome{}, fmt.Errorf(
		"该 market 有 %d 个 outcome，请加 -outcome_id=（可选: %s）或 -list",
		len(outcomes), strings.Join(lines, ","))
}

func bestAskForSize(asks []httpentity.OrderBookEntry, need float64) (float64, bool) {
	type level struct {
		px, sz float64
	}
	levels := make([]level, 0, len(asks))
	for _, a := range asks {
		px, err1 := strconv.ParseFloat(strings.TrimSpace(a.Price), 64)
		sz, err2 := strconv.ParseFloat(strings.TrimSpace(a.Size), 64)
		if err1 != nil || err2 != nil || px <= 0 || sz <= 0 {
			continue
		}
		levels = append(levels, level{px: px, sz: sz})
	}
	if len(levels) == 0 {
		return 0, false
	}
	// asks 通常已按价升序；再保险排序
	for i := 0; i < len(levels); i++ {
		for j := i + 1; j < len(levels); j++ {
			if levels[j].px < levels[i].px {
				levels[i], levels[j] = levels[j], levels[i]
			}
		}
	}
	remain := need
	var worst float64
	for _, lv := range levels {
		take := lv.sz
		if take > remain {
			take = remain
		}
		if take <= 0 {
			continue
		}
		worst = lv.px
		remain -= take
		if remain <= 1e-9 {
			return worst, true
		}
	}
	return 0, false
}

func loadPolyAccount(ctx context.Context, repo persistentrepo.AccountRepo) (httpentity.PolymarketAccount, bool) {
	if repo == nil {
		return httpentity.PolymarketAccount{}, false
	}
	_, list, err := repo.List(ctx, nil, map[string]interface{}{"status = ?": "ACTIVE"}, "id asc", "")
	if err != nil {
		return httpentity.PolymarketAccount{}, false
	}
	for i := range list {
		a := list[i]
		if !strings.EqualFold(strings.TrimSpace(a.Platform), appcommon.PlatformPolymarket) {
			continue
		}
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
		if ok {
			return acc, true
		}
	}
	return httpentity.PolymarketAccount{}, false
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
