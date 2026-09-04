package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wahid/pos-go/internal/models"
	"github.com/wahid/pos-go/internal/repository"
	"github.com/wahid/pos-go/internal/service"
)

// App bundles the live database connection and layers so restore can swap them.
type App struct {
	DB   *sql.DB
	Path string
	Repo *repository.Repository
	Svc  *service.Service
}

type Server struct {
	app *App
	// Port tampil di /api/network agar IP+port di Pengaturan selalu sesuai
	// dengan port server yang sedang jalan. Diisi dari main (flag/env).
	Port string
}

func NewServer(app *App) *Server {
	return &Server{app: app}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/dashboard", s.handleDashboard)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })

	mux.HandleFunc("GET /api/products", s.handleListProducts)
	mux.HandleFunc("POST /api/products", s.handleCreateProduct)
	mux.HandleFunc("GET /api/products/{id}", s.handleGetProduct)
	mux.HandleFunc("PUT /api/products/{id}", s.handleUpdateProduct)
	mux.HandleFunc("DELETE /api/products/{id}", s.handleDeleteProduct)
	mux.HandleFunc("GET /api/products/search/{q}", s.handleSearchProduct)

	mux.HandleFunc("GET /api/categories", s.handleListCategories)
	mux.HandleFunc("POST /api/categories", s.handleCreateCategory)
	mux.HandleFunc("PUT /api/categories/{id}", s.handleUpdateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", s.handleDeleteCategory)

	mux.HandleFunc("POST /api/transactions", s.handleCheckout)
	mux.HandleFunc("GET /api/transactions", s.handleListTransactions)
	mux.HandleFunc("GET /api/transactions/{id}", s.handleGetTransaction)
	mux.HandleFunc("DELETE /api/transactions", s.handleDeleteTransactions)

	mux.HandleFunc("POST /api/stock/adjust", s.handleAdjustStock)
	mux.HandleFunc("GET /api/stock/movements", s.handleStockMovements)

	mux.HandleFunc("GET /api/reports/sales", s.handleSalesReport)
	mux.HandleFunc("GET /api/reports/stock", s.handleStockReport)

	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handleSaveSettings)

	mux.HandleFunc("GET /api/backup/download", s.handleBackup)
	mux.HandleFunc("POST /api/restore", s.handleRestore)

	mux.HandleFunc("GET /api/network", s.handleNetwork)
	mux.HandleFunc("POST /api/unlock", s.handleUnlock)

	return s.requireLANPin(mux)
}

// requireLANPin mengunci /api/* untuk akses non-localhost dengan PIN.
// Desktop (loopback) selalu bebas PIN; HP (LAN) wajib header X-LAN-PIN.
// PIN kosong = tanpa kunci (backward compat).
// ponytail: 1 PIN global, bukan akun per-user; tambah user+session kalau butuh audit.
func (s *Server) requireLANPin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") ||
			r.URL.Path == "/api/unlock" || r.URL.Path == "/api/health" ||
			isLoopback(r) {
			next.ServeHTTP(w, r)
			return
		}
		if want := s.lanPIN(); want != "" && r.Header.Get("X-LAN-PIN") != want {
			writeErr(w, http.StatusUnauthorized, "PIN akses HP salah atau belum diisi")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) lanPIN() string {
	if e := os.Getenv("POS_PIN"); e != "" {
		return e
	}
	v, _ := s.app.Repo.GetSetting("lan_pin")
	return v
}

func isLoopback(r *http.Request) bool {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		h = r.RemoteAddr
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// handleNetwork mengembalikan IP LAN + port server saat ini (live, tidak disimpan)
// agar yang tampil di Pengaturan selalu sesuai IP laptop sekarang.
func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request) {
	port := s.Port
	if port == "" {
		port = os.Getenv("POS_PORT")
	}
	if port == "" {
		port = "2001"
	}
	ips := lanIPs()
	urls := make([]string, 0, len(ips))
	for _, ip := range ips {
		urls = append(urls, "http://"+ip+":"+port+"/#/stok")
	}
	writeJSON(w, 200, map[string]any{"port": port, "ips": ips, "urls": urls})
}

