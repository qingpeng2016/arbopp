-- 双机公网联调（仅 POLY + PRED，无 SXBET）
-- 机器 A（POLY）: 54.65.44.140:9000
-- 机器 B（PRED）: 52.215.36.68:9000
-- 若角色对调，交换下面两套 IP 即可。
-- 执行前请备份；会 DROP node_route / node_config

SET NAMES utf8mb4;

DROP TABLE IF EXISTS `node_route`;
DROP TABLE IF EXISTS `node_config`;

CREATE TABLE `node_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `node_code` varchar(16) NOT NULL,
  `local_platform` varchar(32) NOT NULL,
  `official_ws` varchar(512) NOT NULL DEFAULT '',
  `node_name` varchar(64) NOT NULL DEFAULT '',
  `region` varchar(32) NOT NULL DEFAULT '',
  `vpc_cidr` varchar(32) NOT NULL DEFAULT '',
  `relay_listen` varchar(64) NOT NULL DEFAULT ':9000',
  `peer_ip` varchar(64) NOT NULL DEFAULT '',
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
  (`node_code`, `local_platform`, `official_ws`, `node_name`, `region`, `peer_ip`, `run_conf`, `description`)
VALUES
  ('POLY', 'polymarket', 'wss://ws-subscriptions-clob.polymarket.com/ws/market',
   'arb-poly-test-a', 'eu-west-1', '54.65.44.140:9000', 'prod', '公网联调 A=POLY'),
  ('PRED', 'predict', 'wss://ws.predict.fun/ws',
   'arb-pred-test-b', 'test', '52.215.36.68:9000', 'prod', '公网联调 B=PRED');

INSERT INTO `node_route` (`src_node`, `dst_node`, `route_json`, `description`) VALUES
  ('POLY', 'PRED', JSON_OBJECT(
    'mirror_platform', 'predict',
    'publish_local_book_to', '52.215.36.68:9000',
    'accept_mirror_cidr', '54.65.44.140/32',
    'hedge_order_to', '52.215.36.68:9000'
  ), 'POLY→PRED 公网'),
  ('PRED', 'POLY', JSON_OBJECT(
    'mirror_platform', 'polymarket',
    'publish_local_book_to', '54.65.44.140:9000',
    'accept_mirror_cidr', '52.215.36.68/32',
    'hedge_order_to', '54.65.44.140:9000'
  ), 'PRED→POLY 公网');