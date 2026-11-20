-- market_outcome：子盘口名称列 name → outcome_name（与 market_name 对称）

ALTER TABLE `market_outcome`
  CHANGE COLUMN `name` `outcome_name` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '子盘口名称（question/title）';