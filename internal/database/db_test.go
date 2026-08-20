package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "data.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, path
}

func TestMigration_CreatesSchema(t *testing.T) {
	db, _ := openTestDB(t)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='products'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("products table missing")
	}
	var settings int
	db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&settings)
	if settings < 4 {
		t.Fatalf("expected default settings, got %d", settings)
	}
}

func TestMigration_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.db")
	db1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db1.Exec(`INSERT INTO categories (name) VALUES ('Test')`)
	db1.Close()

	// Reopen the same file: migration must not re-run or reset data.
	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	var n int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("data lost on reopen, categories=%d", n)
	}
}

func TestPersistence_AcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.db")

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO categories (name) VALUES ('Persist')`)
	db.Close()

	// Simulate restart from a completely different working directory.
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(t.TempDir())

	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	var n int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM categories WHERE name='Persist'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("product lost across restart, count=%d", n)
	}
}