-- market_outcome：platform_market_id = 平台场次 ID；market_row_id = market.id 主键关联

ALTER TABLE `market_outcome`
  CHANGE COLUMN `fixture_market_id` `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '父场次平台 ID，对应 market.platform_market_id',
  ADD COLUMN `market_row_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '父场次 market.id' AFTER `platform`;

UPDATE `market_outcome` o
INNER JOIN `market` m
  ON m.`platform` = o.`platform`
 AND m.`platform_market_id` = o.`platform_market_id`
 AND (m.`category` <=> o.`category`)
 AND (m.`sub_category` <=> o.`sub_category`)
SET o.`market_row_id` = m.`id`
WHERE o.`market_row_id` = 0;

ALTER TABLE `market_outcome`
  DROP INDEX `idx_outcome_fixture`,
  ADD KEY `idx_outcome_market_row` (`market_row_id`),
  ADD KEY `idx_outcome_platform_market` (`platform`,`platform_market_id`,`category`,`sub_category`);