-- 实时机会表：记录各腿平台。已有记录可用 backfill_arb_opp_leg_platform.sql 回填。
ALTER TABLE `arb_binary_opportunity`
  ADD COLUMN `leg_1_platform` varchar(32) DEFAULT NULL COMMENT '组合腿1平台' AFTER `leg_1_info`,
  ADD COLUMN `leg_2_platform` varchar(32) DEFAULT NULL COMMENT '组合腿2平台' AFTER `leg_2_info`,
  ADD COLUMN `leg_3_platform` varchar(32) DEFAULT NULL COMMENT '组合腿3平台' AFTER `leg_3_info`;
