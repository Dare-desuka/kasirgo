# AGENTS.md

## Aturan wajib

- **Setiap perubahan/prompt WAJIB memuat dan mengikuti skill `ponytail`** (lazy
  mode: solusi paling sederhana yang benar). Muat skill dengan tool `skill`
  sebelum mulai bekerja. Jangan tambah abstraksi, boilerplate, atau dependency
  yang tidak diminta. Jawaban/penjelasan sesingkat mungkin.
- **Jangan commit** apa pun — repo ini bukan git. Semua build/test harus pakai
  `-buildvcs=false`.
- Jangan menambah komentar di kode kecuali diminta. Tandai shortcut yang
  disengaja dengan `// ponytail: <alasan, batas, jalan upgrade>`.

## Aplikasi

POS Kasir Go (`pos-go`) — single binary, backend Go, frontend vanilla JS (tanpa
framework, tanpa npm). Server melayani API REST + file statis `web/static/`.

- Bahasa: Go, modul `github.com/wahid/pos-go`.
- SQLite via `modernc.org/sqlite` (pure Go, tanpa CGO → aman cross-compile).
- Uang = **integer Rupiah**, semua perhitungan di server-side (frontend tidak
  dipercaya).
- Bukan repo git: `go build -buildvcs=false`, `go test -buildvcs=false`.

## Perintah

```sh
go build -buildvcs=false -o dist/pos-go-linux-amd64 ./cmd/pos   # build cepat
./build.sh                                                      # build 5 target → dist/
go test -buildvcs=false ./...                                   # test
go vet -buildvcs=false ./...                                    # vet
```

## Struktur penting

- `cmd/pos/main.go` — wiring, deteksi LAN, graceful shutdown.
- `cmd/pos/autostart.go` — auto-buka browser, shortcut desktop Windows.
- `cmd/kasirgo-webview/main.go` — WebView2 native window (pure Go, no CGO).
- `internal/api/handlers.go` — semua endpoint REST + backup (`VACUUM INTO`) + restore + hapus transaksi.
- `internal/repository/` — `transactions.go` (Checkout atomic, invoiceNumber, DeleteTransactions),
  `reports.go` (SalesReport/StockReport/Dashboard), `repo.go` (CRUD produk/kategori).
- `internal/service/service.go` — validasi checkout, search barcode/SKU.
- `internal/system/appdir.go` — lokasi DB (app data dir, bukan working dir).
- `internal/database/` — koneksi + migrasi SQL.
- `web/static/app.js` — seluruh frontend SPA (hash routing, modal, semua page).
- `web/static/style.css`, `web/static/index.html`.
- `run.sh` — script jalankan server foreground (tutup terminal = server mati).

## Konvensi & jebakan

- **DB tidak boleh bergantung pada working directory.** Lokasi DB via
  `internal/system/appdir.go`: Linux `~/.local/share/pos-go/data.db`, Windows
  `%APPDATA%\pos-go\data.db`. Saat menjalankan binary manual untuk testing,
  set `HOME=/tmp/...` agar tidak menyentuh DB asli.
- **`pkill -f` mematikan shell sendiri** jika pattern ada di command line-nya.
  Pakai `pgrep -f 'dist/pos-go-linux-[a]md64'` (bracket) lalu `kill`.
- Menjalankan server di background: `nohup env HOME=/tmp/posdata BIN >/tmp/log 2>&1 &`
  dipisah dari perintah build (jangan digabung dalam satu bash call panjang,
  bisa ikut terbunuh saat timeout).
- `api()` di frontend melempar `Error` saat status != ok — handler yang menangani
  hasil not-found harus `.catch(...)`, bukan cek `r.error`.
- Kasir: setelah produk masuk keranjang via klik, fokus lompat ke kolom bayar;
  via scan (Enter di kotak cari) fokus tetap di cari agar bisa scan beruntun.
- Produk: scan barcode tak dikenal → buka form Produk Baru dengan barcode terisi
  (hanya jika query tanpa spasi).
- Test dengan `curl` memakai data seed di `HOME=/tmp/posdata`; barcode contoh
  yang sudah ada: `8991002123456` (Teh Botol).
- **Graceful shutdown**: server handle SIGHUP/SIGINT/SIGTERM → mati saat tutup terminal.
- **Input helpers**: nama auto kapital huruf pertama, number input hapus leading zero.

## Verifikasi

Setelah mengubah kode: rebuild binary linux-amd64, jalankan server test
(`HOME=/tmp/posdata`), verifikasi dengan curl + browser (Chrome DevTools MCP di
`http://localhost:8080`), lalu `./build.sh` untuk semua target.

