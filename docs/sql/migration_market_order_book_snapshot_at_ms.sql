-- 仅补列（代码 INSERT 需要 snapshot_at_ms）；与 migration_market_order_book_history.sql 二选一或顺序执行

ALTER TABLE `market_order_book`
  ADD COLUMN `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT 'REST 拉取毫秒' AFTER `snapshot_second`;