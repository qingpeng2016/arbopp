-- WS 盘口历史：每次落库 INSERT 新行，不按 token 覆盖
-- 若已跑过 migration_market_order_book_ws_unique_snapshot_second.sql，本脚本仍会 DROP 相关唯一键

ALTER TABLE `market_order_book_ws`
  DROP INDEX `uniq_platform_outcome_token`;

ALTER TABLE `market_order_book_ws`
  DROP INDEX `uniq_platform_outcome_token_snap`;

ALTER TABLE `market_order_book_ws`
  ADD COLUMN `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '快照毫秒（平台 WS 时间；0=无）' AFTER `snapshot_second`;

ALTER TABLE `market_order_book_ws`
  ADD KEY `idx_order_book_ws_token_snap_ms` (`platform`, `token_id`, `snapshot_at_ms`);