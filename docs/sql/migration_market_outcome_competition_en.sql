-- market_outcome：与 market 对齐联赛英文名（marketmap 分桶用 market；盘口侧冗余便于查询/落库）
-- category / sub_category / competition 已存在，仅补 competition_en

ALTER TABLE `market_outcome`
  ADD COLUMN `competition_en` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '统一英文名，来自配置 name_en' AFTER `competition`;

UPDATE `market_outcome` AS mo
INNER JOIN `market` AS m ON m.`id` = mo.`market_row_id`
SET
  mo.`competition` = m.`competition`,
  mo.`competition_en` = m.`competition_en`,
  mo.`updated_at` = NOW()
WHERE mo.`market_row_id` > 0;