-- 修正 Predict 官网链接（错误：/category/{slug} 或 /market/{numeric_id}）
-- 正确：https://predict.fun/market/{category_slug}

UPDATE `market`
SET `market_url` = CONCAT('https://predict.fun/market/', LOWER(TRIM(`platform_market_id`))),
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND TRIM(`platform_market_id`) <> '';

UPDATE `market_outcome`
SET `market_url` = CONCAT('https://predict.fun/market/', LOWER(TRIM(`platform_market_id`))),
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND TRIM(`platform_market_id`) <> '';

UPDATE `market`
SET `competition` = NULL, `competition_en` = NULL, `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND (`competition` = '全量' OR `competition_en` LIKE '%tag_ids 未配置%');

UPDATE `market_outcome`
SET `competition` = NULL, `updated_at` = NOW()
WHERE `platform` = 'PREDICT' AND `competition` = '全量';