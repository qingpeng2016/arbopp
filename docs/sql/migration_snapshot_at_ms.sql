-- data_history_order_book：snapshot_ts_ms -> snapshot_at_ms（与机会表对齐）
ALTER TABLE `data_history_order_book`
  CHANGE COLUMN `snapshot_ts_ms` `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '快照毫秒时间戳；0 表示 file 级记录';

ALTER TABLE `data_history_order_book`
  DROP INDEX `idx_history_replay`;

ALTER TABLE `data_history_order_book`
  ADD KEY `idx_history_replay` (`map_key`, `snapshot_at_ms`, `venue_platform`);

-- 实盘机会
ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '命中瞬间毫秒时间戳（与 data_history_order_book.snapshot_at_ms 同义）' AFTER `snapshot_at`;

ALTER TABLE `arb_binary_opportunity`
  ADD KEY `idx_arb_binary_snap_ms` (`map_key`, `snapshot_at_ms`);

-- 回测机会
ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '回放时刻毫秒时间戳（对齐 data_history_order_book.snapshot_at_ms）' AFTER `snapshot_at`;

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD KEY `idx_bt_opp_snap_ms` (`backtest_task_id`, `snapshot_at_ms`);