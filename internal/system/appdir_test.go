package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetDatabasePath_IsAbsoluteAndStable(t *testing.T) {
	p, err := GetDatabasePath()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(p) {
		t.Fatalf("expected absolute path, got %q", p)
	}
	if !strings.HasSuffix(p, "pos-go"+string(os.PathSeparator)+"data.db") {
		t.Fatalf("unexpected path: %q", p)
	}
}

func TestGetDatabasePath_IgnoresWorkingDir(t *testing.T) {
	before, err := GetDatabasePath()
	if err != nil {
		t.Fatal(err)
	}
	// Changing the working directory must not change the database path.
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	after, err := GetDatabasePath()
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("database path changed with working dir: %q vs %q", before, after)
	}
}

func TestGetAppDataDir(t *testing.T) {
	dir, err := GetAppDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "linux" && !strings.Contains(dir, ".local/share/pos-go") {
		t.Fatalf("unexpected linux app data dir: %q", dir)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
}