#!/usr/bin/env bash
set -euo pipefail

# Build pos-go for Linux and Windows. Output goes to ./dist/
cd "$(dirname "$0")"
mkdir -p dist

build() {
  local os=$1 arch=$2 out=$3
  echo "==> $os/$arch -> dist/$out"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -buildvcs=false -trimpath -o "dist/$out" ./cmd/pos
}

build linux amd64  pos-go-linux-amd64
build linux arm64  pos-go-linux-arm64
build windows amd64 pos-go.exe
build windows arm64 pos-go-arm64.exe

# WebView2 (Windows only) — pure Go, no CGO, pakai Edge WebView2
echo "==> windows/amd64 -> dist/kasirgo-webview.exe (WebView2)"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-H windowsgui" -o "dist/kasirgo-webview.exe" ./cmd/kasirgo-webview

echo
echo "Done. Binaries in ./dist/"
ls -la dist/
