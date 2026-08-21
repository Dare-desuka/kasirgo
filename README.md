<div align="center">
  <h1>🛒 KasirGo</h1>
  <p><strong>Aplikasi POS (Point of Sale) Offline-First untuk Warung & Toko Sembako</strong></p>
  <p>
    <a href="#-fitur-utama">Fitur</a> •
    <a href="#-quick-start">Quick Start</a> •
    <a href="#-development">Development</a> •
    <a href="#-deployment">Deployment</a> •
    <a href="#-arsitektur">Arsitektur</a>
  </p>
</div>

---

## 🌟 Fitur Utama

| Area | Detail |
|------|--------|
| **🛍️ Kasir** | Scan barcode (USB/keyboard), keranjang, diskon, 4 metode bayar (Tunai/Transfer/QRIS/Debit), hitung kembalian otomatis, struk print/PDF |
| **📦 Produk & Stok** | CRUD produk, kategori, barcode/SKU, harga beli/jual, stok minimum, satuan custom, penyesuaian stok, riwayat pergerakan |
| **📊 Dashboard** | Penjualan hari ini, donut keuntungan vs modal (conic-gradient), produk hampir habis/habis, transaksi terbaru |
| **📈 Laporan** | Penjualan harian/mingguan/bulan/custom, produk terlaris, stok real-time |
| **⚙️ Transaksi Atomic** | Insert transaksi + item + kurangi stok + catat pergerakan stok dalam 1 transaksi DB (rollback penuh jika gagal) |
| **💾 Backup & Restore** | Snapshot konsisten via `VACUUM INTO`, restore aman (validasi SQLite, rollback otomatis jika gagal) |
| **🎨 Tema & Responsif** | Dark/Light mode (toggle pojok kanan atas, persist localStorage), 3 mode layout: Desktop (>1100px), Tablet (641–1100px), HP (≤640px) |
| **⌨️ Shortcut Keyboard** | `F2` cari produk, `F4` bayar, `ESC` tutup modal, `Enter` konfirmasi |
| **🪟 Windows Native** | Icon embed ke `.exe`, auto-buka browser, shortcut desktop otomatis (minimized console) |

---

## 📸 Screenshots

<div align="center">
  <table>
    <tr>
      <td align="center"><strong>Dashboard</strong><br/><img src="docs/screenshots/dashboard.png" width="300"/></td>
      <td align="center"><strong>Kasir</strong><br/><img src="docs/screenshots/kasir.png" width="300"/></td>
      <td align="center"><strong>Produk</strong><br/><img src="docs/screenshots/produk.png" width="300"/></td>
    </tr>
    <tr>
      <td align="center"><strong>Laporan</strong><br/><img src="docs/screenshots/laporan.png" width="300"/></td>
      <td align="center"><strong>Pengaturan</strong><br/><img src="docs/screenshots/pengaturan.png" width="300"/></td>
      <td align="center"><strong>Mobile</strong><br/><img src="docs/screenshots/mobile.png" width="300"/></td>
    </tr>
  </table>
</div>

> 📝 *Screenshots akan ditambahkan nanti. Placeholder di atas untuk referensi.*

---

## 🚀 Quick Start

### Prasyarat
- Go **1.22+** (hanya untuk build; runtime hanya butuh executable)
- Tidak perlu Node.js, Python, Docker, atau internet saat runtime

### Jalankan (Development)

```bash
# Clone & masuk folder
git clone https://github.com/Dare-desuka/kasirgo.git
cd kasirgo

# Download dependency & jalankan
go mod download
go run ./cmd/pos
```

Buka browser → `http://localhost:8080`

### Akses dari HP (LAN)

```bash
# Bind ke semua interface
./pos-go-linux-amd64 --host 0.0.0.0

# Atau via env
POS_HOST=0.0.0.0 ./pos-go-linux-amd64
```

Server akan menampilkan IP LAN saat startup:
```text
POS Go started
Local:    http://localhost:8080
Network:  http://192.168.1.50:8080
```
Buka URL **Network** dari HP di jaringan yang sama.

---

## 🏗️ Development

```bash
# Setup
go mod download

# Test & lint
go test ./...
go vet ./...

# Run server
go run ./cmd/pos

# Build cepat (linux-amd64)
go build -buildvcs=false -o dist/pos-go-linux-amd64 ./cmd/pos

# Build semua target (Linux + Windows)
./build.sh    # Output di ./dist/
```

### Perintah berguna
```bash
# Test dengan coverage
go test -cover ./...

# Format code
gofmt -l -w .

# Lint
golangci-lint run
```

---

## 📦 Deployment

### Binary Tunggal (Recommended)

Hanya 1 file executable — **tanpa dependency**, tanpa installer.

```bash
# Build semua platform
./build.sh

# Output di ./dist/
# pos-go-linux-amd64     # Linux x86_64
# pos-go-linux-arm64     # Linux ARM64 (Raspberry Pi, dll)
# pos-go.exe             # Windows x86_64
# pos-go-arm64.exe       # Windows ARM64
```

### Windows (.exe)

