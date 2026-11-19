-- 已有 backtest_task 表：增加 progress 列（0-100，每 10% 更新一次）

ALTER TABLE `backtest_task`
  ADD COLUMN `progress` tinyint unsigned NOT NULL DEFAULT 0 COMMENT '回放进度 0-100' AFTER `status`;

-- 监控
SELECT `id`, `map_key`, `status`, `progress`, `error_text`, `updated_at`
FROM `backtest_task`
WHERE `status` = 'running'
ORDER BY `id` DESC;