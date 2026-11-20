-- market_order_book / market_order_book_ws：拉盘/WS 落库时的 Unix 秒（UTC 对齐用 time.Unix）

ALTER TABLE `market_order_book`
  ADD COLUMN `snapshot_second` bigint NOT NULL DEFAULT 0 COMMENT '快照秒级别' AFTER `order_book_bids`;

ALTER TABLE `market_order_book_ws`
  ADD COLUMN `snapshot_second` bigint NOT NULL DEFAULT 0 COMMENT '快照秒级别' AFTER `order_book_bids`;