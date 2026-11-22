# data_analysis

**不连库、不 import 策略包**（规则见 `ws_arb_scan/RULES.md`）。

- 连库 CSV / 中文报告：`ws_arb_scan/data/`，见 [`ws_arb_scan/data/README.md`](ws_arb_scan/data/README.md)
- 附加 jsonl：`ws_arb_scan/out/`

```bash
# 批量：默认 data_analysis/ws_arb_scan/map_keys_batch.txt（20 场）
RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan
# 单场
RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan -map_key='v1|NBA|...'
# WS 落库表
RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan -map_key='...' -book_table=ws
go run ./data_analysis/ws_arb_scan -data=./data_analysis/ws_arb_scan/data
```