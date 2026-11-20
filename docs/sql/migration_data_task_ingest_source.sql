-- data_task：用 ingest_source 区分 PMData / 伪造；去掉冗余 task_type。
-- map_key 仍表示一场跨平台场次（脚本据此解析 Poly slug / Predict market_id）。
-- params JSON 可选，主要给 pmdata 免费档覆盖日期窗口，例如：
--   {"utc_days_before":1,"utc_days_after":1,"min_utc_age_days":21}
--   {"utc_dates":["2026-10-01","2026-10-02"]}

ALTER TABLE `data_task`
  ADD COLUMN `ingest_source` varchar(16) NOT NULL DEFAULT 'pmdata'
    COMMENT 'pmdata=history_data_sync；synthetic=synthetic_history_sync' AFTER `id`,
  ADD COLUMN `params` text NULL COMMENT 'JSON，任务级覆盖项' AFTER `map_key`;

UPDATE `data_task` SET `ingest_source` = 'synthetic' WHERE `task_type` = 'synthetic_l2';
UPDATE `data_task` SET `ingest_source` = 'pmdata' WHERE `task_type` <> 'synthetic_l2' OR `task_type` IS NULL;

ALTER TABLE `data_task` DROP INDEX `idx_data_task_status`;
ALTER TABLE `data_task` DROP COLUMN `task_type`;
ALTER TABLE `data_task`
  ADD KEY `idx_data_task_ingest_status` (`ingest_source`, `status`);