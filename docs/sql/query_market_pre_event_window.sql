-- 与 config strategy_entry.pre_event_scan_minutes 一致（默认 30）
-- 口径：event_start_time(UTC) <= NOW(UTC) + N 分钟（含已开赛、开赛前 N 分钟内将开赛的行）
-- 执行前：SET time_zone = '+00:00';  或让应用/实例默认 UTC

SET @pre_event_minutes := 30;

SELECT
    m.id,
    m.platform,
    TRIM(m.map_key) AS map_key,
    m.competition,
    m.sub_category,
    m.name,
    m.event_start_time,
    (
        COALESCE(
            STR_TO_DATE(
                REPLACE(
                    REPLACE(SUBSTRING_INDEX(TRIM(m.event_start_time), '.', 1), 'Z', ''),
                    'T', ' '
                ),
                '%Y-%m-%d %H:%i:%s'
            ),
            STR_TO_DATE(TRIM(m.event_start_time), '%Y-%m-%d %H:%i:%s')
        )
    ) AS event_start_utc
FROM market m
WHERE m.status = 'ACTIVE'
  AND m.map_verified = 1
  AND m.map_key IS NOT NULL
  AND TRIM(m.map_key) <> ''
  AND m.event_start_time IS NOT NULL
  AND TRIM(m.event_start_time) <> ''
HAVING event_start_utc IS NOT NULL
   AND event_start_utc <= UTC_TIMESTAMP() + INTERVAL @pre_event_minutes MINUTE
ORDER BY event_start_utc ASC, map_key, m.platform;

-- 只要「未开赛且距离开赛 ≤ 30 分钟」：在 HAVING 增加
--   AND event_start_utc >= UTC_TIMESTAMP();