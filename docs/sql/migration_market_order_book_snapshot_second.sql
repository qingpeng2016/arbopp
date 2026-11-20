-- market_order_book / market_order_book_ws：秒级快照标记（0 = REST/file 级当前价）

ALTER TABLE `market_order_book`
  ADD COLUMN `snapshot_second` bigint NOT NULL DEFAULT 0 COMMENT '快照秒级别' AFTER `order_book_bids`;

ALTER TABLE `market_order_book_ws`
  ADD COLUMN `snapshot_second` bigint NOT NULL DEFAULT 0 COMMENT '快照秒级别' AFTER `order_book_bids`;