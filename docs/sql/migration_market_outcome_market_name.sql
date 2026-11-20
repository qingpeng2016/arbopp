-- 父场次展示名冗余列，与 market.name 一致；由 market_sync 写入。

ALTER TABLE `market_outcome`
  ADD COLUMN `market_name` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '父场次名称，同 market.name' AFTER `name`;

UPDATE `market_outcome` AS mo
INNER JOIN `market` AS m ON m.`id` = mo.`market_row_id`
SET mo.`market_name` = m.`name`
WHERE mo.`market_row_id` > 0 AND m.`name` IS NOT NULL;