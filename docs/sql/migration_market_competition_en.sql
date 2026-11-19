-- market.competition 来自配置 name；competition_en 来自同一条配置的 name_en
ALTER TABLE `market`
  ADD COLUMN `competition_en` varchar(128) DEFAULT NULL COMMENT '统一英文名，来自配置 name_en' AFTER `competition`;
