//go:build windows

package truststore

import (
	"context"
	"errors"
	"fmt"

	"github.com/tyemirov/ghttp/internal/certificates"
)

// NewInstaller constructs the platform-specific Installer.
func NewInstaller(commandRunner certificates.CommandRunner, fileSystem certificates.FileSystem, configuration Configuration) (Installer, error) {
	storeName := configuration.WindowsCertificateStoreName
	if storeName == "" {
		storeName = "Root"
	}
	if configuration.CertificateCommonName == "" {
		return nil, errors.New("windows installer requires certificate common name")
	}
	configuration.WindowsCertificateStoreName = storeName
	return &windowsInstaller{
		commandRunner: commandRunner,
		fileSystem:    fileSystem,
		configuration: configuration,
	}, nil
}

type windowsInstaller struct {
	commandRunner certificates.CommandRunner
	fileSystem    certificates.FileSystem
	configuration Configuration
}

func (installer *windowsInstaller) Install(ctx context.Context, certificatePath string) error {
	if certificatePath == "" {
		return errors.New("certificate path is required")
	}
	arguments := []string{"-user", "-addstore", "-f", installer.configuration.WindowsCertificateStoreName, certificatePath}
	err := installer.commandRunner.Run(ctx, commandNameCertutil, arguments)
	if err != nil {
		return fmt.Errorf("install certificate in windows store: %w", err)
	}
	firefoxErr := integrateFirefoxCertificates(ctx, installer.commandRunner, installer.fileSystem, installer.configuration, certificatePath)
	if firefoxErr != nil {
		return fmt.Errorf("configure firefox trust stores: %w", firefoxErr)
	}
	return nil
}

func (installer *windowsInstaller) Uninstall(ctx context.Context) error {
	arguments := []string{"-user", "-delstore", installer.configuration.WindowsCertificateStoreName, installer.configuration.CertificateCommonName}
	err := installer.commandRunner.Run(ctx, commandNameCertutil, arguments)
	if err != nil {
		return fmt.Errorf("remove certificate from windows store: %w", err)
	}
	firefoxErr := removeFirefoxCertificates(ctx, installer.commandRunner, installer.fileSystem, installer.configuration)
	if firefoxErr != nil {
		return fmt.Errorf("remove firefox trust stores: %w", firefoxErr)
	}
	return nil
}
