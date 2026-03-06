#!/usr/bin/env bash
# 快速搭建 env-king 脚手架
# 用法: ./scripts/scaffold.sh [blome|openclaw]
# 可从参考项目复制部分结构到 env-king

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REF="${1:-}"

echo "Env King Scaffold - ROOT=$ROOT"

# 1. 确保 web 目录存在
if [ ! -d "$ROOT/web" ]; then
  echo "Creating web with pnpm create vite..."
  cd "$ROOT" && pnpm create vite@latest web --template react-ts
  cd "$ROOT/web" && pnpm install
fi

# 2. 若指定参考项目，复制部分文件
if [ -n "$REF" ]; then
  REF_ROOT="$(dirname "$ROOT")/$REF"
  if [ -d "$REF_ROOT" ]; then
    echo "Reference project: $REF_ROOT"
    case "$REF" in
      blome)
        [ -d "$REF_ROOT/web2/src/components/ui" ] && \
          cp -r "$REF_ROOT/web2/src/components/ui"/* "$ROOT/web/src/components/ui/" 2>/dev/null || true
        echo "Copied blome UI components (if any)"
        ;;
      openclaw)
        echo "OpenClaw structure reference - see openclaw/ for skills/plugins"
        ;;
    esac
  fi
fi

# 3. 确保目录结构
mkdir -p "$ROOT/web/src/components/ui"
mkdir -p "$ROOT/web/src/pages"
mkdir -p "$ROOT/web/src/api"
mkdir -p "$ROOT/internal/api"

echo "Done. Run: cd web && pnpm dev  (frontend) and ./env-king server (backend)"
