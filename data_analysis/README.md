# data_analysis

**不连库、不 import 策略包**（规则见 `ws_arb_scan/RULES.md`）。数据放 [`data/`](../data/README.md)：

```bash
go run ./data_analysis/ws_arb_scan -data=./data
```

输出：`data_analysis/ws_arb_scan/out/report.jsonl`