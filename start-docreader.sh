#!/usr/bin/env bash
# =============================================================
# WeKnora DocReader 本地服务管理脚本（文档解析 gRPC, 端口 50051）
#
# 用法:
#   ./start-docreader.sh start     启动（默认，幂等：已在运行则跳过）
#   ./start-docreader.sh stop      停止
#   ./start-docreader.sh restart   重启
#   ./start-docreader.sh status    查看状态与最近日志
#
# 说明:
#   - 必须从仓库根目录以 `python -m docreader.main` 方式启动
#     （main.py 在 docreader 包内部，直接跑 main.py 会 ImportError）
#   - 依赖 docreader/.venv（缺失时自动用 uv + 阿里云 PyPI 源安装）
#   - 日志: .docreader.log（仓库根目录）
#   - WeKnora 主服务通过 .env 的 DOCREADER_ADDR=127.0.0.1:50051 连接
# =============================================================
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DC_DIR="$ROOT/docreader"
VENV_PY="$DC_DIR/.venv/bin/python"
LOG="$ROOT/.docreader.log"
PID_FILE="$ROOT/.docreader.pid"
PORT="${DOCREADER_GRPC_PORT:-50051}"

port_pids() { lsof -t -nP -iTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true; }

start() {
  PIDS=$(port_pids)
  if [ -n "$PIDS" ]; then
    echo "✅ DocReader 已在运行 (端口 $PORT, pid: $(echo $PIDS | tr '\n' ' '))"
    exit 0
  fi
  if [ ! -x "$VENV_PY" ]; then
    echo "⚠️  docreader/.venv 不存在，用 uv 安装依赖（阿里云源）..."
    (cd "$DC_DIR" && UV_DEFAULT_INDEX="https://mirrors.aliyun.com/pypi/simple/" uv sync)
  fi
  echo "启动 DocReader (gRPC :$PORT) ..."
  cd "$ROOT"
  nohup "$VENV_PY" -m docreader.main >>"$LOG" 2>&1 &
  echo $! >"$PID_FILE"
  disown || true
  for _ in $(seq 1 30); do
    sleep 1
    if [ -n "$(port_pids)" ]; then
      echo "✅ DocReader 已就绪: 127.0.0.1:$PORT (pid $(cat "$PID_FILE"), 日志 $LOG)"
      exit 0
    fi
  done
  echo "❌ 30 秒内未监听 $PORT，最近日志："
  tail -20 "$LOG"
  exit 1
}

stop() {
  PIDS=$(port_pids)
  if [ -n "$PIDS" ]; then
    kill $PIDS 2>/dev/null || true
    echo "🛑 已停止 (pid: $(echo $PIDS | tr '\n' ' '))"
  else
    echo "DocReader 未在运行"
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
