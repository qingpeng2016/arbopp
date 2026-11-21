#!/usr/bin/env bash
# bot：SSH → git pull → go run main.go --run_conf=test --run_bot=true（见 .deploy.env MALL_BOT_RUN_CONF）
#
# 用法:
#   ./deploy-service.sh <分支>              # deploy
#   ./deploy-service.sh <分支> restart
#   ./deploy-service.sh <分支> stop
#   ./deploy-service.sh <分支> status
#
# 本机可选上传 conf（local 或 DEPLOY_UPLOAD_CONF=1）；代码与 go run 在服务器执行。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export ARBOPP_USE_GO_RUN=1
BRANCH="${1:-}"
ACTION="${2:-deploy}"

if [[ -z "$BRANCH" ]]; then
  echo "用法: $0 <分支> [deploy|restart|stop|status]" >&2
  exit 1
fi

case "$ACTION" in
  deploy|restart|stop|status) ;;
  *)
    echo "未知动作: $ACTION" >&2
    exit 1
    ;;
esac

exec "${SCRIPT_DIR}/deploy-backend.sh" "$BRANCH" "$ACTION" bot
