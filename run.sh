#!/usr/bin/env bash
# KasirGo runner — foreground, Ctrl+C / tutup terminal = server mati
set -e
cd "$(dirname "$0")"

BIN="dist/pos-go-linux-amd64"
if [[ ! -x "$BIN" ]]; then
  echo "❌ Binary belum di-build: $BIN"
  echo "   Jalankan dulu: ./build.sh"
  exit 1
fi

echo "=== KasirGo starting ==="
echo "   Binary: $BIN"
echo "   Tekan Ctrl+C atau tutup terminal untuk berhenti."
echo ""
exec "$BIN" "$@"