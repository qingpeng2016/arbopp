-- map_key / map_verified：从 market_outcome 迁至 market（场次级对齐）。
-- 执行后请重跑 market_map（按场次写 map_key），必要时重跑 orderbook_sync。

ALTER TABLE `market`
  ADD COLUMN `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键（联赛+场次）' AFTER `competition`,
  ADD COLUMN `map_verified` tinyint(1) NOT NULL DEFAULT 0 AFTER `map_key`;

-- 旧 outcome.map_key 为四段 v1|联赛|场次|胜出方 → 场次键取前三段
UPDATE `market` m
INNER JOIN (
  SELECT
    `market_row_id`,
    SUBSTRING_INDEX(MIN(`map_key`), '|', 3) AS `fk`,
    MAX(`map_verified`) AS `mv`
  FROM `market_outcome`
  WHERE `market_row_id` > 0
    AND `map_key` IS NOT NULL
    AND TRIM(`map_key`) <> ''
  GROUP BY `market_row_id`
) x ON x.`market_row_id` = m.`id`
SET
  m.`map_key` = TRIM(x.`fk`),
  m.`map_verified` = IF(x.`mv` > 0, 1, 0);

ALTER TABLE `market_outcome`
  DROP INDEX `idx_outcome_map_key`,
  DROP COLUMN `map_key`,
  DROP COLUMN `map_verified`;

CREATE INDEX `idx_market_map_key` ON `market` (`map_key`, `status`);