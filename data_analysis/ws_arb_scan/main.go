// 套利离线/半离线分析（规则见 RULES.md，不 import 策略包）
//
// 单场连库（默认 market_order_book；WS 表加 -book_table=ws）：
//	RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan -map_key='v1|NBA|nba-mem-min-2026-11-21'
//
// 多场均文件，逐场查库：
//	RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan -keys=./data_analysis/ws_arb_scan/map_keys.txt
//
// 读本地 JSONL：
//	go run ./data_analysis/ws_arb_scan -data=./data_analysis/ws_arb_scan/data
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func main() {
	mapKey := flag.String("map_key", "", "单场 map_key，从库查询并分析")
	keysFile := flag.String("keys", "", "多场：map_key 列表文件，逐场查库")
	dataDir := flag.String("data", "", "读本地 JSONL 目录，默认 "+dataExportDir)
	bookTable := flag.String("book_table", "rest", "订单簿表：rest=market_order_book，ws=market_order_book_ws")
	bookLimit := flag.Int("book_limit", 0, "每场订单簿行数上限；0=该 map_key 全部（兼容旧名 ws_limit）")
	wsLimit := flag.Int("ws_limit", 0, "已废弃，请用 -book_limit")
	out := flag.String("out", "", "附加 jsonl 路径；中文报告与 csv 写入 "+dataExportDir)
	diag := flag.Bool("diag", false, "打印 moneyline 分组与最新盘口是否 ready")
	verifyMs := flag.Int64("verify_ms", 0, "仅核对该毫秒独赢盘口，须配 -map_key")
	flag.Parse()
	setOrderBookTable(*bookTable)
	limit := *bookLimit
	if limit <= 0 && *wsLimit > 0 {
		limit = *wsLimit
	}

	if *verifyMs > 0 {
		if *mapKey == "" {
			fmt.Fprintln(os.Stderr, "verify_ms 需要 -map_key")
			os.Exit(1)
		}
		_ = conf.NewCfg() //nolint - side effect load cfg
		if err := checkMoneylineAtMs(*mapKey, *verifyMs); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	outPath := *out
	if outPath == "" {
		if *mapKey != "" {
			outPath = "data_analysis/ws_arb_scan/out/report_one.jsonl"
		} else {
			outPath = "data_analysis/ws_arb_scan/out/report.jsonl"
		}
	}

	var err error
	switch {
	case *mapKey != "" || *keysFile != "":
		err = runFromDB(*mapKey, *keysFile, outPath, limit, *diag)
	case *dataDir != "":
		err = runScan(*dataDir, *keysFile, outPath)
	default:
		err = fmt.Errorf("请指定 -map_key=... 或 -keys=... 或 -data=%s", dataExportDir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}