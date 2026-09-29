//go:build darwin

package truststore

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tyemirov/ghttp/internal/certificates"
)

// NewInstaller constructs the platform-specific Installer.
func NewInstaller(commandRunner certificates.CommandRunner, fileSystem certificates.FileSystem, configuration Configuration) (Installer, error) {
	if configuration.CertificateCommonName == "" {
		return nil, errors.New("macos installer requires certificate common name")
	}
	keychainPath := configuration.MacOSKeychainPath
	if keychainPath == "" {
		homeDirectory, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, fmt.Errorf("resolve home directory: %w", homeErr)
		}
		keychainPath = filepath.Join(homeDirectory, "Library", "Keychains", "login.keychain-db")
	}
	configuration.MacOSKeychainPath = keychainPath
	return &macOSInstaller{
		commandRunner: commandRunner,
		fileSystem:    fileSystem,
		configuration: configuration,
	}, nil
}

type macOSInstaller struct {
	commandRunner          certificates.CommandRunner
	fileSystem             certificates.FileSystem
	configuration          Configuration
	certificateFingerprint string
}

func (installer *macOSInstaller) Install(ctx context.Context, certificatePath string) error {
	if certificatePath == "" {
		return errors.New("certificate path is required")
	}
	certificatePEM, readErr := installer.fileSystem.ReadFile(certificatePath)
	if readErr != nil {
		return fmt.Errorf("read certificate for macos keychain: %w", readErr)
	}
	certificateBlock, _ := pem.Decode(certificatePEM)
	if certificateBlock == nil || certificateBlock.Type != "CERTIFICATE" {
		return errors.New("macos keychain requires a PEM certificate")
	}
	certificate, parseErr := x509.ParseCertificate(certificateBlock.Bytes)
	if parseErr != nil {
		return fmt.Errorf("parse certificate for macos keychain: %w", parseErr)
	}
	verifyErr := installer.commandRunner.Run(ctx, commandNameSecurity, []string{"verify-cert", "-c", certificatePath, "-p", "basic", "-L", "-l", "-q"})
	if verifyErr != nil {
		var exitError interface{ ExitCode() int }
		if !errors.As(verifyErr, &exitError) || exitError.ExitCode() != 1 {
			return fmt.Errorf("verify certificate in macos keychain: %w", verifyErr)
		}
		arguments := []string{"add-trusted-cert", "-r", "trustRoot", "-k", installer.configuration.MacOSKeychainPath, certificatePath}
		if err := installer.commandRunner.Run(ctx, commandNameSecurity, arguments); err != nil {
			return fmt.Errorf("install certificate in macos keychain: %w", err)
		}
	}
	installer.certificateFingerprint = fmt.Sprintf("%X", sha256.Sum256(certificate.Raw))
	firefoxErr := integrateFirefoxCertificates(ctx, installer.commandRunner, installer.fileSystem, installer.configuration, certificatePath)
	if firefoxErr != nil {
		return fmt.Errorf("configure firefox trust stores: %w", firefoxErr)
	}
	return nil
}

func (installer *macOSInstaller) Uninstall(ctx context.Context) error {
	if installer.certificateFingerprint == "" {
		return errors.New("macos installer has no installed certificate")
	}
	arguments := []string{"delete-certificate", "-Z", installer.certificateFingerprint, "-t", installer.configuration.MacOSKeychainPath}
	err := installer.commandRunner.Run(ctx, commandNameSecurity, arguments)
	if err != nil {
		return fmt.Errorf("remove certificate from macos keychain: %w", err)
	}
	firefoxErr := removeFirefoxCertificates(ctx, installer.commandRunner, installer.fileSystem, installer.configuration)
	if firefoxErr != nil {
		return fmt.Errorf("remove firefox trust stores: %w", firefoxErr)
	}
	return nil
}
