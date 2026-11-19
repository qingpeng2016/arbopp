-- arb 机会表补场次分类/联赛，与 market 对齐
ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `category` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER `map_key`,
  ADD COLUMN `sub_category` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER `category`,
  ADD COLUMN `competition` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）' AFTER `sub_category`,
  ADD COLUMN `competition_en` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '统一英文名，来自配置 name_en' AFTER `competition`;

ALTER TABLE `backtest_arb_binary_opportunity`
  ADD COLUMN `category` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER `map_key`,
  ADD COLUMN `sub_category` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER `category`,
  ADD COLUMN `competition` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）' AFTER `sub_category`,
  ADD COLUMN `competition_en` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '统一英文名，来自配置 name_en' AFTER `competition`;