#!/usr/bin/env bash
# strategy_entry WebSocket 质量快检
set -euo pipefail

LOG="${LOG:-log/error.log}"
MAX_GROUPS="${MAX_GROUPS:-50}"
PLATFORMS="${PLATFORMS:-2}"
PID="${PID:-}"
PGREP_PATTERN="${PGREP_PATTERN:-fgmm|arbopp|main\.go|run_bot|fgmm-strategy|wwwroot/arbopp}"
MYSQL_CMD="${MYSQL_CMD:-}"
SAMPLE_SEC="${SAMPLE_SEC:-0}"

section() { echo; echo "======== $* ========"; }

# 从 ss 里找握 WS 最多的 pid（不依赖进程名）
find_pid_by_ws() {
  local line pid
  line=$(ss -tnp state established 2>/dev/null \
    | grep -E 'polymarket|kalshi|predict' \
    | sed -n 's/.*pid=\([0-9][0-9]*\).*/\1/p' \
    | sort | uniq -c | sort -rn | head -1 || true)
  pid=$(echo "$line" | awk '{print $2}')
  [[ -n "$pid" && "$pid" =~ ^[0-9]+$ ]] && echo "$pid" && return 0
  return 1
}

find_pid_by_fd() {
  local best="" best_n=0 p n
  while read -r p; do
    [[ -z "$p" ]] && continue
    [[ -d "/proc/$p/fd" ]] || continue
    n=$(ls "/proc/$p/fd" 2>/dev/null | wc -l | tr -d ' ')
    if (( n > best_n )); then best_n=$n; best="$p"; fi
  done < <(pgrep -f "$PGREP_PATTERN" 2>/dev/null || true)
  [[ -n "$best" ]] && echo "$best" && return 0
  return 1
}

find_worker_pid() {
  if [[ -n "${PID}" ]]; then
    echo "${PID}"
    return 0
  fi
  local p
  p=$(find_pid_by_ws) && echo "$p" && return 0
  p=$(find_pid_by_fd) && echo "$p" && return 0
  return 1
}

show_pid_help() {
  section "找不到 bot 进程 — 请先确认是否在跑"
  echo "本机候选进程 (pgrep -af):"
  pgrep -af "$PGREP_PATTERN" 2>/dev/null || echo "  (无匹配 PGREP_PATTERN=$PGREP_PATTERN)"
  echo
  echo "或: ps aux | grep -E 'fgmm|arbopp|main.go|run_bot' | grep -v grep"
  ps aux 2>/dev/null | grep -E 'fgmm|arbopp|main\.go|run_bot' | grep -v grep || true
  echo
  echo "当前整机 WS 相关连接 (任意进程):"
  ss -tnp state established 2>/dev/null | grep -E 'polymarket|kalshi|predict' | head -20 || echo "  (无 — bot 可能未启动或未建连)"
  echo
  echo "启动 bot 示例: ./fgmm --run_conf=prod --run_bot=true"
  echo "或: cd /www/wwwroot/arbopp && go run main.go --run_conf=prod --run_bot=true"
  echo
  echo "已知 PID 时: export PID=12345 && bash bash/ws_quality.sh"
  echo "自定义匹配: export PGREP_PATTERN='你的关键字' && bash bash/ws_quality.sh"
}

count_established_linux() {
  local pid="$1" host="$2"
  ss -tnp state established 2>/dev/null | grep "pid=${pid}," | grep -c "$host" || true
}

WORKER_PID=""
if WORKER_PID=$(find_worker_pid); then
  :
else
  show_pid_help
  WORKER_PID=""
fi

EXPECTED=$((MAX_GROUPS * PLATFORMS))

