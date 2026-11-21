# arbopp 部署（bot）

```bash
cp .deploy.env.example .deploy.env
# 编辑 .deploy.env：SSH_HOST、SSH_USER；AWS 用 SSH_KEY=~/.ssh/id_rsa + SSH_USER=ubuntu
chmod +x deploy-service.sh deploy-backend.sh
./deploy-service.sh master
```

SSH：配置了 `SSH_KEY` 则用本机密钥（无需 `sshpass`）；未配置则用 `SSH_PASS` + `sshpass`/`expect`。

服务器执行：`go run main.go --run_conf=test --run_bot=true`（`MALL_BOT_RUN_CONF` 可改）。

`status` / `restart` / `stop`：第二个参数，例如 `./deploy-service.sh master status`。

日志：`$MALL_APP_DIR/logs/arbopp-bot-nohup.log`。

底层：`./deploy-backend.sh <分支> [deploy|restart|stop|status] bot`