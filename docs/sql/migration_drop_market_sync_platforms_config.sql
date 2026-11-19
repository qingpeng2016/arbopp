-- market_sync 参与平台改由 platform.status=ACTIVE 控制，删除冗余 config_strategy 项
DELETE FROM `config_strategy`
WHERE `group_name` = 'market_sync' AND `item_key` = 'platforms';
