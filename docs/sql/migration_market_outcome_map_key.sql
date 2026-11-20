-- market_outcome 冗余父场次 map_key（与 market.map_key 一致；由 market_map 维护，market_sync 不覆盖）
ALTER TABLE `market_outcome`
  ADD COLUMN `map_key` varchar(192) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key' AFTER `competition_en`;

CREATE INDEX `idx_outcome_map_key` ON `market_outcome` (`map_key`);

UPDATE `market_outcome` AS mo
INNER JOIN `market` AS m ON m.`id` = mo.`market_row_id`
SET mo.`map_key` = m.`map_key`,
    mo.`updated_at` = NOW()
WHERE m.`map_key` IS NOT NULL
  AND TRIM(m.`map_key`) <> '';