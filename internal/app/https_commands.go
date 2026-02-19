package app

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tyemirov/ghttp/internal/certificates"
	"github.com/tyemirov/ghttp/internal/certificates/truststore"
	"github.com/tyemirov/ghttp/internal/server"
	"github.com/tyemirov/ghttp/internal/serverdetails"
	"github.com/tyemirov/ghttp/pkg/logging"
)

const (
	certificateAuthorityKeyBits          = 4096
	leafCertificateKeyBits               = 2048
	certificateAuthorityValidityDuration = 5 * 365 * 24 * time.Hour
	certificateAuthorityRenewalWindow    = 30 * 24 * time.Hour
	leafCertificateValidityDuration      = 30 * 24 * time.Hour
	leafCertificateRenewalWindow         = 72 * time.Hour
	logFieldCertificateDirectory         = "certificate_directory"
	logFieldHosts                        = "hosts"
)

func runHTTPSSetup(cmd *cobra.Command) error {
	resources, err := getApplicationResources(cmd)
	if err != nil {
		return err
	}
	certificateDirectory, err := resolveCertificateDirectory(resources.configurationManager)
	if err != nil {
		return err
	}

	fileSystem := certificates.NewOperatingSystemFileSystem()
	certificateConfiguration := buildCertificateAuthorityConfiguration(certificateDirectory)
	manager := certificates.NewCertificateAuthorityManager(fileSystem, certificates.NewSystemClock(), rand.Reader, certificateConfiguration)
	_, ensureErr := manager.EnsureCertificateAuthority(cmd.Context())
	if ensureErr != nil {
		return fmt.Errorf("ensure certificate authority: %w", ensureErr)
	}

	installer, installerErr := buildTrustStoreInstaller(fileSystem)
	if installerErr != nil {
		return installerErr
	}
	installErr := installer.Install(cmd.Context(), filepath.Join(certificateDirectory, certificates.DefaultRootCertificateFileName))
	if installErr != nil {
		return fmt.Errorf("install certificate authority: %w", installErr)
	}

	logCertificateMessage(resources, "certificate authority installed", certificateDirectory)
	return nil
}

func executeHTTPSServe(cmd *cobra.Command, resources *applicationResources, serveConfiguration ServeConfiguration, hosts []string, certificateDirectory string) error {
	fileSystem := certificates.NewOperatingSystemFileSystem()
	certificateAuthorityConfiguration := buildCertificateAuthorityConfiguration(certificateDirectory)
	certificateAuthorityManager := certificates.NewCertificateAuthorityManager(fileSystem, certificates.NewSystemClock(), rand.Reader, certificateAuthorityConfiguration)
	certificateAuthorityMaterial, ensureErr := certificateAuthorityManager.EnsureCertificateAuthority(cmd.Context())
	if ensureErr != nil {
		return fmt.Errorf("ensure certificate authority: %w", ensureErr)
	}

	issuerConfiguration := certificates.ServerCertificateConfiguration{
		CertificateValidityDuration:      leafCertificateValidityDuration,
		CertificateRenewalWindowDuration: leafCertificateRenewalWindow,
		LeafPrivateKeyBitSize:            leafCertificateKeyBits,
		CertificateFilePermissions:       0o600,
		PrivateKeyFilePermissions:        0o600,
	}
	issuer := certificates.NewServerCertificateIssuer(fileSystem, certificates.NewSystemClock(), rand.Reader, issuerConfiguration)
	leafCertificatePath := filepath.Join(certificateDirectory, certificates.DefaultLeafCertificateFileName)
	leafKeyPath := filepath.Join(certificateDirectory, certificates.DefaultLeafPrivateKeyFileName)
	serverCertificateRequest := certificates.ServerCertificateRequest{
		Hosts:                 hosts,
		CertificateOutputPath: leafCertificatePath,
		PrivateKeyOutputPath:  leafKeyPath,
	}
	leafMaterial, issueErr := issuer.IssueServerCertificate(cmd.Context(), certificateAuthorityMaterial, serverCertificateRequest)
	if issueErr != nil {
		return fmt.Errorf("issue server certificate: %w", issueErr)
	}

	tlsCertificate, parseErr := tls.X509KeyPair(leafMaterial.CertificateBytes, leafMaterial.PrivateKeyBytes)
	if parseErr != nil {
		return fmt.Errorf("parse server certificate: %w", parseErr)
	}

	fileServerConfiguration := server.FileServerConfiguration{
		BindAddress:             serveConfiguration.BindAddress,
		Port:                    serveConfiguration.Port,
		DirectoryPath:           serveConfiguration.DirectoryPath,
		ProtocolVersion:         serveConfiguration.ProtocolVersion,
		DisableDirectoryListing: serveConfiguration.DisableDirectoryListing,
		EnableMarkdown:          serveConfiguration.EnableMarkdown,
		BrowseDirectories:       serveConfiguration.BrowseDirectories,
		InitialFileRelativePath: serveConfiguration.InitialFileRelativePath,
		LoggingType:             serveConfiguration.LoggingType,
		ProxyRoutes:             serveConfiguration.ProxyRoutes,
		TLS: &server.TLSConfiguration{
			LoadedCertificate: &tlsCertificate,
		},
	}

	logServingHTTPSMessage(resources, certificateDirectory, hosts)
	servingAddressFormatter := serverdetails.NewServingAddressFormatter()
	fileServerInstance := server.NewFileServer(resources.loggingService, servingAddressFormatter)
	serveContext, cancel := createSignalContext(cmd.Context(), resources.loggingService)
	defer cancel()

	return fileServerInstance.Serve(serveContext, fileServerConfiguration)
}

