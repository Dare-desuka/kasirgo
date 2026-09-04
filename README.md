<div align="center">
  <h1>KasirGo</h1>
  <p><strong>Aplikasi kasir (Point of Sale) offline untuk warung & toko sembako</strong></p>
  <p>
    <a href="#-fitur-utama">Fitur</a> •
    <a href="#-mulai-cepat">Mulai Cepat</a> •
    <a href="#-akses-dari-hp-lan">Akses HP</a> •
    <a href="#-konfigurasi">Konfigurasi</a> •
    <a href="#-arsitektur">Arsitektur</a> •
    <a href="#-build">Build</a>
  </p>
</div>

---

Satu file executable berisi backend Go + frontend web + database SQLite.
Tanpa internet, tanpa instalasi rumit — cukup jalankan, browser terbuka
sendiri, langsung bisa kasir.

- **Uang dalam integer Rupiah**, semua hitungan (subtotal, diskon, kembalian)
  dikerjakan di server. Frontend tidak dipercaya.
- **Database tidak tergantung working directory** — tersimpan di folder data
  aplikasi per OS, pindah executable ke mana pun data tetap sama.
- **Transaksi atomic** — transaksi + item + pengurangan stok + riwayat stok
  tersimpan dalam satu transaksi database (gagal satu = batal semua).

---

## Fitur Utama

| Area | Detail |
|------|--------|
| Kasir | Cari produk (nama/barcode/SKU), scanner barcode USB otomatis masuk keranjang, ubah jumlah, diskon, 4 metode bayar (Cash/Transfer/QRIS/Debit), hitung kembalian otomatis, pratinjau & cetak struk |
| Produk & Stok | CRUD produk & kategori, barcode/SKU, harga beli/jual, stok minimum, satuan bebas, penyesuaian stok per barcode beserta riwayat pergerakan |
| Barcode tak dikenal | Scan barcode baru langsung membuka form Produk Baru dengan barcode terisi |
| Scan kamera | Pindai barcode lewat kamera browser (API `BarcodeDetector` bawaan, tanpa library tambahan) |
| Dasbor | Kartu Keuntungan Hari Ini, diagram donat keuntungan vs modal, stok hampir habis/habis, transaksi terbaru |
| Laporan | Penjualan (harian/mingguan/bulanan/rentang bebas, termasuk produk terlaris) dan stok real-time |
| Keamanan LAN | Akses dari HP dikunci PIN (`X-LAN-PIN`); desktop via loopback bebas PIN |
| Data | Hapus transaksi per rentang tanggal atau semuanya; backup konsisten via `VACUUM INTO`; restore dari file snapshot (validasi SQLite + rollback otomatis bila gagal) |
| Tampilan | Mode gelap/terang (mengikuti OS, pilihan manual tersimpan), 3 tata letak otomatis: Desktop (>1100px), Tablet (641–1100px), HP (≤640px) |
| Shortcut | `F2` cari produk, `F4` bayar, `ESC` tutup modal, `Enter` konfirmasi |

---

## Mulai Cepat

### Prasyarat

- Go **1.26+** — hanya untuk build dari source. Pemakaian harian cukup file executable, tanpa Go, Node.js, Docker, atau internet.

### Dari source (development)

```bash
git clone https://github.com/Dare-desuka/kasirgo.git
cd kasirgo

go mod download
go run -buildvcs=false ./cmd/pos
```

Buka browser → `http://localhost:2001` (browser biasanya terbuka otomatis).

### Dari binary (pemakaian harian)

```bash
./run.sh            # Linux: cek binary, lalu jalankan (Ctrl+C untuk berhenti)
```

Di Windows: cukup double-click `pos-go.exe` (buka browser otomatis) atau
`KasirGo.exe` (jendela native tanpa browser — lihat bawah).

Contoh output saat server jalan:

```text
Database: /home/user/.local/share/pos-go/data.db
Server address: 127.0.0.1:2001
POS Go started
Local:    http://localhost:2001
Network:  http://192.168.1.50:2001/#/stok
```

> Tutup terminal = server berhenti (graceful shutdown via
> SIGHUP/SIGINT/SIGTERM).

---

## Akses dari HP (LAN)

1. Jalankan server dengan bind ke semua interface:

```bash
./dist/pos-go-linux-amd64 --host 0.0.0.0
# atau via env: POS_HOST=0.0.0.0 ./dist/pos-go-linux-amd64
```

