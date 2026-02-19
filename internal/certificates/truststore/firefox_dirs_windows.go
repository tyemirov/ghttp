//go:build windows

package truststore

import (
	"os"
	"path/filepath"
)

func defaultFirefoxProfileDirectories() []string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return nil
	}
	return []string{filepath.Join(appData, "Mozilla", "Firefox", "Profiles")}
}