if [[ -n "$WORKER_PID" ]]; then
  section "进程"
  echo "WORKER_PID=$WORKER_PID"
  ps -p "$WORKER_PID" -o pid,etime,rss,args 2>/dev/null || ps -p "$WORKER_PID" 2>/dev/null || true
  echo "应有 ESTABLISHED 约: $EXPECTED (max_groups=$MAX_GROUPS × platforms=$PLATFORMS)"
  if [[ -d "/proc/$WORKER_PID/fd" ]]; then
    echo "ulimit -n: $(ulimit -n)"
    echo "open fd: $(ls "/proc/$WORKER_PID/fd" 2>/dev/null | wc -l | tr -d ' ')"
    grep -E 'open files|Max open files' "/proc/$WORKER_PID/limits" 2>/dev/null || true
  fi

  section "TCP 实连 (Poly / Kalshi / Predict)"
  if command -v ss >/dev/null 2>&1; then
    total=0
    for h in polymarket kalshi predict; do
      c=$(count_established_linux "$WORKER_PID" "$h")
      echo "$h: $c"
      total=$((total + c))
    done
    echo "合计: $total"
    ss -tnp 2>/dev/null | grep "pid=${WORKER_PID}," | awk '{print $1}' | sort | uniq -c || true
  fi

  if [[ "$SAMPLE_SEC" -gt 0 ]] && command -v ss >/dev/null 2>&1; then
    section "连接采样 (每 30s, 共 ${SAMPLE_SEC} 次)"
    i=0
    while (( i < SAMPLE_SEC )); do
      date '+%F %T'
      ss -tnp state established 2>/dev/null | grep "pid=${WORKER_PID}," | grep -cE 'polymarket|kalshi|predict' || echo 0
      sleep 30
      i=$((i + 1))
    done
  fi
else
  section "TCP 实连 (整机，未绑定 PID)"
  ss -tnp state established 2>/dev/null | grep -E 'polymarket|kalshi|predict' \
    | sed -n 's/.*pid=\([0-9]*\),.*/\1/p' | sort | uniq -c | sort -rn || echo "(无 WS 连接 — bot 未跑或未连上)"
fi

section "出口 RTT (仅 TLS)"
command -v curl >/dev/null 2>&1 && for url in \
  'https://ws-subscriptions-clob.polymarket.com/' \
  'https://api.elections.kalshi.com/' \
  'https://ws.predict.fun/'; do
  curl -o /dev/null -s -m 8 -w "%{url_effective} connect=%{time_connect}s http=%{http_code}\n" "$url" || echo "$url failed"
done

if [[ -f "$LOG" ]]; then
  section "日志断线统计 ($LOG)"
  grep -oE 'strategy_entry-(poly|kalshi|predict)-disconnected' "$LOG" 2>/dev/null | sort | uniq -c | sort -rn || echo "(无 disconnected 记录)"
  grep -E 'strategy_entry-(poly|kalshi|predict)-disconnected' "$LOG" 2>/dev/null \
    | grep -oE '"map_key":"[^"]+"' | sort | uniq -c | sort -rn | head -15 || true
  grep 'disconnected' "$LOG" 2>/dev/null | grep -oE '"gen":[0-9]+' | sort -t: -k2 -n | tail -5 || true
  grep -E 'kalshi-seq-gap|predict-snapshot-failed|event queue full|ws-stagger-read-failed' "$LOG" 2>/dev/null | tail -15 || true
else
  section "日志"
  echo "找不到 LOG=$LOG"
fi

if [[ -n "$MYSQL_CMD" ]]; then
  section "MySQL"
  "$MYSQL_CMD" -e "
SELECT updated_at, JSON_LENGTH(JSON_EXTRACT(item_value, '$.by_map_key')) AS map_keys_in_json
FROM stat_kv WHERE group_name='strategy_entry_detect_latency' AND item_key='ws_to_arb_check_ms';
SELECT platform, COUNT(*) cnt, MAX(updated_at) last_upd,
  TIMESTAMPDIFF(SECOND, MAX(updated_at), NOW()) lag_sec
FROM market_order_book WHERE updated_at > NOW() - INTERVAL 10 MINUTE GROUP BY platform;
" 2>/dev/null || echo "MySQL 失败"
fi

section "判读"
echo "· 无进程且无 polymarket/kalshi 连接 → 先启动 bot (--run_bot=true)"
echo "· 正常应 ≈ $EXPECTED 条 ESTABLISHED（错峰后等 2～3 分钟）"
echo "完成."