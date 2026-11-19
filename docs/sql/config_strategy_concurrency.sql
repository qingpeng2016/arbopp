INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`)
VALUES
  ('market_sync', 'concurrency', '{"polymarket":{"concurrency":5},"kalshi":{"concurrency":3},"predict":{"concurrency":2}}', 'json', '市场同步：polymarket/kalshi/predict 的 concurrency（1-8）', 4, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);