-- 检测 market.map_key 分组是否合理（不按 status 过滤）：
--   1) 同一 map_key 至少 2 个不同平台
--   2) 同一 map_key 下同一 platform 只能出现 1 条 market 行
--
-- 可选：仅核对场次时在各段取消注释 map_verified 条件。

-- ---------------------------------------------------------------------------
-- A) 汇总：各 map_key 平台数、行数、是否合规
-- ---------------------------------------------------------------------------
SELECT
    map_key,
    COUNT(*) AS market_rows,
    COUNT(DISTINCT UPPER(TRIM(platform))) AS platform_cnt,
    GROUP_CONCAT(DISTINCT CONCAT(UPPER(TRIM(platform)), ':', status) ORDER BY platform) AS platform_statuses,
    GROUP_CONCAT(DISTINCT UPPER(TRIM(platform)) ORDER BY UPPER(TRIM(platform))) AS platforms,
    CASE
        WHEN COUNT(DISTINCT UPPER(TRIM(platform))) < 2 THEN '平台不足2'
        WHEN COUNT(*) > COUNT(DISTINCT UPPER(TRIM(platform))) THEN '同平台重复'
        ELSE 'OK'
    END AS check_result
FROM market
WHERE map_key IS NOT NULL
  AND TRIM(map_key) <> ''
  -- AND map_verified = 1
GROUP BY map_key
ORDER BY check_result <> 'OK' DESC, map_key;

-- ---------------------------------------------------------------------------
-- B) 违规：平台数 < 2（单平台孤儿组）
-- ---------------------------------------------------------------------------
SELECT
    map_key,
    UPPER(TRIM(platform)) AS platform,
    status,
    id,
    platform_market_id,
    map_verified,
    competition_en,
    event_start_time
FROM market m
WHERE map_key IS NOT NULL
  AND TRIM(map_key) <> ''
  -- AND map_verified = 1
  AND map_key IN (
      SELECT map_key
      FROM market
      WHERE map_key IS NOT NULL
        AND TRIM(map_key) <> ''
        -- AND map_verified = 1
      GROUP BY map_key
      HAVING COUNT(DISTINCT UPPER(TRIM(platform))) < 2
  )
ORDER BY map_key, platform;

-- ---------------------------------------------------------------------------
-- C) 违规：同一 map_key 下同一平台出现 2 次及以上
-- ---------------------------------------------------------------------------
SELECT
    map_key,
    UPPER(TRIM(platform)) AS platform,
    COUNT(*) AS dup_cnt,
    GROUP_CONCAT(CONCAT('id=', id, ':', status) ORDER BY id) AS rows,
    GROUP_CONCAT(id ORDER BY id) AS market_ids,
    GROUP_CONCAT(platform_market_id ORDER BY id SEPARATOR ' | ') AS platform_market_ids
FROM market
WHERE map_key IS NOT NULL
  AND TRIM(map_key) <> ''
  -- AND map_verified = 1
GROUP BY map_key, UPPER(TRIM(platform))
HAVING COUNT(*) > 1
ORDER BY dup_cnt DESC, map_key, platform;

-- ---------------------------------------------------------------------------
-- D) 一行汇总计数
-- ---------------------------------------------------------------------------
SELECT
    SUM(CASE WHEN platform_cnt < 2 THEN 1 ELSE 0 END) AS bad_lt2_platform_groups,
    SUM(CASE WHEN market_rows > platform_cnt THEN 1 ELSE 0 END) AS bad_dup_platform_groups,
    SUM(CASE WHEN platform_cnt >= 2 AND market_rows = platform_cnt THEN 1 ELSE 0 END) AS ok_groups,
    COUNT(*) AS total_map_keys
FROM (
    SELECT
        map_key,
        COUNT(*) AS market_rows,
        COUNT(DISTINCT UPPER(TRIM(platform))) AS platform_cnt
    FROM market
    WHERE map_key IS NOT NULL
      AND TRIM(map_key) <> ''
      -- AND map_verified = 1
    GROUP BY map_key
) t;

-- ---------------------------------------------------------------------------
-- E) 所有不合规 map_key
-- ---------------------------------------------------------------------------
SELECT
    map_key,
    market_rows,
    platform_cnt,
    platforms,
    platform_statuses,
    violations
FROM (
    SELECT
        map_key,
        COUNT(*) AS market_rows,
        COUNT(DISTINCT UPPER(TRIM(platform))) AS platform_cnt,
        GROUP_CONCAT(DISTINCT UPPER(TRIM(platform)) ORDER BY UPPER(TRIM(platform))) AS platforms,
        GROUP_CONCAT(DISTINCT CONCAT(UPPER(TRIM(platform)), ':', status) ORDER BY platform) AS platform_statuses,
        TRIM(BOTH ',' FROM CONCAT(
            IF(COUNT(DISTINCT UPPER(TRIM(platform))) < 2, '平台不足2,', ''),
            IF(COUNT(*) > COUNT(DISTINCT UPPER(TRIM(platform))), '同平台重复,', '')
        )) AS violations
    FROM market
    WHERE map_key IS NOT NULL
      AND TRIM(map_key) <> ''
      -- AND map_verified = 1
    GROUP BY map_key
) g
WHERE violations <> ''
ORDER BY map_key;