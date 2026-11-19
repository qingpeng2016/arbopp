#!/usr/bin/env bash
# strategy_entry WebSocket 质量快检（上传服务器后直接 bash bash/ws_quality.sh）
set -euo pipefail

# ---------- 可改环境变量 ----------
LOG="${LOG:-log/error.log}"
MAX_GROUPS="${MAX_GROUPS:-50}"
PLATFORMS="${PLATFORMS:-2}" # 2=Poly+Kalshi；开了 Predict 设 3
PID="${PID:-}"
MYSQL_CMD="${MYSQL_CMD:-}"   # 例: mysql -N -h127.0.0.1 -uroot -p'xxx' fgmm
SAMPLE_SEC="${SAMPLE_SEC:-0}" # >0 则采样 N 次（每次间隔 30s）后退出
# ------------------------------------

die() { echo "ERROR: $*" >&2; exit 1; }

find_worker_pid() {
  if [[ -n "${PID}" ]]; then
    echo "${PID}"
    return
  fi
  local best="" best_n=0
  local p n
  while read -r p; do
    [[ -z "$p" ]] && continue
  if [[ -d "/proc/$p/fd" ]]; then
    n=$(ls "/proc/$p/fd" 2>/dev/null | wc -l | tr -d ' ')
    if (( n > best_n )); then best_n=$n; best="$p"; fi
    continue
  fi
  done < <(pgrep -f 'fgmm|strategy|arbopp|less_one' 2>/dev/null || true)
  if [[ -z "$best" ]]; then
    best=$(pgrep -f 'fgmm' 2>/dev/null | head -1 || true)
  fi
  [[ -n "$best" ]] || die "找不到进程：先 export PID=... 或改 pgrep 关键字"
  echo "$best"
}

count_established_linux() {
  local pid="$1" host="$2"
  ss -tnp state established 2>/dev/null | grep "pid=${pid}," | grep -c "$host" || true
}

count_established_mac() {
  local pid="$1"
  lsof -nP -a -p "$pid" -iTCP -sTCP:ESTABLISHED 2>/dev/null | grep -cE 'polymarket|kalshi|predict' || true
}

section() { echo; echo "======== $* ========"; }

WORKER_PID=$(find_worker_pid)
EXPECTED=$((MAX_GROUPS * PLATFORMS))

section "进程"
echo "WORKER_PID=$WORKER_PID"
ps -p "$WORKER_PID" -o pid,etime,rss,command 2>/dev/null || ps -p "$WORKER_PID" 2>/dev/null || true
echo "应有 ESTABLISHED 约: $EXPECTED (max_groups=$MAX_GROUPS × platforms=$PLATFORMS)"
if [[ -d "/proc/$WORKER_PID/fd" ]]; then
  echo "ulimit -n: $(ulimit -n)"
  echo "open fd: $(ls "/proc/$WORKER_PID/fd" 2>/dev/null | wc -l | tr -d ' ')"
  grep -E 'open files|Max open files' "/proc/$WORKER_PID/limits" 2>/dev/null || true
else
  echo "ulimit -n: $(ulimit -n)"
fi

section "TCP 实连 (Poly / Kalshi / Predict)"
if command -v ss >/dev/null 2>&1 && [[ -d "/proc/$WORKER_PID" ]]; then
  total=0
  for h in polymarket kalshi predict; do
    c=$(count_established_linux "$WORKER_PID" "$h")
    echo "$h: $c"
    total=$((total + c))
  done
  echo "合计: $total"
  echo "TCP 状态分布:"
  ss -tnp 2>/dev/null | grep "pid=${WORKER_PID}," | awk '{print $1}' | sort | uniq -c || true
elif command -v lsof >/dev/null 2>&1; then
  c=$(count_established_mac "$WORKER_PID")
  echo "ESTABLISHED (mac 合计): $c"
  lsof -nP -a -p "$WORKER_PID" -iTCP -sTCP:ESTABLISHED 2>/dev/null | grep -E 'polymarket|kalshi|predict' | awk '{print $9}' | sed 's/.*->//' | cut -d: -f1 | sort | uniq -c || true
else
  echo "无 ss/lsof，跳过"
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

section "出口 RTT (仅 TLS, 非 WS)"
for url in \
  'https://ws-subscriptions-clob.polymarket.com/' \
  'https://api.elections.kalshi.com/' \
  'https://ws.predict.fun/'; do
  if command -v curl >/dev/null 2>&1; then
    curl -o /dev/null -s -m 8 -w "%{url_effective} connect=%{time_connect}s http=%{http_code}\n" "$url" || echo "$url failed"
  fi
done

if [[ -f "$LOG" ]]; then
  section "日志断线统计 ($LOG)"
  echo "--- 各平台 disconnected 次数 ---"
  grep -oE 'strategy_entry-(poly|kalshi|predict)-disconnected' "$LOG" 2>/dev/null | sort | uniq -c | sort -rn || echo "(无)"
  echo "--- Top map_key (disconnected) ---"
  grep -E 'strategy_entry-(poly|kalshi|predict)-disconnected' "$LOG" 2>/dev/null \
    | grep -oE '"map_key":"[^"]+"' | sort | uniq -c | sort -rn | head -15 || echo "(无)"
  echo "--- 最近 gen (断线代数, 越大越抖) ---"
  grep 'disconnected' "$LOG" 2>/dev/null | grep -oE '"gen":[0-9]+' | sort -t: -k2 -n | tail -5 || echo "(无)"
  echo "--- 其它 WS 相关 (最近 15 行) ---"
  grep -E 'kalshi-seq-gap|predict-snapshot-failed|event queue full|ws-stagger-read-failed' "$LOG" 2>/dev/null | tail -15 || echo "(无)"
else
  section "日志"
  echo "找不到 LOG=$LOG（export LOG=/path/to/error.log）"
fi

if [[ -n "$MYSQL_CMD" ]]; then
  section "MySQL stat_kv / market_order_book"
  echo "--- detect_latency updated_at + map_key 有 samples 的数量 ---"
  "$MYSQL_CMD" -e "
SELECT updated_at,
  JSON_LENGTH(JSON_EXTRACT(item_value, '$.by_map_key')) AS map_keys_in_json
FROM stat_kv
WHERE group_name='strategy_entry_detect_latency' AND item_key='ws_to_arb_check_ms';
" 2>/dev/null || echo "MySQL 查询失败（检查 MYSQL_CMD 或 JSON 函数）"
  echo "--- market_order_book 近 10 分钟按平台 ---"
  "$MYSQL_CMD" -e "
SELECT platform, COUNT(*) cnt, MAX(updated_at) last_upd,
  TIMESTAMPDIFF(SECOND, MAX(updated_at), NOW()) lag_sec
FROM market_order_book
WHERE updated_at > NOW() - INTERVAL 10 MINUTE
GROUP BY platform;
" 2>/dev/null || true
else
  section "MySQL (可选)"
  echo '未设置 MYSQL_CMD。示例:'
  echo "  export MYSQL_CMD='mysql -N -hHOST -uUSER -pPASS dbname'"
  echo "  bash bash/ws_quality.sh"
fi

section "判读"
echo "· Lark「WS 心跳」每 5min：正常 应 ≈ 应连数 ($EXPECTED)"
echo "· 实连合计明显 < $EXPECTED 或 30s 采样掉下去 → 闪断/代理/限流"
echo "· disconnected 某平台特别多 → 查该平台；少数 map_key → 单场"
echo "· 错峰后至少等 2～3 分钟再对比实连"
echo "完成."