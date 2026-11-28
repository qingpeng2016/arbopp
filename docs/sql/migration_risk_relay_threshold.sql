-- 节点间 TCP relay 风控（断链 / 内网 RTT / 镜像 BOOK 活性 / 定时推全量簿）
INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`, `status`)
VALUES
  ('risk_relay_enabled', 1.00000000, 'DECIMAL', '节点 relay 风控总开关（≥0.5 启用）', 1),
  ('risk_relay_check_interval_sec', 60.00000000, 'DECIMAL', '风控整轮周期（秒）：读阈值+PING+evaluate；与推书间隔无关', 1),
  ('risk_relay_max_rtt_ms', 2000.00000000, 'DECIMAL', '内网 PING-PONG RTT 上限（毫秒）；跨区建议≥1500', 1),
  ('risk_relay_book_stale_sec', 15.00000000, 'DECIMAL', '镜像 BOOK 超过该时长未收到则熔断（秒；建议≥2×推送间隔）', 1),
  ('risk_relay_book_push_interval_sec', 5.00000000, 'DECIMAL', '本机官方平台内存全量簿定时 TCP 推送间隔（秒）', 1),
  ('risk_relay_recover_ok_samples', 3.00000000, 'DECIMAL', '恢复前连续健康评估次数', 1),
  ('risk_relay_alert_repeat_sec', 300.00000000, 'DECIMAL', '熔断期间同一平台 Lark 重复提醒最短间隔（秒）', 1)
ON DUPLICATE KEY UPDATE
  `description` = VALUES(`description`),
  `status` = 1;