#!/usr/bin/env bash
# 一键构建 mihomo 三平台二进制到 ./workdir/mihomo/
# 支持: darwin(amd64/arm64), linux(amd64/arm64), windows(amd64)

set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
OUT="${ROOT}/workdir/mihomo"
MIHOMO_DIR="${ROOT}/mihomo"

mkdir -p "$OUT"

# 平台列表: GOOS/GOARCH
# darwin-arm64 (Apple Silicon), darwin-amd64 (Intel Mac)
# linux-amd64, linux-arm64
# windows-amd64
platforms=(
  "darwin:arm64"
  "darwin:amd64"
  "linux:amd64"
  "linux:arm64"
  "windows:amd64"
)

echo "==> 构建 mihomo 到 $OUT"
cd "$MIHOMO_DIR"

for p in "${platforms[@]}"; do
  GOOS="${p%%:*}"
  GOARCH="${p##*:}"
  BIN="mihomo-${GOOS}-${GOARCH}"
  [[ "$GOOS" == "windows" ]] && BIN="${BIN}.exe"
  echo "  $GOOS/$GOARCH -> $BIN"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="-s -w" -o "${OUT}/${BIN}" .
done

echo "==> 完成: $(ls -la "$OUT"/mihomo-* 2>/dev/null | wc -l | tr -d ' ') 个二进制"
