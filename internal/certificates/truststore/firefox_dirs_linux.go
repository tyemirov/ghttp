//go:build linux

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
	return []string{filepath.Join(home, ".mozilla", "firefox")}
}
