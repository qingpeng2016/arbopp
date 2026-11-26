-- 把卡在 NO_FILL/ERROR 的单重置为 PENDING，供 order_fill_sync 重新回查。
-- 1) Poly FAK 400：从 err_msg 抠 orderID 写入 platform_order_id
UPDATE `orders`
SET `platform_order_id` = TRIM(BOTH '"' FROM SUBSTRING_INDEX(SUBSTRING_INDEX(`err_msg`, '"orderID":"', -1), '"', 1))
WHERE `platform_order_id` = ''
  AND `err_msg` LIKE '%"orderID":"0x%'
  AND TRIM(BOTH '"' FROM SUBSTRING_INDEX(SUBSTRING_INDEX(`err_msg`, '"orderID":"', -1), '"', 1)) LIKE '0x%';

-- 2) 有平台单号、尚无成交的 NO_FILL / ERROR / 空状态 → PENDING（下次 sync 必捞）
UPDATE `orders`
SET `status` = 'PENDING',
    `updated_at` = NOW()
WHERE `platform_order_id` <> ''
  AND `fill_qty` = 0
  AND UPPER(TRIM(`status`)) IN ('NO_FILL', 'ERROR', '', 'PENDING');

-- 3) 查询待回查
SELECT `id`, `platform`, `platform_order_id`, `status`, `fill_qty`, `fee`, `err_msg`, `updated_at`
FROM `orders`
WHERE `platform_order_id` <> ''
  AND UPPER(TRIM(`status`)) = 'PENDING'
ORDER BY `id` ASC;
