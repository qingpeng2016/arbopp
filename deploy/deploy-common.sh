# shellcheck shell=bash
# 本机只做 SSH 跳板；git / build / 启停均在服务器 MALL_APP_DIR 执行。
# SSH：SSH_KEY 密钥登录；否则 sshpass / expect 密码登录

deploy_common_init() {
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  ENV_FILE="${SCRIPT_DIR}/.deploy.env"
  REMOTE_LIB_FILE="${SCRIPT_DIR}/remote-lib.sh"

  [[ -f "$ENV_FILE" ]] || {
    echo "缺少 ${ENV_FILE}，请先: cp .deploy.env.example .deploy.env" >&2
    exit 1
  }
  [[ -f "$REMOTE_LIB_FILE" ]] || {
    echo "缺少 ${REMOTE_LIB_FILE}" >&2
    exit 1
  }

  # shellcheck source=/dev/null
  source "$ENV_FILE"

  : "${SSH_HOST:?请在 .deploy.env 设置 SSH_HOST}"
  : "${SSH_USER:?请在 .deploy.env 设置 SSH_USER}"

  SSH_PORT="${SSH_PORT:-22}"
  SSH_KEY_FILE=""
  if [[ -n "${SSH_KEY:-}" ]]; then
    SSH_KEY_FILE="${SSH_KEY/#\~/$HOME}"
    [[ -f "$SSH_KEY_FILE" ]] || {
      echo "SSH_KEY 文件不存在: ${SSH_KEY_FILE}（.deploy.env 里 SSH_KEY=${SSH_KEY}）" >&2
      exit 1
    }
  else
    : "${SSH_PASS:?请在 .deploy.env 设置 SSH_PASS，或设置 SSH_KEY 使用密钥登录}"
  fi
  MALL_APP_DIR="${MALL_APP_DIR:-/www/wwwroot/arbopp}"
  MALL_USER_RUN_CONF="${MALL_USER_RUN_CONF:-test}"
  MALL_BOT_RUN_CONF="${MALL_BOT_RUN_CONF:-test}"
  MALL_USER_PORT="${MALL_USER_PORT:-8886}"
  # 1=服务器 go run main.go；0=服务器 go build 后跑 bin（deploy-service.sh 会强制为 1）
  ARBOPP_USE_GO_RUN="${ARBOPP_USE_GO_RUN:-${PAPER_USE_GO_RUN:-1}}"
  PAPER_USE_GO_RUN="$ARBOPP_USE_GO_RUN"
  ARBOPP_GO_ROOT="${ARBOPP_GO_ROOT:-${PAPER_GO_ROOT:-}}"
  PAPER_GO_ROOT="$ARBOPP_GO_ROOT"
  ARBOPP_GO_VERSION="${ARBOPP_GO_VERSION:-${PAPER_GO_VERSION:-1.23.0}}"
  PAPER_GO_VERSION="$ARBOPP_GO_VERSION"
  DEPLOY_UPLOAD_CONF="${DEPLOY_UPLOAD_CONF:-0}"
  MALL_BIN_NAME="${MALL_BIN_NAME:-arbopp}"
  GIT_FETCH_TIMEOUT="${GIT_FETCH_TIMEOUT:-120}"
  LOCAL_REPO_DIR="${LOCAL_REPO_DIR:-$(cd "$SCRIPT_DIR/.." && pwd)}"
  GIT_REPO_URL="${GIT_REPO_URL:-git@github.com:qingpeng2016/arbopp.git}"

  APP_DIR="$MALL_APP_DIR"
  SSH_TARGET="${SSH_USER}@${SSH_HOST}"

  if [[ -n "$SSH_KEY_FILE" ]]; then
    SSH_OPTS=(
      -o StrictHostKeyChecking=accept-new
      -o IdentitiesOnly=yes
      -i "$SSH_KEY_FILE"
      -p "$SSH_PORT"
    )
    SCP_OPTS=(
      -o StrictHostKeyChecking=accept-new
      -o IdentitiesOnly=yes
      -i "$SSH_KEY_FILE"
      -P "$SSH_PORT"
    )
  else
    SSH_OPTS=(
      -o StrictHostKeyChecking=accept-new
      -o PreferredAuthentications=password
      -o PubkeyAuthentication=no
      -p "$SSH_PORT"
    )
    # scp 端口是大写 -P；勿把 SSH_OPTS 直接传给 scp（-p 会被当成「保留时间」+ 本地路径 22）
    SCP_OPTS=(
      -o StrictHostKeyChecking=accept-new
      -o PreferredAuthentications=password
      -o PubkeyAuthentication=no
      -P "$SSH_PORT"
    )
  fi
}

