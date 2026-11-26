-- =============================================================================
-- install_database_all.sql — 空库一键导入（自动生成，勿手改）
-- 生成：docs/sql/generate_install_database_all.sh
-- 说明：docs/sql/INSTALL_DATABASE.md
-- =============================================================================
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;


-- ########## 1/5 schema + 全局种子 ##########
-- from: docs/sql/rebuild_database_full.sql

-- =============================================================================
-- 空库 / 删库后：表结构 + 全局配置种子（与当前代码一致，2026-11）
-- 不含：market / market_outcome / market_order_book / stat_kv 运行时数据
--
-- 完整重建请用：
--   mysql DB < docs/sql/install_database_all.sql
-- 或 ./docs/sql/rebuild_database.sh DBNAME
-- 说明：docs/sql/INSTALL_DATABASE.md
-- 若只执行本文件，还须再执行：
--   config_market_category_install.sql  （联赛同步）
--   config_strategy_fee.sql             （手续费 config_strategy_fee，必跑）
--   predict_market_sync.sql             （Predict，可选）
--   migration_predict_all_league_align.sql （Predict 全联赛对齐，推荐）
--
-- 手续费：strategy_entry 读 config_strategy_fee（非 config_strategy_threshold）
--   Poly Sports 默认 0.05；Kalshi Sports 默认 0.07；Predict 见 predict 脚本
-- =============================================================================
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 平台：仅 URL/超时等公共连接配置，不含任何账号密钥
CREATE TABLE IF NOT EXISTS `platform` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(50) NOT NULL COMMENT 'POLYMARKET, KALSHI, PINNACLE, BET365 …',
  `platform_type` varchar(20) NOT NULL DEFAULT 'PREDICTION' COMMENT 'PREDICTION=预测市场, ODDS=赔率(BC)',
  `quote_style` varchar(24) NOT NULL DEFAULT 'PROBABILITY' COMMENT 'PROBABILITY=0~1价|DECIMAL_ODDS=欧赔|AMERICAN_ODDS',
  `display_name` varchar(64) NOT NULL DEFAULT '',
  `timeout` int NOT NULL DEFAULT 30,
  `retry_times` int NOT NULL DEFAULT 3,
  `retry_interval` int NOT NULL DEFAULT 1,
  `market_api_base_url` varchar(512) DEFAULT NULL,
  `order_api_base_url` varchar(512) DEFAULT NULL,
  `websocket_url` varchar(512) DEFAULT NULL,
  `user_websocket_url` varchar(512) DEFAULT NULL,
  `grpc_host` varchar(255) DEFAULT NULL COMMENT 'Foregate',
  `fg_api_host` varchar(255) DEFAULT NULL COMMENT 'Foregate fgopenapi',
  `fg_kms_host` varchar(255) DEFAULT NULL,
  `extra_config` text COMMENT '其余非敏感 JSON',
  `status` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `sort` int NOT NULL DEFAULT 0,
  `remark` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_slug` (`slug`),
  KEY `idx_platform_type` (`platform_type`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='预测市场 + 赔率平台连接配置';

-- 下单凭据：API Key / KMS key_id / Builder / 链上地址 等一律在此表
CREATE TABLE IF NOT EXISTS `account` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL,
  `name` varchar(64) NOT NULL DEFAULT '' COMMENT '内部别名',
  `auth_mode` varchar(32) NOT NULL DEFAULT 'KMS' COMMENT 'KMS/CLOB=预测; BC=赔率平台账密',
  `status` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `api_key` varchar(256) DEFAULT NULL COMMENT 'CLOB / Foregate 等',
  `api_secret_enc` varchar(512) DEFAULT NULL,
  `api_passphrase_enc` varchar(512) DEFAULT NULL,
  `builder_api_key` varchar(128) DEFAULT NULL COMMENT 'Poly Builder 归属（按账号）',
  `builder_api_secret_enc` varchar(512) DEFAULT NULL,
  `builder_api_passphrase_enc` varchar(512) DEFAULT NULL,
  `key_id` varchar(256) DEFAULT NULL COMMENT 'Kalshi ACCESS-KEY / KMS',
  `kms_sign_address` varchar(128) DEFAULT NULL,
  `eoa_address` varchar(128) DEFAULT NULL,
  `proxy_address` varchar(128) DEFAULT NULL,
  `private_key_enc` varchar(1024) DEFAULT NULL,
  `signature_type` int DEFAULT NULL,
  `use_magic_account` tinyint(1) NOT NULL DEFAULT 0,
  `foregate_user_id` int DEFAULT NULL,
  `account_type` varchar(16) DEFAULT NULL COMMENT 'Foregate buy|sell',
  `login_name` varchar(128) DEFAULT NULL COMMENT 'BC 平台',
  `login_password_enc` varchar(512) DEFAULT NULL,
  `fg_token_enc` varchar(512) DEFAULT NULL COMMENT 'Foregate 账号 token',
  `order_max_pending_time` int DEFAULT NULL COMMENT '挂单超时分钟',
  `origin_total_amount` decimal(20,8) DEFAULT NULL COMMENT '总的初始资金',
  `config` text COMMENT '扩展 JSON',
  `remark` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_name` (`platform`,`name`),
  KEY `idx_account_platform_status` (`platform`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='下单账户与全部 API 凭据';

CREATE TABLE IF NOT EXISTS `config_strategy_threshold` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `config_key` varchar(64) NOT NULL,
  `config_value` decimal(20,8) NOT NULL,
  `value_type` varchar(16) NOT NULL DEFAULT 'DECIMAL',
  `description` varchar(500) DEFAULT NULL,
  `status` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='扫描阈值（0.95/1.9 等）';

CREATE TABLE IF NOT EXISTS `config_strategy_fee` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL DEFAULT '' COMMENT 'POLYMARKET / KALSHI',
  `category` varchar(64) NOT NULL DEFAULT '' COMMENT '市场大类，运动为 Sports',
  `sub_category` varchar(64) NOT NULL DEFAULT '' COMMENT '运动。空字符串表示该大类的默认系数',
  `taker_fee_coef` decimal(20,8) NOT NULL COMMENT '吃单系数。费用 = 系数 * 价格 * (1-价格)',
  `description` varchar(500) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_fee_platform_category_sub` (`platform`, `category`, `sub_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='按平台和运动的吃单手续费系数';

CREATE TABLE IF NOT EXISTS `config_strategy` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `group_name` varchar(64) NOT NULL,
  `item_key` varchar(128) NOT NULL,
  `item_value` text NOT NULL,
  `value_type` varchar(32) NOT NULL DEFAULT 'string',
  `description` varchar(500) NOT NULL DEFAULT '',
  `sort_no` int NOT NULL DEFAULT 0,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_fcs_scope_group_item` (`group_name`,`item_key`),
  KEY `idx_fcs_scope_group` (`group_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='策略全局键值';

CREATE TABLE IF NOT EXISTS `bot_schedule_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '任务 ID',
  `module` varchar(50) NOT NULL COMMENT '模块名',
  `task_name` varchar(100) NOT NULL COMMENT '任务名',
  `interval_seconds` float NOT NULL DEFAULT 60 COMMENT '执行间隔（秒）',
  `exe_sort` int DEFAULT NULL COMMENT '同模块内执行顺序',
  `concurrency` int DEFAULT NULL COMMENT '并发度',
  `description` varchar(500) DEFAULT NULL COMMENT '说明',
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  `is_strategy_enabled` tinyint(1) DEFAULT 1 COMMENT '策略是否启用',
  `last_live_time` varchar(255) DEFAULT NULL COMMENT '上次存活时间',
  `last_live_time_by_machine` json DEFAULT NULL COMMENT '各机器上次存活',
  `is_primary_machine_run` int NOT NULL DEFAULT 0 COMMENT '是否主机器执行',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_module_task` (`module`,`task_name`),
  KEY `idx_module` (`module`),
  KEY `idx_bot_schedule_module_enabled` (`module`,`is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Bot 脚本调度';

CREATE TABLE IF NOT EXISTS `market` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL DEFAULT '',
  `name` varchar(2000) DEFAULT NULL,
  `desc` varchar(2000) DEFAULT NULL,
  `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台场次/Event ID',
  `market_url` varchar(255) NOT NULL DEFAULT '',
  `event_start_time` varchar(255) DEFAULT NULL,
  `event_end_time` varchar(255) DEFAULT NULL,
  `status` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `category` varchar(200) DEFAULT NULL,
  `sub_category` varchar(200) DEFAULT NULL,
  `volume_total` double DEFAULT NULL,
  `volume24hr` double DEFAULT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `league` varchar(128) DEFAULT NULL,
  `competition` varchar(64) DEFAULT NULL COMMENT '联赛中文名，来自配置 name',
  `competition_en` varchar(128) DEFAULT NULL COMMENT '统一英文名，来自配置 name_en',
  `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键（联赛+场次）',
  `map_verified` tinyint(1) NOT NULL DEFAULT 0,
  `config` text,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_market_sub` (`platform`,`platform_market_id`,`category`,`sub_category`),
  KEY `idx_market_fixture` (`platform`,`competition`,`status`),
  KEY `idx_market_map_key` (`map_key`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='平台场次/Event';

CREATE TABLE IF NOT EXISTS `market_outcome` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL DEFAULT '',
  `market_row_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '父场次 market.id',
  `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '父场次平台 ID，对应 market.platform_market_id',
  `outcome_name` varchar(2000) DEFAULT NULL COMMENT '子盘口名称（question/title）',
  `market_name` varchar(2000) DEFAULT NULL COMMENT '父场次名称，同 market.name',
  `desc` varchar(2000) DEFAULT NULL,
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台盘口 ID',
  `market_address` varchar(255) DEFAULT NULL,
  `market_url` varchar(255) NOT NULL DEFAULT '',
  `yes_token_id` varchar(255) NOT NULL DEFAULT '',
  `no_token_id` varchar(255) NOT NULL DEFAULT '',
  `selection_id` varchar(255) DEFAULT NULL,
  `market_start_time` varchar(255) DEFAULT NULL,
  `market_end_time` varchar(255) DEFAULT NULL,
  `category` varchar(200) DEFAULT NULL,
  `sub_category` varchar(200) DEFAULT NULL,
  `outcome_type` varchar(32) NOT NULL DEFAULT 'moneyline' COMMENT 'moneyline|handicap|total',
  `volume_total` double DEFAULT NULL,
  `volume24hr` double DEFAULT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `competition` varchar(64) DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）',
  `competition_en` varchar(128) DEFAULT NULL COMMENT '统一英文名，来自配置 name_en',
  `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key',
  `config` text,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_outcome_sub` (`platform`,`platform_market_outcome_id`,`category`,`sub_category`),
  KEY `idx_outcome_market_row` (`market_row_id`),
  KEY `idx_outcome_platform_market` (`platform`,`platform_market_id`,`category`,`sub_category`),
  KEY `idx_outcome_type` (`outcome_type`),
  KEY `idx_outcome_map_key` (`map_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='可交易盘口';

CREATE TABLE IF NOT EXISTS `market_order_book` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `market_row_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '父场次 market.id',
  `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key',
  `category` varchar(200) DEFAULT NULL,
  `sub_category` varchar(200) DEFAULT NULL,
  `competition` varchar(64) DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）',
  `competition_en` varchar(128) DEFAULT NULL COMMENT '统一英文名，来自配置 name_en',
  `platform` varchar(32) NOT NULL DEFAULT '',
  `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '父场次平台 ID',
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台盘口 ID',
  `market_name` text,
  `outcome_name` text COMMENT '子盘口名称 market_outcome.name',
  `token_name` varchar(255) DEFAULT NULL COMMENT 'token 展示名 Yes/No 或队名等',
  `market_url` varchar(255) DEFAULT '',
  `market_address` varchar(255) DEFAULT '',
  `token_id` varchar(160) NOT NULL DEFAULT '' COMMENT '预测=token_id；赔率=selection_id 或空',
  `selection_id` varchar(255) DEFAULT NULL COMMENT '赔率平台选项 ID',
  `order_book_asks` text NOT NULL COMMENT '预测=价位 JSON；赔率=可存 [[decimal_odds,limit],...]',
  `order_book_bids` text NOT NULL,
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '快照毫秒（REST 拉取；0=无）',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_order_book_market_row` (`market_row_id`),
  KEY `idx_order_book_platform_outcome` (`platform`,`platform_market_outcome_id`),
  KEY `idx_order_book_platform_fixture` (`platform`,`platform_market_id`),
  KEY `idx_order_book_map_key` (`map_key`),
  KEY `idx_order_book_token_snap_ms` (`platform`,`token_id`,`snapshot_at_ms`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='REST 报价历史快照（orderbook_sync）';

CREATE TABLE IF NOT EXISTS `market_order_book_ws` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `market_row_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '父场次 market.id',
  `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key',
  `category` varchar(200) DEFAULT NULL,
  `sub_category` varchar(200) DEFAULT NULL,
  `competition` varchar(64) DEFAULT NULL COMMENT '联赛/杯赛名称（如英超）',
  `competition_en` varchar(128) DEFAULT NULL COMMENT '统一英文名，来自配置 name_en',
  `platform` varchar(32) NOT NULL DEFAULT '',
  `platform_market_id` varchar(255) NOT NULL DEFAULT '' COMMENT '父场次平台 ID',
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '' COMMENT '平台盘口 ID',
  `market_name` text,
  `outcome_name` text COMMENT '子盘口名称 market_outcome.name',
  `token_name` varchar(255) DEFAULT NULL COMMENT 'token 展示名 Yes/No 或队名等',
  `market_url` varchar(255) DEFAULT '',
  `market_address` varchar(255) DEFAULT '',
  `token_id` varchar(160) NOT NULL DEFAULT '' COMMENT '预测=token_id；赔率=selection_id 或空',
  `selection_id` varchar(255) DEFAULT NULL COMMENT '赔率平台选项 ID',
  `order_book_asks` text NOT NULL,
  `order_book_bids` text NOT NULL,
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0 COMMENT '快照毫秒（平台 WS 时间；0=无）',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_order_book_ws_market_row` (`market_row_id`),
  KEY `idx_order_book_ws_platform_outcome` (`platform`,`platform_market_outcome_id`),
  KEY `idx_order_book_ws_platform_fixture` (`platform`,`platform_market_id`),
  KEY `idx_order_book_ws_map_key` (`map_key`),
  KEY `idx_order_book_ws_token_snap_ms` (`platform`,`token_id`,`snapshot_at_ms`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='WS 报价快照（strategy_entry）';

-- strategy_entry 机会（二元 + 1X2）
CREATE TABLE IF NOT EXISTS `arb_binary_opportunity` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `opp_kind` varchar(32) NOT NULL COMMENT 'BINARY_YES_NO | ONE_X2_YES_TRIPLE | ONE_X2_NO_TRIPLE',
  `map_key` varchar(192) DEFAULT NULL COMMENT '跨平台场次对齐键 market.map_key',
  `category` varchar(200) DEFAULT NULL,
  `sub_category` varchar(200) DEFAULT NULL,
  `competition` varchar(64) DEFAULT NULL COMMENT '联赛/杯赛',
  `competition_en` varchar(128) DEFAULT NULL,
  `leg_1_info` mediumtext NOT NULL COMMENT '腿1 JSON；BINARY YES；1X2 平局',
  `leg_1_platform` varchar(32) DEFAULT NULL COMMENT '组合腿1平台',
  `leg_1_order_books` mediumtext NOT NULL,
  `leg_1_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿1 卖一（0~1）',
  `leg_2_info` mediumtext NOT NULL COMMENT '腿2 JSON；BINARY NO；1X2 主队',
  `leg_2_platform` varchar(32) DEFAULT NULL COMMENT '组合腿2平台',
  `leg_2_order_books` mediumtext NOT NULL,
  `leg_2_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿2 卖一（0~1）',
  `leg_3_info` mediumtext NOT NULL COMMENT '腿3 JSON；BINARY {}；1X2 客队',
  `leg_3_platform` varchar(32) DEFAULT NULL COMMENT '组合腿3平台',
  `leg_3_order_books` mediumtext NOT NULL,
  `leg_3_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿3 卖一（0~1）',
  `total_cost` decimal(20,8) NOT NULL,
  `profit_space` decimal(20,8) NOT NULL,
  `arb_qty` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '各腿卖一挂单量最小值（可套利份数上限）',
  `arb_profit` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '套利金额 arb_qty * profit_space',
  `arb_profit_amount` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实盘套利利润：min(平台成交量)-总花费',
  `threshold_key` varchar(64) NOT NULL,
  `threshold_value` decimal(20,8) NOT NULL,
  `book_updated_at` datetime DEFAULT NULL,
  `snapshot_at` datetime NOT NULL,
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0,
  `ext_data` text NULL COMMENT '扩展JSON：map_key锁失败、下单前检查失败阈值快照、下单现场',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_arb_binary_time` (`snapshot_at`),
  KEY `idx_arb_binary_map` (`map_key`,`snapshot_at`),
  KEY `idx_arb_binary_kind` (`opp_kind`,`snapshot_at`),
  KEY `idx_arb_binary_snap_ms` (`map_key`,`snapshot_at_ms`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='strategy_entry 套利机会';

CREATE TABLE IF NOT EXISTS `orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `backtest_task_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '0=实盘',
  `strategy_type` varchar(32) NOT NULL DEFAULT 'BINARY_YES_NO',
  `arb_opp_id` bigint unsigned NOT NULL,
  `map_key` varchar(192) NOT NULL,
  `side_no` tinyint unsigned NOT NULL COMMENT '组合腿序号 1..3',
  `platform` varchar(32) NOT NULL,
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '',
  `token_id` varchar(255) NOT NULL DEFAULT '',
  `client_order_id` varchar(64) NOT NULL DEFAULT '' COMMENT '本地生成的客户端单号',
  `platform_order_id` varchar(128) NOT NULL DEFAULT '' COMMENT '平台返回订单 ID/hash',
  `side` varchar(8) NOT NULL COMMENT 'YES|NO',
  `action` varchar(16) NOT NULL DEFAULT 'BUY',
  `price` decimal(20,8) NOT NULL,
  `qty` decimal(20,8) NOT NULL,
  `trade_qty` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实际成交份额（回补；无成交为 0）',
  `trade_price` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实际成交均价（回补；无成交为 0）',
  `trade_amount` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实际成交金额=trade_price*trade_qty',
  `fee` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '手续费（结算币）',
  `status` varchar(32) NOT NULL,
  `err_msg` varchar(1024) NOT NULL DEFAULT '',
  `sent_at` datetime NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_arb_order_opp` (`arb_opp_id`),
  KEY `idx_arb_order_map_time` (`map_key`,`sent_at`),
  KEY `idx_orders_bt_task` (`backtest_task_id`,`sent_at`),
  KEY `idx_orders_strategy` (`strategy_type`,`sent_at`),
  KEY `idx_orders_client_oid` (`client_order_id`),
  KEY `idx_orders_platform_oid` (`platform`,`platform_order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='strategy_entry 下单（实盘）';

CREATE TABLE IF NOT EXISTS `stat_kv` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `group_name` varchar(64) NOT NULL,
  `item_key` varchar(128) NOT NULL,
  `item_value` text NOT NULL,
  `value_type` varchar(32) NOT NULL DEFAULT 'string',
  `description` varchar(500) NOT NULL DEFAULT '',
  `sort_no` int NOT NULL DEFAULT 0,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_stat_kv_group_item` (`group_name`,`item_key`),
  KEY `idx_stat_kv_group` (`group_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `market_category` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sport_cn` varchar(64) NOT NULL,
  `sub_category` varchar(64) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `concurrency` int NOT NULL DEFAULT 1,
  `config_json` longtext,
  `outcome_types_json` varchar(512) NOT NULL DEFAULT '[]',
  `sort_no` int NOT NULL DEFAULT 0,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_category_sub_platform` (`sub_category`, `platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `backtest_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `map_key` varchar(192) NOT NULL DEFAULT '',
  `strategy_code` varchar(32) NOT NULL DEFAULT 'strategy_entry' COMMENT 'strategy_entry|yesno|onextwo|binary_yes_no',
  `status` varchar(24) NOT NULL DEFAULT 'pending',
  `progress` tinyint unsigned NOT NULL DEFAULT 0,
  `error_text` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_backtest_task_status` (`status`, `strategy_code`),
  KEY `idx_backtest_task_map` (`map_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `backtest_arb_binary_opportunity` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `backtest_task_id` bigint unsigned NOT NULL,
  `opp_kind` varchar(32) NOT NULL,
  `map_key` varchar(192) DEFAULT NULL,
  `category` varchar(200) DEFAULT NULL,
  `sub_category` varchar(200) DEFAULT NULL,
  `competition` varchar(64) DEFAULT NULL,
  `competition_en` varchar(128) DEFAULT NULL,
  `leg_1_info` mediumtext NOT NULL COMMENT '腿1 JSON；BINARY YES；1X2 平局',
  `leg_1_platform` varchar(32) DEFAULT NULL COMMENT '组合腿1平台',
  `leg_1_order_books` mediumtext NOT NULL,
  `leg_1_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿1 卖一（0~1）',
  `leg_2_info` mediumtext NOT NULL COMMENT '腿2 JSON；BINARY NO；1X2 主队',
  `leg_2_platform` varchar(32) DEFAULT NULL COMMENT '组合腿2平台',
  `leg_2_order_books` mediumtext NOT NULL,
  `leg_2_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿2 卖一（0~1）',
  `leg_3_info` mediumtext NOT NULL COMMENT '腿3 JSON；BINARY {}；1X2 客队',
  `leg_3_platform` varchar(32) DEFAULT NULL COMMENT '组合腿3平台',
  `leg_3_order_books` mediumtext NOT NULL,
  `leg_3_best_ask` decimal(20,8) DEFAULT NULL COMMENT '腿3 卖一（0~1）',
  `total_cost` decimal(20,8) NOT NULL,
  `profit_space` decimal(20,8) NOT NULL,
  `arb_qty` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '各腿卖一挂单量最小值（可套利份数上限）',
  `arb_profit` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '套利金额 arb_qty * profit_space',
  `arb_profit_amount` decimal(20,8) NOT NULL DEFAULT 0 COMMENT '实盘套利利润：min(平台成交量)-总花费',
  `threshold_key` varchar(64) NOT NULL,
  `threshold_value` decimal(20,8) NOT NULL,
  `book_updated_at` datetime DEFAULT NULL,
  `snapshot_at` datetime NOT NULL,
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_bt_opp_task` (`backtest_task_id`, `snapshot_at`),
  KEY `idx_bt_opp_map` (`map_key`, `snapshot_at`),
  KEY `idx_bt_opp_snap_ms` (`backtest_task_id`, `snapshot_at_ms`),
  UNIQUE KEY `uk_bt_opp_snap_legs` (`backtest_task_id`, `snapshot_at_ms`, `opp_kind`, `map_key`(96), `total_cost`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `backtest_orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `backtest_task_id` bigint unsigned NOT NULL DEFAULT 0,
  `strategy_type` varchar(32) NOT NULL DEFAULT 'BINARY_YES_NO',
  `arb_opp_id` bigint unsigned NOT NULL,
  `map_key` varchar(192) NOT NULL,
  `side_no` tinyint unsigned NOT NULL,
  `platform` varchar(32) NOT NULL,
  `platform_market_outcome_id` varchar(255) NOT NULL DEFAULT '',
  `token_id` varchar(255) NOT NULL DEFAULT '',
  `client_order_id` varchar(64) NOT NULL DEFAULT '',
  `platform_order_id` varchar(128) NOT NULL DEFAULT '',
  `side` varchar(8) NOT NULL,
  `action` varchar(16) NOT NULL DEFAULT 'BUY',
  `price` decimal(20,8) NOT NULL,
  `qty` decimal(20,8) NOT NULL,
  `fill_qty` decimal(20,8) NOT NULL DEFAULT 0,
  `fee` decimal(20,8) NOT NULL DEFAULT 0,
  `status` varchar(32) NOT NULL,
  `err_msg` varchar(1024) NOT NULL DEFAULT '',
  `sent_at` datetime NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_backtest_orders_task` (`backtest_task_id`, `sent_at`),
  KEY `idx_backtest_orders_opp` (`arb_opp_id`),
  KEY `idx_backtest_orders_map_time` (`map_key`, `sent_at`),
  KEY `idx_backtest_orders_strategy` (`strategy_type`, `sent_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `data_platform` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL DEFAULT '',
  `display_name` varchar(128) NOT NULL DEFAULT '',
  `vendor_type` varchar(32) NOT NULL DEFAULT 'third_party',
  `api_base_url` varchar(512) NOT NULL DEFAULT '',
  `api_key` varchar(512) NOT NULL DEFAULT '',
  `extra_config` text,
  `status` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_data_platform_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `data_history_order_book` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `data_platform_id` bigint unsigned NOT NULL DEFAULT 0,
  `map_key` varchar(192) NOT NULL DEFAULT '',
  `market_row_id` bigint unsigned NOT NULL DEFAULT 0,
  `market_outcome_id` bigint unsigned NOT NULL DEFAULT 0,
  `venue_platform` varchar(32) NOT NULL DEFAULT '',
  `vendor_exchange` varchar(32) NOT NULL DEFAULT '',
  `vendor_symbol` varchar(255) NOT NULL DEFAULT '',
  `utc_date` date NOT NULL,
  `data_type` varchar(32) NOT NULL DEFAULT 'l2',
  `source` varchar(16) NOT NULL DEFAULT 'pmdata' COMMENT 'pmdata|synthetic',
  `record_kind` varchar(16) NOT NULL DEFAULT 'snapshot',
  `snapshot_at_ms` bigint NOT NULL DEFAULT 0,
  `order_book_bids` mediumtext NOT NULL,
  `order_book_asks` mediumtext NOT NULL,
  `local_file_path` varchar(512) NOT NULL DEFAULT '',
  `file_bytes` bigint unsigned NOT NULL DEFAULT 0,
  `file_sha256` char(64) NOT NULL DEFAULT '',
  `ingest_status` varchar(24) NOT NULL DEFAULT 'ok',
  `error_text` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_history_ob` (`data_platform_id`,`market_outcome_id`,`utc_date`,`data_type`,`record_kind`,`snapshot_at_ms`),
  KEY `idx_history_map_key` (`map_key`),
  KEY `idx_history_replay` (`map_key`, `snapshot_at_ms`, `venue_platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `data_history_market_event` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `data_platform_id` bigint unsigned NOT NULL DEFAULT 0,
  `map_key` varchar(192) NOT NULL DEFAULT '',
  `market_row_id` bigint unsigned NOT NULL DEFAULT 0,
  `market_outcome_id` bigint unsigned NOT NULL DEFAULT 0,
  `venue_platform` varchar(32) NOT NULL DEFAULT '',
  `vendor_exchange` varchar(32) NOT NULL DEFAULT '',
  `vendor_symbol` varchar(255) NOT NULL DEFAULT '',
  `file_calendar_day` date NOT NULL COMMENT 'Parquet 文件 UTC 日',
  `data_type` varchar(32) NOT NULL DEFAULT 'l2',
  `event_kind` varchar(32) NOT NULL DEFAULT '' COMMENT 'book|price_change|orderbook',
  `event_type` varchar(64) NOT NULL DEFAULT '' COMMENT '源 event_type',
  `event_at_ms` bigint NOT NULL DEFAULT 0,
  `local_event_at_ms` bigint NOT NULL DEFAULT 0,
  `row_index` int unsigned NOT NULL DEFAULT 0,
  `payload_json` mediumtext NOT NULL,
  `local_file_path` varchar(512) NOT NULL DEFAULT '',
  `file_bytes` bigint unsigned NOT NULL DEFAULT 0,
  `file_sha256` char(64) NOT NULL DEFAULT '',
  `ingest_status` varchar(24) NOT NULL DEFAULT 'ok',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_history_market_event` (`data_platform_id`,`market_outcome_id`,`file_sha256`,`row_index`),
  KEY `idx_hme_map_time` (`map_key`,`event_at_ms`,`venue_platform`),
  KEY `idx_hme_kind` (`market_outcome_id`,`file_calendar_day`,`event_kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `data_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ingest_source` varchar(16) NOT NULL DEFAULT 'pmdata' COMMENT 'pmdata|synthetic',
  `map_key` varchar(192) NOT NULL DEFAULT '' COMMENT 'synthetic 用 v1|联赛|slug；pmdata 留空，见 params.fixture_slug',
  `params` text NULL COMMENT 'JSON：pmdata 可 utc_dates/min_utc_age_days 等；synthetic 可扩展',
  `status` varchar(24) NOT NULL DEFAULT 'pending' COMMENT 'pending|running|completed|failed',
  `error_text` varchar(512) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_data_task_ingest_status` (`ingest_source`, `status`),
  KEY `idx_data_task_map_key` (`map_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS = 1;

INSERT INTO `config_strategy_threshold` (`config_key`, `config_value`, `value_type`, `description`) VALUES
  ('binary_yes_no_max_cost', 0.95000000, 'DECIMAL', '二元 YES+NO 最大总成本'),
  ('min_amount', 5.00000000, 'DECIMAL', '最小下单份额金额（÷各平台最低 best_ask → min_qty）'),
  ('max_amount', 5.00000000, 'DECIMAL', '最大下单份额金额（÷各平台最低 best_ask → max_qty）'),
  ('1x2_yes_triple_max_cost', 0.95000000, 'DECIMAL', '1X2 三路 YES'),
  ('1x2_no_triple_max_cost', 1.90000000, 'DECIMAL', '1X2 三路 NO'),
  ('predict_polymarket_sxbet_slippage_bps', 200.00000000, 'DECIMAL', 'Predict+Polymarket+SX 共用滑点 bps（100=1%；Predict→slippageBps，Poly/SX→限价×(1+bps/10000)）'),
  ('predict_is_min_amount_out', 1.00000000, 'DECIMAL', 'Predict MARKET isMinAmountOut（1=true 花固定 USDT、少收份额；0=false 多付钱拿满份额）'),
  ('fill_share_mismatch_pct', 0.01000000, 'DECIMAL', '下单结果判定套利成功：各平台成交份额 (max-min)/max 上限（0.01=1%）'),
  ('certain_min_return_pct', 0.10000000, 'DECIMAL', '必然事件最低收益率'),
  ('stochastic_min_expected_return_pct', 0.10000000, 'DECIMAL', '偶然事件期望收益率'),
  ('stochastic_high_priority_return_pct', 0.20000000, 'DECIMAL', '偶然高优先级')
ON DUPLICATE KEY UPDATE `config_value` = VALUES(`config_value`);

INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('market_sync', 'polymarket_sports', '[{"sport_cn":"足球","sub_category":"Football","items":[{"name":"英超","sport_id":"epl","pair_kalshi_ticker":"KXEPLGAME"},{"name":"欧冠","sport_id":"ucl","pair_kalshi_ticker":"KXUCLGAME"},{"name":"西甲","sport_id":"lal","pair_kalshi_ticker":"KXLALIGAGAME"},{"name":"德甲","sport_id":"bun","pair_kalshi_ticker":"KXBUNDESLIGAGAME"},{"name":"意甲","sport_id":"sea","pair_kalshi_ticker":"KXSERIEAGAME"},{"name":"法甲","sport_id":"fl1"},{"name":"欧联","sport_id":"uel"},{"name":"美职联","sport_id":"mls"},{"name":"国际足球","sport_id":"fif"},{"name":"欧国联","sport_id":"uef"},{"name":"美洲杯","sport_id":"con"},{"name":"非洲杯","sport_id":"caf"},{"name":"金杯赛","sport_id":"cof"},{"name":"巴甲","sport_id":"bra"},{"name":"阿甲","sport_id":"arg"},{"name":"墨超","sport_id":"mex"},{"name":"解放者杯","sport_id":"lib"},{"name":"南美杯","sport_id":"sud"},{"name":"土超","sport_id":"tur"},{"name":"俄超","sport_id":"rus"},{"name":"英格兰足总杯","sport_id":"efa"},{"name":"英格兰联赛杯","sport_id":"efl"},{"name":"荷甲","sport_id":"ere"},{"name":"意大利杯","sport_id":"itc"},{"name":"国王杯","sport_id":"cdr"},{"name":"德国杯","sport_id":"dfb"},{"name":"法国杯","sport_id":"cde"},{"name":"哥伦甲","sport_id":"col"},{"name":"亚冠","sport_id":"afc"},{"name":"大洋洲","sport_id":"ofc"},{"name":"J1","sport_id":"jap"},{"name":"J2","sport_id":"ja2"},{"name":"K联赛","sport_id":"kor"},{"name":"沙特联","sport_id":"spl"},{"name":"中超","sport_id":"chi"},{"name":"澳超","sport_id":"aus"},{"name":"印尼联","sport_id":"idn1"},{"name":"挪超","sport_id":"nor"},{"name":"丹超","sport_id":"den"},{"name":"葡超","sport_id":"por"},{"name":"非洲国家杯","sport_id":"acn"},{"name":"西班牙超级杯","sport_id":"ssc"}]}]', 'json', 'Polymarket 足球联赛 JSON（见 docs/sql/polymarket_sports.json）', 1, 1),
  ('market_sync', 'kalshi_game_series', '[{"sport_cn":"足球","sub_category":"Football","items":[{"name":"国际足球","name_en":"FIFA Friendlies","name_desc":"Intl Friendlies","ticker":"KXINTLFRIENDLYGAME","pair_poly_sport_id":"fif"},{"name":"中北美国家联赛","name_en":"CONCACAF Nations League","name_desc":"CONCACAF Nations League","ticker":"KXCONCACAFNLGAME","pair_poly_sport_id":"conl"},{"name":"欧国联","name_en":"UEFA Nations League","name_desc":"UEFA Nations League","ticker":"KXUEFANLGAME","pair_poly_sport_id":"unl"},{"name":"西甲","name_en":"LaLiga","name_desc":"La Liga","ticker":"KXLALIGAGAME","pair_poly_sport_id":"lal"},{"name":"墨超","name_en":"Liga MX","name_desc":"Liga MX","ticker":"KXLIGAMXGAME","pair_poly_sport_id":"mex"},{"name":"欧联","name_en":"UEFA Europa League","name_desc":"Europa League","ticker":"KXUELGAME","pair_poly_sport_id":"uel"},{"name":"英超","name_en":"Premier League","name_desc":"EPL","ticker":"KXEPLGAME","pair_poly_sport_id":"epl"},{"name":"哥伦甲","name_en":"Categoría Primera A","name_desc":"Colombian Liga DIMAYOR","ticker":"KXDIMAYORGAME","pair_poly_sport_id":"col1"},{"name":"英甲","name_en":"League One","name_desc":"EFL League One","ticker":"KXEFLL1GAME","pair_poly_sport_id":"el1"},{"name":"德甲","name_en":"Bundesliga","name_desc":"Bundesliga","ticker":"KXBUNDESLIGAGAME","pair_poly_sport_id":"bun"},{"name":"法女甲","name_en":"French Women''s Première Ligue","name_desc":"French Women''s Première Ligue","ticker":"KXFRAPLWGAME"},{"name":"法甲","name_en":"Ligue 1","name_desc":"Ligue 1","ticker":"KXLIGUE1GAME","pair_poly_sport_id":"fl1"},{"name":"英格兰女足超","name_en":"Women''s Super League","name_desc":"English Womens Super League","ticker":"KXEWSLGAME","pair_poly_sport_id":"wsl"},{"name":"英乙","name_en":"League Two","name_desc":"EFL League Two","ticker":"KXEFLL2GAME","pair_poly_sport_id":"el2"},{"name":"欧冠","name_en":"UEFA Champions League","name_desc":"Champions League","ticker":"KXUCLGAME","pair_poly_sport_id":"ucl"},{"name":"巴乙","name_en":"Brasileirão Série B","name_desc":"Brasileiro Serie B","ticker":"KXBRASILEIROBGAME","pair_poly_sport_id":"bra2"},{"name":"非洲杯预选赛","name_en":"Africa Cup of Nations Qualifiers","name_desc":"AFCON Qualifiers","ticker":"KXAFCONGAME","pair_poly_sport_id":"afcq"},{"name":"西乙","name_en":"LaLiga2","name_desc":"La Liga 2","ticker":"KXLALIGA2GAME","pair_poly_sport_id":"es2"},{"name":"西女甲","name_en":"Spanish Liga F","name_desc":"Spanish Liga F","ticker":"KXLIGAFGAME"},{"name":"南亚锦标赛","name_en":"SAFF Championship","name_desc":"SAFF Championship","ticker":"KXSAFFGAME"},{"name":"意女甲","name_en":"Serie A Femminile","name_desc":"Serie A Femminile","ticker":"KXSERIEAWGAME"},{"name":"美职联","name_en":"MLS","name_desc":"MLS","ticker":"KXMLSGAME","pair_poly_sport_id":"mls"},{"name":"美国足球冠军联赛","name_en":"USL Championship","name_desc":"USL Championship","ticker":"KXUSLGAME","pair_poly_sport_id":"uslc"},{"name":"德女甲","name_en":"German Women''s Bundesliga","name_desc":"German Women''s Bundesliga","ticker":"KXBUNDESLIGAWGAME"},{"name":"法国杯","name_en":"Coupe de France","name_desc":"Coupe de France","ticker":"KXCOUPEDEFRANCEGAME","pair_poly_sport_id":"cde"},{"name":"巴甲","name_en":"Brasileirão Série A","name_desc":"Brasileiro Serie A","ticker":"KXBRASILEIROGAME","pair_poly_sport_id":"bra"},{"name":"意甲","name_en":"Serie A","name_desc":"Serie A","ticker":"KXSERIEAGAME","pair_poly_sport_id":"sea"},{"name":"阿根廷全国联赛","name_en":"Primera Nacional","name_desc":"Argentine Nacional B","ticker":"KXARGNACBGAME","pair_poly_sport_id":"argpn"},{"name":"斯洛伐克乙","name_en":"Slovakian 2. Liga","name_desc":"Slovakian 2. Liga","ticker":"KXSVK2LGAME"},{"name":"斯洛伐克杯","name_en":"Slovakian Cup","name_desc":"Slovakian Cup","ticker":"KXSVKCUPGAME"},{"name":"荷乙","name_en":"Eerste Divisie","name_desc":"Eerste Divisie","ticker":"KXEERSTEDIVGAME","pair_poly_sport_id":"ned2"},{"name":"美国女足联赛","name_en":"NWSL","name_desc":"NWSL","ticker":"KXNWSLGAME","pair_poly_sport_id":"nwsl"},{"name":"阿根廷杯","name_en":"Copa Argentina","name_desc":"Argentina Cup","ticker":"KXARGCUPGAME","pair_poly_sport_id":"argcopa"},{"name":"J2","name_en":"J2 League","name_desc":"Japan J2 League","ticker":"KXJ2LEAGUEGAME","pair_poly_sport_id":"ja2"},{"name":"意丙","name_en":"Serie C","name_desc":"Serie C","ticker":"KXSERIECGAME","pair_poly_sport_id":"serc"},{"name":"苏冠","name_en":"Scottish Championship","name_desc":"Scottish Championship","ticker":"KXSCOCHAMPGAME"},{"name":"荷丙","name_en":"Tweede Divisie","name_desc":"Tweede Divisie","ticker":"KXTWEEDEDIVGAME"},{"name":"英格兰国家联赛","name_en":"National League","name_desc":"English National League","ticker":"KXENGNLGAME","pair_poly_sport_id":"snhl"},{"name":"欧协联","name_en":"UEFA Conference League","name_desc":"Conference League","ticker":"KXUECLGAME","pair_poly_sport_id":"col"},{"name":"乌拉圭甲","name_en":"Uruguayan Primera División","name_desc":"Uruguay Primera Division","ticker":"KXURYPDGAME","pair_poly_sport_id":"uru1"},{"name":"美国足球甲级","name_en":"USL League One","name_desc":"USL League One","ticker":"KXUSLL1GAME"},{"name":"葡甲","name_en":"Liga Portugal 2","name_desc":"Liga Portugal 2","ticker":"KXLIGAPORTUGAL2GAME","pair_poly_sport_id":"pt2"},{"name":"墨女超","name_en":"Liga MX Women","name_desc":"Liga MX Women","ticker":"KXLIGAMXWGAME"},{"name":"秘鲁杯","name_en":"Peru League Cup","name_desc":"Peru League Cup","ticker":"KXPERLEAGUECUPGAME"},{"name":"巴拉圭杯","name_en":"Paraguay Cup","name_desc":"Paraguay Cup","ticker":"KXPRYCUPGAME"},{"name":"英冠","name_en":"EFL Championship","name_desc":"EFL Championship","ticker":"KXEFLCHAMPIONSHIPGAME","pair_poly_sport_id":"elc"},{"name":"秘鲁甲","name_en":"Liga 1","name_desc":"Peru Liga 1","ticker":"KXPERLIGA1GAME","pair_poly_sport_id":"per1"},{"name":"苏超","name_en":"Scottish Premiership","name_desc":"Scottish Premiership","ticker":"KXSCOTTISHPREMGAME","pair_poly_sport_id":"scop"},{"name":"南美杯","name_en":"Copa Sudamericana","name_desc":"CONMEBOL Sudamericana","ticker":"KXCONMEBOLSUDGAME","pair_poly_sport_id":"sud"},{"name":"荷女甲","name_en":"Eredivisie Vrouwen","name_desc":"Eredivisie Vrouwen","ticker":"KXEREDIVISIEWGAME"},{"name":"墨扩军联赛","name_en":"Liga de Expansión MX","name_desc":"Liga de Expansion MX","ticker":"KXLIGAEXPGAME","pair_poly_sport_id":"mex2"},{"name":"K联赛","name_en":"K League 1","name_desc":"Korea K League 1","ticker":"KXKLEAGUEGAME","pair_poly_sport_id":"kor"},{"name":"澳女超","name_en":"Australia A League Women","name_desc":"Australia A League Women","ticker":"KXALEAGUEWGAME"},{"name":"亚冠","name_en":"AFC Champions League Elite","name_desc":"AFC Champions League","ticker":"KXAFCCLGAME","pair_poly_sport_id":"acle"},{"name":"女足欧冠","name_en":"UEFA Women''s Champions League","name_desc":"Champions League Womens","ticker":"KXUCLWGAME","pair_poly_sport_id":"uwcl"},{"name":"苏女超","name_en":"Scottish Women Premier League","name_desc":"Scottish Women Premier League","ticker":"KXSWPLGAME"},{"name":"J1","name_en":"J1 League","name_desc":"Japan J1 League","ticker":"KXJLEAGUEGAME","pair_poly_sport_id":"jap"},{"name":"日本联赛杯","name_en":"Japan J League Cup","name_desc":"Japan J League Cup","ticker":"KXJLEAGUECUPGAME"},{"name":"亚洲杯","name_en":"AFC Asian Cup","name_desc":"AFC Asian Cup","ticker":"KXAFCACGAME"},{"name":"巴拉圭甲","name_en":"Paraguay Primera División","name_desc":"APF Division de Honor","ticker":"KXAPFDDHGAME"},{"name":"阿甲","name_en":"Liga Profesional de Fútbol","name_desc":"Argentina Primera Division","ticker":"KXARGPREMDIVGAME","pair_poly_sport_id":"arg"},{"name":"澳超","name_en":"A-League Men","name_desc":"Australia A League","ticker":"KXALEAGUEGAME","pair_poly_sport_id":"aus"},{"name":"比甲","name_en":"Belgium Pro League","name_desc":"Belgian Pro League","ticker":"KXBELGIANPLGAME","pair_poly_sport_id":"bel1"},{"name":"解放者杯","name_en":"Copa Libertadores","name_desc":"CONMEBOL Libertadores","ticker":"KXCONMEBOLLIBGAME","pair_poly_sport_id":"lib"},{"name":"世俱杯","name_en":"Club World Cup","name_desc":"Club World Cup","ticker":"KXCLUBWCGAME"},{"name":"国王杯","name_en":"Copa del Rey","name_desc":"Copa del Rey","ticker":"KXCOPADELREYGAME","pair_poly_sport_id":"cdr"},{"name":"巴西杯","name_en":"Copa do Brasil","name_desc":"Copa do Brasil","ticker":"KXCOPADOBRASILGAME"},{"name":"德国杯","name_en":"DFB-Pokal","name_desc":"DFB Pokal","ticker":"KXDFBPOKALGAME","pair_poly_sport_id":"dfb"},{"name":"丹超","name_en":"Danish Superliga","name_desc":"Danish Superliga","ticker":"KXDANISHSUPERLIGAGAME","pair_poly_sport_id":"den"},{"name":"英格兰联赛杯","name_en":"EFL CUP","name_desc":"EFL Cup","ticker":"KXEFLCUPGAME","pair_poly_sport_id":"efl"},{"name":"英格兰联赛奖杯","name_en":"EFL Trophy","name_desc":"EFL Trophy","ticker":"KXEFLTROPHYGAME"},{"name":"波兰超","name_en":"Ekstraklasa","name_desc":"Ekstraklasa","ticker":"KXEKSTRAKLASAGAME","pair_poly_sport_id":"pol"},{"name":"荷甲","name_en":"Eredivisie","name_desc":"Eredivisie","ticker":"KXEREDIVISIEGAME","pair_poly_sport_id":"ere"},{"name":"英格兰足总杯","name_en":"FA Cup","name_desc":"FA Cup","ticker":"KXFACUPGAME","pair_poly_sport_id":"efa"},{"name":"世界杯","name_en":"FIFA World Cup","name_desc":"FIFA World Cup","ticker":"KXWCGAME"},{"name":"荷兰杯","name_en":"KNVB","name_desc":"KNVB","ticker":"KXKNVBCUPGAME"},{"name":"葡超","name_en":"Primeira Liga","name_desc":"Liga Portugal","ticker":"KXLIGAPORTUGALGAME","pair_poly_sport_id":"por"},{"name":"沙特联","name_en":"Saudi Pro League","name_desc":"Saudi Pro League","ticker":"KXSAUDIPLGAME","pair_poly_sport_id":"spl"},{"name":"塞尔维亚杯","name_en":"Serbian Cup","name_desc":"Serbian Cup","ticker":"KXSRBCUPGAME"},{"name":"土超","name_en":"Süper Lig","name_desc":"Super Lig","ticker":"KXSUPERLIGGAME","pair_poly_sport_id":"tur"},{"name":"葡萄牙杯","name_en":"Taça de Portugal","name_desc":"Taca de Portugal","ticker":"KXTACAPORTGAME","pair_poly_sport_id":"ptc"},{"name":"泰超","name_en":"Thai League 1","name_desc":"Thai League 1","ticker":"KXTHAIL1GAME"},{"name":"美国女足超级联赛","name_en":"USL Super League Women","name_desc":"USL Super League Women","ticker":"KXUSLSLWGAME"},{"name":"委内瑞拉甲","name_en":"Venezuela Liga FUTVE","name_desc":"Venezuela Liga FUTVE","ticker":"KXVENFUTVEGAME"}]}]', 'json', 'Kalshi 足球单场 series JSON（name 中文，name_en 统一英文，name_desc 平台展示英文，见 docs/sql/kalshi_game_series.json）', 2, 1),
  ('market_sync', 'sync_outcome_types', '[{"sport_cn":"足球","sub_category":"Football","items":["moneyline","handicap","total"]}]', 'json', '按运动配置盘口。足球：moneyline=胜平负，handicap=让分，total=大小球。其他运动另加一组 sport_cn/sub_category', 3, 1),
  ('orderbook_sync', 'orderbook_fetch', '{"polymarket":{"batch_size":500,"batch_concurrency":3},"kalshi":{"batch_size":100,"batch_concurrency":3}}', 'json', '拉盘口：batch_size 每批条数；batch_concurrency 批间并发（Poly /books 最多 500）', 1, 1),
  ('strategy_entry', 'watch_map_keys', '[]', 'json', '监控场次 map_key 列表（每项应对齐 Poly+Kalshi 两条 market）；strategy_entry 与 orderbook_sync 共用', 1, 1),
  ('strategy_entry', 'max_groups', '50', 'int', 'strategy_entry 同时 WebSocket 监控的 map_key 组数上限；按成交额降序取前 N 组', 2, 1),
  ('strategy_entry', 'ws_connect_stagger_ms', '{"min_ms":0,"max_ms":2000}', 'json', '每场各平台首次 WebSocket 建连前随机等待区间（毫秒）', 3, 1),
  ('strategy_entry', 'pre_event_scan_minutes', '30', 'int', '最早开赛<=now+N(UTC)；0=不按开赛滤', 4, 1),
  ('strategy_entry', 'end_event_scan_minutes', '180', 'int', 'event_end_time 有值则须 end+N>now(UTC)；空不滤；0=不按结束滤', 5, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `description` = VALUES(`description`);

INSERT INTO `platform` (`slug`, `platform_type`, `quote_style`, `display_name`, `sort`, `market_api_base_url`, `order_api_base_url`, `status`, `remark`) VALUES
  ('POLYMARKET', 'PREDICTION', 'PROBABILITY', 'Polymarket', 10, 'https://gamma-api.polymarket.com', 'https://clob.polymarket.com', 'ACTIVE', ''),
  ('KALSHI', 'PREDICTION', 'PROBABILITY', 'Kalshi', 20, 'https://api.elections.kalshi.com', 'https://api.elections.kalshi.com', 'ACTIVE', ''),
  ('FOREGATE', 'PREDICTION', 'PROBABILITY', 'Foregate', 30, NULL, NULL, 'ACTIVE', ''),
  ('PINNACLE', 'ODDS', 'DECIMAL_ODDS', 'Pinnacle', 40, NULL, NULL, 'ACTIVE', '赔率 API 基址写入 extra_config'),
  ('BET365', 'ODDS', 'DECIMAL_ODDS', 'Bet365', 50, NULL, NULL, 'ACTIVE', '赔率 API 基址写入 extra_config')
ON DUPLICATE KEY UPDATE
  `platform_type` = VALUES(`platform_type`),
  `quote_style` = VALUES(`quote_style`),
  `market_api_base_url` = VALUES(`market_api_base_url`),
  `order_api_base_url` = VALUES(`order_api_base_url`);

INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
  ('strategy_entry', 'watch_map_keys', '[]', 'json', '可选：限定监控 map_key；空=LoadVerifiedFixturesNew 扫全库 ACTIVE 核对场次', 1, 1),
  ('strategy_entry', 'max_groups', '1000', 'int', '同时 WebSocket 监控 map_key 上限，按成交额取前 N', 2, 1),
  ('strategy_entry', 'ws_connect_stagger_ms', '{"min_ms":0,"max_ms":2000}', 'json', '首场 WS 建连随机错峰（毫秒）', 3, 1),
  ('strategy_entry', 'pre_event_scan_minutes', '30', 'int', '最早开赛<=now+N(UTC)；0=不按开赛滤', 4, 1),
  ('strategy_entry', 'end_event_scan_minutes', '180', 'int', 'event_end_time 有值则须 end+N>now(UTC)；空不滤；0=不按结束滤', 5, 1),
  ('market_sync', 'concurrency', '{"polymarket":{"concurrency":3},"kalshi":{"concurrency":3},"predict":{"concurrency":2}}', 'json', 'market_sync 各平台并发', 5, 1)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);

INSERT INTO `data_platform` (`code`, `display_name`, `vendor_type`, `api_base_url`, `api_key`, `extra_config`, `status`)
SELECT 'pmdata', 'PMData', 'third_party', 'https://api.pmdata.dev/v1', '',
       '{"storage_dir":"var/pmdata","data_type":"l2","utc_days_before_event":1,"utc_days_after_event":1,"outcome_types":["moneyline"],"polymarket_outcome":"yes","max_snapshots_per_day":500}',
       'ACTIVE'
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM `data_platform` WHERE `code` = 'pmdata');

INSERT INTO `data_platform` (`code`, `display_name`, `vendor_type`, `api_base_url`, `api_key`, `extra_config`, `status`)
SELECT 'synthetic', 'Synthetic L2', 'synthetic', '', '', '{}', 'ACTIVE'
FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM `data_platform` WHERE `code` = 'synthetic');

INSERT INTO `bot_schedule_config` (`module`, `task_name`, `interval_seconds`, `exe_sort`, `concurrency`, `description`, `is_enabled`, `is_strategy_enabled`, `is_primary_machine_run`) VALUES
  ('less_one', 'market_sync', 600, 10, 1, '各平台拉市场列表写入 market', 1, 1, 0),
  ('less_one', 'stat_collect', 300, 25, 1, '刷新 stat_kv', 1, 1, 0),
  ('less_one', 'market_map', 3600, 20, 1, '维护 market.map_key / map_verified', 1, 1, 0),
  ('less_one', 'orderbook_sync', 60, 30, 1, '拉盘口写入 market_order_book', 0, 1, 0),
  ('less_one', 'history_data_sync', 3600, 35, 1, 'data_task pending → PMData L2', 0, 1, 0),
  ('less_one', 'synthetic_history_sync', 3600, 36, 1, 'data_task synthetic_l2 伪造秒级簿', 0, 1, 0),
  ('less_one', 'backtest_replay', 3600, 37, 1, 'backtest_task pending 重放历史簿', 0, 1, 0),
  ('less_one', 'strategy_entry', 20, 40, 1, '已核对场次 WebSocket：二元 + 1X2', 1, 1, 0),
  ('less_one', 'order_fill_sync', 30, 45, 1, '回查 Poly/Predict/SX 成交，补全 orders.trade_qty/trade_price/trade_amount/status/fee', 1, 1, 0),
  ('less_one', 'ws_book_flush', 5, 31, 1, 'Redis 队列批量写 market_order_book_ws', 1, 1, 0),
  ('less_one', 'trade_execute', 5, 60, 1, '下单与敞口（预留）', 0, 1, 0),
  ('less_one', 'task_alert', 300, 70, 1, '告警', 0, 1, 0),
  ('less_one', 'report_daily', 86400, 80, 1, '日复盘', 0, 1, 0),
  ('less_one', 'orderbook', 60, 5, 1, '盘口探针', 0, 1, 0)
ON DUPLICATE KEY UPDATE
  `interval_seconds` = VALUES(`interval_seconds`),
  `exe_sort` = VALUES(`exe_sort`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);

INSERT INTO `platform` (`slug`, `platform_type`, `quote_style`, `display_name`, `sort`, `market_api_base_url`, `order_api_base_url`, `websocket_url`, `status`, `remark`) VALUES
  ('PREDICT', 'PREDICTION', 'PROBABILITY', 'Predict.fun', 25, 'https://api.predict.fun/v1', 'https://api.predict.fun/v1', 'wss://ws.predict.fun/ws', 'INACTIVE', '配置 api_key 后改 ACTIVE')
ON DUPLICATE KEY UPDATE
  `market_api_base_url` = VALUES(`market_api_base_url`),
  `order_api_base_url` = VALUES(`order_api_base_url`),
  `websocket_url` = VALUES(`websocket_url`);



-- ########## 2/5 market_category 联赛 ##########
-- from: docs/sql/config_market_category_install.sql

CREATE TABLE IF NOT EXISTS `market_category` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sport_cn` varchar(64) NOT NULL,
  `sub_category` varchar(64) NOT NULL,
  `platform` varchar(32) NOT NULL,
  `concurrency` int NOT NULL DEFAULT 1,
  `config_json` longtext,
  `outcome_types_json` varchar(512) NOT NULL DEFAULT '[]',
  `sort_no` int NOT NULL DEFAULT 0,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_category_sub_platform` (`sub_category`, `platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
INSERT INTO `market_category` (`id`, `sport_cn`, `sub_category`, `platform`, `concurrency`, `config_json`, `outcome_types_json`, `sort_no`, `is_enabled`, `created_at`, `updated_at`)
VALUES
	(1, '足球', 'Football', 'POLYMARKET', 5, '{"sport_cn":"足球","sub_category":"Football","sync_all_soccer_games":true,"soccer_game_tag_ids":[],"items":[{"name":"英超","name_en":"Premier League","name_desc":"Premier League","sport_id":"epl","pair_kalshi_ticker":"KXEPLGAME"},{"name":"欧冠","name_en":"UEFA Champions League","name_desc":"UEFA Champions League","sport_id":"ucl","pair_kalshi_ticker":"KXUCLGAME","tag_ids":[100977]},{"name":"西甲","name_en":"LaLiga","name_desc":"LaLiga","sport_id":"lal","pair_kalshi_ticker":"KXLALIGAGAME"},{"name":"德甲","name_en":"Bundesliga","name_desc":"Bundesliga","sport_id":"bun","pair_kalshi_ticker":"KXBUNDESLIGAGAME"},{"name":"意甲","name_en":"Serie A","name_desc":"Serie A","sport_id":"sea","pair_kalshi_ticker":"KXSERIEAGAME"},{"name":"法甲","name_en":"Ligue 1","name_desc":"Ligue 1","sport_id":"fl1"},{"name":"欧联","name_en":"UEFA Europa League","name_desc":"UEFA Europa League","sport_id":"uel"},{"name":"荷甲","name_en":"Eredivisie","name_desc":"Eredivisie","sport_id":"ere"},{"name":"葡超","name_en":"Primeira Liga","name_desc":"Primeira Liga","sport_id":"por"},{"name":"土超","name_en":"Süper Lig","name_desc":"Süper Lig","sport_id":"tur"},{"name":"俄超","name_en":"Russian Premier League","name_desc":"Russian Premier League","sport_id":"rus"},{"name":"美职联","name_en":"MLS","name_desc":"MLS","sport_id":"mls"},{"name":"墨超","name_en":"Liga MX","name_desc":"Liga MX","sport_id":"mex"},{"name":"巴甲","name_en":"Brasileirão Série A","name_desc":"Brasileirão Série A","sport_id":"bra"},{"name":"阿甲","name_en":"Liga Profesional de Fútbol","name_desc":"Liga Profesional de Fútbol","sport_id":"arg"},{"name":"沙特联","name_en":"Saudi Pro League","name_desc":"Saudi Pro League","sport_id":"spl"},{"name":"K联赛","name_en":"K League 1","name_desc":"K League 1","sport_id":"kor"},{"name":"J1","name_en":"J1 League","name_desc":"J1 League","sport_id":"jap"},{"name":"J2","name_en":"J2 League","name_desc":"J2 League","sport_id":"ja2"},{"name":"澳超","name_en":"A-League Men","name_desc":"A-League Men","sport_id":"aus"},{"name":"印尼联","name_en":"Indonesia Liga 1","name_desc":"Indonesia Liga 1","sport_id":"idn1"},{"name":"挪超","name_en":"Eliteserien","name_desc":"Eliteserien","sport_id":"nor"},{"name":"丹超","name_en":"Danish Superliga","name_desc":"Danish Superliga","sport_id":"den"},{"name":"欧国联","name_en":"UEFA Nations League","name_desc":"UEFA Nations League","sport_id":"unl"},{"name":"解放者杯","name_en":"Copa Libertadores","name_desc":"Copa Libertadores","sport_id":"lib"},{"name":"中超","name_en":"Chinese Super League","name_desc":"Chinese Super League","sport_id":"chi"},{"name":"南美杯","name_en":"Copa Sudamericana","name_desc":"Copa Sudamericana","sport_id":"sud"},{"name":"国王杯","name_en":"Copa del Rey","name_desc":"Copa del Rey","sport_id":"cdr"},{"name":"法国杯","name_en":"Coupe de France","name_desc":"Coupe de France","sport_id":"cde"},{"name":"德国杯","name_en":"DFB-Pokal","name_desc":"DFB-Pokal","sport_id":"dfb"},{"name":"意大利杯","name_en":"Coppa Italia","name_desc":"Coppa Italia","sport_id":"itc"},{"name":"英格兰足总杯","name_en":"FA Cup","name_desc":"FA Cup","sport_id":"efa"},{"name":"英格兰联赛杯","name_en":"EFL CUP","name_desc":"EFL CUP","sport_id":"efl"},{"name":"西班牙超级杯","name_en":"Supercopa de España","name_desc":"Supercopa de España","sport_id":"ssc"},{"name":"美洲杯","name_en":"Copa America","name_desc":"Copa America","sport_id":"copaam"},{"name":"金杯赛","name_en":"CONCACAF Gold Cup","name_desc":"CONCACAF Gold Cup","sport_id":"ncag"},{"name":"非洲国家杯","name_en":"Africa Cup of Nations","name_desc":"Africa Cup of Nations","sport_id":"acn"},{"name":"哥伦甲","name_en":"Categoría Primera A","name_desc":"Categoría Primera A","sport_id":"col1"},{"name":"欧协联","name_en":"UEFA Conference League","name_desc":"UEFA Conference League","sport_id":"col"},{"name":"亚冠","name_en":"AFC Champions League Elite","name_desc":"AFC Champions League Elite","sport_id":"acle"},{"name":"德乙","name_en":"2. Bundesliga","name_desc":"2. Bundesliga","sport_id":"bl2"},{"name":"德丙","name_en":"3. Liga","name_desc":"3. Liga","sport_id":"bl3"},{"name":"非洲杯预选赛","name_en":"Africa Cup of Nations Qualifiers","name_desc":"Africa Cup of Nations Qualifiers","sport_id":"afcq"},{"name":"比甲","name_en":"Belgium Pro League","name_desc":"Belgium Pro League","sport_id":"bel1"},{"name":"玻利维亚甲","name_en":"Bolivian Primera División","name_desc":"Bolivian Primera División","sport_id":"bol1"},{"name":"巴乙","name_en":"Brasileirão Série B","name_desc":"Brasileirão Série B","sport_id":"bra2"},{"name":"中北美国家联赛","name_en":"CONCACAF Nations League","name_desc":"CONCACAF Nations League","sport_id":"conl"},{"name":"捷克联赛","name_en":"Chance Liga","name_desc":"Chance Liga","sport_id":"cze1"},{"name":"俱乐部友谊赛","name_en":"Club Friendlies","name_desc":"Club Friendlies","sport_id":"clf"},{"name":"阿根廷杯","name_en":"Copa Argentina","name_desc":"Copa Argentina","sport_id":"argcopa"},{"name":"英冠","name_en":"EFL Championship","name_desc":"EFL Championship","sport_id":"elc"},{"name":"荷乙","name_en":"Eerste Divisie","name_desc":"Eerste Divisie","sport_id":"ned2"},{"name":"埃及超","name_en":"Egyptian Premier League","name_desc":"Egyptian Premier League","sport_id":"egy1"},{"name":"波兰超","name_en":"Ekstraklasa","name_desc":"Ekstraklasa","sport_id":"pol"},{"name":"印度超","name_en":"Indian Super League","name_desc":"Indian Super League","sport_id":"isp"},{"name":"J3联赛","name_en":"J3 League","name_desc":"J3 League","sport_id":"ja3"},{"name":"西乙","name_en":"LaLiga2","name_desc":"LaLiga2","sport_id":"es2"},{"name":"英甲","name_en":"League One","name_desc":"League One","sport_id":"el1"},{"name":"英乙","name_en":"League Two","name_desc":"League Two","sport_id":"el2"},{"name":"秘鲁甲","name_en":"Liga 1","name_desc":"Liga 1","sport_id":"per1"},{"name":"哥斯达黎加甲","name_en":"Liga FPD","name_desc":"Liga FPD","sport_id":"fpd"},{"name":"危地马拉甲","name_en":"Liga Nacional de Guatemala","name_desc":"Liga Nacional de Guatemala","sport_id":"gtm"},{"name":"葡甲","name_en":"Liga Portugal 2","name_desc":"Liga Portugal 2","sport_id":"pt2"},{"name":"墨扩军联赛","name_en":"Liga de Expansión MX","name_desc":"Liga de Expansión MX","sport_id":"mex2"},{"name":"智利甲","name_en":"Liga de Primera","name_desc":"Liga de Primera","sport_id":"chi1"},{"name":"法乙","name_en":"Ligue 2","name_desc":"Ligue 2","sport_id":"fr2"},{"name":"美国女足联赛","name_en":"NWSL","name_desc":"NWSL","sport_id":"nwsl"},{"name":"英格兰国家联赛","name_en":"National League","name_desc":"National League","sport_id":"snhl"},{"name":"斯洛伐克超","name_en":"Niké liga","name_desc":"Niké liga","sport_id":"svk1"},{"name":"挪威甲","name_en":"OBOS-ligaen","name_desc":"OBOS-ligaen","sport_id":"nor2"},{"name":"巴拉圭甲","name_en":"Paraguay Primera División","name_desc":"Paraguay Primera División","sport_id":"par1"},{"name":"智利乙","name_en":"Primera B (Chile)","name_desc":"Primera B (Chile)","sport_id":"chl2"},{"name":"西协甲","name_en":"Primera Federación","name_desc":"Primera Federación","sport_id":"es3"},{"name":"阿根廷全国联赛","name_en":"Primera Nacional","name_desc":"Primera Nacional","sport_id":"argpn"},{"name":"克罗地亚甲","name_en":"Prva Liga","name_desc":"Prva Liga","sport_id":"hr1"},{"name":"苏超","name_en":"Scottish Premiership","name_desc":"Scottish Premiership","sport_id":"scop"},{"name":"意乙","name_en":"Serie B","name_desc":"Serie B","sport_id":"itsb"},{"name":"意丙","name_en":"Serie C","name_desc":"Serie C","sport_id":"serc"},{"name":"罗马尼亚超","name_en":"SuperLiga","name_desc":"SuperLiga","sport_id":"rou1"},{"name":"葡萄牙杯","name_en":"Taça de Portugal","name_desc":"Taça de Portugal","sport_id":"ptc"},{"name":"阿联酋超","name_en":"UAE Pro League","name_desc":"UAE Pro League","sport_id":"uae1"},{"name":"女足欧冠","name_en":"UEFA Women''s Champions League","name_desc":"UEFA Women''s Champions League","sport_id":"uwcl"},{"name":"美国足球冠军联赛","name_en":"USL Championship","name_desc":"USL Championship","sport_id":"uslc"},{"name":"乌克兰超","name_en":"Ukrainian Premier League","name_desc":"Ukrainian Premier League","sport_id":"ukr1"},{"name":"乌拉圭甲","name_en":"Uruguayan Primera División","name_desc":"Uruguayan Primera División","sport_id":"uru1"},{"name":"乌兹别克超","name_en":"Uzbekistan Super League","name_desc":"Uzbekistan Super League","sport_id":"uzb1"},{"name":"拉脱维亚超","name_en":"Virslīga","name_desc":"Virslīga","sport_id":"lva1"},{"name":"英格兰女足超","name_en":"Women''s Super League","name_desc":"Women''s Super League","sport_id":"wsl"},{"name":"英格兰女足甲级","name_en":"Women''s Super League 2","name_desc":"Women''s Super League 2","sport_id":"wsl2"},{"name":"保加利亚超","name_en":"efbet Liga","name_desc":"efbet Liga","sport_id":"bul"},{"name":"国际足球","name_en":"FIFA Friendlies","name_desc":"FIFA Friendlies","sport_id":"fif"}]}', '["moneyline","handicap","total"]', 1, 1, NOW(), NOW()),
	(2, '足球', 'Football', 'PREDICT', 2, '{"sport_cn":"足球","sub_category":"Football","sync_all_soccer":true,"soccer_root_tag_id":"14","market_variants":["SPORTS_MATCH","SPORTS_TEAM_MATCH","SPORTS_PROPS"],"items":[{"name":"英超","name_en":"Premier League","name_desc":"Premier League","sport_id":"epl","tag_ids":["546"]},{"name":"欧冠","name_en":"UEFA Champions League","name_desc":"UEFA Champions League","sport_id":"ucl","tag_ids":["342"]},{"name":"西甲","name_en":"LaLiga","name_desc":"LaLiga","sport_id":"lal","tag_ids":["852"]},{"name":"德甲","name_en":"Bundesliga","name_desc":"Bundesliga","sport_id":"bun","tag_ids":["80"]},{"name":"意甲","name_en":"Serie A","name_desc":"Serie A","sport_id":"sea","tag_ids":["545"]},{"name":"法甲","name_en":"Ligue 1","name_desc":"Ligue 1","sport_id":"fl1","tag_ids":["544"]},{"name":"欧联","name_en":"UEFA Europa League","name_desc":"UEFA Europa League","sport_id":"uel","tag_ids":["343"]},{"name":"荷甲","name_en":"Eredivisie","name_desc":"Eredivisie","sport_id":"ere","tag_ids":["1923"]},{"name":"葡超","name_en":"Primeira Liga","name_desc":"Primeira Liga","sport_id":"por","tag_ids":["1921"]},{"name":"土超","name_en":"Süper Lig","name_desc":"Süper Lig","sport_id":"tur","tag_ids":["1924"]},{"name":"俄超","name_en":"Russian Premier League","name_desc":"Russian Premier League","sport_id":"rus","tag_ids":[]},{"name":"美职联","name_en":"MLS","name_desc":"MLS","sport_id":"mls","tag_ids":["376"]},{"name":"墨超","name_en":"Liga MX","name_desc":"Liga MX","sport_id":"mex","tag_ids":["3858"]},{"name":"巴甲","name_en":"Brasileirão Série A","name_desc":"Brasileirão Série A","sport_id":"bra","tag_ids":["1919"]},{"name":"阿甲","name_en":"Liga Profesional de Fútbol","name_desc":"Liga Profesional de Fútbol","sport_id":"arg","tag_ids":["3854"]},{"name":"沙特联","name_en":"Saudi Pro League","name_desc":"Saudi Pro League","sport_id":"spl","tag_ids":[]},{"name":"K联赛","name_en":"K League 1","name_desc":"K League 1","sport_id":"kor","tag_ids":["1926"]},{"name":"J1","name_en":"J1 League","name_desc":"J1 League","sport_id":"jap","tag_ids":["1925"]},{"name":"J2","name_en":"J2 League","name_desc":"J2 League","sport_id":"ja2","tag_ids":[]},{"name":"澳超","name_en":"A-League Men","name_desc":"A-League Men","sport_id":"aus","tag_ids":["3855"]},{"name":"印尼联","name_en":"Indonesia Liga 1","name_desc":"Indonesia Liga 1","sport_id":"idn1","tag_ids":[]},{"name":"挪超","name_en":"Eliteserien","name_desc":"Eliteserien","sport_id":"nor","tag_ids":["1922"]},{"name":"丹超","name_en":"Danish Superliga","name_desc":"Danish Superliga","sport_id":"den","tag_ids":["3857"]},{"name":"欧国联","name_en":"UEFA Nations League","name_desc":"UEFA Nations League","sport_id":"unl","tag_ids":["2417"]},{"name":"解放者杯","name_en":"Copa Libertadores","name_desc":"Copa Libertadores","sport_id":"lib","tag_ids":[]},{"name":"中超","name_en":"Chinese Super League","name_desc":"Chinese Super League","sport_id":"chi","tag_ids":["443"]},{"name":"南美杯","name_en":"Copa Sudamericana","name_desc":"Copa Sudamericana","sport_id":"sud","tag_ids":[]},{"name":"国王杯","name_en":"Copa del Rey","name_desc":"Copa del Rey","sport_id":"cdr","tag_ids":[]},{"name":"法国杯","name_en":"Coupe de France","name_desc":"Coupe de France","sport_id":"cde","tag_ids":[]},{"name":"德国杯","name_en":"DFB-Pokal","name_desc":"DFB-Pokal","sport_id":"dfb","tag_ids":[]},{"name":"意大利杯","name_en":"Coppa Italia","name_desc":"Coppa Italia","sport_id":"itc","tag_ids":[]},{"name":"英格兰足总杯","name_en":"FA Cup","name_desc":"FA Cup","sport_id":"efa","tag_ids":[]},{"name":"英格兰联赛杯","name_en":"EFL CUP","name_desc":"EFL CUP","sport_id":"efl","tag_ids":["1918"]},{"name":"西班牙超级杯","name_en":"Supercopa de España","name_desc":"Supercopa de España","sport_id":"ssc","tag_ids":[]},{"name":"美洲杯","name_en":"Copa America","name_desc":"Copa America","sport_id":"copaam","tag_ids":[]},{"name":"金杯赛","name_en":"CONCACAF Gold Cup","name_desc":"CONCACAF Gold Cup","sport_id":"ncag","tag_ids":[]},{"name":"非洲国家杯","name_en":"Africa Cup of Nations","name_desc":"Africa Cup of Nations","sport_id":"acn","tag_ids":[]},{"name":"哥伦甲","name_en":"Categoría Primera A","name_desc":"Categoría Primera A","sport_id":"col1","tag_ids":[]},{"name":"欧协联","name_en":"UEFA Conference League","name_desc":"UEFA Conference League","sport_id":"col","tag_ids":[]},{"name":"亚冠","name_en":"AFC Champions League Elite","name_desc":"AFC Champions League Elite","sport_id":"acle","tag_ids":["2163"]},{"name":"德乙","name_en":"2. Bundesliga","name_desc":"2. Bundesliga","sport_id":"bl2","tag_ids":[]},{"name":"德丙","name_en":"3. Liga","name_desc":"3. Liga","sport_id":"bl3","tag_ids":[]},{"name":"非洲杯预选赛","name_en":"Africa Cup of Nations Qualifiers","name_desc":"Africa Cup of Nations Qualifiers","sport_id":"afcq","tag_ids":[]},{"name":"比甲","name_en":"Belgium Pro League","name_desc":"Belgium Pro League","sport_id":"bel1","tag_ids":["3856"]},{"name":"玻利维亚甲","name_en":"Bolivian Primera División","name_desc":"Bolivian Primera División","sport_id":"bol1","tag_ids":[]},{"name":"巴乙","name_en":"Brasileirão Série B","name_desc":"Brasileirão Série B","sport_id":"bra2","tag_ids":[]},{"name":"中北美国家联赛","name_en":"CONCACAF Nations League","name_desc":"CONCACAF Nations League","sport_id":"conl","tag_ids":[]},{"name":"捷克联赛","name_en":"Chance Liga","name_desc":"Chance Liga","sport_id":"cze1","tag_ids":[]},{"name":"俱乐部友谊赛","name_en":"Club Friendlies","name_desc":"Club Friendlies","sport_id":"clf","tag_ids":[]},{"name":"阿根廷杯","name_en":"Copa Argentina","name_desc":"Copa Argentina","sport_id":"argcopa","tag_ids":[]},{"name":"英冠","name_en":"EFL Championship","name_desc":"EFL Championship","sport_id":"elc","tag_ids":[]},{"name":"荷乙","name_en":"Eerste Divisie","name_desc":"Eerste Divisie","sport_id":"ned2","tag_ids":[]},{"name":"埃及超","name_en":"Egyptian Premier League","name_desc":"Egyptian Premier League","sport_id":"egy1","tag_ids":[]},{"name":"波兰超","name_en":"Ekstraklasa","name_desc":"Ekstraklasa","sport_id":"pol","tag_ids":[]},{"name":"印度超","name_en":"Indian Super League","name_desc":"Indian Super League","sport_id":"isp","tag_ids":[]},{"name":"J3联赛","name_en":"J3 League","name_desc":"J3 League","sport_id":"ja3","tag_ids":[]},{"name":"西乙","name_en":"LaLiga2","name_desc":"LaLiga2","sport_id":"es2","tag_ids":[]},{"name":"英甲","name_en":"League One","name_desc":"League One","sport_id":"el1","tag_ids":[]},{"name":"英乙","name_en":"League Two","name_desc":"League Two","sport_id":"el2","tag_ids":[]},{"name":"秘鲁甲","name_en":"Liga 1","name_desc":"Liga 1","sport_id":"per1","tag_ids":[]},{"name":"哥斯达黎加甲","name_en":"Liga FPD","name_desc":"Liga FPD","sport_id":"fpd","tag_ids":[]},{"name":"危地马拉甲","name_en":"Liga Nacional de Guatemala","name_desc":"Liga Nacional de Guatemala","sport_id":"gtm","tag_ids":[]},{"name":"葡甲","name_en":"Liga Portugal 2","name_desc":"Liga Portugal 2","sport_id":"pt2","tag_ids":[]},{"name":"墨扩军联赛","name_en":"Liga de Expansión MX","name_desc":"Liga de Expansión MX","sport_id":"mex2","tag_ids":[]},{"name":"智利甲","name_en":"Liga de Primera","name_desc":"Liga de Primera","sport_id":"chi1","tag_ids":[]},{"name":"法乙","name_en":"Ligue 2","name_desc":"Ligue 2","sport_id":"fr2","tag_ids":[]},{"name":"美国女足联赛","name_en":"NWSL","name_desc":"NWSL","sport_id":"nwsl","tag_ids":[]},{"name":"英格兰国家联赛","name_en":"National League","name_desc":"National League","sport_id":"snhl","tag_ids":[]},{"name":"斯洛伐克超","name_en":"Niké liga","name_desc":"Niké liga","sport_id":"svk1","tag_ids":[]},{"name":"挪威甲","name_en":"OBOS-ligaen","name_desc":"OBOS-ligaen","sport_id":"nor2","tag_ids":[]},{"name":"巴拉圭甲","name_en":"Paraguay Primera División","name_desc":"Paraguay Primera División","sport_id":"par1","tag_ids":[]},{"name":"智利乙","name_en":"Primera B (Chile)","name_desc":"Primera B (Chile)","sport_id":"chl2","tag_ids":[]},{"name":"西协甲","name_en":"Primera Federación","name_desc":"Primera Federación","sport_id":"es3","tag_ids":[]},{"name":"阿根廷全国联赛","name_en":"Primera Nacional","name_desc":"Primera Nacional","sport_id":"argpn","tag_ids":[]},{"name":"克罗地亚甲","name_en":"Prva Liga","name_desc":"Prva Liga","sport_id":"hr1","tag_ids":[]},{"name":"苏超","name_en":"Scottish Premiership","name_desc":"Scottish Premiership","sport_id":"scop","tag_ids":["954"]},{"name":"意乙","name_en":"Serie B","name_desc":"Serie B","sport_id":"itsb","tag_ids":[]},{"name":"意丙","name_en":"Serie C","name_desc":"Serie C","sport_id":"serc","tag_ids":[]},{"name":"罗马尼亚超","name_en":"SuperLiga","name_desc":"SuperLiga","sport_id":"rou1","tag_ids":[]},{"name":"葡萄牙杯","name_en":"Taça de Portugal","name_desc":"Taça de Portugal","sport_id":"ptc","tag_ids":[]},{"name":"阿联酋超","name_en":"UAE Pro League","name_desc":"UAE Pro League","sport_id":"uae1","tag_ids":[]},{"name":"女足欧冠","name_en":"UEFA Women''s Champions League","name_desc":"UEFA Women''s Champions League","sport_id":"uwcl","tag_ids":[]},{"name":"美国足球冠军联赛","name_en":"USL Championship","name_desc":"USL Championship","sport_id":"uslc","tag_ids":[]},{"name":"乌克兰超","name_en":"Ukrainian Premier League","name_desc":"Ukrainian Premier League","sport_id":"ukr1","tag_ids":[]},{"name":"乌拉圭甲","name_en":"Uruguayan Primera División","name_desc":"Uruguayan Primera División","sport_id":"uru1","tag_ids":[]},{"name":"乌兹别克超","name_en":"Uzbekistan Super League","name_desc":"Uzbekistan Super League","sport_id":"uzb1","tag_ids":[]},{"name":"拉脱维亚超","name_en":"Virslīga","name_desc":"Virslīga","sport_id":"lva1","tag_ids":[]},{"name":"英格兰女足超","name_en":"Women''s Super League","name_desc":"Women''s Super League","sport_id":"wsl","tag_ids":[]},{"name":"英格兰女足甲级","name_en":"Women''s Super League 2","name_desc":"Women''s Super League 2","sport_id":"wsl2","tag_ids":[]},{"name":"保加利亚超","name_en":"efbet Liga","name_desc":"efbet Liga","sport_id":"bul","tag_ids":[]},{"name":"国际足球","name_en":"FIFA Friendlies","name_desc":"FIFA Friendlies","sport_id":"fif","tag_ids":["1336"]}]}', '["moneyline","handicap","total"]', 1, 1, NOW(), NOW()),
	(3, '板球', 'Cricket', 'POLYMARKET', 3, '{"sport_cn":"板球","sub_category":"Cricket","items":[{"name":"IPL","name_en":"IPL","name_desc":"IPL","sport_id":"ipl","tag_ids":[101977]},{"name":"ODI","name_en":"ODI","name_desc":"ODI","sport_id":"odi","tag_ids":[102815]},{"name":"T20","name_en":"T20 Internationals","name_desc":"T20 Internationals","sport_id":"t20","tag_ids":[102810]},{"name":"Big Bash League","name_en":"Big Bash League","name_desc":"Big Bash League","sport_id":"abb","tag_ids":[102813]},{"name":"CSA T20 Challenge","name_en":"CSA T20 Challenge","name_desc":"CSA T20 Challenge","sport_id":"csa","tag_ids":[102808]},{"name":"Cricket Bangladesh","name_en":"Cricket Bangladesh","name_desc":"Cricket Bangladesh","sport_id":"crban","tag_ids":[103014]},{"name":"International","name_en":"International","name_desc":"International","sport_id":"crint","tag_ids":[102803]},{"name":"Cricket Australia","name_en":"Cricket Australia","name_desc":"Cricket Australia","sport_id":"craus","tag_ids":[102946]},{"name":"Cricket England","name_en":"Cricket England","name_desc":"Cricket England","sport_id":"creng","tag_ids":[103017]},{"name":"Cricket New Zealand","name_en":"Cricket New Zealand","name_desc":"Cricket New Zealand","sport_id":"crnew","tag_ids":[102949]},{"name":"Cricket India","name_en":"Cricket India","name_desc":"Cricket India","sport_id":"crind","tag_ids":[103018]},{"name":"Cricket South Africa","name_en":"Cricket South Africa","name_desc":"Cricket South Africa","sport_id":"crsou","tag_ids":[103015]},{"name":"Cricket Pakistan","name_en":"Cricket Pakistan","name_desc":"Cricket Pakistan","sport_id":"crpak","tag_ids":[103016]},{"name":"Cricket UAE","name_en":"Cricket UAE","name_desc":"Cricket UAE","sport_id":"cruae","tag_ids":[102948]},{"name":"Test Cricket","name_en":"Test Cricket","name_desc":"Test Cricket","sport_id":"test","tag_ids":[102842]},{"name":"Sheffield Shield","name_en":"Sheffield Shield","name_desc":"Sheffield Shield","sport_id":"she","tag_ids":[102817]},{"name":"SA20","name_en":"SA20","name_desc":"SA20","sport_id":"sasa","tag_ids":[103811]},{"name":"Lanka Premier League","name_en":"Lanka Premier League","name_desc":"Lanka Premier League","sport_id":"lpl","tag_ids":[102812]},{"name":"U19 Cricket World Cup","name_en":"U19 Cricket World Cup","name_desc":"U19 Cricket World Cup","sport_id":"cru19wc","tag_ids":[103178]},{"name":"Women''s Premier League","name_en":"Women''s Premier League","name_desc":"Women''s Premier League","sport_id":"crwpl20","tag_ids":[103179]}]}', '["moneyline","handicap","total"]', 2, 1, NOW(), NOW()),
	(4, '板球', 'Cricket', 'PREDICT', 2, '{"sport_cn":"板球","sub_category":"Cricket","sport_en":"Cricket","market_variants":["SPORTS_MATCH","SPORTS_PROPS"],"items":[],"sync_all_sub":true,"sync_root_tag_id":"97"}', '["moneyline","handicap","total"]', 2, 1, NOW(), NOW()),
	(5, '网球', 'Tennis', 'POLYMARKET', 3, '{"sport_cn":"网球","sub_category":"Tennis","items":[{"name":"ATP","name_en":"ATP Tour","name_desc":"ATP Tour","sport_id":"atp","tag_ids":[101232]},{"name":"WTA","name_en":"WTA Tour","name_desc":"WTA Tour","sport_id":"wta","tag_ids":[102123]}]}', '["moneyline","handicap","total"]', 3, 1, NOW(), NOW()),
	(6, '网球', 'Tennis', 'PREDICT', 2, '{"sport_cn":"网球","sub_category":"Tennis","sport_en":"Tennis","market_variants":["SPORTS_MATCH","SPORTS_PROPS"],"items":[],"sync_all_sub":true,"sync_root_tag_id":"85"}', '["moneyline","handicap","total"]', 3, 1, NOW(), NOW()),
	(7, '篮球', 'Basketball', 'POLYMARKET', 3, '{"sport_cn":"篮球","sub_category":"Basketball","items":[{"name":"NBA","name_en":"NBA","name_desc":"NBA","sport_id":"nba","tag_ids":[745]},{"name":"NCAA男子篮球","name_en":"NCAA Men''s Basketball","name_desc":"NCAA Men''s Basketball","sport_id":"ncaab","tag_ids":[100149]},{"name":"NCAA篮球","name_en":"College Basketball","name_desc":"College Basketball","sport_id":"cbb","tag_ids":[101178]},{"name":"WNBA","name_en":"WNBA","name_desc":"WNBA","sport_id":"wnba","tag_ids":[100254]},{"name":"NCAA女子篮球","name_en":"Women''s College Basketball","name_desc":"Women''s College Basketball","sport_id":"cwbb","tag_ids":[102003]},{"name":"Liga ACB","name_en":"Liga ACB","name_desc":"Liga ACB","sport_id":"bkligend","tag_ids":[103094]},{"name":"NBL","name_en":"NBL","name_desc":"NBL","sport_id":"bknbl","tag_ids":[103096]},{"name":"Basketball Champions League","name_en":"Basketball Champions League","name_desc":"Basketball Champions League","sport_id":"bkcl","tag_ids":[103093]},{"name":"Lega Basket Serie A","name_en":"Lega Basket Serie A","name_desc":"Lega Basket Serie A","sport_id":"bkseriea","tag_ids":[103095]},{"name":"CBA","name_en":"CBA","name_desc":"CBA","sport_id":"bkcba","tag_ids":[103097]},{"name":"Betclic Élite","name_en":"Betclic Élite","name_desc":"Betclic Élite","sport_id":"bkfr1","tag_ids":[103098]},{"name":"Liga Nacional de Básquet","name_en":"Liga Nacional de Básquet","name_desc":"Liga Nacional de Básquet","sport_id":"bkarg","tag_ids":[103099]},{"name":"KBL","name_en":"KBL","name_desc":"KBL","sport_id":"bkkbl","tag_ids":[103100]}]}', '["moneyline","handicap","total"]', 4, 1, NOW(), NOW()),
	(8, '篮球', 'Basketball', 'PREDICT', 2, '{"sport_cn":"篮球","sub_category":"Basketball","sport_en":"Basketball","market_variants":["SPORTS_MATCH","SPORTS_PROPS"],"items":[{"name":"NBA","name_en":"NBA","name_desc":"NBA","sport_id":"nba","tag_ids":["78"]},{"name":"NCAA男子篮球","name_en":"NCAA Men''s Basketball","name_desc":"NCAA Men''s Basketball","sport_id":"ncaab","tag_ids":["82"]},{"name":"FIBA","name_en":"FIBA","name_desc":"FIBA","sport_id":"fiba","tag_ids":["98"]}]}', '["moneyline","handicap","total"]', 4, 1, NOW(), NOW()),
	(9, '美式足球', 'NFL', 'POLYMARKET', 3, '{"sport_cn":"美式足球","sub_category":"NFL","items":[{"name":"NFL","name_en":"NFL","name_desc":"NFL","sport_id":"nfl","tag_ids":[450]},{"name":"NCAA橄榄球","name_en":"College Football","name_desc":"College Football","sport_id":"cfb","tag_ids":[100351]}]}', '["moneyline","handicap","total"]', 5, 1, NOW(), NOW()),
	(10, '美式足球', 'NFL', 'PREDICT', 2, '{"sport_cn":"美式足球","sub_category":"NFL","sport_en":"American Football","market_variants":["SPORTS_MATCH","SPORTS_PROPS"],"items":[{"name":"NFL","name_en":"NFL","name_desc":"NFL","sport_id":"nfl","tag_ids":["45"]}]}', '["moneyline","handicap","total"]', 5, 1, NOW(), NOW()),
	(11, '冰球', 'NHL', 'POLYMARKET', 3, '{"sport_cn":"冰球","sub_category":"NHL","items":[{"name":"NHL","name_en":"NHL","name_desc":"NHL","sport_id":"nhl","tag_ids":[899]},{"name":"AHL","name_en":"AHL","name_desc":"AHL","sport_id":"ahl","tag_ids":[102907]},{"name":"SHL","name_en":"SHL","name_desc":"SHL","sport_id":"shl","tag_ids":[102906]},{"name":"Czech Extraliga","name_en":"Czech Extraliga","name_desc":"Czech Extraliga","sport_id":"cehl","tag_ids":[102910]},{"name":"DEL","name_en":"DEL","name_desc":"DEL","sport_id":"dehl","tag_ids":[102909]},{"name":"National League","name_en":"National League","name_desc":"National League","sport_id":"snhl","tag_ids":[102911]},{"name":"KHL","name_en":"KHL","name_desc":"KHL","sport_id":"khl","tag_ids":[102908]}]}', '["moneyline","handicap","total"]', 6, 1, NOW(), NOW()),
	(12, '冰球', 'NHL', 'PREDICT', 2, '{"sport_cn":"冰球","sub_category":"NHL","sport_en":"Ice Hockey","market_variants":["SPORTS_MATCH","SPORTS_PROPS"],"items":[{"name":"NHL","name_en":"NHL","name_desc":"NHL","sport_id":"nhl","tag_ids":["79"]}]}', '["moneyline","handicap","total"]', 6, 1, NOW(), NOW()),
	(13, 'UFC', 'MMA', 'POLYMARKET', 3, '{"sport_cn":"UFC","sub_category":"MMA","items":[{"name":"UFC","name_en":"UFC","name_desc":"UFC","sport_id":"ufc","tag_ids":[279]}]}', '["moneyline","handicap","total"]', 7, 1, NOW(), NOW()),
	(14, 'UFC', 'MMA', 'PREDICT', 2, '{"sport_cn":"UFC","sub_category":"MMA","sport_en":"UFC","market_variants":["SPORTS_MATCH","SPORTS_PROPS"],"items":[],"sync_all_sub":true,"sync_root_tag_id":"177"}', '["moneyline","handicap","total"]', 7, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
`sport_cn`=VALUES(`sport_cn`), `concurrency`=VALUES(`concurrency`), `config_json`=VALUES(`config_json`),
`outcome_types_json`=VALUES(`outcome_types_json`),
`sort_no`=VALUES(`sort_no`), `is_enabled`=1, `updated_at`=NOW();


-- ########## 3/5 config_strategy_fee ##########
-- from: docs/sql/config_strategy_fee.sql

-- 吃单手续费系数。费用 = taker_fee_coef * 价格 * (1 - 价格)，按每个份额。
-- sub_category 为空：该平台这个 category 下，没有单独行的运动用这一行。
--
-- Polymarket 官方 Sports 类别吃单系数 0.05，各运动相同。Maker 为 0。
--   https://docs.polymarket.com/trading/fees
-- Kalshi 吃单费用 = 系列乘数 * 0.07 * 份额 * 价格 * (1-价格)。
--   2026-07-07 价目表里已列出的运动系列，乘数多为 1，所以系数写成 0.07。
--   某个系列改了乘数，改对应运动这一行。
--   https://kalshi.com/docs/kalshi-fee-schedule.pdf

CREATE TABLE IF NOT EXISTS `config_strategy_fee` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `platform` varchar(32) NOT NULL DEFAULT '' COMMENT 'POLYMARKET / KALSHI',
  `category` varchar(64) NOT NULL DEFAULT '' COMMENT '市场大类，运动为 Sports',
  `sub_category` varchar(64) NOT NULL DEFAULT '' COMMENT '运动。空字符串表示该大类的默认系数',
  `taker_fee_coef` decimal(20,8) NOT NULL COMMENT '吃单系数。费用 = 系数 * 价格 * (1-价格)',
  `description` varchar(500) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_fee_platform_category_sub` (`platform`, `category`, `sub_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='按平台和运动的吃单手续费系数';

INSERT INTO `config_strategy_fee` (`platform`, `category`, `sub_category`, `taker_fee_coef`, `description`, `status`) VALUES
  ('POLYMARKET', 'Sports', '', 0.05000000, 'Polymarket 体育默认。官方 Sports 吃单系数 0.05，各运动相同', 1),
  ('POLYMARKET', 'Sports', 'Football', 0.05000000, '足球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Basketball', 0.05000000, '篮球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'NFL', 0.05000000, '美式橄榄球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'MLB', 0.05000000, '棒球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'NHL', 0.05000000, '冰球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Tennis', 0.05000000, '网球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'MMA', 0.05000000, '综合格斗。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Boxing', 0.05000000, '拳击。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Golf', 0.05000000, '高尔夫。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Esports', 0.05000000, '电竞。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Rugby', 0.05000000, '橄榄球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Cricket', 0.05000000, '板球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'WinterOlympics', 0.05000000, '冬奥。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'TableTennis', 0.05000000, '乒乓球。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Formula1', 0.05000000, 'F1。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Chess', 0.05000000, '国际象棋。Polymarket Sports 吃单系数 0.05', 1),
  ('POLYMARKET', 'Sports', 'Pickleball', 0.05000000, '匹克球。Polymarket Sports 吃单系数 0.05', 1),
  ('KALSHI', 'Sports', '', 0.07000000, 'Kalshi 体育默认。系列乘数 1 时吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Football', 0.07000000, '足球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Basketball', 0.07000000, '篮球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'NFL', 0.07000000, '美式橄榄球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'MLB', 0.07000000, '棒球。Kalshi 系列乘数 1，吃单系数 0.07。某系列乘数变了就改这一行', 1),
  ('KALSHI', 'Sports', 'NHL', 0.07000000, '冰球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Tennis', 0.07000000, '网球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'MMA', 0.07000000, '综合格斗。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Boxing', 0.07000000, '拳击。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Golf', 0.07000000, '高尔夫。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Esports', 0.07000000, '电竞。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Rugby', 0.07000000, '橄榄球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Cricket', 0.07000000, '板球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'WinterOlympics', 0.07000000, '冬奥。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'TableTennis', 0.07000000, '乒乓球。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Formula1', 0.07000000, 'F1。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Chess', 0.07000000, '国际象棋。Kalshi 系列乘数 1，吃单系数 0.07', 1),
  ('KALSHI', 'Sports', 'Pickleball', 0.07000000, '匹克球。Kalshi 系列乘数 1，吃单系数 0.07', 1)
ON DUPLICATE KEY UPDATE
  `taker_fee_coef` = VALUES(`taker_fee_coef`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

UPDATE `config_strategy_threshold`
SET `status` = 0,
    `description` = '已停用。手续费改看 config_strategy_fee'
WHERE `config_key` IN ('binary_yes_no_poly_fee_coef', 'binary_yes_no_kalshi_fee_coef');


-- ########## 4/5 predict_market_sync ##########
-- from: docs/sql/predict_market_sync.sql

-- Predict.fun 接入 market_sync（平台 + 配置 + 并发）
-- 前置：REST 需 x-api-key（任选其一）
--   1) platform.extra_config = '{"api_key":"你的key"}'
--   2) account 表 platform=PREDICT, status=ACTIVE, api_key=你的key
-- tag_ids：登录后 GET https://api.predict.fun/v1/tags，按联赛 name 填到 predict_sports.items[].tag_ids

-- 1) 平台（默认不 ACTIVE，填好 api_key 后再改为 ACTIVE）
INSERT INTO `platform` (
  `slug`, `platform_type`, `quote_style`, `display_name`, `sort`,
  `market_api_base_url`, `order_api_base_url`, `websocket_url`, `status`, `remark`, `extra_config`
) VALUES (
  'PREDICT', 'PREDICTION', 'PROBABILITY', 'Predict.fun', 25,
  'https://api.predict.fun/v1', 'https://api.predict.fun/v1', 'wss://ws.predict.fun/ws',
  'INACTIVE', 'market_sync 需 api_key；extra_config 或 account 表',
  NULL
)
ON DUPLICATE KEY UPDATE
  `platform_type` = VALUES(`platform_type`),
  `quote_style` = VALUES(`quote_style`),
  `display_name` = VALUES(`display_name`),
  `market_api_base_url` = VALUES(`market_api_base_url`),
  `order_api_base_url` = VALUES(`order_api_base_url`),
  `websocket_url` = VALUES(`websocket_url`),
  `remark` = VALUES(`remark`);

-- 2) 联赛 tag 配置（与 Poly/Kalshi 足球子类对齐；tag_ids 请按环境补全）
INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`) VALUES
(
  'market_sync',
  'predict_sports',
  '[{"sport_cn":"足球","sub_category":"Football","market_variants":["SPORTS_MATCH","SPORTS_TEAM_MATCH"],"items":[{"name":"欧国联","name_en":"UEFA Nations League","sport_id":"unl","tag_ids":[]},{"name":"英超","name_en":"Premier League","sport_id":"epl","tag_ids":[]},{"name":"欧冠","name_en":"UEFA Champions League","sport_id":"ucl","tag_ids":[]},{"name":"西甲","name_en":"LaLiga","sport_id":"lal","tag_ids":[]},{"name":"德甲","name_en":"Bundesliga","sport_id":"bun","tag_ids":[]},{"name":"意甲","name_en":"Serie A","sport_id":"sea","tag_ids":[]},{"name":"法甲","name_en":"Ligue 1","sport_id":"fl1","tag_ids":[]},{"name":"欧联","name_en":"UEFA Europa League","sport_id":"uel","tag_ids":[]},{"name":"美职联","name_en":"MLS","sport_id":"mls","tag_ids":[]}]}]',
  'json',
  'Predict 足球：按 tag_ids 过滤 GET /v1/categories；market_variants 默认 SPORTS_MATCH',
  5,
  1
)
ON DUPLICATE KEY UPDATE
  `item_value` = VALUES(`item_value`),
  `value_type` = VALUES(`value_type`),
  `description` = VALUES(`description`),
  `is_enabled` = VALUES(`is_enabled`);

-- 3) 联赛拉取并发（在原有 poly/kalshi 上增加 predict）
UPDATE `config_strategy`
SET `item_value` = JSON_SET(
  COALESCE(NULLIF(`item_value`, ''), '{"polymarket":{"concurrency":3},"kalshi":{"concurrency":3}}'),
  '$.predict',
  JSON_OBJECT('concurrency', 2)
),
    `description` = '市场同步：polymarket/kalshi/predict 的 concurrency（1-8）'
WHERE `group_name` = 'market_sync' AND `item_key` IN ('market_league_fetch', 'league_concurrency', 'concurrency');

INSERT INTO `config_strategy` (`group_name`, `item_key`, `item_value`, `value_type`, `description`, `sort_no`, `is_enabled`)
SELECT 'market_sync', 'concurrency',
  '{"polymarket":{"concurrency":3},"kalshi":{"concurrency":3},"predict":{"concurrency":2}}',
  'json', '市场同步：polymarket/kalshi/predict 的 concurrency（1-8）', 4, 1
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM `config_strategy` WHERE `group_name` = 'market_sync' AND `item_key` IN ('market_league_fetch', 'league_concurrency', 'concurrency')
);

-- 4) 手续费占位（与 Poly 足球一致，可后续按 Predict feeRateBps 调整）
INSERT INTO `config_strategy_fee` (`platform`, `category`, `sub_category`, `taker_fee_coef`, `description`, `status`) VALUES
  ('PREDICT', 'Sports', '', 0.02000000, 'Predict 体育默认占位（按市场 feeRateBps 微调）', 1),
  ('PREDICT', 'Sports', 'Football', 0.02000000, '足球', 1)
ON DUPLICATE KEY UPDATE
  `taker_fee_coef` = VALUES(`taker_fee_coef`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

-- 5) 启用平台（确认 api_key 已配置后执行）
-- UPDATE `platform` SET `status` = 'ACTIVE' WHERE `slug` = 'PREDICT';

-- ########## 5/5 predict 全联赛对齐 ##########
-- from: docs/sql/migration_predict_all_league_align.sql

-- Predict 全子类联赛名与 POLYMARKET 对齐（config items + 存量 slug 回填）
-- 前提：migration_market_outcome_competition_en.sql 已执行
-- 已单独跑过：migration_predict_cricket_league_items.sql / migration_predict_tennis_mma_league_items.sql 可跳过对应段或重复执行无害
-- 执行后：market_map

-- ========== Basketball：补全 items（保留 nba/ncaab/fiba 原有 tag_ids）==========
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "篮球",
  "sub_category": "Basketball",
  "sport_en": "Basketball",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "items": [
    {"name": "NBA", "name_en": "NBA", "name_desc": "NBA", "sport_id": "nba", "tag_ids": ["78"]},
    {"name": "NCAA篮球", "name_en": "NCAA Basketball", "name_desc": "NCAA Basketball", "sport_id": "cbb", "tag_ids": []},
    {"name": "NCAA篮球", "name_en": "NCAA Basketball", "name_desc": "NCAA Basketball", "sport_id": "cwbb", "tag_ids": []},
    {"name": "NCAA篮球", "name_en": "NCAA Basketball", "name_desc": "NCAA Basketball", "sport_id": "ncaab", "tag_ids": ["82"]},
    {"name": "WNBA", "name_en": "WNBA", "name_desc": "WNBA", "sport_id": "wnba", "tag_ids": []},
    {"name": "FIBA", "name_en": "FIBA", "name_desc": "FIBA", "sport_id": "fiba", "tag_ids": ["98"]},
    {"name": "Liga ACB", "name_en": "Liga ACB", "name_desc": "Liga ACB", "sport_id": "bkligend", "tag_ids": []},
    {"name": "NBL", "name_en": "NBL", "name_desc": "NBL", "sport_id": "bknbl", "tag_ids": []},
    {"name": "Basketball Champions League", "name_en": "Basketball Champions League", "name_desc": "Basketball Champions League", "sport_id": "bkcl", "tag_ids": []},
    {"name": "Lega Basket Serie A", "name_en": "Lega Basket Serie A", "name_desc": "Lega Basket Serie A", "sport_id": "bkseriea", "tag_ids": []},
    {"name": "CBA", "name_en": "CBA", "name_desc": "CBA", "sport_id": "bkcba", "tag_ids": []},
    {"name": "Betclic Élite", "name_en": "Betclic Élite", "name_desc": "Betclic Élite", "sport_id": "bkfr1", "tag_ids": []},
    {"name": "Liga Nacional de Básquet", "name_en": "Liga Nacional de Básquet", "name_desc": "Liga Nacional de Básquet", "sport_id": "bkarg", "tag_ids": []},
    {"name": "KBL", "name_en": "KBL", "name_desc": "KBL", "sport_id": "bkkbl", "tag_ids": []},
    {"name": "EuroLeague", "name_en": "EuroLeague", "name_desc": "EuroLeague", "sport_id": "euroleague", "tag_ids": []},
    {"name": "EuroCup", "name_en": "EuroCup", "name_desc": "EuroCup", "sport_id": "bkeurocup", "tag_ids": []},
    {"name": "B.League", "name_en": "B.League", "name_desc": "B.League", "sport_id": "bkjpn", "tag_ids": []},
    {"name": "NBB", "name_en": "NBB", "name_desc": "NBB", "sport_id": "bknbb", "tag_ids": []},
    {"name": "VTB United League", "name_en": "VTB United League", "name_desc": "VTB United League", "sport_id": "bkvtb", "tag_ids": []},
    {"name": "ABA League", "name_en": "ABA League", "name_desc": "ABA League", "sport_id": "bkaba", "tag_ids": []},
    {"name": "Liga Argentina", "name_en": "Liga Argentina", "name_desc": "Liga Argentina", "sport_id": "bkligarg", "tag_ids": []},
    {"name": "LKL", "name_en": "LKL", "name_desc": "LKL", "sport_id": "bklkl", "tag_ids": []},
    {"name": "Basketball Bundesliga", "name_en": "Basketball Bundesliga", "name_desc": "Basketball Bundesliga", "sport_id": "bkbbl", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT' AND `sub_category` = 'Basketball';


UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'euroleague-%' OR LOWER(`slug`) = 'euroleague' THEN 'EuroLeague'
    WHEN LOWER(`slug`) LIKE 'bkeurocup-%' OR LOWER(`slug`) = 'bkeurocup' THEN 'EuroCup'
    WHEN LOWER(`slug`) LIKE 'bkligend-%' OR LOWER(`slug`) = 'bkligend' THEN 'Liga ACB'
    WHEN LOWER(`slug`) LIKE 'bkseriea-%' OR LOWER(`slug`) = 'bkseriea' THEN 'Lega Basket Serie A'
    WHEN LOWER(`slug`) LIKE 'bkligarg-%' OR LOWER(`slug`) = 'bkligarg' THEN 'Liga Argentina'
    WHEN LOWER(`slug`) LIKE 'ncaab-%' OR LOWER(`slug`) = 'ncaab' THEN 'NCAA篮球'
    WHEN LOWER(`slug`) LIKE 'bknbl-%' OR LOWER(`slug`) = 'bknbl' THEN 'NBL'
    WHEN LOWER(`slug`) LIKE 'bkcba-%' OR LOWER(`slug`) = 'bkcba' THEN 'CBA'
    WHEN LOWER(`slug`) LIKE 'bkfr1-%' OR LOWER(`slug`) = 'bkfr1' THEN 'Betclic Élite'
    WHEN LOWER(`slug`) LIKE 'bkarg-%' OR LOWER(`slug`) = 'bkarg' THEN 'Liga Nacional de Básquet'
    WHEN LOWER(`slug`) LIKE 'bkkbl-%' OR LOWER(`slug`) = 'bkkbl' THEN 'KBL'
    WHEN LOWER(`slug`) LIKE 'bkjpn-%' OR LOWER(`slug`) = 'bkjpn' THEN 'B.League'
    WHEN LOWER(`slug`) LIKE 'bknbb-%' OR LOWER(`slug`) = 'bknbb' THEN 'NBB'
    WHEN LOWER(`slug`) LIKE 'bkvtb-%' OR LOWER(`slug`) = 'bkvtb' THEN 'VTB United League'
    WHEN LOWER(`slug`) LIKE 'bkaba-%' OR LOWER(`slug`) = 'bkaba' THEN 'ABA League'
    WHEN LOWER(`slug`) LIKE 'bklkl-%' OR LOWER(`slug`) = 'bklkl' THEN 'LKL'
    WHEN LOWER(`slug`) LIKE 'bkbbl-%' OR LOWER(`slug`) = 'bkbbl' THEN 'Basketball Bundesliga'
    WHEN LOWER(`slug`) LIKE 'cwbb-%' OR LOWER(`slug`) = 'cwbb' THEN 'NCAA篮球'
    WHEN LOWER(`slug`) LIKE 'wnba-%' OR LOWER(`slug`) = 'wnba' THEN 'WNBA'
    WHEN LOWER(`slug`) LIKE 'fiba-%' OR LOWER(`slug`) = 'fiba' THEN 'FIBA'
    WHEN LOWER(`slug`) LIKE 'bkcl-%' OR LOWER(`slug`) = 'bkcl' THEN 'Basketball Champions League'
    WHEN LOWER(`slug`) LIKE 'nba-%' OR LOWER(`slug`) = 'nba' THEN 'NBA'
    WHEN LOWER(`slug`) LIKE 'cbb-%' OR LOWER(`slug`) = 'cbb' THEN 'NCAA篮球'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'euroleague-%' OR LOWER(`slug`) = 'euroleague' THEN 'EuroLeague'
    WHEN LOWER(`slug`) LIKE 'bkeurocup-%' OR LOWER(`slug`) = 'bkeurocup' THEN 'EuroCup'
    WHEN LOWER(`slug`) LIKE 'bkligend-%' OR LOWER(`slug`) = 'bkligend' THEN 'Liga ACB'
    WHEN LOWER(`slug`) LIKE 'bkseriea-%' OR LOWER(`slug`) = 'bkseriea' THEN 'Lega Basket Serie A'
    WHEN LOWER(`slug`) LIKE 'bkligarg-%' OR LOWER(`slug`) = 'bkligarg' THEN 'Liga Argentina'
    WHEN LOWER(`slug`) LIKE 'ncaab-%' OR LOWER(`slug`) = 'ncaab' THEN 'NCAA Basketball'
    WHEN LOWER(`slug`) LIKE 'bknbl-%' OR LOWER(`slug`) = 'bknbl' THEN 'NBL'
    WHEN LOWER(`slug`) LIKE 'bkcba-%' OR LOWER(`slug`) = 'bkcba' THEN 'CBA'
    WHEN LOWER(`slug`) LIKE 'bkfr1-%' OR LOWER(`slug`) = 'bkfr1' THEN 'Betclic Élite'
    WHEN LOWER(`slug`) LIKE 'bkarg-%' OR LOWER(`slug`) = 'bkarg' THEN 'Liga Nacional de Básquet'
    WHEN LOWER(`slug`) LIKE 'bkkbl-%' OR LOWER(`slug`) = 'bkkbl' THEN 'KBL'
    WHEN LOWER(`slug`) LIKE 'bkjpn-%' OR LOWER(`slug`) = 'bkjpn' THEN 'B.League'
    WHEN LOWER(`slug`) LIKE 'bknbb-%' OR LOWER(`slug`) = 'bknbb' THEN 'NBB'
    WHEN LOWER(`slug`) LIKE 'bkvtb-%' OR LOWER(`slug`) = 'bkvtb' THEN 'VTB United League'
    WHEN LOWER(`slug`) LIKE 'bkaba-%' OR LOWER(`slug`) = 'bkaba' THEN 'ABA League'
    WHEN LOWER(`slug`) LIKE 'bklkl-%' OR LOWER(`slug`) = 'bklkl' THEN 'LKL'
    WHEN LOWER(`slug`) LIKE 'bkbbl-%' OR LOWER(`slug`) = 'bkbbl' THEN 'Basketball Bundesliga'
    WHEN LOWER(`slug`) LIKE 'cwbb-%' OR LOWER(`slug`) = 'cwbb' THEN 'NCAA Basketball'
    WHEN LOWER(`slug`) LIKE 'wnba-%' OR LOWER(`slug`) = 'wnba' THEN 'WNBA'
    WHEN LOWER(`slug`) LIKE 'fiba-%' OR LOWER(`slug`) = 'fiba' THEN 'FIBA'
    WHEN LOWER(`slug`) LIKE 'bkcl-%' OR LOWER(`slug`) = 'bkcl' THEN 'Basketball Champions League'
    WHEN LOWER(`slug`) LIKE 'nba-%' OR LOWER(`slug`) = 'nba' THEN 'NBA'
    WHEN LOWER(`slug`) LIKE 'cbb-%' OR LOWER(`slug`) = 'cbb' THEN 'NCAA Basketball'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Basketball'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== NHL：Poly 7 联赛词典（仅 nhl 保留 tag 79）==========
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "冰球",
  "sub_category": "NHL",
  "sport_en": "Ice Hockey",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "items": [
    {"name": "NHL", "name_en": "NHL", "name_desc": "NHL", "sport_id": "nhl", "tag_ids": ["79"]},
    {"name": "AHL", "name_en": "AHL", "name_desc": "AHL", "sport_id": "ahl", "tag_ids": []},
    {"name": "SHL", "name_en": "SHL", "name_desc": "SHL", "sport_id": "shl", "tag_ids": []},
    {"name": "Czech Extraliga", "name_en": "Czech Extraliga", "name_desc": "Czech Extraliga", "sport_id": "cehl", "tag_ids": []},
    {"name": "DEL", "name_en": "DEL", "name_desc": "DEL", "sport_id": "dehl", "tag_ids": []},
    {"name": "National League", "name_en": "National League", "name_desc": "National League", "sport_id": "snhl", "tag_ids": []},
    {"name": "KHL", "name_en": "KHL", "name_desc": "KHL", "sport_id": "khl", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT' AND `sub_category` = 'NHL';


UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'cehl-%' OR LOWER(`slug`) = 'cehl' THEN 'Czech Extraliga'
    WHEN LOWER(`slug`) LIKE 'dehl-%' OR LOWER(`slug`) = 'dehl' THEN 'DEL'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN 'National League'
    WHEN LOWER(`slug`) LIKE 'nhl-%' OR LOWER(`slug`) = 'nhl' THEN 'NHL'
    WHEN LOWER(`slug`) LIKE 'ahl-%' OR LOWER(`slug`) = 'ahl' THEN 'AHL'
    WHEN LOWER(`slug`) LIKE 'shl-%' OR LOWER(`slug`) = 'shl' THEN 'SHL'
    WHEN LOWER(`slug`) LIKE 'khl-%' OR LOWER(`slug`) = 'khl' THEN 'KHL'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'cehl-%' OR LOWER(`slug`) = 'cehl' THEN 'Czech Extraliga'
    WHEN LOWER(`slug`) LIKE 'dehl-%' OR LOWER(`slug`) = 'dehl' THEN 'DEL'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN 'National League'
    WHEN LOWER(`slug`) LIKE 'nhl-%' OR LOWER(`slug`) = 'nhl' THEN 'NHL'
    WHEN LOWER(`slug`) LIKE 'ahl-%' OR LOWER(`slug`) = 'ahl' THEN 'AHL'
    WHEN LOWER(`slug`) LIKE 'shl-%' OR LOWER(`slug`) = 'shl' THEN 'SHL'
    WHEN LOWER(`slug`) LIKE 'khl-%' OR LOWER(`slug`) = 'khl' THEN 'KHL'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'NHL'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== NFL：补 cfb（College Football）==========
UPDATE `market_category`
SET `config_json` = '{
  "sport_cn": "美式足球",
  "sub_category": "NFL",
  "sport_en": "American Football",
  "market_variants": ["SPORTS_MATCH", "SPORTS_PROPS"],
  "items": [
    {"name": "NFL", "name_en": "NFL", "name_desc": "NFL", "sport_id": "nfl", "tag_ids": ["45"]},
    {"name": "NCAA橄榄球", "name_en": "College Football", "name_desc": "College Football", "sport_id": "cfb", "tag_ids": []}
  ]
}',
    `updated_at` = NOW()
WHERE `platform` = 'PREDICT' AND `sub_category` = 'NFL';


UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'nfl-%' OR LOWER(`slug`) = 'nfl' THEN 'NFL'
    WHEN LOWER(`slug`) LIKE 'cfb-%' OR LOWER(`slug`) = 'cfb' THEN 'NCAA橄榄球'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'nfl-%' OR LOWER(`slug`) = 'nfl' THEN 'NFL'
    WHEN LOWER(`slug`) LIKE 'cfb-%' OR LOWER(`slug`) = 'cfb' THEN 'College Football'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'NFL'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== Football：config 已齐，仅存量回填 ==========

UPDATE `market`
SET
  `competition` = CASE
    WHEN LOWER(`slug`) LIKE 'argcopa-%' OR LOWER(`slug`) = 'argcopa' THEN '阿根廷杯'
    WHEN LOWER(`slug`) LIKE 'copaam-%' OR LOWER(`slug`) = 'copaam' THEN '美洲杯'
    WHEN LOWER(`slug`) LIKE 'argpn-%' OR LOWER(`slug`) = 'argpn' THEN '阿根廷全国联赛'
    WHEN LOWER(`slug`) LIKE 'afcq-%' OR LOWER(`slug`) = 'afcq' THEN '非洲杯预选赛'
    WHEN LOWER(`slug`) LIKE 'bra2-%' OR LOWER(`slug`) = 'bra2' THEN '巴乙'
    WHEN LOWER(`slug`) LIKE 'chl2-%' OR LOWER(`slug`) = 'chl2' THEN '智利乙'
    WHEN LOWER(`slug`) LIKE 'mex2-%' OR LOWER(`slug`) = 'mex2' THEN '墨扩军联赛'
    WHEN LOWER(`slug`) LIKE 'chi1-%' OR LOWER(`slug`) = 'chi1' THEN '智利甲'
    WHEN LOWER(`slug`) LIKE 'nor2-%' OR LOWER(`slug`) = 'nor2' THEN '挪威甲'
    WHEN LOWER(`slug`) LIKE 'conl-%' OR LOWER(`slug`) = 'conl' THEN '中北美国家联赛'
    WHEN LOWER(`slug`) LIKE 'serc-%' OR LOWER(`slug`) = 'serc' THEN '意丙'
    WHEN LOWER(`slug`) LIKE 'itsb-%' OR LOWER(`slug`) = 'itsb' THEN '意乙'
    WHEN LOWER(`slug`) LIKE 'scop-%' OR LOWER(`slug`) = 'scop' THEN '苏超'
    WHEN LOWER(`slug`) LIKE 'wsl2-%' OR LOWER(`slug`) = 'wsl2' THEN '英格兰女足甲级'
    WHEN LOWER(`slug`) LIKE 'uwcl-%' OR LOWER(`slug`) = 'uwcl' THEN '女足欧冠'
    WHEN LOWER(`slug`) LIKE 'uslc-%' OR LOWER(`slug`) = 'uslc' THEN '美国足球冠军联赛'
    WHEN LOWER(`slug`) LIKE 'ukr1-%' OR LOWER(`slug`) = 'ukr1' THEN '乌克兰超'
    WHEN LOWER(`slug`) LIKE 'uru1-%' OR LOWER(`slug`) = 'uru1' THEN '乌拉圭甲'
    WHEN LOWER(`slug`) LIKE 'uzb1-%' OR LOWER(`slug`) = 'uzb1' THEN '乌兹别克超'
    WHEN LOWER(`slug`) LIKE 'lva1-%' OR LOWER(`slug`) = 'lva1' THEN '拉脱维亚超'
    WHEN LOWER(`slug`) LIKE 'bel1-%' OR LOWER(`slug`) = 'bel1' THEN '比甲'
    WHEN LOWER(`slug`) LIKE 'bol1-%' OR LOWER(`slug`) = 'bol1' THEN '玻利维亚甲'
    WHEN LOWER(`slug`) LIKE 'cze1-%' OR LOWER(`slug`) = 'cze1' THEN '捷克联赛'
    WHEN LOWER(`slug`) LIKE 'egy1-%' OR LOWER(`slug`) = 'egy1' THEN '埃及超'
    WHEN LOWER(`slug`) LIKE 'ned2-%' OR LOWER(`slug`) = 'ned2' THEN '荷乙'
    WHEN LOWER(`slug`) LIKE 'nwsl-%' OR LOWER(`slug`) = 'nwsl' THEN '美国女足联赛'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN '英格兰国家联赛'
    WHEN LOWER(`slug`) LIKE 'svk1-%' OR LOWER(`slug`) = 'svk1' THEN '斯洛伐克超'
    WHEN LOWER(`slug`) LIKE 'par1-%' OR LOWER(`slug`) = 'par1' THEN '巴拉圭甲'
    WHEN LOWER(`slug`) LIKE 'per1-%' OR LOWER(`slug`) = 'per1' THEN '秘鲁甲'
    WHEN LOWER(`slug`) LIKE 'rou1-%' OR LOWER(`slug`) = 'rou1' THEN '罗马尼亚超'
    WHEN LOWER(`slug`) LIKE 'uae1-%' OR LOWER(`slug`) = 'uae1' THEN '阿联酋超'
    WHEN LOWER(`slug`) LIKE 'acle-%' OR LOWER(`slug`) = 'acle' THEN '亚冠'
    WHEN LOWER(`slug`) LIKE 'col1-%' OR LOWER(`slug`) = 'col1' THEN '哥伦甲'
    WHEN LOWER(`slug`) LIKE 'idn1-%' OR LOWER(`slug`) = 'idn1' THEN '印尼联'
    WHEN LOWER(`slug`) LIKE 'ncag-%' OR LOWER(`slug`) = 'ncag' THEN '金杯赛'
    WHEN LOWER(`slug`) LIKE 'es3-%' OR LOWER(`slug`) = 'es3' THEN '西协甲'
    WHEN LOWER(`slug`) LIKE 'elc-%' OR LOWER(`slug`) = 'elc' THEN '英冠'
    WHEN LOWER(`slug`) LIKE 'el1-%' OR LOWER(`slug`) = 'el1' THEN '英甲'
    WHEN LOWER(`slug`) LIKE 'el2-%' OR LOWER(`slug`) = 'el2' THEN '英乙'
    WHEN LOWER(`slug`) LIKE 'es2-%' OR LOWER(`slug`) = 'es2' THEN '西乙'
    WHEN LOWER(`slug`) LIKE 'fr2-%' OR LOWER(`slug`) = 'fr2' THEN '法乙'
    WHEN LOWER(`slug`) LIKE 'fpd-%' OR LOWER(`slug`) = 'fpd' THEN '哥斯达黎加甲'
    WHEN LOWER(`slug`) LIKE 'gtm-%' OR LOWER(`slug`) = 'gtm' THEN '危地马拉甲'
    WHEN LOWER(`slug`) LIKE 'pt2-%' OR LOWER(`slug`) = 'pt2' THEN '葡甲'
    WHEN LOWER(`slug`) LIKE 'ptc-%' OR LOWER(`slug`) = 'ptc' THEN '葡萄牙杯'
    WHEN LOWER(`slug`) LIKE 'bul-%' OR LOWER(`slug`) = 'bul' THEN '保加利亚超'
    WHEN LOWER(`slug`) LIKE 'fif-%' OR LOWER(`slug`) = 'fif' THEN '国际足球'
    WHEN LOWER(`slug`) LIKE 'ja3-%' OR LOWER(`slug`) = 'ja3' THEN 'J3联赛'
    WHEN LOWER(`slug`) LIKE 'isp-%' OR LOWER(`slug`) = 'isp' THEN '印度超'
    WHEN LOWER(`slug`) LIKE 'clf-%' OR LOWER(`slug`) = 'clf' THEN '俱乐部友谊赛'
    WHEN LOWER(`slug`) LIKE 'bl2-%' OR LOWER(`slug`) = 'bl2' THEN '德乙'
    WHEN LOWER(`slug`) LIKE 'bl3-%' OR LOWER(`slug`) = 'bl3' THEN '德丙'
    WHEN LOWER(`slug`) LIKE 'wsl-%' OR LOWER(`slug`) = 'wsl' THEN '英格兰女足超'
    WHEN LOWER(`slug`) LIKE 'hr1-%' OR LOWER(`slug`) = 'hr1' THEN '克罗地亚甲'
    WHEN LOWER(`slug`) LIKE 'epl-%' OR LOWER(`slug`) = 'epl' THEN '英超'
    WHEN LOWER(`slug`) LIKE 'ucl-%' OR LOWER(`slug`) = 'ucl' THEN '欧冠'
    WHEN LOWER(`slug`) LIKE 'lal-%' OR LOWER(`slug`) = 'lal' THEN '西甲'
    WHEN LOWER(`slug`) LIKE 'bun-%' OR LOWER(`slug`) = 'bun' THEN '德甲'
    WHEN LOWER(`slug`) LIKE 'sea-%' OR LOWER(`slug`) = 'sea' THEN '意甲'
    WHEN LOWER(`slug`) LIKE 'fl1-%' OR LOWER(`slug`) = 'fl1' THEN '法甲'
    WHEN LOWER(`slug`) LIKE 'uel-%' OR LOWER(`slug`) = 'uel' THEN '欧联'
    WHEN LOWER(`slug`) LIKE 'ere-%' OR LOWER(`slug`) = 'ere' THEN '荷甲'
    WHEN LOWER(`slug`) LIKE 'por-%' OR LOWER(`slug`) = 'por' THEN '葡超'
    WHEN LOWER(`slug`) LIKE 'tur-%' OR LOWER(`slug`) = 'tur' THEN '土超'
    WHEN LOWER(`slug`) LIKE 'rus-%' OR LOWER(`slug`) = 'rus' THEN '俄超'
    WHEN LOWER(`slug`) LIKE 'mls-%' OR LOWER(`slug`) = 'mls' THEN '美职联'
    WHEN LOWER(`slug`) LIKE 'mex-%' OR LOWER(`slug`) = 'mex' THEN '墨超'
    WHEN LOWER(`slug`) LIKE 'bra-%' OR LOWER(`slug`) = 'bra' THEN '巴甲'
    WHEN LOWER(`slug`) LIKE 'arg-%' OR LOWER(`slug`) = 'arg' THEN '阿甲'
    WHEN LOWER(`slug`) LIKE 'spl-%' OR LOWER(`slug`) = 'spl' THEN '沙特联'
    WHEN LOWER(`slug`) LIKE 'kor-%' OR LOWER(`slug`) = 'kor' THEN 'K联赛'
    WHEN LOWER(`slug`) LIKE 'jap-%' OR LOWER(`slug`) = 'jap' THEN 'J1'
    WHEN LOWER(`slug`) LIKE 'ja2-%' OR LOWER(`slug`) = 'ja2' THEN 'J2'
    WHEN LOWER(`slug`) LIKE 'aus-%' OR LOWER(`slug`) = 'aus' THEN '澳超'
    WHEN LOWER(`slug`) LIKE 'nor-%' OR LOWER(`slug`) = 'nor' THEN '挪超'
    WHEN LOWER(`slug`) LIKE 'den-%' OR LOWER(`slug`) = 'den' THEN '丹超'
    WHEN LOWER(`slug`) LIKE 'unl-%' OR LOWER(`slug`) = 'unl' THEN '欧国联'
    WHEN LOWER(`slug`) LIKE 'lib-%' OR LOWER(`slug`) = 'lib' THEN '解放者杯'
    WHEN LOWER(`slug`) LIKE 'chi-%' OR LOWER(`slug`) = 'chi' THEN '中超'
    WHEN LOWER(`slug`) LIKE 'sud-%' OR LOWER(`slug`) = 'sud' THEN '南美杯'
    WHEN LOWER(`slug`) LIKE 'cdr-%' OR LOWER(`slug`) = 'cdr' THEN '国王杯'
    WHEN LOWER(`slug`) LIKE 'cde-%' OR LOWER(`slug`) = 'cde' THEN '法国杯'
    WHEN LOWER(`slug`) LIKE 'dfb-%' OR LOWER(`slug`) = 'dfb' THEN '德国杯'
    WHEN LOWER(`slug`) LIKE 'itc-%' OR LOWER(`slug`) = 'itc' THEN '意大利杯'
    WHEN LOWER(`slug`) LIKE 'efa-%' OR LOWER(`slug`) = 'efa' THEN '英格兰足总杯'
    WHEN LOWER(`slug`) LIKE 'efl-%' OR LOWER(`slug`) = 'efl' THEN '英格兰联赛杯'
    WHEN LOWER(`slug`) LIKE 'ssc-%' OR LOWER(`slug`) = 'ssc' THEN '西班牙超级杯'
    WHEN LOWER(`slug`) LIKE 'acn-%' OR LOWER(`slug`) = 'acn' THEN '非洲国家杯'
    WHEN LOWER(`slug`) LIKE 'col-%' OR LOWER(`slug`) = 'col' THEN '欧协联'
    WHEN LOWER(`slug`) LIKE 'pol-%' OR LOWER(`slug`) = 'pol' THEN '波兰超'
    ELSE `competition`
  END,
  `competition_en` = CASE
    WHEN LOWER(`slug`) LIKE 'argcopa-%' OR LOWER(`slug`) = 'argcopa' THEN 'Copa Argentina'
    WHEN LOWER(`slug`) LIKE 'copaam-%' OR LOWER(`slug`) = 'copaam' THEN 'Copa America'
    WHEN LOWER(`slug`) LIKE 'argpn-%' OR LOWER(`slug`) = 'argpn' THEN 'Primera Nacional'
    WHEN LOWER(`slug`) LIKE 'afcq-%' OR LOWER(`slug`) = 'afcq' THEN 'Africa Cup of Nations Qualifiers'
    WHEN LOWER(`slug`) LIKE 'bra2-%' OR LOWER(`slug`) = 'bra2' THEN 'Brasileirão Série B'
    WHEN LOWER(`slug`) LIKE 'chl2-%' OR LOWER(`slug`) = 'chl2' THEN 'Primera B (Chile)'
    WHEN LOWER(`slug`) LIKE 'mex2-%' OR LOWER(`slug`) = 'mex2' THEN 'Liga de Expansión MX'
    WHEN LOWER(`slug`) LIKE 'chi1-%' OR LOWER(`slug`) = 'chi1' THEN 'Liga de Primera'
    WHEN LOWER(`slug`) LIKE 'nor2-%' OR LOWER(`slug`) = 'nor2' THEN 'OBOS-ligaen'
    WHEN LOWER(`slug`) LIKE 'conl-%' OR LOWER(`slug`) = 'conl' THEN 'CONCACAF Nations League'
    WHEN LOWER(`slug`) LIKE 'serc-%' OR LOWER(`slug`) = 'serc' THEN 'Serie C'
    WHEN LOWER(`slug`) LIKE 'itsb-%' OR LOWER(`slug`) = 'itsb' THEN 'Serie B'
    WHEN LOWER(`slug`) LIKE 'scop-%' OR LOWER(`slug`) = 'scop' THEN 'Scottish Premiership'
    WHEN LOWER(`slug`) LIKE 'wsl2-%' OR LOWER(`slug`) = 'wsl2' THEN 'Women''s Super League 2'
    WHEN LOWER(`slug`) LIKE 'uwcl-%' OR LOWER(`slug`) = 'uwcl' THEN 'UEFA Women''s Champions League'
    WHEN LOWER(`slug`) LIKE 'uslc-%' OR LOWER(`slug`) = 'uslc' THEN 'USL Championship'
    WHEN LOWER(`slug`) LIKE 'ukr1-%' OR LOWER(`slug`) = 'ukr1' THEN 'Ukrainian Premier League'
    WHEN LOWER(`slug`) LIKE 'uru1-%' OR LOWER(`slug`) = 'uru1' THEN 'Uruguayan Primera División'
    WHEN LOWER(`slug`) LIKE 'uzb1-%' OR LOWER(`slug`) = 'uzb1' THEN 'Uzbekistan Super League'
    WHEN LOWER(`slug`) LIKE 'lva1-%' OR LOWER(`slug`) = 'lva1' THEN 'Virslīga'
    WHEN LOWER(`slug`) LIKE 'bel1-%' OR LOWER(`slug`) = 'bel1' THEN 'Belgium Pro League'
    WHEN LOWER(`slug`) LIKE 'bol1-%' OR LOWER(`slug`) = 'bol1' THEN 'Bolivian Primera División'
    WHEN LOWER(`slug`) LIKE 'cze1-%' OR LOWER(`slug`) = 'cze1' THEN 'Chance Liga'
    WHEN LOWER(`slug`) LIKE 'egy1-%' OR LOWER(`slug`) = 'egy1' THEN 'Egyptian Premier League'
    WHEN LOWER(`slug`) LIKE 'ned2-%' OR LOWER(`slug`) = 'ned2' THEN 'Eerste Divisie'
    WHEN LOWER(`slug`) LIKE 'nwsl-%' OR LOWER(`slug`) = 'nwsl' THEN 'NWSL'
    WHEN LOWER(`slug`) LIKE 'snhl-%' OR LOWER(`slug`) = 'snhl' THEN 'National League'
    WHEN LOWER(`slug`) LIKE 'svk1-%' OR LOWER(`slug`) = 'svk1' THEN 'Niké liga'
    WHEN LOWER(`slug`) LIKE 'par1-%' OR LOWER(`slug`) = 'par1' THEN 'Paraguay Primera División'
    WHEN LOWER(`slug`) LIKE 'per1-%' OR LOWER(`slug`) = 'per1' THEN 'Liga 1'
    WHEN LOWER(`slug`) LIKE 'rou1-%' OR LOWER(`slug`) = 'rou1' THEN 'SuperLiga'
    WHEN LOWER(`slug`) LIKE 'uae1-%' OR LOWER(`slug`) = 'uae1' THEN 'UAE Pro League'
    WHEN LOWER(`slug`) LIKE 'acle-%' OR LOWER(`slug`) = 'acle' THEN 'AFC Champions League Elite'
    WHEN LOWER(`slug`) LIKE 'col1-%' OR LOWER(`slug`) = 'col1' THEN 'Categoría Primera A'
    WHEN LOWER(`slug`) LIKE 'idn1-%' OR LOWER(`slug`) = 'idn1' THEN 'Indonesia Liga 1'
    WHEN LOWER(`slug`) LIKE 'ncag-%' OR LOWER(`slug`) = 'ncag' THEN 'CONCACAF Gold Cup'
    WHEN LOWER(`slug`) LIKE 'es3-%' OR LOWER(`slug`) = 'es3' THEN 'Primera Federación'
    WHEN LOWER(`slug`) LIKE 'elc-%' OR LOWER(`slug`) = 'elc' THEN 'EFL Championship'
    WHEN LOWER(`slug`) LIKE 'el1-%' OR LOWER(`slug`) = 'el1' THEN 'League One'
    WHEN LOWER(`slug`) LIKE 'el2-%' OR LOWER(`slug`) = 'el2' THEN 'League Two'
    WHEN LOWER(`slug`) LIKE 'es2-%' OR LOWER(`slug`) = 'es2' THEN 'LaLiga2'
    WHEN LOWER(`slug`) LIKE 'fr2-%' OR LOWER(`slug`) = 'fr2' THEN 'Ligue 2'
    WHEN LOWER(`slug`) LIKE 'fpd-%' OR LOWER(`slug`) = 'fpd' THEN 'Liga FPD'
    WHEN LOWER(`slug`) LIKE 'gtm-%' OR LOWER(`slug`) = 'gtm' THEN 'Liga Nacional de Guatemala'
    WHEN LOWER(`slug`) LIKE 'pt2-%' OR LOWER(`slug`) = 'pt2' THEN 'Liga Portugal 2'
    WHEN LOWER(`slug`) LIKE 'ptc-%' OR LOWER(`slug`) = 'ptc' THEN 'Taça de Portugal'
    WHEN LOWER(`slug`) LIKE 'bul-%' OR LOWER(`slug`) = 'bul' THEN 'efbet Liga'
    WHEN LOWER(`slug`) LIKE 'fif-%' OR LOWER(`slug`) = 'fif' THEN 'FIFA Friendlies'
    WHEN LOWER(`slug`) LIKE 'ja3-%' OR LOWER(`slug`) = 'ja3' THEN 'J3 League'
    WHEN LOWER(`slug`) LIKE 'isp-%' OR LOWER(`slug`) = 'isp' THEN 'Indian Super League'
    WHEN LOWER(`slug`) LIKE 'clf-%' OR LOWER(`slug`) = 'clf' THEN 'Club Friendlies'
    WHEN LOWER(`slug`) LIKE 'bl2-%' OR LOWER(`slug`) = 'bl2' THEN '2. Bundesliga'
    WHEN LOWER(`slug`) LIKE 'bl3-%' OR LOWER(`slug`) = 'bl3' THEN '3. Liga'
    WHEN LOWER(`slug`) LIKE 'wsl-%' OR LOWER(`slug`) = 'wsl' THEN 'Women''s Super League'
    WHEN LOWER(`slug`) LIKE 'hr1-%' OR LOWER(`slug`) = 'hr1' THEN 'Prva Liga'
    WHEN LOWER(`slug`) LIKE 'epl-%' OR LOWER(`slug`) = 'epl' THEN 'Premier League'
    WHEN LOWER(`slug`) LIKE 'ucl-%' OR LOWER(`slug`) = 'ucl' THEN 'UEFA Champions League'
    WHEN LOWER(`slug`) LIKE 'lal-%' OR LOWER(`slug`) = 'lal' THEN 'LaLiga'
    WHEN LOWER(`slug`) LIKE 'bun-%' OR LOWER(`slug`) = 'bun' THEN 'Bundesliga'
    WHEN LOWER(`slug`) LIKE 'sea-%' OR LOWER(`slug`) = 'sea' THEN 'Serie A'
    WHEN LOWER(`slug`) LIKE 'fl1-%' OR LOWER(`slug`) = 'fl1' THEN 'Ligue 1'
    WHEN LOWER(`slug`) LIKE 'uel-%' OR LOWER(`slug`) = 'uel' THEN 'UEFA Europa League'
    WHEN LOWER(`slug`) LIKE 'ere-%' OR LOWER(`slug`) = 'ere' THEN 'Eredivisie'
    WHEN LOWER(`slug`) LIKE 'por-%' OR LOWER(`slug`) = 'por' THEN 'Primeira Liga'
    WHEN LOWER(`slug`) LIKE 'tur-%' OR LOWER(`slug`) = 'tur' THEN 'Süper Lig'
    WHEN LOWER(`slug`) LIKE 'rus-%' OR LOWER(`slug`) = 'rus' THEN 'Russian Premier League'
    WHEN LOWER(`slug`) LIKE 'mls-%' OR LOWER(`slug`) = 'mls' THEN 'MLS'
    WHEN LOWER(`slug`) LIKE 'mex-%' OR LOWER(`slug`) = 'mex' THEN 'Liga MX'
    WHEN LOWER(`slug`) LIKE 'bra-%' OR LOWER(`slug`) = 'bra' THEN 'Brasileirão Série A'
    WHEN LOWER(`slug`) LIKE 'arg-%' OR LOWER(`slug`) = 'arg' THEN 'Liga Profesional de Fútbol'
    WHEN LOWER(`slug`) LIKE 'spl-%' OR LOWER(`slug`) = 'spl' THEN 'Saudi Pro League'
    WHEN LOWER(`slug`) LIKE 'kor-%' OR LOWER(`slug`) = 'kor' THEN 'K League 1'
    WHEN LOWER(`slug`) LIKE 'jap-%' OR LOWER(`slug`) = 'jap' THEN 'J1 League'
    WHEN LOWER(`slug`) LIKE 'ja2-%' OR LOWER(`slug`) = 'ja2' THEN 'J2 League'
    WHEN LOWER(`slug`) LIKE 'aus-%' OR LOWER(`slug`) = 'aus' THEN 'A-League Men'
    WHEN LOWER(`slug`) LIKE 'nor-%' OR LOWER(`slug`) = 'nor' THEN 'Eliteserien'
    WHEN LOWER(`slug`) LIKE 'den-%' OR LOWER(`slug`) = 'den' THEN 'Danish Superliga'
    WHEN LOWER(`slug`) LIKE 'unl-%' OR LOWER(`slug`) = 'unl' THEN 'UEFA Nations League'
    WHEN LOWER(`slug`) LIKE 'lib-%' OR LOWER(`slug`) = 'lib' THEN 'Copa Libertadores'
    WHEN LOWER(`slug`) LIKE 'chi-%' OR LOWER(`slug`) = 'chi' THEN 'Chinese Super League'
    WHEN LOWER(`slug`) LIKE 'sud-%' OR LOWER(`slug`) = 'sud' THEN 'Copa Sudamericana'
    WHEN LOWER(`slug`) LIKE 'cdr-%' OR LOWER(`slug`) = 'cdr' THEN 'Copa del Rey'
    WHEN LOWER(`slug`) LIKE 'cde-%' OR LOWER(`slug`) = 'cde' THEN 'Coupe de France'
    WHEN LOWER(`slug`) LIKE 'dfb-%' OR LOWER(`slug`) = 'dfb' THEN 'DFB-Pokal'
    WHEN LOWER(`slug`) LIKE 'itc-%' OR LOWER(`slug`) = 'itc' THEN 'Coppa Italia'
    WHEN LOWER(`slug`) LIKE 'efa-%' OR LOWER(`slug`) = 'efa' THEN 'FA Cup'
    WHEN LOWER(`slug`) LIKE 'efl-%' OR LOWER(`slug`) = 'efl' THEN 'EFL CUP'
    WHEN LOWER(`slug`) LIKE 'ssc-%' OR LOWER(`slug`) = 'ssc' THEN 'Supercopa de España'
    WHEN LOWER(`slug`) LIKE 'acn-%' OR LOWER(`slug`) = 'acn' THEN 'Africa Cup of Nations'
    WHEN LOWER(`slug`) LIKE 'col-%' OR LOWER(`slug`) = 'col' THEN 'UEFA Conference League'
    WHEN LOWER(`slug`) LIKE 'pol-%' OR LOWER(`slug`) = 'pol' THEN 'Ekstraklasa'
    ELSE `competition_en`
  END,
  `map_key` = NULL,
  `map_verified` = 0,
  `updated_at` = NOW()
WHERE `platform` = 'PREDICT'
  AND `sub_category` = 'Football'
  AND `slug` IS NOT NULL
  AND TRIM(`slug`) <> '';

-- ========== market_outcome ==========
UPDATE `market_outcome` AS mo
INNER JOIN `market` AS m ON m.`id` = mo.`market_row_id`
SET
  mo.`competition` = m.`competition`,
  mo.`competition_en` = m.`competition_en`,
  mo.`updated_at` = NOW()
WHERE m.`platform` = 'PREDICT'
  AND m.`sub_category` IN ('Football', 'Basketball', 'Cricket', 'Tennis', 'NFL', 'NHL', 'MMA');

-- SELECT platform, sub_category, competition_en, COUNT(*) FROM market
-- WHERE platform='PREDICT' GROUP BY 1,2,3 ORDER BY 2,3;


SET FOREIGN_KEY_CHECKS = 1;
-- done
