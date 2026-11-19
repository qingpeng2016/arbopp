-- arb / backtest 机会表：记录组合 YES/NO 价腿对应的 market_outcome.id

ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `yes_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'YES 价腿 market_outcome.id' AFTER `map_key`,
  ADD COLUMN `no_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'NO 价腿 market_outcome.id' AFTER `yes_market_outcome_id`,
  ADD KEY `idx_arb_binary_leg_outcomes` (`yes_market_outcome_id`, `no_market_outcome_id`);

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `yes_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'YES 价腿 market_outcome.id' AFTER `map_key`,
  ADD COLUMN `no_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT 'NO 价腿 market_outcome.id' AFTER `yes_market_outcome_id`,
  ADD UNIQUE KEY `uk_bt_opp_snap_legs` (`backtest_task_id`, `snapshot_at_ms`, `yes_market_outcome_id`, `no_market_outcome_id`);