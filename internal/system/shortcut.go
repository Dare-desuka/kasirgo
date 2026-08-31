package system

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

// EnsureDesktopShortcut membuat shortcut "KasirGo" di Desktop Windows pada
// first run. // ponytail: lewat PowerShell COM (WScript.Shell) karena stdlib
// tidak bisa menulis .lnk.
func EnsureDesktopShortcut(exe string) {
	if runtime.GOOS != "windows" {
		return
	}
	if os.Getenv("POS_NO_SHORTCUT") != "" {
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
