-- 跨平台 map_key 漏链 / 错链粗查（MySQL 8+）
-- 在 fgmm_strategy_less_one 执行；队名精确匹配需 go run ./cmd/mapkey_cross_audit

-- ---------------------------------------------------------------------------
-- 0) 概况
-- ---------------------------------------------------------------------------
SELECT
  SUM(CASE WHEN map_key IS NULL OR TRIM(map_key) = '' THEN 1 ELSE 0 END) AS active_no_map_key,
  SUM(CASE WHEN map_key IS NOT NULL AND TRIM(map_key) <> '' AND map_verified = 0 THEN 1 ELSE 0 END) AS has_key_unverified,
  SUM(CASE WHEN map_key IS NOT NULL AND TRIM(map_key) <> '' AND map_verified = 1 THEN 1 ELSE 0 END) AS has_key_verified
FROM market
WHERE status = 'ACTIVE';

-- ---------------------------------------------------------------------------
-- 1) 漏链：同联赛桶 + 同一天开赛 + 不同平台 + map_key 不同（或一方为空）
--    用 title 里是否同时含两队简写做粗筛（会漏/误报，最终以 mapkey_cross_audit 为准）
-- ---------------------------------------------------------------------------
SELECT
  a.competition_en,
  a.category,
  a.sub_category,
  LEFT(a.event_start_time, 10) AS event_day,
  UPPER(a.platform) AS platform_a,
  a.id AS id_a,
  a.map_key AS map_key_a,
  a.map_verified AS verified_a,
  a.name AS name_a,
  UPPER(b.platform) AS platform_b,
  b.id AS id_b,
  b.map_key AS map_key_b,
  b.map_verified AS verified_b,
  b.name AS name_b
FROM market a
INNER JOIN market b
  ON a.status = 'ACTIVE'
 AND b.status = 'ACTIVE'
 AND a.id < b.id
 AND NOT (UPPER(TRIM(a.platform)) <=> UPPER(TRIM(b.platform)))
 AND a.category <=> b.category
 AND a.sub_category <=> b.sub_category
 AND a.competition_en <=> b.competition_en
 AND a.competition_en IS NOT NULL
 AND TRIM(a.competition_en) <> ''
 AND LEFT(COALESCE(a.event_start_time, ''), 10) = LEFT(COALESCE(b.event_start_time, ''), 10)
 AND LEFT(COALESCE(a.event_start_time, ''), 10) <> ''
 AND (
       (a.map_key IS NULL OR TRIM(a.map_key) = '' OR b.map_key IS NULL OR TRIM(b.map_key) = '')
       OR TRIM(a.map_key) <> TRIM(b.map_key)
     )
-- 粗队名：两边 name 都含 vs，且取 vs 左侧首词与右侧首词有交集（极简）
 AND LOWER(a.name) LIKE '%vs%'
 AND LOWER(b.name) LIKE '%vs%'
 AND (
       LOWER(a.name) LIKE CONCAT('%', SUBSTRING_INDEX(SUBSTRING_INDEX(LOWER(b.name), 'vs', 1), ' ', -1), '%')
    OR LOWER(b.name) LIKE CONCAT('%', SUBSTRING_INDEX(SUBSTRING_INDEX(LOWER(a.name), 'vs', 1), ' ', -1), '%')
     )
ORDER BY a.competition_en, event_day, a.id
LIMIT 100;

-- ---------------------------------------------------------------------------
-- 2) 错链疑义：同一 map_key 下开赛日不同
-- ---------------------------------------------------------------------------
SELECT
  TRIM(map_key) AS map_key,
  COUNT(DISTINCT LEFT(event_start_time, 10)) AS distinct_start_days,
  COUNT(DISTINCT UPPER(TRIM(platform))) AS platform_cnt,
  GROUP_CONCAT(DISTINCT CONCAT(UPPER(platform), ':', LEFT(event_start_time, 16), ':', id) ORDER BY platform SEPARATOR ' | ') AS samples
FROM market
WHERE status = 'ACTIVE'
  AND map_key IS NOT NULL
  AND TRIM(map_key) <> ''
GROUP BY TRIM(map_key)
HAVING distinct_start_days > 1
ORDER BY distinct_start_days DESC, map_key
LIMIT 50;

-- ---------------------------------------------------------------------------
-- 3) 错链疑义：同一 map_key + 同平台 多条（应对统计 dup）
-- ---------------------------------------------------------------------------
SELECT
  TRIM(map_key) AS map_key,
  UPPER(TRIM(platform)) AS platform,
  COUNT(*) AS dup_cnt,
  GROUP_CONCAT(id ORDER BY id) AS market_ids,
  GROUP_CONCAT(LEFT(name, 60) ORDER BY id SEPARATOR ' || ') AS names
FROM market
WHERE status = 'ACTIVE'
  AND map_key IS NOT NULL
  AND TRIM(map_key) <> ''
GROUP BY TRIM(map_key), UPPER(TRIM(platform))
HAVING dup_cnt > 1
ORDER BY dup_cnt DESC
LIMIT 50;

-- ---------------------------------------------------------------------------
-- 4) 联赛桶不一致却共用一个 map_key（competition_en 不同）
-- ---------------------------------------------------------------------------
SELECT
  TRIM(map_key) AS map_key,
  COUNT(DISTINCT competition_en) AS distinct_league_en,
  COUNT(DISTINCT CONCAT(category, '|', sub_category, '|', competition_en)) AS distinct_scopes,
  GROUP_CONCAT(DISTINCT competition_en ORDER BY competition_en) AS leagues
FROM market
WHERE status = 'ACTIVE'
  AND map_key IS NOT NULL
  AND TRIM(map_key) <> ''
GROUP BY TRIM(map_key)
HAVING distinct_league_en > 1
ORDER BY distinct_league_en DESC
LIMIT 50;

-- ---------------------------------------------------------------------------
-- 5) SXBET 有 map_key 但未与 Poly/Predict/Kalshi 同键（按 name+同日复查）
-- ---------------------------------------------------------------------------
SELECT
  sx.competition_en,
  LEFT(sx.event_start_time, 10) AS event_day,
  sx.id AS sx_id,
  sx.map_key AS sx_map_key,
  sx.name AS sx_name,
  other.platform,
  other.id AS other_id,
  other.map_key AS other_map_key,
  other.name AS other_name
FROM market sx
INNER JOIN market other
  ON sx.platform = 'SXBET'
 AND sx.status = 'ACTIVE'
 AND other.status = 'ACTIVE'
 AND other.platform IN ('POLYMARKET', 'PREDICT', 'KALSHI')
 AND sx.competition_en <=> other.competition_en
 AND sx.category <=> other.category
 AND sx.sub_category <=> other.sub_category
 AND LEFT(COALESCE(sx.event_start_time, ''), 10) = LEFT(COALESCE(other.event_start_time, ''), 10)
 AND LEFT(COALESCE(sx.event_start_time, ''), 10) <> ''
 AND (
       other.map_key IS NULL OR TRIM(other.map_key) = ''
       OR sx.map_key IS NULL OR TRIM(sx.map_key) = ''
       OR TRIM(sx.map_key) <> TRIM(other.map_key)
     )
 AND LOWER(sx.name) LIKE '%vs%'
 AND LOWER(other.name) LIKE '%vs%'
ORDER BY sx.competition_en, event_day
LIMIT 80;