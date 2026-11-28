# arbopp 部署（bot）

```bash
cp .deploy.env.example .deploy.env
# 爱尔兰 POLY + 东京 PRED：NODE_*_HOST=公网 SSH；NODE_*_SSH_KEY=各区域 pem
chmod +x deploy-service.sh deploy-backend.sh
./deploy-service.sh v2 deploy POLY
./deploy-service.sh v2 deploy PRED
```

## 用法

```text
./deploy-service.sh <分支> [deploy|restart|stop|status] [POLY|PRED|SXBET|ALL]
```

| 第三参 | SSH 目标 | 远端启动 |
|--------|----------|----------|
| `POLY` | `NODE_POLY_HOST` | `--node=POLY` |
| `PRED` | `NODE_PRED_HOST` | `--node=PRED` |
| `SXBET` | `NODE_SXBET_HOST` | `--node=SXBET` |
| 省略 / `ALL` | `SSH_HOST` | `--node=ALL`（默认） |

- 爱尔兰 Poly + 东京 Predict（**relay 私网**）：`docs/sql/node_route_install_2node_ie_jp_private.sql`
- 公网联调 POLY + SXBET：`docs/sql/node_route_install_2node_public_test.sql`

部署 **SSH 用公网**；`node_route` 里 `publish_local_book_to` / `hedge_order_to` 为对端 **私网 IP:9000**。

例：

```bash
./deploy-service.sh v2 status SXBET
./deploy-service.sh v2 restart POLY
```

SSH：默认 `SSH_KEY`；指定 `POLY|PRED|SXBET` 时用对应 `NODE_*_SSH_KEY`（可覆盖 `SSH_KEY`）。未配置密钥则用 `SSH_PASS` + `sshpass`/`expect`。

服务器执行：`go run main.go --run_conf=test --run_bot=true --node=POLY`（由第三参决定）。

日志：`$MALL_APP_DIR/logs/arbopp-bot-nohup.log`。