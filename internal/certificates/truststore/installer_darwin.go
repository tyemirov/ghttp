//go:build darwin

package truststore

import (
	"context"
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
	commandRunner certificates.CommandRunner
	fileSystem    certificates.FileSystem
	configuration Configuration
}

func (installer *macOSInstaller) Install(ctx context.Context, certificatePath string) error {
	if certificatePath == "" {
		return errors.New("certificate path is required")
	}
	arguments := []string{"add-trusted-cert", "-r", "trustRoot", "-k", installer.configuration.MacOSKeychainPath, certificatePath}
	err := installer.commandRunner.Run(ctx, commandNameSecurity, arguments)
	if err != nil {
		return fmt.Errorf("install certificate in macos keychain: %w", err)
	}
	firefoxErr := integrateFirefoxCertificates(ctx, installer.commandRunner, installer.fileSystem, installer.configuration, certificatePath)
	if firefoxErr != nil {
		return fmt.Errorf("configure firefox trust stores: %w", firefoxErr)
	}
	return nil
}

func (installer *macOSInstaller) Uninstall(ctx context.Context) error {
	arguments := []string{"delete-certificate", "-c", installer.configuration.CertificateCommonName, installer.configuration.MacOSKeychainPath}
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
