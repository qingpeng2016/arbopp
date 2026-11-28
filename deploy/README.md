# arbopp 部署（bot）

```bash
cp .deploy.env.example .deploy.env
# 编辑 NODE_POLY_HOST / NODE_SXBET_HOST（AWS：同一 SSH_KEY + ubuntu + REMOTE_SUDO=1）
chmod +x deploy-service.sh deploy-backend.sh
./deploy-service.sh v2 deploy POLY
./deploy-service.sh v2 deploy SXBET
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

双机公网联调（POLY + SXBET）前在 MySQL 执行：`docs/sql/node_route_install_2node_public_test.sql`。

例：

```bash
./deploy-service.sh v2 status SXBET
./deploy-service.sh v2 restart POLY
```

SSH：配置了 `SSH_KEY` 则用本机密钥；未配置则用 `SSH_PASS` + `sshpass`/`expect`。

服务器执行：`go run main.go --run_conf=test --run_bot=true --node=POLY`（由第三参决定）。

日志：`$MALL_APP_DIR/logs/arbopp-bot-nohup.log`。