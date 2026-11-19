-- market_sync 可拉取的盘口类型；见 common/enum/crypto/prediction/outcome_market_type.go
INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('market_sync', 'sync_outcome_types', '[{"sport_cn":"足球","sub_category":"Football","items":["moneyline","handicap","total"]}]', 'json', '按运动配置盘口。足球：moneyline=胜平负，handicap=让分，total=大小球。其他运动另加一组 sport_cn/sub_category', 3, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);

ALTER TABLE `market_outcome`
  ADD COLUMN `outcome_type` varchar(32) NOT NULL DEFAULT 'moneyline' COMMENT 'moneyline|handicap|total' AFTER `sub_category`,
  ADD KEY `idx_outcome_type` (`outcome_type`,`status`);