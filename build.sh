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

echo
echo "Done. Binaries in ./dist/"
ls -la dist/