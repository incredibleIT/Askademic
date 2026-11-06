#!/usr/bin/env bash
# =============================================================
# WeKnora 本地开发环境总控脚本
#
# 一键拉起全部本地服务（幂等，已在运行会跳过）:
#   ./start-weknora.sh            # 启动全部
#   ./start-weknora.sh status     # 查看全部状态
#   ./start-weknora.sh stop       # 停止本地进程（不动 Docker）
#
# 组件:
#   1. PostgreSQL + Redis        → Docker (docker-compose.dev-academic.yml)
#   2. weknora-server :8080      → ./weknora-server 二进制 (ENV_FILE=.env)
#   3. docreader      :50051     → docreader/.venv (调 start-docreader.sh)
#   4. embedding      :8791      → .embed-venv + fastembed bge-small-zh-v1.5
#
# 前端: http://127.0.0.1:8080  (WEKNORA_WEB_DIR=frontend/dist)
# =============================================================
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

port_pids() { lsof -t -nP -iTCP:"$1" -sTCP:LISTEN 2>/dev/null || true; }

ensure_docker() {
  local missing=0
  for c in weknora-pg weknora-redis; do
    docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$c" || missing=1
  done
  if [ "$missing" = "1" ]; then
    echo "启动 Docker 依赖 (pg + redis)..."
    docker compose -f docker-compose.dev-academic.yml up -d >/dev/null 2>&1
    sleep 3
  fi
  docker ps --format '{{.Names}}' 2>/dev/null | grep -qx weknora-pg && echo "✅ PostgreSQL (docker)" || echo "❌ PostgreSQL 未启动"
  docker ps --format '{{.Names}}' 2>/dev/null | grep -qx weknora-redis && echo "✅ Redis (docker)" || echo "❌ Redis 未启动"
}

start_server() {
  if [ -n "$(port_pids 8080)" ]; then
    echo "✅ weknora-server 运行中 (:8080)"
    return 0
  fi
  echo "启动 weknora-server (:8080) ..."
  ENV_FILE=.env nohup ./weknora-server > .weknora-server.log 2>&1 &
  echo $! > .weknora-server.pid
  disown || true
  for _ in $(seq 1 20); do
    sleep 1
    [ -n "$(port_pids 8080)" ] && { echo "✅ weknora-server 已就绪 (日志 .weknora-server.log)"; return 0; }
  done
  echo "❌ weknora-server 启动失败："; tail -10 .weknora-server.log
  return 1
}

stop_server() {
  local pids; pids=$(port_pids 8080)
  if [ -n "$pids" ]; then kill $pids 2>/dev/null || true; echo "🛑 weknora-server 已停止"; else echo "weknora-server 未在运行"; fi
  rm -f .weknora-server.pid
}

case "${1:-start}" in
  start)
    ensure_docker
    ./start-docreader.sh start
    ./start-embedding-server.sh start
    start_server
    echo ""
    echo "🌐 前端入口: http://127.0.0.1:8080"
    ;;
  stop)
    stop_server
    ./start-docreader.sh stop
    ./start-embedding-server.sh stop
    echo "(Docker 容器未动；需要停用: docker compose -f docker-compose.dev-academic.yml stop)"
    ;;
  status)
    [ -n "$(port_pids 8080)" ]  && echo "✅ weknora-server :8080" || echo "⭕ weknora-server :8080"
    [ -n "$(port_pids 50051)" ] && echo "✅ docreader      :50051" || echo "⭕ docreader      :50051"
    [ -n "$(port_pids 8791)" ]  && echo "✅ embedding      :8791" || echo "⭕ embedding      :8791"
    docker ps --format '{{.Names}}' 2>/dev/null | grep -qx weknora-pg    && echo "✅ PostgreSQL (docker)" || echo "⭕ PostgreSQL"
    docker ps --format '{{.Names}}' 2>/dev/null | grep -qx weknora-redis && echo "✅ Redis (docker)"      || echo "⭕ Redis"
    ;;
  *)
    echo "用法: $0 [start|stop|status]"
    ;;
esac
