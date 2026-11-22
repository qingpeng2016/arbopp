-- 仅此一段：删 leg3_market_outcome_id，加 draw_info / draw_order_books / draw_best_ask

ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `draw_info` mediumtext NULL AFTER `no_order_books`,
  ADD COLUMN `draw_order_books` mediumtext NULL AFTER `draw_info`,
  ADD COLUMN `draw_best_ask` decimal(20,8) NULL AFTER `no_best_ask`;

UPDATE `arb_binary_opportunity` SET
  draw_info = COALESCE(draw_info, '{}\n'),
  draw_order_books = COALESCE(draw_order_books, '{}\n');

ALTER TABLE `arb_binary_opportunity`
  MODIFY COLUMN draw_info mediumtext NOT NULL,
  MODIFY COLUMN draw_order_books mediumtext NOT NULL,
  DROP COLUMN leg3_market_outcome_id;

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `draw_info` mediumtext NULL AFTER `no_order_books`,
  ADD COLUMN `draw_order_books` mediumtext NULL AFTER `draw_info`,
  ADD COLUMN `draw_best_ask` decimal(20,8) NULL AFTER `no_best_ask`;

UPDATE `backtest_arb_binary_opportunity` SET
  draw_info = COALESCE(draw_info, '{}\n'),
  draw_order_books = COALESCE(draw_order_books, '{}\n');

ALTER TABLE `backtest_arb_binary_opportunity`
  MODIFY COLUMN draw_info mediumtext NOT NULL,
  MODIFY COLUMN draw_order_books mediumtext NOT NULL,
  DROP INDEX uk_bt_opp_snap_legs,
  DROP COLUMN leg3_market_outcome_id;

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD UNIQUE KEY `uk_bt_opp_snap_legs` (`backtest_task_id`, `snapshot_at_ms`, `opp_kind`, `map_key`(96), `total_cost`);