func lanIPs() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok {
			ip := ipn.IP.To4()
			if ip != nil && !ip.IsLoopback() {
				out = append(out, ip.String())
			}
		}
	}
	return out
}

func (s *Server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pin string `json:"pin"`
	}
	if decode(w, r, &body) != nil {
		return
	}
	if want := s.lanPIN(); want != "" && body.Pin != want {
		writeErr(w, http.StatusUnauthorized, "PIN salah")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	log.Printf("api error %d: %s", status, msg)
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return err
	}
	return nil
}

func mapErr(err error) (int, string) {
	if errors.Is(err, repository.ErrNotFound) {
		return http.StatusNotFound, "not found"
	}
	return http.StatusBadRequest, err.Error()
}

// ---------- dashboard ----------

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.app.Repo.Dashboard()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat dashboard")
		return
	}
	writeJSON(w, 200, d)
}

// ---------- products ----------

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var catID int64
	fmt.Sscanf(r.URL.Query().Get("category"), "%d", &catID)
	products, err := s.app.Repo.ListProducts(repository.ProductFilter{Search: q, CategoryID: catID})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat produk")
		return
	}
	writeJSON(w, 200, products)
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.Repo.GetProduct(parseID(r))
	if err != nil {
		status, msg := mapErr(err)
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	if decode(w, r, &p) != nil {
		return
	}
	p.ID = 0
	created, err := s.app.Repo.CreateProduct(&p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	if decode(w, r, &p) != nil {
		return
	}
	p.ID = parseID(r)
	updated, err := s.app.Repo.UpdateProduct(&p)
	if err != nil {
		status, msg := mapErr(err)
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, 200, updated)
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Repo.DeleteProduct(parseID(r)); err != nil {
		status, msg := mapErr(err)
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handleSearchProduct(w http.ResponseWriter, r *http.Request) {
	q := r.PathValue("q")
	p, err := s.app.Svc.SearchProduct(q)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeJSON(w, 200, map[string]string{"error": "Produk tidak ditemukan"})
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, 200, p)
}

// ---------- categories ----------

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cs, err := s.app.Repo.ListCategories()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat kategori")
		return
	}
	writeJSON(w, 200, cs)
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if decode(w, r, &in) != nil {
		return
	}
	c, err := s.app.Repo.CreateCategory(in.Name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if decode(w, r, &in) != nil {
		return
	}
	if err := s.app.Repo.UpdateCategory(parseID(r), in.Name); err != nil {
		status, msg := mapErr(err)
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Repo.DeleteCategory(parseID(r)); err != nil {
		status, msg := mapErr(err)
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// ---------- transactions ----------

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	var req models.CheckoutRequest
	if decode(w, r, &req) != nil {
		return
	}
	tx, err := s.app.Svc.Checkout(req)
	if err != nil {
		status, msg := mapErr(err)
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusCreated, tx)
}

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	txs, err := s.app.Repo.ListTransactions(from, to, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat transaksi")
		return
	}
	writeJSON(w, 200, txs)
}

func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	t, err := s.app.Repo.GetTransaction(parseID(r))
	if err != nil {
		status, msg := mapErr(err)
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, 200, t)
}

// ---------- stock ----------

func (s *Server) handleAdjustStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProductID int64  `json:"product_id"`
		Delta     int64  `json:"delta"`
		Note      string `json:"note"`
	}
	if decode(w, r, &in) != nil {
		return
	}
	if in.ProductID <= 0 || in.Delta == 0 {
		writeErr(w, http.StatusBadRequest, "product_id dan delta wajib (delta != 0)")
		return
	}
	if err := s.app.Repo.AdjustStock(in.ProductID, in.Delta, strings.TrimSpace(in.Note)); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handleStockMovements(w http.ResponseWriter, r *http.Request) {
	ms, err := s.app.Repo.ListStockMovements(100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat riwayat stok")
		return
	}
	writeJSON(w, 200, ms)
}

// ---------- reports ----------

