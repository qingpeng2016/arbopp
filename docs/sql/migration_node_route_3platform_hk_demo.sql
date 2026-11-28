-- 三平台示例：Polymarket + Predict + SXBET，节点均在「香港」（演示用 IP / 网段可随便改）
-- 依赖表：node_config、node_route（见 migration_config_node_route.sql）
-- 演示拓扑：3 节点同 VPC 10.10.0.0/16，私网 10.10.0.10 / .11 / .12，book 与 HEDGE 均 1→2 推送

SET NAMES utf8mb4;

-- 若表尚未创建，取消下面两行注释并执行 migration_config_node_route.sql 中的 CREATE 段

DELETE FROM `node_route` WHERE `node_code` IN ('POLY', 'PRED', 'SXBET');
DELETE FROM `node_config` WHERE `node_code` IN ('POLY', 'PRED', 'SXBET');

INSERT INTO `node_config`
  (`node_code`, `node_name`, `region`, `vpc_cidr`, `run_conf`, `relay_listen`, `description`)
VALUES
  ('POLY', 'arb-poly-hk', 'ap-east-1', '10.10.0.0/16', 'local', ':9000', '香港区 Poly 本地腿（示例）'),
  ('PRED', 'arb-pred-hk', 'ap-east-1', '10.10.0.0/16', 'local', ':9000', '香港区 Predict 本地腿（示例）'),
  ('SXBET', 'arb-sxbet-hk', 'ap-east-1', '10.10.0.0/16', 'local', ':9000', '香港区 SXBet 本地腿（示例）');

-- route_kind 说明：
--   ws_official / ws_mirror：行情来源
--   relay_publish_book / relay_accept_book：内网镜像 1→N
--   order_local / order_peer：本机官方下单 vs 内网 HEDGE 对端下单

-- ===================== POLY 节点（本地 Polymarket）=====================
INSERT INTO `node_route`
  (`node_code`, `route_kind`, `platform`, `target`, `triggers_arb`, `sort_no`, `description`)
VALUES
  ('POLY', 'ws_official', 'polymarket', 'wss://ws-subscriptions-clob.polymarket.com/ws/market', 1, 10, 'Poly 官方 WS'),
  ('POLY', 'ws_mirror', 'predict', 'peer', 0, 20, 'Predict 内网镜像'),
  ('POLY', 'ws_mirror', 'sxbet', 'peer', 0, 21, 'SXBet 内网镜像'),
  ('POLY', 'relay_publish_book', 'polymarket', '10.10.0.11:9000', 0, 30, '推 Poly book → PRED'),
  ('POLY', 'relay_publish_book', 'polymarket', '10.10.0.12:9000', 0, 31, '推 Poly book → SXBET'),
  ('POLY', 'relay_accept_book', 'predict', '10.10.0.0/16', 0, 40, '收 Predict 镜像'),
  ('POLY', 'relay_accept_book', 'sxbet', '10.10.0.0/16', 0, 41, '收 SXBet 镜像'),
  ('POLY', 'order_local', 'polymarket', 'official', 0, 50, '本机 Poly 官方下单'),
  ('POLY', 'order_peer', 'predict', '10.10.0.11:9000', 0, 60, 'HEDGE → PRED 下 Predict'),
  ('POLY', 'order_peer', 'sxbet', '10.10.0.12:9000', 0, 61, 'HEDGE → SXBET 下 SX');

-- ===================== PRED 节点（本地 Predict）=====================
INSERT INTO `node_route`
  (`node_code`, `route_kind`, `platform`, `target`, `triggers_arb`, `sort_no`, `description`)
VALUES
  ('PRED', 'ws_official', 'predict', 'wss://ws.predict.fun/ws', 1, 10, 'Predict 官方 WS'),
  ('PRED', 'ws_mirror', 'polymarket', 'peer', 0, 20, 'Poly 内网镜像'),
  ('PRED', 'ws_mirror', 'sxbet', 'peer', 0, 21, 'SXBet 内网镜像'),
  ('PRED', 'relay_publish_book', 'predict', '10.10.0.10:9000', 0, 30, '推 Predict book → POLY'),
  ('PRED', 'relay_publish_book', 'predict', '10.10.0.12:9000', 0, 31, '推 Predict book → SXBET'),
  ('PRED', 'relay_accept_book', 'polymarket', '10.10.0.0/16', 0, 40, '收 Poly 镜像'),
  ('PRED', 'relay_accept_book', 'sxbet', '10.10.0.0/16', 0, 41, '收 SXBet 镜像'),
  ('PRED', 'order_local', 'predict', 'official', 0, 50, '本机 Predict 官方下单'),
  ('PRED', 'order_peer', 'polymarket', '10.10.0.10:9000', 0, 60, 'HEDGE → POLY 下 Poly'),
  ('PRED', 'order_peer', 'sxbet', '10.10.0.12:9000', 0, 61, 'HEDGE → SXBET 下 SX');

-- ===================== SXBET 节点（本地 SXBet）=====================
INSERT INTO `node_route`
  (`node_code`, `route_kind`, `platform`, `target`, `triggers_arb`, `sort_no`, `description`)
VALUES
  ('SXBET', 'ws_official', 'sxbet', 'wss://realtime.sx.bet/connection/websocket', 1, 10, 'SXBet 官方 WS'),
  ('SXBET', 'ws_mirror', 'polymarket', 'peer', 0, 20, 'Poly 内网镜像'),
  ('SXBET', 'ws_mirror', 'predict', 'peer', 0, 21, 'Predict 内网镜像'),
  ('SXBET', 'relay_publish_book', 'sxbet', '10.10.0.10:9000', 0, 30, '推 SX book → POLY'),
  ('SXBET', 'relay_publish_book', 'sxbet', '10.10.0.11:9000', 0, 31, '推 SX book → PRED'),
  ('SXBET', 'relay_accept_book', 'polymarket', '10.10.0.0/16', 0, 40, '收 Poly 镜像'),
  ('SXBET', 'relay_accept_book', 'predict', '10.10.0.0/16', 0, 41, '收 Predict 镜像'),
  ('SXBET', 'order_local', 'sxbet', 'official', 0, 50, '本机 SX 官方下单'),
  ('SXBET', 'order_peer', 'polymarket', '10.10.0.10:9000', 0, 60, 'HEDGE → POLY 下 Poly'),
  ('SXBET', 'order_peer', 'predict', '10.10.0.11:9000', 0, 61, 'HEDGE → PRED 下 Predict');