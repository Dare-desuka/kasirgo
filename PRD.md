# PRD — POS Kasir Go

## 1. Ringkasan

Buat aplikasi POS (Point of Sale) untuk toko sembako menggunakan Go.

Aplikasi harus:

* Berjalan native di Windows dan Linux.
* Menggunakan SQLite sebagai database lokal.
* Tidak bergantung pada working directory untuk lokasi database.
* Bisa digunakan melalui browser di PC/laptop.
* Bisa diakses dari HP melalui jaringan lokal.
* Tetap bisa digunakan tanpa internet.
* Memiliki sistem stok, transaksi, barcode, laporan, dan manajemen produk.
* Mudah di-build dan didistribusikan sebagai aplikasi standalone.

Nama proyek sementara: `pos-go`.

---

## 2. Tujuan

Tujuan utama aplikasi:

1. Mempercepat proses kasir.
2. Mengelola stok barang.
3. Menyimpan seluruh transaksi secara persisten.
4. Menyediakan laporan penjualan.
5. Bisa digunakan di komputer Windows maupun Linux.
6. Bisa diakses dari perangkat HP dalam jaringan lokal.
7. Tidak kehilangan data ketika aplikasi dipindahkan, restart, atau working directory berubah.

---

## 3. Target Pengguna

Target utama:

* Toko sembako.
* Warung.
* Toko kelontong.
* Usaha kecil dengan satu komputer kasir.

Tidak perlu membuat sistem multi-cabang atau cloud pada versi awal.

---

# 4. Teknologi

## Backend

Gunakan:

* Go
* HTTP server bawaan Go atau router ringan
* SQLite
* SQL migrations
* JSON API

Hindari dependency yang tidak diperlukan.

Prioritaskan library yang stabil dan mudah dirawat.

## Database

Gunakan SQLite.

Database harus disimpan pada lokasi data aplikasi yang stabil.

Jangan menggunakan:

```text
./data/database.db
```

atau path relatif terhadap working directory.

Lokasi database harus dihitung berdasarkan OS.

Contoh konsep:

Linux:

```text
~/.local/share/pos-go/data.db
```

Windows:

```text
%APPDATA%/pos-go/data.db
```

Implementasikan melalui fungsi khusus seperti:

```go
GetAppDataDir()
GetDatabasePath()
```

Semua komponen aplikasi harus menggunakan fungsi tersebut.

Jangan ada kode lain yang menentukan lokasi database sendiri.

---

# 5. Arsitektur

Gunakan arsitektur sederhana dan modular.

Contoh:

```text
cmd/
    pos/
        main.go

internal/
    config/
    database/
    migrations/
    models/
    repository/
    service/
    api/
    barcode/
    report/
    system/

web/
    static/
    templates/

data/
```

Struktur boleh disesuaikan jika ada alasan teknis yang lebih baik.

Pisahkan:

```text
API
Service
Repository
Database
```

Jangan memasukkan SQL langsung ke HTTP handler jika bisa dihindari.

---

# 6. Database

Minimal tabel:

## products

Field:

* id
* barcode
* sku
* name
* category_id
* purchase_price
* selling_price
* stock
* minimum_stock
* unit
* created_at
* updated_at
* deleted_at

Barcode harus bisa kosong.

Barcode harus unique jika diisi.

---

## categories

Field:

* id
* name
* created_at
* updated_at

---

## transactions

Field:

* id
* invoice_number
* subtotal
* discount
* total
* paid
* change
* payment_method
* cashier
* created_at

---

## transaction_items

Field:

* id
* transaction_id
* product_id
* product_name
* quantity
* price
* subtotal

Simpan snapshot nama dan harga produk pada saat transaksi.

Jangan hanya mengandalkan data produk saat ini.

---

## stock_movements

Field:

* id
* product_id
* type
* quantity
* reference_id
* note
* created_at

Type contoh:

```text
sale
purchase
adjustment
return
```

---

## settings

Untuk konfigurasi aplikasi.

Contoh:

```text
store_name
store_address
store_phone
currency
receipt_footer
```

---

# 7. Fitur MVP

## Dashboard

Tampilkan:

* Total penjualan hari ini.
* Jumlah transaksi hari ini.
* Produk hampir habis.
* Produk habis.
* Penjualan terbaru.

---

# 8. Kasir

Halaman kasir harus menjadi fitur utama.

Flow:

```text
Cari produk
    ↓
Tambah ke keranjang
    ↓
Ubah quantity
    ↓
Hitung subtotal
    ↓
Diskon
    ↓
Total
    ↓
Input pembayaran
    ↓
Hitung kembalian
    ↓
Simpan transaksi
    ↓
Kurangi stok
    ↓
Tampilkan struk
```

---

## Pencarian Produk

Bisa mencari berdasarkan:

* Barcode
* SKU
* Nama produk

