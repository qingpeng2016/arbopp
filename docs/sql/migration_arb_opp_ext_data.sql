-- arb_binary_opportunity：锁失败 / 下单前检查失败 / 下单现场快照
ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `ext_data` text NULL COMMENT '扩展JSON：map_key锁失败、下单前检查失败阈值快照、下单现场' AFTER `snapshot_at_ms`;