2. Di PC, buka **Pengaturan → kartu Akses HP**: tampil URL live sesuai IP
   laptop saat ini (diambil dari `GET /api/network`, tidak disimpan) plus
   kolom untuk mengatur **PIN LAN** (`lan_pin`).
3. Di HP (satu Wi-Fi yang sama), buka URL tersebut, masukkan PIN saat diminta.
   PIN dikirim via header `X-LAN-PIN` (`POST /api/unlock` untuk verifikasi awal).

Aturan kunci:

- Akses dari PC sendiri (loopback) **bebas PIN**.
- Akses non-loopback wajib PIN bila PIN diisi; PIN kosong = tanpa kunci.
- Override darurat via env `POS_PIN` (mengalahkan setting database).

---

## Konfigurasi

### Lokasi database (otomatis per OS)

| OS | Path |
|----|------|
| Linux | `~/.local/share/pos-go/data.db` |
| Windows | `%APPDATA%\pos-go\data.db` |

Path absolut dicetak di log setiap startup.

### Flag & environment variable

| Nama | Bentuk | Deskripsi | Default |
|------|--------|-----------|---------|
| `--host` | flag | Host listen (`0.0.0.0` = buka akses LAN) | `127.0.0.1` (`POS_HOST` menang atas flag bila keduanya diisi... lihat catatan) |
| `--port` | flag | Port HTTP | `2001` |
| `--no-browser` | flag | Matikan buka-browser-otomatis | mati (browser dibuka) |
| `POS_HOST` | env | Sama dengan `--host` (prioritas tertinggi) | `127.0.0.1` |
| `POS_PORT` | env | Sama dengan `--port` (prioritas tertinggi) | `2001` |
| `POS_PIN` | env | Override PIN LAN (mengalahkan `lan_pin` di database) | (kosong) |
| `POS_NO_BROWSER` | env | `1` = jangan buka browser otomatis | (kosong) |
| `POS_NO_SHORTCUT` | env | `1` = jangan buat shortcut desktop (Windows) | (kosong) |

> Urutan prioritas host/port: env → flag → default.

---

## Arsitektur

```text
┌──────────────────────────────────────────────────────────┐
│                      Single Binary                       │
├──────────────────────────────────────────────────────────┤
│  cmd/pos                 → server + browser otomatis     │
│  cmd/kasirgo-webview     → jendela native Windows        │
│  internal/api/           → handler REST + backup/restore │
│                            + hapus transaksi + kunci PIN │
│  internal/service/       → validasi checkout, cari produk│
│  internal/repository/    → akses DB (transaksi atomic,   │
│                            invoice, laporan, dasbor)     │
│  internal/models/        → struct Produk, Transaksi, dll │
│  internal/database/      → koneksi SQLite + migrasi SQL  │
│  internal/system/        → path data per OS + shortcut   │
│  web/                    → frontend (embed ke binary)    │
│    static/               → SPA vanilla (hash routing)    │
└──────────────────────────────────────────────────────────┘
```

### Teknologi

| Lapisan | Teknologi |
|---------|-----------|
| Bahasa | Go 1.26+ |
| Database | SQLite via `modernc.org/sqlite` (murni Go, tanpa CGO → aman cross-compile) |
| Frontend | HTML/CSS/JS vanilla tanpa framework & tanpa npm (di-embed ke binary) |
| Jendela native | WebView2 via `github.com/jchv/go-webview2` (Windows saja, butuh runtime Edge WebView2) |
| Ikon `.exe` | `kasirgo.ico` dari `assets/icon.jpg`, di-embed via `go-winres` |

### Skema database (ringkas)

```sql
products (id, barcode, sku, name, category_id, purchase_price, selling_price,
          stock, minimum_stock, unit, created_at, updated_at, deleted_at)
transactions (id, invoice_number, subtotal, discount, total, paid, change,
              payment_method, cashier, created_at)
transaction_items (id, transaction_id, product_id, product_name, quantity, price, subtotal)
stock_movements (id, product_id, type, quantity, reference_id, note, created_at)
categories, settings
```

### API (ringkas)

```text
GET/POST      /api/products            PUT/DELETE /api/products/:id
GET           /api/products/search/:q   (barcode/SKU/nama; tak ketemu → {"error": ...})
GET/POST      /api/categories           PUT/DELETE /api/categories/:id
POST          /api/transactions         (checkout atomic)
GET           /api/transactions[?from&to]   GET /api/transactions/:id
DELETE        /api/transactions[?from&to]   (hapus per rentang YYYY-MM-DD / semua)
POST          /api/stock/adjust         GET /api/stock/movements
GET           /api/reports/sales?from&to (termasuk produk terlaris)
GET           /api/reports/stock        GET /api/dashboard
GET/PUT       /api/settings             (termasuk lan_pin)
GET           /api/backup/download      POST /api/restore (field form: backup)
GET           /api/network              POST /api/unlock (verifikasi PIN HP)
```

