package truststore

import (
	"context"
	"io/fs"
)

const (
	commandNameSecurity = "security"
	commandNameCertutil = "certutil"
	commandNameTrust    = "trust"
)

// Installer provisions and removes certificates from operating system trust stores.
type Installer interface {
	Install(ctx context.Context, certificatePath string) error
	Uninstall(ctx context.Context) error
}

// Configuration controls installer behavior across platforms.
type Configuration struct {
	CertificateCommonName           string
	MacOSKeychainPath               string
	LinuxCertificateDestinationPath string
	LinuxCertificateFilePermissions fs.FileMode
	WindowsCertificateStoreName     string
	FirefoxProfileDirectories       []string
}
