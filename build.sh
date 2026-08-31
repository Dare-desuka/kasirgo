#!/usr/bin/env bash
set -euo pipefail

# Build pos-go for Linux and Windows. Output goes to ./dist/
cd "$(dirname "$0")"
mkdir -p dist

# Generate icon dari assets/icon.jpg → .ico → .syso (kedua cmd/pos & cmd/kasirgo-webview)
GO_WINRES=~/go/bin/go-winres
if [[ -f assets/icon.jpg && -x "$GO_WINRES" ]]; then
  echo "==> Generating icon from assets/icon.jpg"
  # Convert jpg → ico multi-size (16,32,48,256)
  magick assets/icon.jpg -resize 256x256 /tmp/_icon_256.png
  magick assets/icon.jpg -resize 48x48 /tmp/_icon_48.png
  magick assets/icon.jpg -resize 32x32 /tmp/_icon_32.png
  magick assets/icon.jpg -resize 16x16 /tmp/_icon_16.png
  magick /tmp/_icon_256.png /tmp/_icon_48.png /tmp/_icon_32.png /tmp/_icon_16.png assets/kasirgo.ico
  rm -f /tmp/_icon_*.png

  # Generate .syso untuk cmd/pos (amd64 + arm64)
  cd cmd/pos
  "$GO_WINRES" simply --icon ../../assets/kasirgo.ico --arch amd64,arm64 --manifest gui \
    --product-name "KasirGo" --file-description "KasirGo POS"
  cd ../..

  # Generate .syso untuk cmd/kasirgo-webview (amd64 only)
  cd cmd/kasirgo-webview
  "$GO_WINRES" simply --icon ../../assets/kasirgo.ico --arch amd64 --manifest gui \
    --product-name "KasirGo" --file-description "KasirGo POS WebView"
  cd ../..

  echo "    Icon generated: assets/kasirgo.ico + .syso files"
fi

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
