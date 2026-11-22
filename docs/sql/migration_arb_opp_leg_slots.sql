-- arb / backtest 机会表：yes/no/draw 腿列 → leg_1/2/3（按槽位，语义见 opp_kind）
-- 映射仅改列名，不搬数据：draw→leg_1，yes→leg_2，no→leg_3

ALTER TABLE `arb_binary_opportunity`
  CHANGE COLUMN `draw_info` `leg_1_info` mediumtext NOT NULL COMMENT '组合腿1 JSON；BINARY 为 {}；1X2 为平局',
  CHANGE COLUMN `draw_order_books` `leg_1_order_books` mediumtext NOT NULL COMMENT '腿1 全量盘口',
  CHANGE COLUMN `draw_best_ask` `leg_1_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿1 卖一（0~1）',
  CHANGE COLUMN `yes_info` `leg_2_info` mediumtext NOT NULL COMMENT '腿2 JSON；BINARY=YES；1X2=主队',
  CHANGE COLUMN `yes_order_books` `leg_2_order_books` mediumtext NOT NULL,
  CHANGE COLUMN `yes_best_ask` `leg_2_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿2 卖一（0~1）',
  CHANGE COLUMN `no_info` `leg_3_info` mediumtext NOT NULL COMMENT '腿3 JSON；BINARY=NO；1X2=客队',
  CHANGE COLUMN `no_order_books` `leg_3_order_books` mediumtext NOT NULL,
  CHANGE COLUMN `no_best_ask` `leg_3_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿3 卖一（0~1）';

ALTER TABLE `backtest_arb_binary_opportunity`
  CHANGE COLUMN `draw_info` `leg_1_info` mediumtext NOT NULL COMMENT '组合腿1 JSON；BINARY 为 {}；1X2 为平局',
  CHANGE COLUMN `draw_order_books` `leg_1_order_books` mediumtext NOT NULL COMMENT '腿1 全量盘口',
  CHANGE COLUMN `draw_best_ask` `leg_1_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿1 卖一（0~1）',
  CHANGE COLUMN `yes_info` `leg_2_info` mediumtext NOT NULL COMMENT '腿2 JSON；BINARY=YES；1X2=主队',
  CHANGE COLUMN `yes_order_books` `leg_2_order_books` mediumtext NOT NULL,
  CHANGE COLUMN `yes_best_ask` `leg_2_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿2 卖一（0~1）',
  CHANGE COLUMN `no_info` `leg_3_info` mediumtext NOT NULL COMMENT '腿3 JSON；BINARY=NO；1X2=客队',
  CHANGE COLUMN `no_order_books` `leg_3_order_books` mediumtext NOT NULL,
  CHANGE COLUMN `no_best_ask` `leg_3_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿3 卖一（0~1）';