Barcode scanner USB harus diperlakukan seperti keyboard.

Saat barcode scanner mengetik barcode dan menekan Enter:

```text
barcode → cari produk → otomatis masuk keranjang
```

Jika produk sudah ada di keranjang:

```text
quantity + 1
```

Jangan membuat item duplikat.

---

# 9. Keranjang

Setiap item menampilkan:

```text
Nama
Harga
Quantity
Subtotal
```

Fungsi:

* Tambah quantity.
* Kurangi quantity.
* Input quantity manual.
* Hapus item.
* Kosongkan keranjang.

---

# 10. Pembayaran

Minimal:

```text
Cash
Transfer
QRIS
Debit
```

Untuk cash:

```text
Total: 25.000
Bayar: 30.000
Kembali: 5.000
```

Validasi:

```text
paid >= total
```

Jika kurang:

```text
Pembayaran tidak cukup.
```

---

# 11. Stok

Ketika transaksi berhasil:

```text
stock -= quantity
```

Setiap perubahan stok harus membuat record `stock_movements`.

Jangan hanya mengubah angka stock tanpa histori.

---

# 12. Manajemen Produk

Fitur:

* Tambah produk.
* Edit produk.
* Hapus produk.
* Cari produk.
* Filter kategori.
* Atur harga beli.
* Atur harga jual.
* Atur stok.
* Atur minimum stok.
* Atur barcode.
* Atur SKU.
* Atur satuan.

Satuan contoh:

```text
pcs
kg
gram
liter
botol
dus
bungkus
```

---

# 13. Manajemen Stok

Sediakan fitur penyesuaian stok.

Contoh:

```text
Produk: Indomie Goreng
Stok sekarang: 20
Penyesuaian: -2
Alasan: Barang rusak
```

Sistem harus mencatat:

```text
stock_movements
type = adjustment
```

---

# 14. Laporan

Minimal:

## Laporan Penjualan

Filter:

* Hari ini.
* Kemarin.
* Minggu ini.
* Bulan ini.
* Custom date range.

Tampilkan:

* Jumlah transaksi.
* Total penjualan.
* Total diskon.
* Total pembayaran.
* Produk terjual.

---

## Produk Terlaris

Tampilkan:

```text
Produk
Jumlah terjual
Total penjualan
```

---

## Laporan Stok

Tampilkan:

* Semua produk.
* Stok saat ini.
* Produk hampir habis.
* Produk habis.

---

# 15. Struk

Setelah transaksi:

* Tampilkan preview struk.
* Bisa print.
* Bisa menyimpan sebagai PDF jika memungkinkan.

Format struk sederhana:

```text
TOKO SEMBAKO
Alamat
------------------------
Indomie Goreng
2 x 3.000        6.000

Beras 5kg
1 x 75.000      75.000
------------------------
Subtotal        81.000
Diskon           0
TOTAL           81.000
Bayar           90.000
Kembali          9.000
------------------------
Terima kasih
```

---

# 16. API

Backend menyediakan REST API.

Contoh:

```text
GET    /api/products
POST   /api/products
GET    /api/products/:id
PUT    /api/products/:id
DELETE /api/products/:id

GET    /api/categories
POST   /api/categories

POST   /api/transactions
GET    /api/transactions
GET    /api/transactions/:id

GET    /api/reports/sales
GET    /api/reports/products
GET    /api/reports/stock
```

API harus memiliki validasi input.

Jangan mempercayai data dari frontend.

---

# 17. Network / LAN

Default:

```text
127.0.0.1
```

Sediakan konfigurasi agar server dapat listen:

```text
0.0.0.0
```

sehingga HP dalam jaringan yang sama dapat mengakses:

```text
http://IP-PC:PORT
```

Tampilkan URL LAN ketika server berjalan.

Contoh:

```text
POS Go running

Local:
http://localhost:8080

Network:
http://192.168.1.10:8080
```

Jangan menganggap IP selalu sama.

---

# 18. Keamanan LAN

Versi MVP tidak membutuhkan login kompleks.

Namun:

* Jangan expose database langsung.
* Semua akses database melalui backend.
* Validasi request.
* Gunakan transaction database untuk transaksi kasir.
* Jangan menjalankan SQL berdasarkan input mentah user.
* Jangan expose endpoint filesystem.

---

# 19. Konsistensi Transaksi

Transaksi kasir harus atomic.

Saat menyimpan transaksi:

```text
BEGIN TRANSACTION

insert transaction
insert transaction_items
update stock
insert stock_movements

COMMIT
```

Jika salah satu gagal:

```text
ROLLBACK
```

Jangan sampai transaksi tersimpan tetapi stok gagal berubah.

---

# 20. Backup

Sediakan fitur backup database.

Contoh:

```text
Backup Database
```

User dapat memilih lokasi file backup.

