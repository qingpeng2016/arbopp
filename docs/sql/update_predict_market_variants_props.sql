-- 让分/大小在独立 category（SPORTS_PROPS，slug 后缀 -more-markets）；须拉取该 variant 才能同步 handicap/total

UPDATE `config_strategy`
SET
  `item_value` = JSON_SET(
    `item_value`,
    '$[0].market_variants',
    JSON_ARRAY('SPORTS_MATCH', 'SPORTS_TEAM_MATCH', 'SPORTS_PROPS')
  ),
  `updated_at` = NOW()
WHERE `group_name` = 'market_sync'
  AND `item_key` = 'predict_sports';