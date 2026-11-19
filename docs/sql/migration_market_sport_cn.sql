-- market 运动中文分类（UI 上的「足球/篮球」，与 market_category.sport_cn 一致）
-- category 仍为顶层 Sports；sub_category 仍为 Football/Basketball 等

ALTER TABLE `market`
  ADD COLUMN `sport_cn` varchar(64) DEFAULT NULL COMMENT '运动分类中文名（足球/篮球）' AFTER `sub_category`;

UPDATE `market` m
INNER JOIN `market_category` c
  ON c.`platform` = m.`platform` AND c.`sub_category` = m.`sub_category` AND c.`is_enabled` = 1
SET m.`sport_cn` = c.`sport_cn`
WHERE m.`sport_cn` IS NULL OR TRIM(m.`sport_cn`) = '';