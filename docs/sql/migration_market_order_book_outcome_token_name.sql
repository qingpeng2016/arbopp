-- market_order_book / market_order_book_ws：盘口名与 token 展示名

ALTER TABLE `market_order_book`
  ADD COLUMN `outcome_name` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '子盘口名称 market_outcome.outcome_name' AFTER `market_name`,
  ADD COLUMN `token_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'token 展示名 Yes/No 或队名等' AFTER `outcome_name`;

ALTER TABLE `market_order_book_ws`
  ADD COLUMN `outcome_name` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '子盘口名称 market_outcome.outcome_name' AFTER `market_name`,
  ADD COLUMN `token_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'token 展示名 Yes/No 或队名等' AFTER `outcome_name`;

UPDATE `market_order_book` AS ob
INNER JOIN `market_outcome` AS mo
  ON mo.`platform` = ob.`platform`
 AND mo.`platform_market_outcome_id` = ob.`platform_market_outcome_id`
 AND mo.`category` <=> ob.`category`
 AND mo.`sub_category` <=> ob.`sub_category`
SET
  ob.`outcome_name` = mo.`outcome_name`,
  ob.`market_name` = COALESCE(mo.`market_name`, ob.`market_name`),
  ob.`token_name` = CASE
    WHEN ob.`token_id` = mo.`yes_token_id` THEN 'Yes'
    WHEN ob.`token_id` = mo.`no_token_id` THEN 'No'
    ELSE ob.`token_name`
  END
WHERE mo.`id` > 0;

UPDATE `market_order_book_ws` AS ob
INNER JOIN `market_outcome` AS mo
  ON mo.`platform` = ob.`platform`
 AND mo.`platform_market_outcome_id` = ob.`platform_market_outcome_id`
 AND mo.`category` <=> ob.`category`
 AND mo.`sub_category` <=> ob.`sub_category`
SET
  ob.`outcome_name` = mo.`outcome_name`,
  ob.`market_name` = COALESCE(mo.`market_name`, ob.`market_name`),
  ob.`token_name` = CASE
    WHEN ob.`token_id` = mo.`yes_token_id` THEN 'Yes'
    WHEN ob.`token_id` = mo.`no_token_id` THEN 'No'
    ELSE ob.`token_name`
  END
WHERE mo.`id` > 0;