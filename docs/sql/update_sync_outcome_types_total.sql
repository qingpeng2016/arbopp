UPDATE `config_strategy`
SET `item_value` = '[{"sport_cn":"足球","sub_category":"Football","items":["moneyline","handicap","total"]}]',
    `description` = '按运动配置盘口。足球：moneyline=胜平负，handicap=让分，total=大小球。其他运动另加一组 sport_cn/sub_category'
WHERE `group_name` = 'market_sync'
  AND `item_key` = 'sync_outcome_types';

ALTER TABLE `market_outcome`
  MODIFY COLUMN `outcome_type` varchar(32) NOT NULL DEFAULT 'moneyline' COMMENT 'moneyline|handicap|total';
