-- 把 market_order_book 对齐到线上一致的 market_order_book_ws：
-- 仅 snapshot_at_ms、无 token 唯一键、历史 INSERT

ALTER TABLE `market_order_book` DROP INDEX `uniq_platform_outcome_token`;

-- 若仍无 snapshot_at_ms（有 snapshot_second 时先加列再迁数据）：
-- ALTER TABLE `market_order_book`
--   ADD COLUMN `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '快照毫秒（REST 拉取）' AFTER `snapshot_second`;
-- UPDATE `market_order_book` SET `snapshot_at_ms` = `snapshot_second` * 1000 WHERE `snapshot_at_ms` = 0 AND `snapshot_second` > 0;
-- ALTER TABLE `market_order_book` DROP COLUMN `snapshot_second`;

ALTER TABLE `market_order_book`
  ADD KEY `idx_order_book_token_snap_ms` (`platform`, `token_id`, `snapshot_at_ms`);