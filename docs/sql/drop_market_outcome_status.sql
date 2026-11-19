-- 盘口是否有效跟着 market.status，market_outcome 不再单独记状态。
ALTER TABLE `market_outcome`
  DROP INDEX `idx_outcome_type`,
  DROP COLUMN `status`,
  ADD KEY `idx_outcome_type` (`outcome_type`);
