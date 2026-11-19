INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('orderbook_sync', 'orderbook_fetch', '{"polymarket":{"batch_size":500,"batch_concurrency":3},"kalshi":{"batch_size":100,"batch_concurrency":3}}', 'json', '拉盘口：每批条数 batch_size；多批并发 batch_concurrency（Poly /books 单请求最多 500 token）', 1, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);