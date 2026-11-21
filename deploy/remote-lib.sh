# 仅在远程服务器上执行（由 deploy-*.sh 通过 SSH 注入，勿在本机 source）

git_prepare_for_pull() {
  local clean_bin="${1:-0}"
  cd "${APP_DIR}"

  echo ">>> [服务器] git checkout .（丢弃已跟踪文件的本地改动）"
  git checkout .

  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git restore --worktree log/info.log log/error.log 2>/dev/null \
      || git checkout HEAD -- log/info.log log/error.log 2>/dev/null \
      || true
  fi
  if [[ "${clean_bin}" == "1" ]]; then
    git clean -fd bin/ 2>/dev/null || true
  fi
}

git_fetch_with_timeout() {
  export GIT_TERMINAL_PROMPT=0
  if command -v timeout >/dev/null 2>&1; then
    timeout "${GIT_FETCH_TIMEOUT}" git fetch origin --prune --progress
  else
    git fetch origin --prune --progress
  fi
}

ensure_app_git_repo() {
  mkdir -p "$(dirname "${APP_DIR}")"
  if [[ -d "${APP_DIR}/.git" ]]; then
    return 0
  fi
  local url="${GIT_REPO_URL:-}"
  if [[ -z "${url}" ]]; then
    echo "ERROR: ${APP_DIR} 不是 git 仓库，且未设置 GIT_REPO_URL" >&2
    exit 1
  fi
  echo ">>> [服务器] git clone ${url} → ${APP_DIR}"
  rm -rf "${APP_DIR}"
  git clone "${url}" "${APP_DIR}"
  cd "${APP_DIR}"
}

git_sync_branch() {
  ensure_app_git_repo
  cd "${APP_DIR}"
  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "ERROR: ${APP_DIR} 不是 git 仓库（或当前用户无权限 git，目录属主为 root 时请在 .deploy.env 设 REMOTE_SUDO=1）" >&2
    git status 2>&1 | head -5 >&2 || true
    exit 1
  fi
  git_prepare_for_pull "${GIT_CLEAN_BIN:-0}"
  echo ">>> [服务器] 目录: ${APP_DIR}"
  echo ">>> [服务器] git fetch + checkout ${BRANCH} + pull"
  git_fetch_with_timeout
  if git show-ref --verify --quiet "refs/heads/${BRANCH}"; then
    git checkout "${BRANCH}"
  else
    git checkout -B "${BRANCH}" "origin/${BRANCH}"
  fi
  if ! git pull --ff-only origin "${BRANCH}"; then
    echo "ERROR: git pull --ff-only 失败（已执行 git checkout .，请检查服务器仓库状态）" >&2
    git status -sb >&2 || true
    exit 1
  fi
  git log -1 --oneline
}

# 非交互 SSH 不会读 .bashrc 里的 PATH，必须用绝对路径或显式 PATH
setup_go_path() {
  local d
  export PATH="/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:${PATH}"
  local go_root="${ARBOPP_GO_ROOT:-${PAPER_GO_ROOT:-}}"
  if [[ -n "${go_root}" && -d "${go_root}/bin" ]]; then
    export PATH="${go_root}/bin:${PATH}"
  fi
  for d in /usr/local/go/bin /opt/go/bin /www/server/go/bin; do
    if [[ -x "${d}/go" ]]; then
      export PATH="${d}:${PATH}"
    fi
  done
  if [[ -s /etc/profile ]]; then
    # shellcheck disable=SC1091
    source /etc/profile 2>/dev/null || true
  fi
  if [[ -s /etc/environment ]]; then
    set -a
    # shellcheck disable=SC1091
    source /etc/environment 2>/dev/null || true
    set +a
  fi
}

resolve_go_bin() {
  local c found go_root="${ARBOPP_GO_ROOT:-${PAPER_GO_ROOT:-}}"
  setup_go_path
  GO_BIN=""
  for c in \
    "${go_root:+"${go_root}/bin/go"}" \
    /usr/local/go/bin/go \
    /opt/go/bin/go \
    /usr/bin/go; do
    [[ -n "${c}" && -x "${c}" ]] || continue
    GO_BIN="${c}"
    export PATH="$(dirname "${c}"):${PATH}"
    export GO_BIN
    return 0
  done
  found="$(type -P go 2>/dev/null || true)"
  if [[ -n "${found}" && -x "${found}" ]]; then
    GO_BIN="${found}"
    export GO_BIN
    export PATH="$(dirname "${found}"):${PATH}"
    return 0
  fi
  return 1
}

install_go_on_server() {
  if [[ -x /usr/local/go/bin/go ]]; then
    return 0
  fi
  local ver="${ARBOPP_GO_VERSION:-${PAPER_GO_VERSION:-1.23.0}}"
  local tarball="go${ver}.linux-amd64.tar.gz"
  local tmp="/tmp/${tarball}"
  local url
  echo ">>> [服务器] 未找到 go，尝试安装 Go ${ver} → /usr/local/go ..."
  for url in \
    "https://go.dev/dl/${tarball}" \
    "https://golang.google.cn/dl/${tarball}"; do
    if command -v wget >/dev/null 2>&1; then
      wget -qO "${tmp}" "${url}" && break
    elif command -v curl >/dev/null 2>&1; then
      curl -fsSL -o "${tmp}" "${url}" && break
    else
      echo "ERROR: [服务器] 需要 wget 或 curl 才能自动安装 Go" >&2
      return 1
    fi
    url=""
  done
  [[ -s "${tmp}" ]] || return 1
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "${tmp}"
  rm -f "${tmp}"
  setup_go_path
}

ensure_go() {
  local gv
  if resolve_go_bin; then
    gv=$("${GO_BIN}" version 2>/dev/null) || gv="(version failed)"
    echo ">>> [服务器] ${GO_BIN} ${gv}"
    return 0
  fi
  install_go_on_server || true
  if resolve_go_bin; then
    gv=$("${GO_BIN}" version 2>/dev/null) || gv="(version failed)"
    echo ">>> [服务器] ${GO_BIN} ${gv}"
    return 0
  fi
  echo "ERROR: [服务器] 未找到 go（非交互 SSH 不会加载 .bashrc 的 PATH）。" >&2
  echo "  请确认: ls -l /usr/local/go/bin/go" >&2
  echo "  或在 deploy/.deploy.env 设置: ARBOPP_GO_ROOT=/usr/local/go" >&2
  exit 1
}

