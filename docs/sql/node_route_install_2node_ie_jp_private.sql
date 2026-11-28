-- 双机跨区私网 relay（POLY 爱尔兰 + PRED 东京）
-- POLY 公网 46.51.144.94  私网 10.249.16.56  → eu-west-1 / Polymarket
-- PRED 公网 18.176.194.218 私网 10.249.17.50  → ap-northeast-1 / Predict
-- 前置：跨区 VPC/对等连通，安全组放行 TCP 9000（建议仅对端私网 /32）
-- 执行：两台共用同一 MySQL 时执行一次；若各机本地 MySQL 各执行一次。
-- 全量重装（会 DROP 两表）见下方；仅更新数据用「仅 INSERT」段。

SET NAMES utf8mb4;

-- -------- 可选：干净安装（会删 SXBET 等同表旧数据）--------
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
  (`node_code`, `local_platform`, `official_ws`, `node_name`, `region`, `vpc_cidr`, `relay_listen`, `peer_ip`, `run_conf`, `description`)
VALUES
  ('POLY', 'polymarket', 'wss://ws-subscriptions-clob.polymarket.com/ws/market',
   'arb-poly-ie', 'eu-west-1', '10.249.16.0/24', ':9000', '10.249.16.56:9000', 'test',
   '爱尔兰 Poly；relay 本机私网监听，对端 PRED 10.249.17.50'),
  ('PRED', 'predict', 'wss://ws.predict.fun/ws',
   'arb-pred-jp', 'ap-northeast-1', '10.249.17.0/24', ':9000', '10.249.17.50:9000', 'test',
   '东京 Predict；对端 POLY 10.249.16.56');

INSERT INTO `node_route` (`src_node`, `dst_node`, `route_json`, `description`) VALUES
  ('POLY', 'PRED', JSON_OBJECT(
    'mirror_platform', 'predict',
    'publish_local_book_to', '10.249.17.50:9000',
    'accept_mirror_cidr', '10.249.17.50/32',
    'hedge_order_to', '10.249.17.50:9000'
  ), 'POLY→PRED 跨区私网'),
  ('PRED', 'POLY', JSON_OBJECT(
    'mirror_platform', 'polymarket',
    'publish_local_book_to', '10.249.16.56:9000',
    'accept_mirror_cidr', '10.249.16.56/32',
    'hedge_order_to', '10.249.16.56:9000'
  ), 'PRED→POLY 跨区私网');