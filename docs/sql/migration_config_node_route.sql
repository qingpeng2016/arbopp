-- 双主节点 + WS/内网/下单路由（node_config + node_route）
-- 执行前：将 10.1.0.10 / 10.2.0.10 换成爱尔兰 / 东京 ECS 私网 IP
-- 用法：mysql -h ... -u ... -p your_db < docs/sql/migration_config_node_route.sql

SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS `node_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `node_code` varchar(16) NOT NULL COMMENT '启动参数 --node，如 IR / TK',
  `node_name` varchar(64) NOT NULL DEFAULT '' COMMENT '人类可读名',
  `region` varchar(32) NOT NULL DEFAULT '' COMMENT '云上区域 eu-west-1 / ap-northeast-1',
  `vpc_cidr` varchar(32) NOT NULL DEFAULT '' COMMENT '本 VPC 网段',
  `run_conf` varchar(32) NOT NULL DEFAULT 'local' COMMENT '对应 --run_conf',
  `relay_listen` varchar(64) NOT NULL DEFAULT ':9000' COMMENT '内网 relay 监听',
  `description` varchar(500) NOT NULL DEFAULT '',
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_node_code` (`node_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='双主节点定义';

CREATE TABLE IF NOT EXISTS `node_route` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `node_code` varchar(16) NOT NULL COMMENT 'node_config.node_code',
  `route_kind` varchar(32) NOT NULL COMMENT 'ws_official|ws_mirror|relay_publish_book|relay_accept_book|order_local|order_peer',
  `platform` varchar(32) NOT NULL DEFAULT '' COMMENT 'polymarket|predict',
  `target` varchar(512) NOT NULL DEFAULT '' COMMENT 'official=WS URL；peer=host:port；accept=CIDR',
  `triggers_arb` tinyint(1) NOT NULL DEFAULT 0 COMMENT '仅 ws_official 通常为 1',
  `sort_no` int NOT NULL DEFAULT 0,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `description` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_node_kind_platform_target` (`node_code`, `route_kind`, `platform`, `target`(191)),
  KEY `idx_node_enabled` (`node_code`, `is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点 WS/内网/下单路由';

-- 重复执行时先清示例数据（按需注释掉）
DELETE FROM `node_route` WHERE `node_code` IN ('IR', 'TK');
DELETE FROM `node_config` WHERE `node_code` IN ('IR', 'TK');

INSERT INTO `node_config`
  (`node_code`, `node_name`, `region`, `vpc_cidr`, `run_conf`, `relay_listen`, `description`)
VALUES
  ('IR', 'arb-poly-ir', 'eu-west-1', '10.1.0.0/16', 'prod', ':9000', '欧洲 Poly 本地腿'),
  ('TK', 'arb-pred-tk', 'ap-northeast-1', '10.2.0.0/16', 'prod', ':9000', '东京 Predict 本地腿');

INSERT INTO `node_route`
  (`node_code`, `route_kind`, `platform`, `target`, `triggers_arb`, `sort_no`, `description`)
VALUES
  ('IR', 'ws_official', 'polymarket', 'wss://ws-subscriptions-clob.polymarket.com/ws/market', 1, 10, 'Poly 官方 WS'),
  ('IR', 'ws_mirror', 'predict', 'peer', 0, 20, 'Predict 只吃内网镜像'),
  ('IR', 'relay_publish_book', 'polymarket', '10.2.0.10:9000', 0, 30, '镜像 Poly 盘口到东京'),
  ('IR', 'relay_accept_book', 'predict', '10.2.0.0/16', 0, 40, '收东京发来的 Predict book'),
  ('IR', 'order_local', 'polymarket', 'official', 0, 50, '触发后本机直连 Poly 下单'),
  ('IR', 'order_peer', 'predict', '10.2.0.10:9000', 0, 60, '对冲腿：内网 HEDGE 东京下 Predict'),

  ('TK', 'ws_official', 'predict', 'wss://ws.predict.fun/ws', 1, 10, 'Predict 官方 WS'),
  ('TK', 'ws_mirror', 'polymarket', 'peer', 0, 20, 'Poly 只吃内网镜像'),
  ('TK', 'relay_publish_book', 'predict', '10.1.0.10:9000', 0, 30, '镜像 Predict 盘口到爱尔兰'),
  ('TK', 'relay_accept_book', 'polymarket', '10.1.0.0/16', 0, 40, '收爱尔兰发来的 Poly book'),
  ('TK', 'order_local', 'predict', 'official', 0, 50, '触发后本机直连 Predict 下单'),
  ('TK', 'order_peer', 'polymarket', '10.1.0.10:9000', 0, 60, '对冲腿：内网 HEDGE 爱尔兰下 Poly');