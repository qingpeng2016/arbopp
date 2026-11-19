-- 联赛展示名（如英超、欧冠），与 config JSON 的 name 一致，便于跨平台对照
ALTER TABLE `market`
  ADD COLUMN `competition` varchar(64) DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）' AFTER `league`;
