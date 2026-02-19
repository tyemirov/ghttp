//go:build darwin

package truststore

import (
	"os"
	"path/filepath"
)

func defaultFirefoxProfileDirectories() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")}
}
