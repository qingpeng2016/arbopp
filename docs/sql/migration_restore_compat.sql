-- 从旧备份恢复库后，与当前代码对齐（2026-11）
-- 已在 fgmm_strategy_less_one @ 54.151.158.221 跑过；空库请用 rebuild_database_full.sql

-- 1) 回测订单独立表（代码写 backtest_orders，不再用 backtest_arb_binary_order）
-- 见 migration_backtest_orders.sql（完整 CREATE）

-- 2) backtest_task 进度
ALTER TABLE `backtest_task`
  ADD COLUMN `progress` tinyint unsigned NOT NULL DEFAULT 0 COMMENT '回放进度 0-100' AFTER `status`;

-- 3) 回测机会去重（含 1X2 opp_kind + leg3）
ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `leg3_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '第三腿；二元为 0' AFTER `no_market_outcome_id`;
ALTER TABLE `backtest_arb_binary_opportunity` DROP INDEX `uk_bt_opp_snap_legs`;
ALTER TABLE `backtest_arb_binary_opportunity` ADD UNIQUE KEY `uk_bt_opp_snap_legs` (
  `backtest_task_id`, `snapshot_at_ms`, `opp_kind`,
  `yes_market_outcome_id`, `no_market_outcome_id`, `leg3_market_outcome_id`
);

-- 4) stat_collect 调度（若缺失）
INSERT INTO `bot_schedule_config` (`module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`, `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`)
SELECT 'less_one', 'stat_collect', 300, 25, 1, '刷新 stat_kv', 1, 1, 0
FROM DUAL WHERE NOT EXISTS (
  SELECT 1 FROM `bot_schedule_config` WHERE `module` = 'less_one' AND `task_name` = 'stat_collect'
);

-- 本地执行：RUN_CONF=local go run ./cmd/apply_db_migrations/
-- 审计：RUN_CONF=local go run ./cmd/schema_audit/