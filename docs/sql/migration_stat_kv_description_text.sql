-- stat_kv.description 改为 TEXT，存完整 market_running_count_strategy groups JSON
ALTER TABLE `stat_kv`
  MODIFY COLUMN `description` text NOT NULL COMMENT '说明或扩展 JSON';