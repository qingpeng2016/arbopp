UPDATE `config_strategy`
SET `item_key` = 'concurrency',
    `item_value` = REPLACE(REPLACE(`item_value`, '"league_concurrency"', '"concurrency"'), 'league_concurrency', 'concurrency'),
    `description` = '市场同步：polymarket/kalshi/predict 的 concurrency（1-8）'
WHERE `group_name` = 'market_sync'
  AND `item_key` IN ('market_league_fetch', 'league_concurrency', 'concurrency');

UPDATE `market_category`
SET `config_json` = REPLACE(`config_json`, '"league_concurrency"', '"concurrency"')
WHERE `config_json` LIKE '%league_concurrency%';

UPDATE `market_category`
SET `concurrency` = GREATEST(1, LEAST(8, CAST(JSON_UNQUOTE(JSON_EXTRACT(`config_json`, '$.concurrency')) AS SIGNED)))
WHERE `concurrency` = 1 AND JSON_EXTRACT(`config_json`, '$.concurrency') IS NOT NULL;

UPDATE `market_category`
SET `config_json` = JSON_REMOVE(JSON_REMOVE(`config_json`, '$.concurrency'), '$.league_concurrency')
WHERE JSON_CONTAINS_PATH(`config_json`, 'one', '$.concurrency')
   OR JSON_CONTAINS_PATH(`config_json`, 'one', '$.league_concurrency');