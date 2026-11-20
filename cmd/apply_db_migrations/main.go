package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"gorm.io/gorm"
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	ctx := context.Background()

	steps := []struct {
		name string
		fn   func(*gorm.DB) error
	}{
		{"backtest_orders", ensureBacktestOrders},
		{"backtest_task.progress", ensureBacktestTaskProgress},
		{"backtest_opp_uk", ensureBacktestOppUniqueKey},
		{"bot stat_collect", ensureBotStatCollect},
		{"data_task.ingest_source", ensureDataTaskIngestSource},
	}

	for _, s := range steps {
		if err := s.fn(db.WithContext(ctx)); err != nil {
			fmt.Printf("FAIL %s: %v\n", s.name, err)
			os.Exit(1)
		}
		fmt.Printf("OK %s\n", s.name)
	}
	fmt.Println("done")
}

func hasTable(db *gorm.DB, name string) bool {
	var n int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, name).Scan(&n)
	return n > 0
}

func hasColumn(db *gorm.DB, table, col string) bool {
	var n int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, col).Scan(&n)
	return n > 0
}

func hasIndex(db *gorm.DB, table, index string) bool {
	var n int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`, table, index).Scan(&n)
	return n > 0
}

func ensureBacktestOrders(db *gorm.DB) error {
	if hasTable(db, "backtest_orders") {
		return nil
	}
	// docs/sql/migration_backtest_orders.sql
	return db.Exec(`
CREATE TABLE IF NOT EXISTS backtest_orders (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  backtest_task_id bigint unsigned NOT NULL DEFAULT 0,
  strategy_type varchar(32) NOT NULL DEFAULT 'BINARY_YES_NO',
  arb_opp_id bigint unsigned NOT NULL,
  map_key varchar(192) NOT NULL,
  side_no tinyint unsigned NOT NULL,
  platform varchar(32) NOT NULL,
  platform_market_outcome_id varchar(255) NOT NULL DEFAULT '',
  token_id varchar(255) NOT NULL DEFAULT '',
  side varchar(8) NOT NULL,
  action varchar(16) NOT NULL DEFAULT 'BUY',
  price decimal(20,8) NOT NULL,
  qty decimal(20,8) NOT NULL,
  fill_qty decimal(20,8) NOT NULL DEFAULT 0,
  status varchar(32) NOT NULL,
  err_msg varchar(1024) NOT NULL DEFAULT '',
  sent_at datetime NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_backtest_orders_task (backtest_task_id, sent_at),
  KEY idx_backtest_orders_opp (arb_opp_id),
  KEY idx_backtest_orders_map_time (map_key, sent_at),
  KEY idx_backtest_orders_strategy (strategy_type, sent_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`).Error
}

func ensureBacktestTaskProgress(db *gorm.DB) error {
	if !hasTable(db, "backtest_task") {
		return nil
	}
	if hasColumn(db, "backtest_task", "progress") {
		return nil
	}
	return db.Exec(`ALTER TABLE backtest_task ADD COLUMN progress tinyint unsigned NOT NULL DEFAULT 0 COMMENT '回放进度 0-100' AFTER status`).Error
}

func ensureBacktestOppUniqueKey(db *gorm.DB) error {
	if !hasTable(db, "backtest_arb_binary_opportunity") {
		return nil
	}
	if !hasColumn(db, "backtest_arb_binary_opportunity", "leg3_market_outcome_id") {
		if err := db.Exec(`ALTER TABLE backtest_arb_binary_opportunity ADD COLUMN leg3_market_outcome_id bigint unsigned NOT NULL DEFAULT 0 AFTER no_market_outcome_id`).Error; err != nil {
			return err
		}
	}
	want := "uk_bt_opp_snap_legs"
	if hasIndex(db, "backtest_arb_binary_opportunity", want) {
		return nil
	}
	// drop old uk without opp_kind/leg3 if present (ignore errors)
	_ = db.Exec(`ALTER TABLE backtest_arb_binary_opportunity DROP INDEX uk_bt_opp_snap_legs`).Error
	return db.Exec(`ALTER TABLE backtest_arb_binary_opportunity ADD UNIQUE KEY uk_bt_opp_snap_legs (
    backtest_task_id, snapshot_at_ms, opp_kind, yes_market_outcome_id, no_market_outcome_id, leg3_market_outcome_id
  )`).Error
}

func ensureDataTaskIngestSource(db *gorm.DB) error {
	if !hasTable(db, "data_task") {
		return nil
	}
	if !hasColumn(db, "data_task", "ingest_source") {
		if err := db.Exec(`
ALTER TABLE data_task
  ADD COLUMN ingest_source varchar(16) NOT NULL DEFAULT 'pmdata'
    COMMENT 'pmdata|synthetic' AFTER id`).Error; err != nil {
			return err
		}
	}
	if !hasColumn(db, "data_task", "params") {
		if err := db.Exec(`
ALTER TABLE data_task
  ADD COLUMN params text NULL COMMENT 'JSON task overrides' AFTER map_key`).Error; err != nil {
			return err
		}
	}
	if hasColumn(db, "data_task", "task_type") {
		if err := db.Exec(`UPDATE data_task SET ingest_source = 'synthetic' WHERE task_type = 'synthetic_l2'`).Error; err != nil {
			return err
		}
		if err := db.Exec(`UPDATE data_task SET ingest_source = 'pmdata' WHERE task_type <> 'synthetic_l2'`).Error; err != nil {
			return err
		}
		if hasIndex(db, "data_task", "idx_data_task_status") {
			if err := db.Exec(`ALTER TABLE data_task DROP INDEX idx_data_task_status`).Error; err != nil {
				return err
			}
		}
		if err := db.Exec(`ALTER TABLE data_task DROP COLUMN task_type`).Error; err != nil {
			return err
		}
	}
	if !hasIndex(db, "data_task", "idx_data_task_ingest_status") {
		if err := db.Exec(`ALTER TABLE data_task ADD KEY idx_data_task_ingest_status (ingest_source, status)`).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureBotStatCollect(db *gorm.DB) error {
	var n int64
	db.Raw(`SELECT COUNT(*) FROM bot_schedule_config WHERE module='less_one' AND task_name='stat_collect'`).Scan(&n)
	if n > 0 {
		return nil
	}
	return db.Exec(`
INSERT INTO bot_schedule_config (module, task_name, interval_seconds, exe_sort, concurrency, description, is_enabled, is_strategy_enabled, is_primary_machine_run)
VALUES ('less_one', 'stat_collect', 300, 25, 1, '刷新 stat_kv', 1, 1, 0)`).Error
}