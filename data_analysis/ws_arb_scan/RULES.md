# 套利判定（本工具自实现，不 import 策略包）

对照 `strategy_entry` 行为手写等价逻辑：

## 1. BINARY_YES_NO（跨所两两）

- 同一 **OutcomeGroupKey**（`map_key|赛果norm` 或导出字段 `outcome_group_key`）
- 平台 A 买 **YES 卖一** + 平台 B 买 **NO 卖一**
- Poly：YES/NO 各用对应 `token_id` 的 **asks** 顶价
- Predict / SX / Kalshi：YES=asks 顶价，NO=**1 − bids 顶价**
- 每份成本：`per = yesPx + noPx + fee_yes + fee_no`，`fee = coef * p * (1-p)`
- 默认 `coef`：Poly 0.05，Kalshi 0.07，Predict/SX 0.02
- 命中：`per < 0.95`，且 `min(yesSz,noSz) * 2 >= orderAmount/(yesPx+noPx)`（order=100，factor=2）

## 2. ONE_X2 三路

- 仅 `outcome_type=moneyline` 的赛果组
- 三组：draw + 两队；各腿选该组内 **最便宜** 平台报价（买 YES 或买 NO 三路）
- YES 三路和 `< 0.95`；NO 三路和 `< 1.90`；深度规则同上

## 赛果分组

- 优先 JSONL 字段 **`outcome_group_key`**
- 否则 **`marketmap.OutcomeGroupKey`**（与 strategy_entry 一致）；失败才用 `groupkey.go` 启发式

## 历史套利

- **`yesno_replay`**：按落库 `id` 升序回放内存盘口（默认表 `market_order_book`；`-book_table=ws` 为 `market_order_book_ws`）
- **`yesno_snapshot`**：同 `snapshot_at_ms` 拼盘（较保守，可能漏跨毫秒组合）
- **`yesno_hist_min`**：各腿全程最低价混搭，**不能**当真实机会