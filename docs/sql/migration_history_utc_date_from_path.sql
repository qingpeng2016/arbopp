-- 一次性修正：从 local_file_path 恢复 utc_date（Parquet 日历日）。
-- 路径形如 var/pmdata/.../l2/2026-10-17/...parquet
-- utc_date 为保留字，列名必须用反引号。
UPDATE `data_history_order_book`
SET `utc_date` = STR_TO_DATE(
  SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(`local_file_path`, '\\', '/'), '/l2/', -1), '/', 1),
  '%Y-%m-%d'
)
WHERE `record_kind` IN ('snapshot', 'file')
  AND `local_file_path` LIKE '%/l2/20%/%'
  AND SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(`local_file_path`, '\\', '/'), '/l2/', -1), '/', 1) REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2}$';