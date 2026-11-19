-- market_order_book：补场次关联列（盘口列见 migration_market_order_book_platform_outcome_id.sql）

ALTER TABLE `market_order_book`
  ADD COLUMN `market_row_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '父场次 market.id' AFTER `platform`,
  ADD COLUMN `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '父场次平台 ID，对应 market.platform_market_id' AFTER `market_row_id`;

UPDATE `market_order_book` ob
INNER JOIN `market_outcome` o
  ON o.`platform` = ob.`platform`
 AND o.`platform_market_outcome_id` = ob.`market_id`
SET ob.`market_row_id` = o.`market_row_id`,
    ob.`platform_market_id` = o.`platform_market_id`
WHERE ob.`market_row_id` = 0;

ALTER TABLE `market_order_book`
  ADD KEY `idx_order_book_market_row` (`market_row_id`),
  ADD KEY `idx_order_book_platform_fixture` (`platform`,`platform_market_id`);