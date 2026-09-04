//go:build windows

package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	webview "github.com/jchv/go-webview2"
	"github.com/wahid/pos-go/internal/api"
	"github.com/wahid/pos-go/internal/database"
	"github.com/wahid/pos-go/internal/repository"
	"github.com/wahid/pos-go/internal/service"
	"github.com/wahid/pos-go/internal/system"
	"github.com/wahid/pos-go/web"
)

func main() {
	dbPath, err := system.GetDatabasePath()
	if err != nil {
		log.Fatalf("resolve database path: %v", err)
	}
	if err := system.EnsureDir(filepath.Dir(dbPath)); err != nil {
		log.Fatalf("app data dir: %v", err)
	}

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	app := &api.App{DB: db, Path: dbPath, Repo: repository.New(db)}
	app.Svc = service.New(app.Repo)

	server := api.NewServer(app)
	handler := web.WithStatic(server.Routes())

	// Shortcut desktop Windows di first run
	if exe, err := os.Executable(); err == nil {
		system.EnsureDesktopShortcut(exe)
	}

	// ponytail: selalu 0.0.0.0:2001 (sama dgn pos-go) agar HP bisa konek;
	// window desktop tetap lewat loopback (bebas PIN), HP wajib PIN via middleware.
	ln, err := net.Listen("tcp", "0.0.0.0:2001")
	if err != nil {
		log.Fatalf("listen 0.0.0.0:2001: %v (port dipakai aplikasi lain?)", err)
	}
	server.Port = "2001"

	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)

	// WebView2 window — pure Go, no CGO, icon dari .syso embed
	w := webview.NewWithOptions(webview.WebViewOptions{
		Debug: false,
		WindowOptions: webview.WindowOptions{
			Title:  "KasirGo",
			Width:  1024,
			Height: 700,
			Center: true,
		},
	})
	if w == nil {
		log.Fatal("failed to create webview — pasti WebView2 runtime terinstall (Windows 10/11)")
	}
	defer w.Destroy()

	w.Navigate("http://127.0.0.1:2001")
	w.Run()

	// Tutup window → shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