deploy_remote_lib() {
  cat "${SCRIPT_DIR}/remote-lib.sh"
}

# 远程执行一条 shell 命令（已拼好、由调用方转义）
_deploy_ssh_run() {
  local remote_cmd="$1"
  local rc=0

  if [[ -n "$SSH_KEY_FILE" ]]; then
    ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "$remote_cmd"
    return $?
  fi

  if command -v sshpass >/dev/null 2>&1; then
    sshpass -p "$SSH_PASS" ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "$remote_cmd"
    rc=$?
  elif command -v expect >/dev/null 2>&1; then
    # shellcheck disable=SC2016
    expect <<EOF
set timeout 600
log_user 1
spawn ssh -o StrictHostKeyChecking=accept-new -o PreferredAuthentications=password -o PubkeyAuthentication=no -p ${SSH_PORT} ${SSH_TARGET} "${remote_cmd}"
expect {
  -re "(?i)password:" {
    send "${SSH_PASS}\r"
    exp_continue
  }
  eof
}
set wait_result [wait]
set exit_code [lindex \$wait_result 3]
if {\$exit_code eq ""} { set exit_code 0 }
exit \$exit_code
EOF
    rc=$?
  else
    echo "未找到 sshpass 或 expect。Mac 一般自带 expect；或: brew install hudochenkov/sshpass/sshpass" >&2
    return 1
  fi
  return "$rc"
}

run_ssh() {
  local remote_cmd="$*"
  _deploy_ssh_run "$remote_cmd"
}

run_remote_bash() {
  local script_body="$1"
  echo ">>> SSH ${SSH_TARGET} — 以下在服务器 ${APP_DIR} 执行"
  local b64 one_liner
  b64=$(printf '%s' "$script_body" | base64 | tr -d '\n')
  one_liner="echo ${b64} | base64 -d | bash"
  _deploy_ssh_run "$one_liner"
}

resolve_backend_mode() {
  local mode="$1"
  case "$mode" in
    user|api) echo "user" ;;
    bot|robot) echo "bot" ;;
    *)
      echo "ERROR: 模式只能是 user/api 或 bot/robot，当前: ${mode}" >&2
      exit 1
      ;;
  esac
}

backend_run_vars() {
  local mode="$1"
  case "$mode" in
    user|api)
      RUN_CONF="$MALL_USER_RUN_CONF"
      RUN_BOT="false"
      LISTEN_PORT="$MALL_USER_PORT"
      ;;
    bot|robot)
      RUN_CONF="$MALL_BOT_RUN_CONF"
      RUN_BOT="true"
      LISTEN_PORT=""
      ;;
  esac
}

parse_action_mode_args() {
  ACTION="${1:-deploy}"
  MODE="${2:-user}"
  case "$ACTION" in
    user|api|bot|robot)
      MODE="$ACTION"
      ACTION="deploy"
      ;;
  esac
}

# conf/<run_conf>.conf.yaml：默认随 git；local 或 DEPLOY_UPLOAD_CONF=1 时从本机上传
deploy_sync_run_conf() {
  local run_conf="${1:-}"
  [[ -n "$run_conf" ]] || return 0
  local upload=0
  if [[ "$run_conf" == "local" ]]; then
    upload=1
  elif [[ "${DEPLOY_UPLOAD_CONF}" == "1" ]]; then
    upload=1
  fi
  [[ "$upload" -eq 1 ]] || return 0
  local src="${LOCAL_REPO_DIR}/conf/${run_conf}.conf.yaml"
  [[ -f "$src" ]] || {
    echo "ERROR: 缺少 ${src}（run_conf=${run_conf} 需要本机配置文件）" >&2
    exit 1
  }
  echo ">>> [本机] 上传 ${run_conf}.conf.yaml → ${APP_DIR}/conf/"
  run_ssh "mkdir -p '${APP_DIR}/conf'"
  local b64
  b64=$(base64 <"$src" | tr -d '\n')
  if [[ ${#b64} -gt 120000 ]]; then
    echo "ERROR: ${run_conf}.conf.yaml 过大，请改用 scp 手动上传" >&2
    exit 1
  fi
  run_ssh "echo '${b64}' | base64 -d > '${APP_DIR}/conf/${run_conf}.conf.yaml'"
}

deploy_sync_local_conf() {
  deploy_sync_run_conf "$@"
}