func runHTTPSUninstall(cmd *cobra.Command) error {
	resources, err := getApplicationResources(cmd)
	if err != nil {
		return err
	}
	certificateDirectory, err := resolveCertificateDirectory(resources.configurationManager)
	if err != nil {
		return err
	}

	fileSystem := certificates.NewOperatingSystemFileSystem()
	installer, installerErr := buildTrustStoreInstaller(fileSystem)
	if installerErr != nil {
		return installerErr
	}
	uninstallErr := installer.Uninstall(cmd.Context())
	if uninstallErr != nil {
		return fmt.Errorf("uninstall certificate authority: %w", uninstallErr)
	}
	removalTargets := []string{
		filepath.Join(certificateDirectory, certificates.DefaultRootCertificateFileName),
		filepath.Join(certificateDirectory, certificates.DefaultRootPrivateKeyFileName),
		filepath.Join(certificateDirectory, certificates.DefaultLeafCertificateFileName),
		filepath.Join(certificateDirectory, certificates.DefaultLeafPrivateKeyFileName),
	}
	for _, target := range removalTargets {
		_ = fileSystem.Remove(target)
	}
	logCertificateMessage(resources, "certificate authority uninstalled", certificateDirectory)
	return nil
}

func prepareHTTPSContext(cmd *cobra.Command) error {
	resources, err := getApplicationResources(cmd)
	if err != nil {
		return err
	}
	hosts := sanitizeHosts(resources.configurationManager.GetStringSlice(configKeyHTTPSHosts))
	if len(hosts) == 0 {
		return errors.New("at least one host must be specified")
	}
	certificateDirectory, err := resolveCertificateDirectory(resources.configurationManager)
	if err != nil {
		return err
	}
	updatedContext := context.WithValue(cmd.Context(), contextKeyHTTPSHosts, hosts)
	updatedContext = context.WithValue(updatedContext, contextKeyHTTPSCertificateDir, certificateDirectory)
	cmd.SetContext(updatedContext)
	return nil
}

func resolveCertificateDirectory(configurationManager *viper.Viper) (string, error) {
	directoryValue := strings.TrimSpace(configurationManager.GetString(configKeyHTTPSCertificateDir))
	if directoryValue == "" {
		return "", errors.New("certificate directory is not configured")
	}
	absoluteDirectory, err := filepath.Abs(directoryValue)
	if err != nil {
		return "", fmt.Errorf("resolve certificate directory: %w", err)
	}
	return absoluteDirectory, nil
}

func buildCertificateAuthorityConfiguration(certificateDirectory string) certificates.CertificateAuthorityConfiguration {
	return certificates.CertificateAuthorityConfiguration{
		DirectoryPath:                    certificateDirectory,
		CertificateFileName:              certificates.DefaultRootCertificateFileName,
		PrivateKeyFileName:               certificates.DefaultRootPrivateKeyFileName,
		DirectoryPermissions:             0o700,
		CertificateFilePermissions:       0o600,
		PrivateKeyFilePermissions:        0o600,
		RSAKeyBitSize:                    certificateAuthorityKeyBits,
		CertificateValidityDuration:      certificateAuthorityValidityDuration,
		CertificateRenewalWindowDuration: certificateAuthorityRenewalWindow,
		SubjectCommonName:                certificates.DefaultCertificateAuthorityCommonName,
		SubjectOrganizationalUnit:        certificates.DefaultCertificateAuthorityOrganizationalUnit,
		SubjectOrganization:              certificates.DefaultCertificateAuthorityOrganization,
	}
}

func buildTrustStoreInstaller(fileSystem certificates.FileSystem) (truststore.Installer, error) {
	commandRunner := certificates.NewExecutableRunner()
	linuxDestinationPath := ""
	if runtime.GOOS == "linux" {
		homeDirectory, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, fmt.Errorf("resolve home directory: %w", homeErr)
		}
		linuxDestinationPath = filepath.Join(homeDirectory, ".local", "share", "ca-certificates", certificates.DefaultRootCertificateFileName)
	}
	configuration := truststore.Configuration{
		CertificateCommonName:           certificates.DefaultCertificateAuthorityCommonName,
		LinuxCertificateDestinationPath: linuxDestinationPath,
		LinuxCertificateFilePermissions: 0o644,
		WindowsCertificateStoreName:     "Root",
	}
	return truststore.NewInstaller(commandRunner, fileSystem, configuration)
}

func sanitizeHosts(hosts []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(hosts))
	for _, host := range hosts {
		normalizedHost := strings.TrimSpace(host)
		if normalizedHost == "" {
			continue
		}
		if _, exists := seen[normalizedHost]; exists {
			continue
		}
		seen[normalizedHost] = struct{}{}
		result = append(result, normalizedHost)
	}
	return result
}

func certificateDirectoryField(path string) logging.Field {
	return logging.String(logFieldCertificateDirectory, path)
}

func logCertificateMessage(resources *applicationResources, message string, directory string) {
	if resources.loggingService == nil {
		return
	}
	resources.loggingService.Info(message, certificateDirectoryField(directory))
}

func logServingHTTPSMessage(resources *applicationResources, directory string, hosts []string) {
	if resources.loggingService == nil {
		return
	}
	resources.loggingService.Info("serving https", certificateDirectoryField(directory), logging.Strings(logFieldHosts, hosts))
}
