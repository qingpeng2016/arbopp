-- 各平台展示层统一：NCAA 大学篮球统称「NCAA篮球」（不区分男子/女子配置名）
-- Poly：日常单场走 sport_id cbb/cwbb；去掉无单场的 ncaab 配置项
-- 执行后建议再跑 market_sync / stat_collect

UPDATE `market`
SET `competition` = 'NCAA篮球',
    `competition_en` = 'NCAA Basketball',
    `updated_at` = NOW()
WHERE TRIM(`competition`) IN ('NCAA男子篮球', 'NCAA女子篮球', 'NCAA Men''s Basketball', 'Women''s College Basketball', 'College Basketball')
   OR TRIM(`competition_en`) IN ('NCAA Men''s Basketball', 'Women''s College Basketball', 'College Basketball');

UPDATE `market_outcome`
SET `competition` = 'NCAA篮球',
    `updated_at` = NOW()
WHERE TRIM(`competition`) IN ('NCAA男子篮球', 'NCAA女子篮球', 'NCAA Men''s Basketball', 'Women''s College Basketball', 'College Basketball');

UPDATE `arb_binary_opportunity`
SET `competition` = 'NCAA篮球',
    `competition_en` = 'NCAA Basketball',
    `updated_at` = NOW()
WHERE TRIM(`competition`) IN ('NCAA男子篮球', 'NCAA女子篮球');