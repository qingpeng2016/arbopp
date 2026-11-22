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
			IFNULL(mo.slug,'') AS slug
		FROM market_outcome mo
		JOIN market m ON m.id = mo.market_row_id
		WHERE m.map_key = ?
	`, mapKey).Scan(&rows).Error
	return rows, err
}

func loadWSDB(mapKey string, wsLimit int) ([]wsRow, error) {
	if wsLimit <= 0 {
		wsLimit = 8000
	}
	db := boot.NewDBClient()
	var rows []wsRow
	err := db.Raw(`
		SELECT map_key, platform, platform_market_outcome_id, token_id,
			order_book_asks, order_book_bids, snapshot_at_ms, id
		FROM market_order_book_ws
		WHERE map_key = ?
		ORDER BY id DESC
		LIMIT ?
	`, mapKey, wsLimit).Scan(&rows).Error
	return rows, err
}

func runScanDB(keys []string, outPath string, wsLimit int) error {
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
		wsRows, err := loadWSDB(mk, wsLimit)
		if err != nil {
			return fmt.Errorf("%s ws: %w", mk, err)
		}
		latest, hist := splitLatestHist(wsRows)
		reports = append(reports, scanFixture(mk, outcomes, latest, hist))
		fmt.Printf("[%d/%d] %s outcomes=%d ws=%d\n", i+1, len(keys), mk, len(outcomes), len(wsRows))
	}
	return writeReports(outPath, reports)
}

func runFromDB(mapKey, keysFile, outPath string, wsLimit int) error {
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
	return runScanDB(keys, outPath, wsLimit)
}