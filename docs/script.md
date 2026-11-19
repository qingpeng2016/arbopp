# less_one 策略脚本（整套需求，对照套利文档）

> 调度：`Entry` 调 `Scheduler`，任务名与周期在库表 `bot_schedule_config`（`module=less_one`）。

| 阶段 | 任务名 | 建议周期 | 功能 | 主要读写 |
|------|--------|----------|------|----------|
| 二 | market_sync | 5–30m | 各平台拉**体育单场**：场次写 `market`，盘口写 `market_outcome`；盘口类型由 `sync_outcome_types` 按运动分组（足球：`moneyline` / `handicap` / `total`） | `market`、`market_outcome`、`config_strategy` |
| 二 | market_map | 定时 | 跨平台匹配成功写 `map_key` 并 **`map_verified=1`**（仅处理 `map_verified=0`） | `market_outcome` |
| 二 | orderbook_sync | 10–60s | `config_strategy.strategy_entry.watch_map_keys` 指定场次；`orderbook_fetch` 控制每批条数与批间并发（默认 Poly 500×3、Kalshi 100×3、Predict 100×1），**BatchUpsert** `market_order_book` | `market`、`market_outcome`、`market_order_book`、`config_strategy` |
| 二 | history_data_sync | 按需 | `data_task` 中 `status=pending` 的 `map_key`；PMData 拉 L2，完成后 `data_task.status=completed` | `data_task`、`data_platform`、`data_history_order_book`、`market`、`market_outcome` |
| 二 | synthetic_history_sync | 按需 | `data_task.task_type=synthetic_l2`：每 outcome 1000 秒×1 条/秒，批插 100 秒；多平台 `snapshot_at_ms` 对齐 | `data_task`、`data_platform(synthetic)`、`data_history_order_book` |
| 二 | backtest_replay | 按需 | `backtest_task.status=pending`：按 `snapshot_at_ms` 重放 `data_history_order_book`，驱动 `yesno`+`onextwo`（不发真实单）；机会 `backtest_arb_binary_opportunity`，订单 `orders`（`backtest_task_id`） | `backtest_task`、`data_history_order_book`、`backtest_arb_binary_opportunity`、`orders` |
| 三 | strategy_entry | 20s | `watch_map_keys` 配置 map_key（每场 Poly+Kalshi 两条 market）；`max_groups`（默认 50）按成交额取前 N 组；`ws_connect_stagger_ms`（默认 `{"min_ms":0,"max_ms":2000}`）首场建连错峰。WebSocket 比较卖一后发两笔 FAK；一腿成交会反向平掉。落库 `market_order_book`、`arb_binary_opportunity`、`arb_binary_order` | `market`、`market_outcome`、`config_strategy`、`config_strategy_threshold`、`account` |
| 五–六 | trade_execute | 事件/5s | 下单（FAK 两腿/三腿）+ **内嵌敞口处理**：部分成交/单腿失败在同一执行上下文里补腿、反向平仓、记台账，避免与另一定时任务并发改同一笔执行 | `account`、`arb_binary_opportunity`、执行/敞口表（规划） |
| — | task_alert | 1–5m | 机会、敞口、拒单率等告警 | 日志、通知 |
| — | report_daily | 1d | 机会数、成交率、滑点、复盘报表 | 各业务表 |

**合计：7 个业务任务**（无独立 `fair_price`、`risk_filter`、`exposure_manage`）。已实现：**market_sync**、**market_map**（写 `map_key`）、**strategy_entry**（5.2 二元全覆盖检测）、**orderbook** 探针。

`market_sync` 代码目录：`application/strategy/scripts/marketsync/`（根目录编排 + 落库）、`polymarket/`、`kalshi/` 各平台拉数；一次同步写 **`market`（场次）** 与 **`market_outcome`（盘口）** 两张表。

### market_map 规则（自动）

1. 扫描 `status=ACTIVE` 且 `map_verified=0` 的行（`competition` 非空）。
2. 先按 **`competition` 相同**。
3. **同一逻辑盘**：`competition` 相同 + slug 对齐场次（Poly `epl-ast-sun-2026-11-21-*` ↔ Kalshi `kxeplgame-26nov21avlsun`）+ 标题解析 **胜出方**（`Will X win` / `X wins` / 平局）。
4. 生成 `map_key`：`v1|{联赛}|{场次slug}|{球队slug}`；slug/队名匹配后，若两边均有 **`event_start_time`**，须 **UTC 同日** 且相差 ≤18h 才成对；仅 **≥2 平台** 时写入并 **`map_verified=1`**。
5. 已是 `map_verified=1` 的行不参与扫描；错配需人工改回 `0` 后重跑。
