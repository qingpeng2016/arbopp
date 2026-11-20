-- REST / strategy_entry：与 market_order_book_ws 相同语义 — 每次 INSERT 历史整快照，不按 token 覆盖

ALTER TABLE `market_order_book`
  DROP INDEX `uniq_platform_outcome_token`;

ALTER TABLE `market_order_book`
  ADD COLUMN `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT 'REST 拉取毫秒' AFTER `snapshot_second`;

ALTER TABLE `market_order_book`
  ADD KEY `idx_order_book_token_snap_ms` (`platform`, `token_id`, `snapshot_at_ms`);