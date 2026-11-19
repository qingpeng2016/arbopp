-- Predict 非足球：去掉 config_json.market_variants（API 会误过滤 SPORTS_NBA / SPORTS_TEAM_MATCH 等）
-- 足球行若存在则保留；同步逻辑对非足球已不传 marketVariants，此迁移与旧配置兼容。
-- 2026-11-18

UPDATE `market_category`
SET `config_json` = JSON_REMOVE(`config_json`, '$.market_variants'),
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` <> 'Football';