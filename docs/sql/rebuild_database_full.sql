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
  `yes_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0,
  `no_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0,
  `leg3_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '1X2 第三腿；二元为 0',
  `leg_1_info` mediumtext NOT NULL,
  `leg_1_platform` varchar(32) DEFAULT NULL COMMENT '组合腿1平台',
  `leg_1_order_books` mediumtext NOT NULL,
  `leg_1_best_ask` decimal(20,8) DEFAULT NULL,
  `leg_2_info` mediumtext NOT NULL,
  `leg_2_platform` varchar(32) DEFAULT NULL COMMENT '组合腿2平台',
  `leg_2_order_books` mediumtext NOT NULL,
  `leg_2_best_ask` decimal(20,8) DEFAULT NULL,
  `leg_3_info` mediumtext NOT NULL,
  `leg_3_platform` varchar(32) DEFAULT NULL COMMENT '组合腿3平台',
  `leg_3_order_books` mediumtext NOT NULL,
  `leg_3_best_ask` decimal(20,8) DEFAULT NULL,
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
  `ext_data` text NULL COMMENT '扩展JSON：map_key锁失败、下单前检查失败参数快照、下单现场',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_arb_binary_time` (`snapshot_at`),
  KEY `idx_arb_binary_map` (`map_key`,`snapshot_at`),
  KEY `idx_arb_binary_kind` (`opp_kind`,`snapshot_at`),
  KEY `idx_arb_binary_snap_ms` (`map_key`,`snapshot_at_ms`),
  KEY `idx_arb_binary_leg_outcomes` (`yes_market_outcome_id`,`no_market_outcome_id`)
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
  `yes_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0,
  `no_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0,
  `leg3_market_outcome_id` bigint unsigned NOT NULL DEFAULT 0,
  `leg_1_info` mediumtext NOT NULL,
  `leg_1_platform` varchar(32) DEFAULT NULL COMMENT '组合腿1平台',
  `leg_1_order_books` mediumtext NOT NULL,
  `leg_1_best_ask` decimal(20,8) DEFAULT NULL,
  `leg_2_info` mediumtext NOT NULL,
  `leg_2_platform` varchar(32) DEFAULT NULL COMMENT '组合腿2平台',
  `leg_2_order_books` mediumtext NOT NULL,
  `leg_2_best_ask` decimal(20,8) DEFAULT NULL,
  `leg_3_info` mediumtext NOT NULL,
  `leg_3_platform` varchar(32) DEFAULT NULL COMMENT '组合腿3平台',
  `leg_3_order_books` mediumtext NOT NULL,
  `leg_3_best_ask` decimal(20,8) DEFAULT NULL,
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
  UNIQUE KEY `uk_bt_opp_snap_legs` (`backtest_task_id`, `snapshot_at_ms`, `opp_kind`, `yes_market_outcome_id`, `no_market_outcome_id`, `leg3_market_outcome_id`)
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
  ('predict_polymarket_sxbet_slippage_bps', 0.03000000, 'DECIMAL', 'Predict+Poly+SX 绝对价格滑点（0.03：卖一 0.50→最差 0.53；键名 _bps 为历史命名）'),
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

