package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"unicode/utf16"
)

// openBrowser membuka URL di browser default (best-effort).
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("buka browser: %v", err)
	}
}

// ensureDesktopShortcut membuat shortcut "KasirGo" di Desktop Windows pada
// first run, berjalan minimized (console disembunyikan). // ponytail: lewat
// PowerShell COM (WScript.Shell) karena stdlib tidak bisa menulis .lnk.
func ensureDesktopShortcut(exe string) {
	if runtime.GOOS != "windows" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	lnk := filepath.Join(home, "Desktop", "KasirGo.lnk")
	if _, err := os.Stat(lnk); err == nil {
		return
	}
	script := fmt.Sprintf(
		`$ws=New-Object -ComObject WScript.Shell;$s=$ws.CreateShortcut('%s');`+
			`$s.TargetPath='%s';$s.WorkingDirectory='%s';$s.WindowStyle=7;$s.IconLocation='%s,0';$s.Save()`,
		lnk, exe, filepath.Dir(exe), exe)
	// -EncodedCommand menghindari masalah quoting path ber-spasi.
	enc := base64.StdEncoding.EncodeToString(toUTF16LE(script))
	if err := exec.Command("powershell", "-NoProfile", "-EncodedCommand", enc).Run(); err != nil {
		log.Printf("buat shortcut: %v", err)
	}
}

func toUTF16LE(s string) []byte {
	u := utf16.Encode([]rune(s))
	var b []byte
	for _, r := range u {
		b = append(b, byte(r), byte(r>>8))
	}
	return b
}