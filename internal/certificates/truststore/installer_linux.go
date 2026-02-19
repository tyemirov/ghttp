//go:build linux

package truststore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/tyemirov/ghttp/internal/certificates"
)

// NewInstaller constructs the platform-specific Installer.
func NewInstaller(commandRunner certificates.CommandRunner, fileSystem certificates.FileSystem, configuration Configuration) (Installer, error) {
	if configuration.LinuxCertificateFilePermissions == 0 {
		configuration.LinuxCertificateFilePermissions = 0o644
	}
	return &linuxInstaller{
		commandRunner: commandRunner,
		fileSystem:    fileSystem,
		configuration: configuration,
	}, nil
}

type linuxInstaller struct {
	commandRunner certificates.CommandRunner
	fileSystem    certificates.FileSystem
	configuration Configuration
	installedPath string
}

func (installer *linuxInstaller) Install(ctx context.Context, certificatePath string) error {
	if certificatePath == "" {
		return errors.New("certificate path is required")
	}
	exists, existsErr := installer.fileSystem.FileExists(certificatePath)
	if existsErr != nil {
		return fmt.Errorf("check certificate path: %w", existsErr)
	}
	if !exists {
		return fmt.Errorf("certificate path does not exist: %s", certificatePath)
	}
	anchorPath := certificatePath
	if installer.configuration.LinuxCertificateDestinationPath != "" {
		destinationDirectory := filepath.Dir(installer.configuration.LinuxCertificateDestinationPath)
		if err := installer.fileSystem.EnsureDirectory(destinationDirectory, 0o755); err != nil {
			return fmt.Errorf("ensure linux certificate directory: %w", err)
		}
		content, readErr := installer.fileSystem.ReadFile(certificatePath)
		if readErr != nil {
			return fmt.Errorf("read certificate: %w", readErr)
		}
		if err := installer.fileSystem.WriteFile(installer.configuration.LinuxCertificateDestinationPath, content, installer.configuration.LinuxCertificateFilePermissions); err != nil {
			return fmt.Errorf("write linux trust certificate: %w", err)
		}
		anchorPath = installer.configuration.LinuxCertificateDestinationPath
	}
	installErr := installer.commandRunner.Run(ctx, commandNameTrust, []string{"anchor", anchorPath})
	if installErr != nil {
		return fmt.Errorf("configure linux trust store: %w", installErr)
	}
	installer.installedPath = anchorPath
	firefoxErr := integrateFirefoxCertificates(ctx, installer.commandRunner, installer.fileSystem, installer.configuration, certificatePath)
	if firefoxErr != nil {
		return fmt.Errorf("configure firefox trust stores: %w", firefoxErr)
	}
	return nil
}

func (installer *linuxInstaller) Uninstall(ctx context.Context) error {
	anchorPath := installer.installedPath
	if anchorPath == "" {
		anchorPath = installer.configuration.LinuxCertificateDestinationPath
	}
	if anchorPath == "" {
		return errors.New("linux installer has no recorded certificate path")
	}
	removeErr := installer.commandRunner.Run(ctx, commandNameTrust, []string{"anchor", "--remove", anchorPath})
	if removeErr != nil {
		return fmt.Errorf("remove linux trust store certificate: %w", removeErr)
	}
	if installer.configuration.LinuxCertificateDestinationPath != "" {
		if err := installer.fileSystem.Remove(installer.configuration.LinuxCertificateDestinationPath); err != nil {
			return fmt.Errorf("remove linux certificate file: %w", err)
		}
	}
	firefoxErr := removeFirefoxCertificates(ctx, installer.commandRunner, installer.fileSystem, installer.configuration)
	if firefoxErr != nil {
		return fmt.Errorf("remove firefox trust stores: %w", firefoxErr)
	}
	return nil
}