| Fitur | Detail |
|-------|--------|
| **Icon** | Embed di `.exe` (taskbar, shortcut, alt-tab) |
| **Auto Browser** | Buka browser otomatis saat double-click (`--no-browser` untuk disable) |
| **Shortcut Desktop** | Otomatis dibuat di *first run* (minimized console) |
| **Database** | `%APPDATA%\pos-go\data.db` |
| **Firewall** | Minta izin akses jaringan saat pertama kali jalan |

> 💡 **Untuk user non-teknis (tante, warung):**  
> Kirim `pos-go.exe` via WhatsApp/USB → taruh di Desktop → double-click → browser terbuka otomatis → siap kasir.

### Linux (systemd service)

```ini
# /etc/systemd/system/kasirgo.service
[Unit]
Description=KasirGo POS Server
After=network.target

[Service]
Type=simple
User=kasir
WorkingDirectory=/opt/kasirgo
ExecStart=/opt/kasirgo/pos-go-linux-amd64 --host 0.0.0.0
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now kasirgo
```

---

## 🗄️ Database & Konfigurasi

### Lokasi Database (Otomatis per OS)

| OS | Path |
|----|------|
| Linux | `~/.local/share/pos-go/data.db` |
| Windows | `%APPDATA%\pos-go\data.db` |

> **Penting:** Database **tidak bergantung pada working directory**. Pindahkan executable ke folder manapun, database tetap sama. Path absolut dicetak di log saat startup.

### Environment Variables

| Variable | Deskripsi | Default |
|----------|-----------|---------|
| `POS_HOST` | Host listen (`0.0.0.0` = LAN) | `127.0.0.1` |
| `POS_PORT` | Port HTTP | `8080` |
| `POS_NO_BROWSER` | `1` = disable auto-open browser | (unset) |
| `POS_NO_SHORTCUT` | `1` = disable shortcut desktop | (unset) |

---

## 🏛️ Arsitektur

```
┌─────────────────────────────────────────────────────────────┐
│                      Single Binary                          │
├─────────────────────────────────────────────────────────────┤
│  cmd/pos/main.go          → Entry point, wiring, LAN detect │
│  internal/api/            → REST handlers, static files     │
│  internal/service/        → Business logic, validation      │
│  internal/repository/     → DB access (transactions, repo)  │
│  internal/models/         → Structs (Product, Transaction)  │
│  internal/database/       → SQLite connection & migration   │
│  internal/system/         → App dir, platform paths         │
│  web/static/              → Frontend (HTML/CSS/JS vanilla)  │
└─────────────────────────────────────────────────────────────┘
```

### Stack Teknologi
| Layer | Teknologi |
|-------|-----------|
| Language | Go 1.22+ |
| Database | SQLite (`modernc.org/sqlite` — pure Go, no CGO) |
| Frontend | Vanilla HTML/CSS/JS (ES6 modules, hash routing) |
| Build | `go build` (CGO_ENABLED=0, cross-compile ready) |
| Icons | Embedded via `go-winres` (`.syso` resource) |

### Database Schema (Ringkas)
```sql
products (id, barcode, sku, name, category_id, purchase_price, selling_price,
          stock, minimum_stock, unit, created_at, updated_at, deleted_at)
transactions (id, invoice_number, subtotal, discount, total, paid, change,
              payment_method, cashier, created_at)
transaction_items (id, transaction_id, product_id, product_name, quantity, price, subtotal)
stock_movements (id, product_id, type, quantity, reference_id, note, created_at)
categories, settings
```

---

## 🧪 Testing

```bash
# Semua test
go test ./... -v

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Cakupan test:**
- Perhitungan uang (subtotal, diskon, total, kembalian, pembulatan)
- Validasi stok (cukup/tidak, rollback otomatis)
- Lookup barcode/SKU (exact match + prefix)
- Database path absolut (tidak tergantung cwd)
- Migration idempotent (aman dijalankan berulang)
- Integrasi: produk → transaksi → stok turun → movement tercatat
- Rollback saat stok tidak cukup / error DB
- Persistensi lintas restart server

---

## 🔧 Build Cross-Platform

```bash
# Otomatis (Linux + Windows)
./build.sh

# Manual
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o pos-go-linux-amd64     ./cmd/pos
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o pos-go-linux-arm64     ./cmd/pos
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o pos-go.exe             ./cmd/pos
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o pos-go-arm64.exe      ./cmd/pos
```

**Output:** `dist/pos-go-linux-amd64`, `dist/pos-go-linux-arm64`, `dist/pos-go.exe`, `dist/pos-go-arm64.exe`

> **Catatan:** `modernc.org/sqlite` (pure Go) memungkinkan cross-compile Windows dari Linux tanpa CGO/mingw.

---

## 📄 Lisensi

MIT License — bebas digunakan, dimodifikasi, dan didistribusikan.

---

## 🤝 Kontribusi

1. Fork repo
2. Buat branch fitur (`git checkout -b feat/nama-fitur`)
3. Commit perubahan (`git commit -m 'feat: tambah fitur X'`)
4. Push ke branch (`git push origin feat/nama-fitur`)
5. Buat Pull Request

---

<div align="center">
  <p>Dibuat dengan ❤️ untuk UMKM Indonesia</p>
  <p><a href="https://github.com/Dare-desuka/kasirgo/issues">Laporkan Bug</a> • <a href="https://github.com/Dare-desuka/kasirgo/discussions">Diskusi</a></p>
</div>