func (s *Server) handleSalesReport(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeErr(w, http.StatusBadRequest, "from dan to wajib (YYYY-MM-DD)")
		return
	}
	rep, err := s.app.Repo.SalesReport(from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat laporan")
		return
	}
	writeJSON(w, 200, rep)
}

func (s *Server) handleStockReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.app.Repo.StockReport()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat laporan stok")
		return
	}
	writeJSON(w, 200, rep)
}

// ---------- settings ----------

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	kv := map[string]string{}
	for _, key := range []string{"store_name", "store_address", "store_phone", "currency", "receipt_footer", "lan_pin"} {
		v, err := s.app.Repo.GetSetting(key)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "gagal memuat setting")
			return
		}
		kv[key] = v
	}
	writeJSON(w, 200, kv)
}

func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	var kv map[string]string
	if decode(w, r, &kv) != nil {
		return
	}
	if err := s.app.Repo.SaveSettings(kv); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan setting")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// handleDeleteTransactions menghapus transaksi (semua atau rentang dari/to, format YYYY-MM-DD).
func (s *Server) handleDeleteTransactions(w http.ResponseWriter, r *http.Request) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	for _, d := range []string{from, to} {
		if d != "" && !dateRe.MatchString(d) {
			writeErr(w, http.StatusBadRequest, "format tanggal salah (YYYY-MM-DD)")
			return
		}
	}
	n, err := s.app.Repo.DeleteTransactions(from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menghapus transaksi: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]int64{"deleted": n})
}

// ---------- backup / restore ----------
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	tmp := filepath.Join(s.app.Path + ".backup.tmp")
	defer os.Remove(tmp)
	if _, err := s.app.DB.Exec(fmt.Sprintf("VACUUM INTO '%s'", strings.ReplaceAll(tmp, "'", "''"))); err != nil {
		writeErr(w, http.StatusInternalServerError, "backup gagal: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-sqlite3")
	w.Header().Set("Content-Disposition", `attachment; filename="pos-go-backup-`+time.Now().Format("20060102-150405")+`.db"`)
	http.ServeFile(w, r, tmp)
}

// Restore replaces the database with an uploaded backup.
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
	file, _, err := r.FormFile("backup")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "file backup wajib (field 'backup')")
		return
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "pos-go-restore-*.db")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "restore gagal")
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		writeErr(w, http.StatusBadRequest, "file backup tidak valid")
		return
	}
	tmp.Close()

	// Validate it really is a SQLite database.
	head := make([]byte, 16)
	f, _ := os.Open(tmp.Name())
	_, err = io.ReadFull(f, head)
	f.Close()
	if err != nil || string(head) != "SQLite format 3\x00" {
		writeErr(w, http.StatusBadRequest, "file bukan database SQLite")
		return
	}

	if err := s.app.RestoreFrom(tmp.Name()); err != nil {
		log.Printf("restore failed: %v", err)
		writeErr(w, http.StatusInternalServerError, "restore gagal: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok", "message": "database dipulihkan"})
}

// RestoreFrom swaps the live database for the file at src. The current database is
// kept as .bak so a failed restore can be rolled back.
func (a *App) RestoreFrom(src string) error {
	bak := a.Path + ".bak"
	if err := a.DB.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	if err := os.Rename(a.Path, bak); err != nil {
		return fmt.Errorf("backup current db: %w", err)
	}
	if err := os.Rename(src, a.Path); err != nil {
		os.Rename(bak, a.Path)
		return fmt.Errorf("move restore file: %w", err)
	}
	if err := a.reopen(); err != nil {
		os.Remove(a.Path)
		os.Rename(bak, a.Path)
		_ = a.reopen()
		return fmt.Errorf("reopen restored db: %w", err)
	}
	os.Remove(bak)
	return nil
}

func (a *App) reopen() error {
	db, err := databaseOpen(a.Path)
	if err != nil {
		return err
	}
	a.DB = db
	a.Repo = repository.New(db)
	a.Svc = service.New(a.Repo)
	return nil
}

func parseID(r *http.Request) int64 {
	var id int64
	fmt.Sscanf(r.PathValue("id"), "%d", &id)
	return id
}