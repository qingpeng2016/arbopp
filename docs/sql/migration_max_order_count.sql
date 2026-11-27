-- strategy_entry 进程内实盘下单次数上限（内存计数；≤0 或不配=不限制；20s reconcile 重读）
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('max_order_count', 10.00000000, 'DECIMAL', 'strategy_entry 进程实盘下单次数上限（达限后只写机会表；≤0=不限制；热路径只读内存）', 1)
ON DUPLICATE KEY UPDATE
  `description` = VALUES(`description`),
  `status` = 1;
