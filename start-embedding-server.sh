#!/usr/bin/env bash
# =============================================================
# WeKnora 本地嵌入服务管理脚本（fastembed / bge-small-zh-v1.5, 端口 8791）
#
# 用法:
#   ./start-embedding-server.sh [start|stop|status|restart]
#
# 说明:
#   - venv 在 .embed-venv/（缺失时自动用 uv + 阿里云 PyPI 源安装）
#   - 模型首次启动从 hf-mirror.com 下载 ~90MB，之后走本地缓存
#   - WeKnora 模型配置: base_url=http://127.0.0.1:8791/v1 (provider=generic)
#   - 日志: .embedding-server.log（仓库根目录）
# =============================================================
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_PY="$ROOT/.embed-venv/bin/python"
LOG="$ROOT/.embedding-server.log"
PID_FILE="$ROOT/.embedding-server.pid"
PORT="${EMBED_PORT:-8791}"

port_pids() { lsof -t -nP -iTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true; }

start() {
  PIDS=$(port_pids)
  if [ -n "$PIDS" ]; then
    echo "✅ 嵌入服务已在运行 (端口 $PORT, pid: $(echo $PIDS | tr '\n' ' '))"
    exit 0
  fi
  if [ ! -x "$VENV_PY" ]; then
    echo "⚠️  .embed-venv 不存在，用 uv 安装依赖（阿里云源）..."
    (cd "$ROOT" \
      && UV_CACHE_DIR="$ROOT/.uv-cache" UV_DEFAULT_INDEX="https://mirrors.aliyun.com/pypi/simple/" \
         uv venv .embed-venv \
      && UV_CACHE_DIR="$ROOT/.uv-cache" UV_DEFAULT_INDEX="https://mirrors.aliyun.com/pypi/simple/" \
         uv pip install --python "$VENV_PY" fastembed fastapi uvicorn)
  fi
  echo "启动嵌入服务 (:$PORT)，首次会下载模型约 90MB ..."
  cd "$ROOT"
  HF_ENDPOINT="${HF_ENDPOINT:-https://hf-mirror.com}" \
  nohup "$VENV_PY" "$ROOT/embedding_server.py" >>"$LOG" 2>&1 &
  echo $! >"$PID_FILE"
  disown || true
  for _ in $(seq 1 120); do
    sleep 1
    if [ -n "$(port_pids)" ]; then
      echo "✅ 嵌入服务已就绪: 127.0.0.1:$PORT (pid $(cat "$PID_FILE"), 日志 $LOG)"
      exit 0
    fi
  done
  echo "❌ 120 秒内未监听 $PORT，最近日志："
  tail -20 "$LOG"
  exit 1
}

stop() {
  PIDS=$(port_pids)
  if [ -n "$PIDS" ]; then
    kill $PIDS 2>/dev/null || true
    echo "🛑 已停止 (pid: $(echo $PIDS | tr '\n' ' '))"
  else
    echo "嵌入服务未在运行"
  fi
  rm -f "$PID_FILE"
}

status() {
  PIDS=$(port_pids)
  if [ -n "$PIDS" ]; then
    echo "✅ 运行中 (端口 $PORT, pid: $(echo $PIDS | tr '\n' ' '))"
  else
    echo "⭕ 未运行"
  fi
  [ -f "$LOG" ] && echo "--- 最近日志 ---" && tail -3 "$LOG"
}

case "${1:-start}" in
  start) start ;;
  stop) stop ;;
  restart) stop; sleep 1; start ;;
  status) status ;;
  *) echo "用法: $0 [start|stop|status|restart]"; exit 1 ;;
esac
