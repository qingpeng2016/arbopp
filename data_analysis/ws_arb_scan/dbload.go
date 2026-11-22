// 按 map_key 从库拉数（单场/逐场，避免一次 IN 全表）。
package main

import (
	"fmt"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func loadOutcomesDB(mapKey string) ([]outcomeRec, error) {
	db := boot.NewDBClient()
	var rows []outcomeRec
	err := db.Raw(`
		SELECT m.map_key AS map_key, mo.id, mo.platform, mo.platform_market_outcome_id,
			IFNULL(mo.outcome_name,'') AS outcome_name,
			IFNULL(mo.outcome_type,'') AS outcome_type,
			IFNULL(mo.yes_token_id,'') AS yes_token_id,
			IFNULL(mo.no_token_id,'') AS no_token_id,
			IFNULL(mo.slug,'') AS slug,
			IFNULL(mo.competition,'') AS competition
		FROM market_outcome mo
		JOIN market m ON m.id = mo.market_row_id
		WHERE m.map_key = ?
	`, mapKey).Scan(&rows).Error
	return rows, err
}

func loadBookDB(mapKey string, wsLimit int) ([]wsRow, error) {
	db := boot.NewDBClient()
	var rows []wsRow
	q := `
		SELECT map_key, platform, IFNULL(platform_market_id,'') AS platform_market_id,
			platform_market_outcome_id,
			IFNULL(outcome_name,'') AS outcome_name, IFNULL(token_name,'') AS token_name,
			token_id, order_book_asks, order_book_bids, snapshot_at_ms, id
		FROM ` + orderBookTable + `
		WHERE map_key = ?
		ORDER BY id DESC`
	if wsLimit > 0 {
		err := db.Raw(q+`
		LIMIT ?`, mapKey, wsLimit).Scan(&rows).Error
		return rows, err
	}
	err := db.Raw(q, mapKey).Scan(&rows).Error
	return rows, err
}

func loadBookDBAtMs(mapKey string, ms int64, outcomeIDs []string) ([]wsRow, error) {
	if ms <= 0 || len(outcomeIDs) == 0 {
		return nil, fmt.Errorf("参数无效")
	}
	db := boot.NewDBClient()
	var rows []wsRow
	err := db.Raw(`
		SELECT map_key, platform, IFNULL(platform_market_id,'') AS platform_market_id,
			platform_market_outcome_id,
			IFNULL(outcome_name,'') AS outcome_name, IFNULL(token_name,'') AS token_name,
			token_id, order_book_asks, order_book_bids, snapshot_at_ms, id
		FROM ` + orderBookTable + `
		WHERE map_key = ? AND snapshot_at_ms = ?
		  AND platform_market_outcome_id IN ?
		ORDER BY platform, id`, mapKey, ms, outcomeIDs).Scan(&rows).Error
	return rows, err
}

func runScanDB(keys []string, outPath string, wsLimit int, diag bool) error {
	if len(keys) == 0 {
		return fmt.Errorf("无 map_key")
	}
	var reports []fixtureReport
	for i, mk := range keys {
		mk = strings.TrimSpace(mk)
		if mk == "" {
			continue
		}
		outcomes, err := loadOutcomesDB(mk)
		if err != nil {
			return fmt.Errorf("%s outcomes: %w", mk, err)
		}
		wsRows, err := loadBookDB(mk, wsLimit)
		if err != nil {
			return fmt.Errorf("%s ws: %w", mk, err)
		}
		byHash := outcomeIndex(outcomes)
		latest, hist := splitLatestHist(wsRows, byHash)
		if diag {
			printDiag(mk, outcomes, latest)
		}
		rep := scanFixture(mk, outcomes, latest, hist)
		rep.OutcomeCount = len(outcomes)
		rep.OutcomesByPlatform = countOutcomesByPlatform(outcomes)
		rep.WSByPlatform = countByPlatformPlatforms(wsRows)

		outcomeCSV, wsCSV, err := exportFixtureCSV(mk, outcomes, wsRows)
		if err != nil {
			return fmt.Errorf("%s 导出csv: %w", mk, err)
		}
		reportCN, err := writeChineseReport(mk, rep, outcomeCSV, wsCSV)
		if err != nil {
			return fmt.Errorf("%s 分析报告: %w", mk, err)
		}

		reports = append(reports, rep)
		fmt.Printf("[%d/%d] 进度 %s\n", i+1, len(keys), mk)
		printChineseSummary(mk, rep, outcomeCSV, wsCSV, reportCN)
	}
	return writeReports(outPath, reports)
}

func runFromDB(mapKey, keysFile, outPath string, wsLimit int, diag bool) error {
	_ = conf.NewCfg()
	var keys []string
	if mapKey != "" {
		keys = []string{strings.TrimSpace(mapKey)}
	} else {
		var err error
		if keysFile == "" {
			keysFile = "data_analysis/ws_arb_scan/map_keys.txt"
		}
		keys, err = loadMapKeys(keysFile)
		if err != nil {
			return err
		}
	}
	return runScanDB(keys, outPath, wsLimit, diag)
}