-- market_order_book：market_id → platform_market_outcome_id

ALTER TABLE `market_order_book`
  CHANGE COLUMN `market_id` `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台盘口 ID';

-- 列已改名时可只执行索引重命名段
ALTER TABLE `market_order_book`
  DROP INDEX `uniq_market_token_platform`,
  ADD UNIQUE KEY `uniq_platform_outcome_token` (`platform`,`platform_market_outcome_id`,`token_id`);

ALTER TABLE `market_order_book`
  DROP INDEX `idx_order_book_platform_market`,
  ADD KEY `idx_order_book_platform_outcome` (`platform`,`platform_market_outcome_id`);