## Fitur yang sudah selesai

- **Core POS**: kasir cepat, barcode scanner (fokus otomatis ke bayar saat klik tile, fokus tetap di cari saat scan enter), tambah produk langsung saat barcode tak dikenal, CRUD produk/kategori, penyesuaian stok per-barcode (`#stok-scan`).
- **Akses HP via LAN**: jalankan `--host 0.0.0.0`, kartu "Akses HP" di Pengaturan tampilkan URL live (`GET /api/network`, tidak disimpan) + atur `lan_pin`; middleware kunci `/api/*` non-loopback via header `X-LAN-PIN` (`POST /api/unlock`), desktop loopback bebas PIN; override env `POS_PIN`/`POS_PORT`.
- **Scan kamera browser** (`openCamScan`): native `BarcodeDetector` saja (live + foto `capture`); tanpa library. Scan serius via aplikasi HP KasirGo Stok (`~/Projects/kasirgo-stok`, Expo + EAS, kamera native).
- **Dashboard**: kartu hero Keuntungan Hari Ini, grafik donat keuntungan vs modal harian (`conic-gradient`), ringkasan stok hampir habis/habis, transaksi terbaru.
- **Pengaturan Data**:
  - Hapus transaksi per rentang tanggal (`YYYY-MM-DD`) atau semua transaksi (aman FK, hapus movement sale & items, tidak sentuh stok fisik).
  - Backup SQLite via `VACUUM INTO` + restore upload snapshot.
  - Layout 2 kolom desktop (Informasi Toko | Data Transaksi, Database di bawah).
- **Tema & Responsif**:
  - Dark/Light mode toggle switch fixed di pojok kanan atas (default ikut OS, manual persist di `localStorage`).
  - 3 mode layout otomatis: Desktop (>1100px), Tablet (641–1100px), HP (≤640px dengan navbar slider horizontal & tabel scrollable).
- **Windows Standalone Experience**:
  - Icon embedded langsung ke `.exe` (`kasirgo.ico` via `go-winres` `.syso`).
  - Auto buka browser default saat dijalankan.
  - Shortcut desktop otomatis pada first run (minimized console).
- **WebView2** (`kasirgo-webview.exe`): native window Edge WebView2, no browser, no CMD hitam, pure Go no CGO, port random, tutup window = server mati.
- **Branding**: `KasirGo`.

## TODO (belum dikerjakan)

### APK WebView "KasirGo" — klien tipis ke server PC via LAN

Status: **plan disetujui, belum dieksekusi.** Lanjut besok.

Tujuan: APK Android tipis (tanpa backend di HP). User memilih jalur **B
(WebView)** + **alamat server bisa diisi manual** (bukan hardcode).

Arsitektur: server PC tetap jalan, HP akses via LAN → semua data terpusat.
Cocok dengan desain kasirgo. (Jalur A=PWA & C=mandiri sudah ditolak.)

Rencana build (SDK sudah terpasang di `$HOME/Android/Sdk`: platform
android-36, build-tools 35/36, cmdline-tools; `java` ada; **tanpa gradle**):

File baru di `android/`:
1. `AndroidManifest.xml` — permission `INTERNET`, `usesCleartextTraffic`,
   1 Activity, minSdk 21 / targetSdk 36.
2. `src/io/kasirgo/MainActivity.java`:
   - Belum ada alamat tersimpan → layar isian (EditText prefill
     `http://192.168.1.150:8080` + tombol Hubungkan), simpan ke
     `SharedPreferences`.
   - Sudah tersimpan → WebView langsung ke `http://<alamat>` (JS + DOM
     storage aktif agar toggle theme / localStorage jalan).
   - Tombol kecil "Ganti Server" pojok kiri atas → kembali ke layar isian.
   - Tombol back Android → navigasi mundur WebView.
3. `res/values/strings.xml` — app_name "KasirGo".
4. `res/xml/network_security_config.xml` — izinkan cleartext HTTP.
5. `build-apk.sh` — pipeline manual: `aapt2 compile`+`link` → `javac`
   (`-classpath platforms/android-36/android.jar`) → `d8` → tambah
   `classes.dex` ke APK → `zipalign` → `apksigner` (keystore debug
   `android/debug.keystore` dibuat sekali via `keytool`, password `android`,
   alias `kasirgo`). Output **`dist/kasirgo.apk`**.

Verifikasi: `apksigner verify --print-certs`, `aapt2 dump badging`, `adb
devices` (kemungkinan tanpa device → hanya cek struktur APK). Install manual
di HP.

Dilewati (tambah kalau butuh): ikon launcher custom, mode mandiri offline.
