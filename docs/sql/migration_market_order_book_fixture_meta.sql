-- market_order_book 冗余场次维度，便于按 map_key / 联赛筛选
ALTER TABLE `market_order_book`
  ADD COLUMN `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key' AFTER `market_row_id`,
  ADD COLUMN `category` varchar(200) DEFAULT NULL AFTER `map_key`,
  ADD COLUMN `sub_category` varchar(200) DEFAULT NULL AFTER `category`,
  ADD COLUMN `competition` varchar(64) DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）' AFTER `sub_category`,
  ADD COLUMN `competition_en` varchar(128) DEFAULT NULL COMMENT '统一英文名，来自配置 name_en' AFTER `competition`,
  ADD KEY `idx_order_book_map_key` (`map_key`);