---

## Build

```bash
./build.sh    # build 5 target → dist/ (termasuk generate ikon bila tool tersedia)
```

Hasil di `dist/`:

| File | Keterangan |
|------|------------|
| `pos-go-linux-amd64` | Linux x86_64 (server + browser otomatis) |
| `pos-go-linux-arm64` | Linux ARM64 (Raspberry Pi, dll) |
| `pos-go.exe` | Windows x86_64 (server + browser otomatis) |
| `pos-go-arm64.exe` | Windows ARM64 |
| `kasirgo-webview.exe` | Windows x86_64, jendela native Edge WebView2: tanpa browser, tanpa console hitam, port tetap `0.0.0.0:2001`, tutup jendela = server berhenti |

Catatan build:

- Semua perintah build/test/vet memakai `-buildvcs=false`.
- `modernc.org/sqlite` murni Go sehingga cross-compile Windows dari Linux
  tanpa CGO/mingw.
- Pembuatan ikon butuh `go-winres` (`~/go/bin/go-winres`) dan ImageMagick
  (`magick`); sumber ikon `assets/icon.jpg`. Tanpa keduanya, build tetap
  jalan tanpa ikon baru.

Build manual (tanpa `./build.sh`):

```bash
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -o dist/pos-go-linux-amd64 ./cmd/pos
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -o dist/pos-go-linux-arm64 ./cmd/pos
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -o dist/pos-go.exe         ./cmd/pos
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -o dist/pos-go-arm64.exe   ./cmd/pos
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-H windowsgui" -o dist/kasirgo-webview.exe ./cmd/kasirgo-webview
```

### Windows (.exe)

| Fitur | Detail |
|-------|--------|
| Ikon | Tertanam di `.exe` (taskbar, shortcut, alt-tab) |
| Browser otomatis | Terbuka saat double-click (`--no-browser` untuk mematikan) |
| Shortcut desktop | Dibuat otomatis saat pertama jalan (console diminimize) |
| Database | `%APPDATA%\pos-go\data.db` |
| Firewall | Windows meminta izin akses jaringan saat pertama jalan |

> Untuk user non-teknis: kirim `pos-go.exe` via WhatsApp/USB → taruh di
> Desktop → double-click → browser terbuka → siap kasir.

### Linux (systemd, opsional)

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

### File turunan di `dist/` (bukan output `build.sh`)

- **`KasirGo.exe`** — salinan rename dari `kasirgo-webview.exe` (nama ramah
  user untuk dibagikan ke Windows).
- **`KasirGo-stok.apk`** — aplikasi HP pendamping **khusus isi/update stok
  via kamera**, di-build dari proyek sebelah (`~/Projects/kasirgo-stok`,
  Expo + EAS) dan ditaruh di sini agar satu tempat dengan installer PC.

---

## Testing

```bash
go test -buildvcs=false ./...          # semua test
go test -buildvcs=false -cover ./...   # dengan coverage
go vet -buildvcs=false ./...           # vet
gofmt -l -w .                          # format
```

Cakupan test:

- Hitungan uang (subtotal, diskon, total, kembalian; non-tunai kembalian nol)
- Validasi stok & rollback saat stok kurang / DB error
- Lookup barcode/SKU, keunikan barcode, soft-delete produk
- Path database absolut & tidak tergantung working directory
- Migrasi idempotent & persistensi lintas restart
- Integrasi penuh: produk → transaksi → stok berkurang → movement tercatat → hapus transaksi

---

## Lisensi

Lisensi MIT — bebas dipakai, diubah, dan disebarkan.

## Kontribusi

1. Fork repo ini
2. Buat branch fitur (`git checkout -b feat/nama-fitur`)
3. Commit (`git commit -m 'feat: tambah fitur X'`)
4. Push & buat Pull Request

---

<div align="center">
  <p>Dibuat untuk UMKM Indonesia</p>
  <p><a href="https://github.com/Dare-desuka/kasirgo/issues">Laporkan Bug</a> • <a href="https://github.com/Dare-desuka/kasirgo/discussions">Diskusi</a></p>
</div>
