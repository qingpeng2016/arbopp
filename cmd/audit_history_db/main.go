// 只读审计 data_history_order_book（本场 Everton vs Chelsea）。
package main

import (
	"fmt"
	"os"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

const mapKey = "v1|英超|epl-eve-che-2026-10-17"

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()

	type rk struct {
		RecordKind    string
		VenuePlatform string
		N             int64
	}
	var rows []rk
	if err := db.Raw(`
SELECT record_kind, venue_platform, COUNT(*) AS n
FROM data_history_order_book
WHERE map_key = ?
GROUP BY record_kind, venue_platform
ORDER BY venue_platform, record_kind`, mapKey).Scan(&rows).Error; err != nil {
		fmt.Println("ERR summary:", err)
		os.Exit(1)
	}
	fmt.Println("=== by record_kind / venue ===")
	for _, r := range rows {
		fmt.Printf("  %-12s %-12s %d\n", r.VenuePlatform, r.RecordKind, r.N)
	}

	type snap struct {
		VenuePlatform   string `gorm:"column:venue_platform"`
		MarketOutcomeID uint   `gorm:"column:market_outcome_id"`
		DayUTC          string `gorm:"column:day_utc"`
		SnapN           int64  `gorm:"column:snap_n"`
		MaxBidLen       int    `gorm:"column:max_bid_len"`
		BidSample       string `gorm:"column:bid_sample"`
		AskSample       string `gorm:"column:ask_sample"`
	}
	var snaps []snap
	if err := db.Raw(`
SELECT venue_platform, market_outcome_id, DATE_FORMAT(`+"`utc_date`"+`,'%Y-%m-%d') AS day_utc,
  COUNT(*) AS snap_n,
  MAX(LENGTH(order_book_bids)) AS max_bid_len,
  MIN(LEFT(order_book_bids, 100)) AS bid_sample,
  MIN(LEFT(order_book_asks, 100)) AS ask_sample
FROM data_history_order_book
WHERE map_key = ? AND record_kind = 'snapshot'
GROUP BY venue_platform, market_outcome_id, `+"`utc_date`"+`
ORDER BY venue_platform, market_outcome_id, `+"`utc_date`"+``, mapKey).Scan(&snaps).Error; err != nil {
		fmt.Println("ERR snaps:", err)
		os.Exit(1)
	}
	fmt.Println("\n=== snapshot groups (per outcome × day) ===")
	for _, s := range snaps {
		fmt.Printf("  %s outcome=%d day=%s snaps=%d max_bid_json_len=%d\n    bid: %s\n    ask: %s\n",
			s.VenuePlatform, s.MarketOutcomeID, s.DayUTC, s.SnapN, s.MaxBidLen, s.BidSample, s.AskSample)
	}

	type depth struct {
		VenuePlatform string
		OneLevel      int64
		MultiLevel    int64
		MaxPairs      int
	}
	var depths []depth
	if err := db.Raw(`
SELECT venue_platform,
  SUM(CASE WHEN order_book_bids NOT LIKE '%,%' AND order_book_asks NOT LIKE '%,%' THEN 1 ELSE 0 END) AS one_level,
  SUM(CASE WHEN order_book_bids LIKE '%,%' OR order_book_asks LIKE '%,%' THEN 1 ELSE 0 END) AS multi_level,
  MAX((LENGTH(order_book_bids) - LENGTH(REPLACE(order_book_bids, '],', '')))) AS max_pairs
FROM data_history_order_book
WHERE map_key = ? AND record_kind = 'snapshot'
GROUP BY venue_platform`, mapKey).Scan(&depths).Error; err != nil {
		fmt.Println("ERR depth:", err)
		os.Exit(1)
	}
	fmt.Println("\n=== snapshot depth (heuristic) ===")
	for _, d := range depths {
		fmt.Printf("  %s one_level~%d multi_level~%d max_level_markers~%d\n", d.VenuePlatform, d.OneLevel, d.MultiLevel, d.MaxPairs)
	}

	type task struct {
		ID         uint
		Status     string
		ErrorText  string
		UpdatedAt  string
	}
	var tasks []task
	_ = db.Raw(`SELECT id, status, IFNULL(error_text,'') AS error_text, updated_at
FROM data_task WHERE ingest_source='pmdata' ORDER BY id DESC LIMIT 3`).Scan(&tasks)
	fmt.Println("\n=== latest data_task (pmdata) ===")
	for _, t := range tasks {
		fmt.Printf("  id=%d status=%s updated=%s err=%s\n", t.ID, t.Status, t.UpdatedAt, t.ErrorText)
	}
}