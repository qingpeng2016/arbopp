# data_analysis

**不连库、不 import 策略包**（规则见 `ws_arb_scan/RULES.md`）。

- 连库 CSV / 中文报告：`ws_arb_scan/data/`，见 [`ws_arb_scan/data/README.md`](ws_arb_scan/data/README.md)
- 附加 jsonl：`ws_arb_scan/out/`

```bash
RUN_CONF=conf/local.conf.yaml go run ./data_analysis/ws_arb_scan -map_key='v1|NBA|...'
go run ./data_analysis/ws_arb_scan -data=./data_analysis/ws_arb_scan/data
```