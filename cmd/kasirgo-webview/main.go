//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
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

	// Listen di port random
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port

	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)

	// WebView2 window — pure Go, no CGO
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

	w.Navigate(fmt.Sprintf("http://127.0.0.1:%d", actualPort))
	w.Run()

	// Tutup window → shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
