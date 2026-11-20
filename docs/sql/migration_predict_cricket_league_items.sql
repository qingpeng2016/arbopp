-- Predict 板球：与 POLYMARKET 同名的 league items（sync_all_sub 仍用根 tag 97 全量拉取；items 供 slug 前缀 → competition_en，marketmap 严格分桶）
-- 执行后：跑 Predict market_sync（可选，新数据自动对）+ 下方存量回填 + market_map

UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "板球",
  "sub_category": "Cricket",
  "sport_en": "Cricket",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "sync_all_sub": true,
  "sync_root_tag_id": "97",
  "items": [
    {"name": "IPL", "name_en": "IPL", "name_desc": "IPL", "sport_id": "ipl", "tag_ids": []},
    {"name": "ODI", "name_en": "ODI", "name_desc": "ODI", "sport_id": "odi", "tag_ids": []},
    {"name": "T20", "name_en": "T20 Internationals", "name_desc": "T20 Internationals", "sport_id": "t20", "tag_ids": []},
    {"name": "Big Bash League", "name_en": "Big Bash League", "name_desc": "Big Bash League", "sport_id": "abb", "tag_ids": []},
    {"name": "CSA T20 Challenge", "name_en": "CSA T20 Challenge", "name_desc": "CSA T20 Challenge", "sport_id": "csa", "tag_ids": []},
    {"name": "Cricket Bangladesh", "name_en": "Cricket Bangladesh", "name_desc": "Cricket Bangladesh", "sport_id": "crban", "tag_ids": []},
    {"name": "International", "name_en": "International", "name_desc": "International", "sport_id": "crint", "tag_ids": []},
    {"name": "Cricket Australia", "name_en": "Cricket Australia", "name_desc": "Cricket Australia", "sport_id": "craus", "tag_ids": []},
    {"name": "Cricket England", "name_en": "Cricket England", "name_desc": "Cricket England", "sport_id": "creng", "tag_ids": []},
    {"name": "Cricket New Zealand", "name_en": "Cricket New Zealand", "name_desc": "Cricket New Zealand", "sport_id": "crnew", "tag_ids": []},
    {"name": "Cricket India", "name_en": "Cricket India", "name_desc": "Cricket India", "sport_id": "crind", "tag_ids": []},
    {"name": "Cricket South Africa", "name_en": "Cricket South Africa", "name_desc": "Cricket South Africa", "sport_id": "crsou", "tag_ids": []},
    {"name": "Cricket Pakistan", "name_en": "Cricket Pakistan", "name_desc": "Cricket Pakistan", "sport_id": "crpak", "tag_ids": []},
    {"name": "Cricket UAE", "name_en": "Cricket UAE", "name_desc": "Cricket UAE", "sport_id": "cruae", "tag_ids": []},
    {"name": "Test Cricket", "name_en": "Test Cricket", "name_desc": "Test Cricket", "sport_id": "test", "tag_ids": []},
    {"name": "Sheffield Shield", "name_en": "Sheffield Shield", "name_desc": "Sheffield Shield", "sport_id": "she", "tag_ids": []},
    {"name": "SA20", "name_en": "SA20", "name_desc": "SA20", "sport_id": "sasa", "tag_ids": []},
    {"name": "Lanka Premier League", "name_en": "Lanka Premier League", "name_desc": "Lanka Premier League", "sport_id": "lpl", "tag_ids": []},
    {"name": "U19 Cricket World Cup", "name_en": "U19 Cricket World Cup", "name_desc": "U19 Cricket World Cup", "sport_id": "cru19wc", "tag_ids": []},
    {"name": "Women''s Premier League", "name_en": "Women''s Premier League", "name_desc": "Women''s Premier League", "sport_id": "crwpl20", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Cricket';

-- 存量 PREDICT 板球场次：按 slug 前缀写 competition / competition_en（与 Poly items 一致；CASE 顺序：较长 sport_id 在前）
UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'crwpl20-%' OR LOWER(`slug`) = 'crwpl20' THEN 'Women''s Premier League'
    WHEN LOWER(`slug`) LIKE 'cru19wc-%' OR LOWER(`slug`) = 'cru19wc' THEN 'U19 Cricket World Cup'
    WHEN LOWER(`slug`) LIKE 'crban-%' OR LOWER(`slug`) = 'crban' THEN 'Cricket Bangladesh'
    WHEN LOWER(`slug`) LIKE 'crint-%' OR LOWER(`slug`) = 'crint' THEN 'International'
    WHEN LOWER(`slug`) LIKE 'craus-%' OR LOWER(`slug`) = 'craus' THEN 'Cricket Australia'
    WHEN LOWER(`slug`) LIKE 'creng-%' OR LOWER(`slug`) = 'creng' THEN 'Cricket England'
    WHEN LOWER(`slug`) LIKE 'crnew-%' OR LOWER(`slug`) = 'crnew' THEN 'Cricket New Zealand'
    WHEN LOWER(`slug`) LIKE 'crind-%' OR LOWER(`slug`) = 'crind' THEN 'Cricket India'
    WHEN LOWER(`slug`) LIKE 'crsou-%' OR LOWER(`slug`) = 'crsou' THEN 'Cricket South Africa'
    WHEN LOWER(`slug`) LIKE 'crpak-%' OR LOWER(`slug`) = 'crpak' THEN 'Cricket Pakistan'
    WHEN LOWER(`slug`) LIKE 'cruae-%' OR LOWER(`slug`) = 'cruae' THEN 'Cricket UAE'
    WHEN LOWER(`slug`) LIKE 'csa-%' OR LOWER(`slug`) = 'csa' THEN 'CSA T20 Challenge'
    WHEN LOWER(`slug`) LIKE 'abb-%' OR LOWER(`slug`) = 'abb' THEN 'Big Bash League'
    WHEN LOWER(`slug`) LIKE 'sasa-%' OR LOWER(`slug`) = 'sasa' THEN 'SA20'
    WHEN LOWER(`slug`) LIKE 'she-%' OR LOWER(`slug`) = 'she' THEN 'Sheffield Shield'
    WHEN LOWER(`slug`) LIKE 'test-%' OR LOWER(`slug`) = 'test' THEN 'Test Cricket'
    WHEN LOWER(`slug`) LIKE 'lpl-%' OR LOWER(`slug`) = 'lpl' THEN 'Lanka Premier League'
    WHEN LOWER(`slug`) LIKE 'ipl-%' OR LOWER(`slug`) = 'ipl' THEN 'IPL'
    WHEN LOWER(`slug`) LIKE 'odi-%' OR LOWER(`slug`) = 'odi' THEN 'ODI'
    WHEN LOWER(`slug`) LIKE 't20-%' OR LOWER(`slug`) = 't20' THEN 'T20'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'crwpl20-%' OR LOWER(`slug`) = 'crwpl20' THEN 'Women''s Premier League'
    WHEN LOWER(`slug`) LIKE 'cru19wc-%' OR LOWER(`slug`) = 'cru19wc' THEN 'U19 Cricket World Cup'
    WHEN LOWER(`slug`) LIKE 'crban-%' OR LOWER(`slug`) = 'crban' THEN 'Cricket Bangladesh'
    WHEN LOWER(`slug`) LIKE 'crint-%' OR LOWER(`slug`) = 'crint' THEN 'International'
    WHEN LOWER(`slug`) LIKE 'craus-%' OR LOWER(`slug`) = 'craus' THEN 'Cricket Australia'
    WHEN LOWER(`slug`) LIKE 'creng-%' OR LOWER(`slug`) = 'creng' THEN 'Cricket England'
    WHEN LOWER(`slug`) LIKE 'crnew-%' OR LOWER(`slug`) = 'crnew' THEN 'Cricket New Zealand'
    WHEN LOWER(`slug`) LIKE 'crind-%' OR LOWER(`slug`) = 'crind' THEN 'Cricket India'
    WHEN LOWER(`slug`) LIKE 'crsou-%' OR LOWER(`slug`) = 'crsou' THEN 'Cricket South Africa'
    WHEN LOWER(`slug`) LIKE 'crpak-%' OR LOWER(`slug`) = 'crpak' THEN 'Cricket Pakistan'
    WHEN LOWER(`slug`) LIKE 'cruae-%' OR LOWER(`slug`) = 'cruae' THEN 'Cricket UAE'
    WHEN LOWER(`slug`) LIKE 'csa-%' OR LOWER(`slug`) = 'csa' THEN 'CSA T20 Challenge'
    WHEN LOWER(`slug`) LIKE 'abb-%' OR LOWER(`slug`) = 'abb' THEN 'Big Bash League'
    WHEN LOWER(`slug`) LIKE 'sasa-%' OR LOWER(`slug`) = 'sasa' THEN 'SA20'
    WHEN LOWER(`slug`) LIKE 'she-%' OR LOWER(`slug`) = 'she' THEN 'Sheffield Shield'
    WHEN LOWER(`slug`) LIKE 'test-%' OR LOWER(`slug`) = 'test' THEN 'Test Cricket'
    WHEN LOWER(`slug`) LIKE 'lpl-%' OR LOWER(`slug`) = 'lpl' THEN 'Lanka Premier League'
    WHEN LOWER(`slug`) LIKE 'ipl-%' OR LOWER(`slug`) = 'ipl' THEN 'IPL'
    WHEN LOWER(`slug`) LIKE 'odi-%' OR LOWER(`slug`) = 'odi' THEN 'ODI'
    WHEN LOWER(`slug`) LIKE 't20-%' OR LOWER(`slug`) = 't20' THEN 'T20 Internationals'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Cricket'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

UPDATE `market_outcome` AS mo
INNER JOIN `market` AS m ON m.`id` = mo.`market_row_id`
SET
  mo.`competition` = m.`competition`,
  mo.`competition_en` = m.`competition_en`,
  mo.`updated_at` = NOW()
WHERE m.`platform` = 'PREDICT'
  AND m.`sub_category` = 'Cricket';

-- 自检：仍落在根 sport_en 的场次（需补 sport_id 或手工核对）
-- SELECT id, slug, competition, competition_en FROM market
-- WHERE platform = 'PREDICT' AND sub_category = 'Cricket' AND competition_en IN ('Cricket', '');