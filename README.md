# pos-go

POS (Point of Sale) untuk toko sembako / warung, berjalan **offline** di PC
laptop, atau HP dalam jaringan lokal. Backend Go + SQLite, frontend HTML/CSS/JS
ringan (tanpa Node, tanpa build step). Satu executable — semua aset tertanam di
dalamnya.

## Fitur

- Dashboard: penjualan hari ini, produk hampir habis / habis, transaksi terbaru.
- Kasir: pencarian + scan barcode (USB scanner = keyboard), keranjang,
  diskon, 4 metode bayar (cash/transfer/qris/debit), hitung kembalian, struk
  yang bisa diprint / simpan PDF.
- Produk: CRUD, kategori, barcode/SKU, harga beli/jual, stok minimum, satuan.
- Stok: penyesuaian stok, riwayat `stock_movements`, status Normal/Menipis/Habis.
- Transaksi atomic: insert transaksi + item + kurangi stok + catat pergerakan
  stok dalam satu transaksi database (rollback penuh jika ada yang gagal).
- Laporan: penjualan (hari ini/kemarin/minggu/bulan/custom), produk terlaris,
  laporan stok.
- Pengaturan toko: nama, alamat, telepon, mata uang, footer struk.
- Backup & restore database.
- Pintasan keyboard: `F2` cari, `F4` pembayaran, `ESC` tutup, `Enter` konfirmasi.

## Requirement

- Go 1.22+ untuk build (hanya untuk development; runtime hanya butuh executable).
- Tanpa internet, tanpa Node/Python/Docker untuk menjalankan hasil build.

## Development

```sh
go mod download
go test ./...
go vet ./...
go run ./cmd/pos
# buka http://localhost:8080
```

## Run

```sh
go run ./cmd/pos
# atau binary hasil build:
./pos-go-linux-amd64
```

Server default `127.0.0.1:8080`. Untuk akses dari HP di jaringan yang sama:

```sh
./pos-go-linux-amd64 --host 0.0.0.0
# atau env: POS_HOST=0.0.0.0 POS_PORT=8080
```

Saat server jalan, IP LAN yang terdeteksi ikut ditampilkan di log:

```text
POS Go started
Local:    http://localhost:8080
Network:  http://192.168.1.10:8080
```

Buka URL Network dari HP untuk kasir via browser.

## Database location

Database **tidak bergantung pada working directory**. Lokasi dihitung dari
sistem operasi:

| OS      | Path                                  |
|---------|---------------------------------------|
| Linux   | `~/.local/share/pos-go/data.db`       |
| Windows | `%APPDATA%\pos-go\data.db`            |

Path absolut database dicetak saat startup (`Database: /…/data.db`). Jika file
sudah ada, dibuka apa adanya (tidak direset); hanya migration yang belum
dijalankan yang diterapkan. Memindahkan executable atau menjalankannya dari
folder berbeda **tidak** membuat database baru.

## Configuration

| Setting   | Keterangan                        | Default   |
|-----------|-----------------------------------|-----------|
| `POS_HOST` | host listen (`0.0.0.0` untuk LAN) | `127.0.0.1` |
| `POS_PORT` | port                              | `8080`    |

Keduanya juga bisa lewat flag `--host` / `--port`.

## LAN access

Jalankan dengan `--host 0.0.0.0` lalu buka `http://<IP-PC>:<port>` dari HP.
Versi MVP tanpa autentikasi; semua akses lewat API tervalidasi, parameterized
SQL, dan database tidak pernah diekspos langsung.

## Backup

Halaman **Pengaturan → Backup Database** mengunduh snapshot database yang
konsisten (via `VACUUM INTO`, aman saat database aktif). **Restore Database**
menerima file backup (divalidasi sebagai SQLite); database lama disimpan sebagai
`.bak` dan dipakai lagi otomatis jika restore gagal.

## Testing

```sh
go test ./...
```

Cakupan: perhitungan uang (subtotal/diskon/total/kembalian), pengecekan stok,
lookup barcode, database path (absolut & tidak tergantung cwd), migration
(idempotent), integrasi (produk → transaksi → stok turun → pergerakan stok
tercatat), rollback saat stok tidak cukup, dan persistensi lintas restart.

## Cross-platform build

```sh
./build.sh        # Linux + Windows, keluar ke ./dist/
```

Target: linux/amd64, linux/arm64, windows/amd64, windows/arm64. Output:
`pos-go-linux-amd64`, `pos-go-linux-arm64`, `pos-go.exe`, `pos-go-arm64.exe`.

Atau build manual:

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o pos-go.exe ./cmd/pos
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o pos-go ./cmd/pos
```

Driver SQLite (`modernc.org/sqlite`) murni Go, jadi tanpa CGO — aman
di-cross-compile ke Windows.