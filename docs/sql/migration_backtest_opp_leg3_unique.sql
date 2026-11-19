-- 1X2 回测机会：第三腿 id + opp_kind 纳入去重（与二元 yesno 共用 backtest_arb_binary_opportunity）

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `leg3_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '第三腿 market_outcome.id，二元为 0' AFTER `no_market_outcome_id`;

ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `leg3_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '第三腿 market_outcome.id，二元为 0' AFTER `no_market_outcome_id`;

ALTER TABLE `backtest_arb_binary_opportunity`
  DROP INDEX `uk_bt_opp_snap_legs`,
  ADD UNIQUE KEY `uk_bt_opp_snap_legs` (
    `backtest_task_id`,
    `snapshot_at_ms`,
    `opp_kind`,
    `yes_market_outcome_id`,
    `no_market_outcome_id`,
    `leg3_market_outcome_id`
  );