Backup harus dilakukan dengan cara yang aman untuk SQLite.

Tambahkan juga:

```text
Restore Database
```

jika memungkinkan pada MVP/versi berikutnya.

---

# 21. Database Persistence

Ini sangat penting.

Aplikasi TIDAK BOLEH:

* Membuat database baru ketika working directory berubah.
* Membuat database baru ketika executable dipindahkan.
* Menggunakan path seperti `./data.db`.
* Menghapus database saat aplikasi restart.
* Menggunakan temporary directory sebagai database utama.
* Membuat database berbeda untuk setiap lokasi startup.

Saat aplikasi startup, log:

```text
Database: /absolute/path/to/data.db
```

Jika database belum ada:

```text
create database
run migrations
```

Jika sudah ada:

```text
open existing database
run pending migrations
```

Jangan reset database.

---

# 22. Migration

Gunakan migration versioning.

Contoh:

```text
001_initial.sql
002_add_settings.sql
003_add_stock_movements.sql
```

Aplikasi harus mengetahui migration terakhir yang sudah dijalankan.

Migration baru tidak boleh menghapus data existing kecuali memang diperlukan dan aman.

---

# 23. Error Handling

Error harus jelas.

Backend:

```text
HTTP status code
JSON error
```

Contoh:

```json
{
  "error": "product not found"
}
```

Frontend menampilkan pesan yang mudah dimengerti.

Jangan menampilkan stack trace kepada user.

Stack trace hanya untuk log developer.

---

# 24. Logging

Gunakan structured/simple logging.

Minimal log:

* Startup.
* Database path.
* Migration.
* Server address.
* Transaction error.
* Database error.
* Backup error.

Jangan log data sensitif.

---

# 25. UI/UX

Prioritaskan kecepatan penggunaan kasir.

Desktop:

```text
+--------------------------------------+
| Search / Barcode                     |
+----------------------+---------------+
| Product              | Cart          |
|                      |               |
|                      |               |
|                      |               |
+----------------------+---------------+
|                      | TOTAL         |
|                      | [BAYAR]       |
+----------------------+---------------+
```

Keyboard shortcut sangat diutamakan.

Contoh:

```text
F2  → Search
F4  → Payment
ESC → Clear/close modal
Enter → Confirm
```

Jangan membuat UI terlalu ramai.

---

# 26. Responsive

UI harus tetap usable pada:

* Desktop.
* Laptop.
* Tablet.
* HP.

Kasir desktop menjadi prioritas utama.

---

# 27. Offline

Core POS harus berjalan tanpa internet.

Internet hanya diperlukan jika nanti ditambahkan fitur cloud/integrasi eksternal.

---

# 28. Testing

Minimal:

## Unit test

Test:

* Perhitungan subtotal.
* Diskon.
* Total.
* Pembayaran.
* Kembalian.
* Stok.
* Barcode lookup.
* Database path.
* Migration.

## Integration test

Test:

```text
Create product
↓
Create transaction
↓
Stock decreases
↓
Transaction saved
↓
Stock movement created
```

Test rollback ketika salah satu operasi gagal.

---

# 29. Build

Target:

```text
Linux amd64
Linux arm64
Windows amd64
Windows arm64 jika memungkinkan
```

Output sederhana:

```text
pos-go
pos-go.exe
```

Aplikasi harus bisa dijalankan tanpa:

```text
npm
python
node
docker
```

setelah build production.

---

# 30. Non-Goals MVP

Jangan membuat dulu:

* Cloud sync.
* Multi-cabang.
* Multi-user kompleks.
* Akuntansi penuh.
* Payroll.
* Marketplace.
* Payment gateway.
* AI.
* Subscription.
* Server cloud.

Fokus pada POS lokal yang stabil.

---

# 31. Prioritas

Urutan implementasi:

1. Project structure.
2. Database.
3. Migration.
4. Product CRUD.
5. Category CRUD.
6. Inventory.
7. Transaction engine.
8. Cart.
9. Payment.
10. Barcode.
11. Dashboard.
12. Reports.
13. Receipt.
14. Backup.
15. LAN access.
16. Testing.
17. Build Windows/Linux.

---

# 32. Definition of Done

MVP dianggap selesai jika:

* Aplikasi bisa berjalan di Linux.
* Aplikasi bisa berjalan di Windows.
* Database persistent.
* Working directory tidak memengaruhi database.
* Produk bisa dibuat/edit/delete.
* Barcode bisa digunakan.
* Transaksi bisa dilakukan.
* Stok otomatis berkurang.
* Histori stok tersimpan.
* Pembayaran dan kembalian benar.
* Laporan penjualan tersedia.
* Backup database tersedia.
* HP bisa mengakses POS melalui LAN.
* Tidak membutuhkan internet.
* Test utama lulus.
* Build production berhasil.
