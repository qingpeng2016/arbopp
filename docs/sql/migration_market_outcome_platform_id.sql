-- market_outcome：outcome_market_id → platform_market_outcome_id

ALTER TABLE `market_outcome`
  CHANGE COLUMN `outcome_market_id` `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台盘口 ID';

ALTER TABLE `market_outcome`
  DROP INDEX `uk_platform_outcome_sub`,
  ADD UNIQUE KEY `uk_platform_outcome_sub` (`platform`,`platform_market_outcome_id`,`category`,`sub_category`);