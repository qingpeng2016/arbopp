-- Predict 网球 / MMA：与 POLYMARKET 对齐 competition_en（严格 marketmap 分桶）
-- Tennis：Poly ATP Tour / WTA Tour；Predict 原为 sync_all_sub + items 空 → competition_en=Tennis
-- MMA：Poly UFC；Predict 原为 sync_all_sub + items 空 → 易落 API tag 非 UFC
-- NFL / NHL 已有按 tag 的 nfl/nhl items，无需本脚本
-- 执行后：market_map；（可选）Predict market_sync

-- ---------- Tennis ----------
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "网球",
  "sub_category": "Tennis",
  "sport_en": "Tennis",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "sync_all_sub": true,
  "sync_root_tag_id": "85",
  "items": [
    {"name": "ATP", "name_en": "ATP Tour", "name_desc": "ATP Tour", "sport_id": "atp", "tag_ids": []},
    {"name": "WTA", "name_en": "WTA Tour", "name_desc": "WTA Tour", "sport_id": "wta", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Tennis';

UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'atp-%' OR LOWER(`slug`) = 'atp' THEN 'ATP'
    WHEN LOWER(`slug`) LIKE 'wta-%' OR LOWER(`slug`) = 'wta' THEN 'WTA'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'atp-%' OR LOWER(`slug`) = 'atp' THEN 'ATP Tour'
    WHEN LOWER(`slug`) LIKE 'wta-%' OR LOWER(`slug`) = 'wta' THEN 'WTA Tour'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Tennis'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ---------- MMA ----------
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "UFC",
  "sub_category": "MMA",
  "sport_en": "UFC",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "sync_all_sub": true,
  "sync_root_tag_id": "177",
  "items": [
    {"name": "UFC", "name_en": "UFC", "name_desc": "UFC", "sport_id": "ufc", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'MMA';

UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'ufc-%' OR LOWER(`slug`) = 'ufc' THEN 'UFC'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'ufc-%' OR LOWER(`slug`) = 'ufc' THEN 'UFC'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'MMA'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ---------- market_outcome（需已执行 migration_market_outcome_competition_en.sql）----------
UPDATE `market_outcome` AS mo
INNER JOIN `market` AS m ON m.`id` = mo.`market_row_id`
SET
  mo.`competition` = m.`competition`,
  mo.`competition_en` = m.`competition_en`,
  mo.`updated_at` = NOW()
WHERE m.`platform` = 'PREDICT'
  AND m.`sub_category` IN ('Tennis', 'MMA');

-- 自检
-- SELECT platform, sub_category, competition_en, COUNT(*) FROM market
-- WHERE platform = 'PREDICT' AND sub_category IN ('Tennis','MMA') GROUP BY 1,2,3;