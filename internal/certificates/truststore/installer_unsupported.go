//go:build !linux && !darwin && !windows

package truststore

import (
	"fmt"
	"runtime"

	"github.com/tyemirov/ghttp/internal/certificates"
)

// NewInstaller constructs the platform-specific Installer.
func NewInstaller(commandRunner certificates.CommandRunner, fileSystem certificates.FileSystem, configuration Configuration) (Installer, error) {
	return nil, fmt.Errorf("unsupported operating system %s", runtime.GOOS)
}
