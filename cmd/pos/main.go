package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wahid/pos-go/internal/api"
	"github.com/wahid/pos-go/internal/database"
	"github.com/wahid/pos-go/internal/repository"
	"github.com/wahid/pos-go/internal/service"
	"github.com/wahid/pos-go/internal/system"
	"github.com/wahid/pos-go/web"
)

func main() {
	host := flag.String("host", "", "listen host (e.g. 0.0.0.0 for LAN access)")
	port := flag.String("port", "", "listen port")
	noBrowser := flag.Bool("no-browser", false, "jangan auto-buka browser")
	flag.Parse()

	// env menang, lalu flag, lalu default.
	hostV := "127.0.0.1"
	if e := os.Getenv("POS_HOST"); e != "" {
		hostV = e
	}
	if f := *host; f != "" {
		hostV = f
	}
	portV := "8080"
	if e := os.Getenv("POS_PORT"); e != "" {
		portV = e
	}
	if f := *port; f != "" {
		portV = f
	}

	dbPath, err := system.GetDatabasePath()
	if err != nil {
		log.Fatalf("resolve database path: %v", err)
	}
	if err := system.EnsureDir(filepath.Dir(dbPath)); err != nil {
		log.Fatalf("app data dir: %v", err)
	}
	log.Printf("Database: %s", dbPath)

	// If the file already exists, Open keeps it (no reset) and applies pending migrations.
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	app := &api.App{DB: db, Path: dbPath, Repo: repository.New(db)}
	app.Svc = service.New(app.Repo)

	server := api.NewServer(app)
	handler := web.WithStatic(server.Routes())

	addr := hostV + ":" + portV
	log.Printf("Server address: %s", addr)
	log.Printf("POS Go started\nLocal:    http://localhost:%s", portV)
	if hostV != "127.0.0.1" {
		for _, ip := range lanIPs() {
			log.Printf("Network:  http://%s:%s", ip, portV)
		}
	}

	// Auto-buka browser (untuk user awam) + shortcut desktop Windows di first run.
	if !*noBrowser && os.Getenv("POS_NO_BROWSER") == "" {
		go func() {
			time.Sleep(400 * time.Millisecond)
			openBrowser("http://localhost:" + portV)
		}()
	}
	if os.Getenv("POS_NO_SHORTCUT") == "" {
		if exe, err := os.Executable(); err == nil {
			ensureDesktopShortcut(exe)
		}
	}

	if err := http.ListenAndServe(addr, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}

// lanIPs returns non-loopback IPv4 addresses of this machine.
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