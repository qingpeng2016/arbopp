DROP TABLE IF EXISTS `config_market_category`;
DROP TABLE IF EXISTS `market_category`;

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
	(13, 'UFC', 'MMA', 'POLYMARKET', 3, '{"sport_cn":"UFC","sub_category":"MMA","items":[{"name":"UFC","name_en":"UFC","name_desc":"UFC","sport_id":"ufc","tag_ids":[279]}]}', '["moneyline","handicap","total"]', 7, 1, NOW(), NOW()),
	(14, 'UFC', 'MMA', 'PREDICT', 2, '{"sport_cn":"UFC","sub_category":"MMA","sport_en":"UFC","market_variants":["SPORTS_MATCH","SPORTS_PROPS"],"items":[],"sync_all_sub":true,"sync_root_tag_id":"177"}', '["moneyline","handicap","total"]', 7, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
`sport_cn`=VALUES(`sport_cn`), `concurrency`=VALUES(`concurrency`), `config_json`=VALUES(`config_json`),
`outcome_types_json`=VALUES(`outcome_types_json`),
`sort_no`=VALUES(`sort_no`), `is_enabled`=1, `updated_at`=NOW();
