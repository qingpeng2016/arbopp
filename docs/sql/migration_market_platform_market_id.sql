-- market 表：market_id → platform_market_id（与 market_outcome / market_order_book 命名一致）

ALTER TABLE `market`
  CHANGE COLUMN `market_id` `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台场次/Event ID';

ALTER TABLE `market`
  DROP INDEX `uk_platform_market_sub`,
  ADD UNIQUE KEY `uk_platform_market_sub` (`platform`,`platform_market_id`,`category`,`sub_category`);