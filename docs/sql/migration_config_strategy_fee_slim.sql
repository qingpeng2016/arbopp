-- 线上已有 config_strategy_fee：补 SXBET + 收成每平台一行默认（表结构不变）。
-- 必跑：节 1。节 2 可选，删掉与默认系数重复的运动分行。

-- 1) 四平台 Sports 默认（sub_category 空 = 全场运动兜底）
INSERT INTO `config_strategy_fee` (`platform`, `category`, `sub_category`, `taker_fee_coef`, `description`, `status`) VALUES
  ('POLYMARKET', 'Sports', '', 0.05000000, 'Poly 体育吃单：每份 fee=coef×p×(1−p)，官方 Sports coef=0.05', 1),
  ('KALSHI', 'Sports', '', 0.07000000, 'Kalshi 吃单 coef=0.07（系列乘数 1）；改乘数时加 sub_category 行覆盖', 1),
  ('PREDICT', 'Sports', '', 0.02000000, '无 market feeRateBps 时回退；有 bps 时策略用 bps/10000', 1),
  ('SXBET', 'Sports', '', 0.01000000, 'SX 赢家净利试算 coef；与代码 defaultSxbetSports=0.01 一致，Premium 可改为 0.02', 1)
ON DUPLICATE KEY UPDATE
  `taker_fee_coef` = VALUES(`taker_fee_coef`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

-- 2) 可选：删除「按运动拆分、系数与同平台 Sports 默认相同」的重复行
-- DELETE f FROM config_strategy_fee f
-- INNER JOIN config_strategy_fee d
--   ON d.platform = f.platform AND d.category = f.category AND d.sub_category = ''
--  AND d.status = 1 AND f.status = 1
-- WHERE f.sub_category <> '' AND f.taker_fee_coef = d.taker_fee_coef;