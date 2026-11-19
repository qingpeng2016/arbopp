-- 去掉 leg 命名：legs_json → outcomes_json，leg_no → side_no
-- 已有 order_books_json 内 JSON 键由 legs/leg_no 改为 outcomes/side_no，新写入由 strategy_entry 负责；历史行需自行迁移 JSON 或保留旧格式只读。

ALTER TABLE `arb_binary_opportunity`
  CHANGE COLUMN `legs_json` `outcomes_json` text NOT NULL COMMENT '组合两边摘要 JSON 数组：platform、market_id、side(YES|NO)、best_ask、size、fill';

ALTER TABLE `arb_binary_opportunity`
  MODIFY COLUMN `order_books_json` mediumtext NOT NULL COMMENT '命中时盘口快照 JSON：{outcomes:[{side_no,platform,platform_market_outcome_id,books:[{book_key,role,bids,asks}]}]}';

ALTER TABLE `arb_binary_order`
  CHANGE COLUMN `leg_no` `side_no` tinyint unsigned NOT NULL COMMENT '1=组合第一边 2=组合第二边';