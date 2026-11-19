-- 移除未使用的组合盘机会表与调度项（已有库执行）
DROP TABLE IF EXISTS `arb_combo_opportunity`;

DELETE FROM `bot_schedule_config`
WHERE `module` = 'less_one' AND `task_name` = 'arb_scan_combo';