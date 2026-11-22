# 套利判定

**直接 import `strategy_entry`**：`book.Store`、`yesno.Runner`（`TopHit` / `TryPickForOutcome` / `OutcomeGroupKey` 与线上一致）。

门槛与手续费默认 `strategyentry.LoadParams`；`hist_min` / 三路仍用本包辅助（非线上发现路径）。

## 扫描模式

- **latest**：最新 REST/WS 行灌入 `book.Store` 后 `yesno.AllPicksInGroup`
- **replay**：按 `id` 升序 `applyBookRow` + `TryPickForOutcome`（对齐 `OnBookUpdate`）
- **snapshot**：同 `snapshot_at_ms` 灌盘再试算
- **hist_min**：各腿全程最低价（仅参考，非 strategy 路径）