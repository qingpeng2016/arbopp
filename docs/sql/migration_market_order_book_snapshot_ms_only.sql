-- 两表只保留 snapshot_at_ms；历史快照 INSERT，去掉 token 唯一键

-- -------- market_order_book --------
ALTER TABLE `market_order_book` DROP INDEX `uniq_platform_outcome_token`;

ALTER TABLE `market_order_book`
  ADD COLUMN `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '快照毫秒（REST 拉取）' AFTER `snapshot_second`;

UPDATE `market_order_book`
SET `snapshot_at_ms` = `snapshot_second` * 1000
WHERE `snapshot_at_ms` = 0 AND `snapshot_second` > 0;

ALTER TABLE `market_order_book` DROP COLUMN `snapshot_second`;

ALTER TABLE `market_order_book`
  ADD KEY `idx_order_book_token_snap_ms` (`platform`, `token_id`, `snapshot_at_ms`);

-- -------- market_order_book_ws（若仍有 snapshot_second）--------
UPDATE `market_order_book_ws`
SET `snapshot_at_ms` = `snapshot_second` * 1000
WHERE `snapshot_at_ms` = 0 AND `snapshot_second` > 0;

ALTER TABLE `market_order_book_ws` DROP COLUMN `snapshot_second`;