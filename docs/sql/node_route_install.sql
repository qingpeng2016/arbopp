-- node_config 3 行 + node_route 6 行（src→dst 一条边）
-- 重新执行会 DROP 两表，请先备份

SET NAMES utf8mb4;

DROP TABLE IF EXISTS `node_route`;
DROP TABLE IF EXISTS `node_config`;

CREATE TABLE `node_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `node_code` varchar(16) NOT NULL COMMENT 'POLY|PRED|SXBET',
  `local_platform` varchar(32) NOT NULL,
  `official_ws` varchar(512) NOT NULL DEFAULT '',
  `node_name` varchar(64) NOT NULL DEFAULT '',
  `region` varchar(32) NOT NULL DEFAULT '',
  `vpc_cidr` varchar(32) NOT NULL DEFAULT '',
  `relay_listen` varchar(64) NOT NULL DEFAULT ':9000',
  `peer_ip` varchar(64) NOT NULL DEFAULT '' COMMENT '本机 私网:9000',
  `run_conf` varchar(32) NOT NULL DEFAULT 'local',
  `description` varchar(500) NOT NULL DEFAULT '',
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_node_code` (`node_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `node_route` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `src_node` varchar(16) NOT NULL,
  `dst_node` varchar(16) NOT NULL,
  `route_json` json NOT NULL,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `description` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_src_dst` (`src_node`, `dst_node`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `node_config`
  (`node_code`, `local_platform`, `official_ws`, `node_name`, `region`, `vpc_cidr`, `peer_ip`, `run_conf`, `description`)
VALUES
  ('POLY',  'polymarket', 'wss://ws-subscriptions-clob.polymarket.com/ws/market', 'arb-poly-ir',  'eu-west-1',      '10.1.0.0/16',  '10.1.0.10:9000',  'prod', '爱尔兰'),
  ('PRED',  'predict',    'wss://ws.predict.fun/ws',                             'arb-pred-tk',  'ap-northeast-1', '10.2.0.0/16',  '10.2.0.10:9000',  'prod', '东京'),
  ('SXBET', 'sxbet',      'wss://realtime.sx.bet/connection/websocket',          'arb-sxbet-hk', 'ap-east-1',      '10.10.0.0/16', '10.10.0.10:9000', 'prod', '香港');

INSERT INTO `node_route` (`src_node`, `dst_node`, `route_json`, `description`)
VALUES
  ('POLY', 'PRED', JSON_OBJECT(
    'mirror_platform', 'predict',
    'publish_local_book_to', '10.2.0.10:9000',
    'accept_mirror_cidr', '10.2.0.0/16',
    'hedge_order_to', '10.2.0.10:9000'
  ), 'POLY→PRED'),

  ('POLY', 'SXBET', JSON_OBJECT(
    'mirror_platform', 'sxbet',
    'publish_local_book_to', '10.10.0.10:9000',
    'accept_mirror_cidr', '10.10.0.0/16',
    'hedge_order_to', '10.10.0.10:9000'
  ), 'POLY→SXBET'),

  ('PRED', 'POLY', JSON_OBJECT(
    'mirror_platform', 'polymarket',
    'publish_local_book_to', '10.1.0.10:9000',
    'accept_mirror_cidr', '10.1.0.0/16',
    'hedge_order_to', '10.1.0.10:9000'
  ), 'PRED→POLY'),

  ('PRED', 'SXBET', JSON_OBJECT(
    'mirror_platform', 'sxbet',
    'publish_local_book_to', '10.10.0.10:9000',
    'accept_mirror_cidr', '10.10.0.0/16',
    'hedge_order_to', '10.10.0.10:9000'
  ), 'PRED→SXBET'),

  ('SXBET', 'POLY', JSON_OBJECT(
    'mirror_platform', 'polymarket',
    'publish_local_book_to', '10.1.0.10:9000',
    'accept_mirror_cidr', '10.1.0.0/16',
    'hedge_order_to', '10.1.0.10:9000'
  ), 'SXBET→POLY'),

  ('SXBET', 'PRED', JSON_OBJECT(
    'mirror_platform', 'predict',
    'publish_local_book_to', '10.2.0.10:9000',
    'accept_mirror_cidr', '10.2.0.0/16',
    'hedge_order_to', '10.2.0.10:9000'
  ), 'SXBET→PRED');