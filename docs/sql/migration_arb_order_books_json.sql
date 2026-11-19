-- 已建 arb_*_opportunity 且无 order_books_json 列时执行
ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `order_books_json` mediumtext NULL COMMENT '命中时盘口快照 asks/bids' AFTER `outcomes_json`;

UPDATE `arb_binary_opportunity` SET `order_books_json` = '[]' WHERE `order_books_json` IS NULL;

ALTER TABLE `arb_binary_opportunity`
  MODIFY COLUMN `order_books_json` mediumtext NOT NULL COMMENT '命中时盘口快照 asks/bids';
