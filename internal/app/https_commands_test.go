package app

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/tyemirov/ghttp/pkg/logging"
)

func TestPrepareHTTPSContext(testingInstance *testing.T) {
	testCases := []struct {
		name          string
		hosts         []string
		expectedHosts []string
		expectError   bool
	}{
		{
			name:          "sanitizes and stores hosts",
			hosts:         []string{"localhost", " example.test ", "", "localhost", "127.0.0.1"},
			expectedHosts: []string{"localhost", "example.test", "127.0.0.1"},
			expectError:   false,
		},
		{
			name:          "rejects empty hosts",
			hosts:         []string{" ", ""},
			expectedHosts: nil,
			expectError:   true,
		},
	}

	for _, testCase := range testCases {
		testingInstance.Run(testCase.name, func(testingInstance *testing.T) {
			configurationManager := viper.New()
			configurationManager.Set(configKeyHTTPSHosts, testCase.hosts)
			certificateDirectory := testingInstance.TempDir()
			configurationManager.Set(configKeyHTTPSCertificateDir, certificateDirectory)

			resources := &applicationResources{
				configurationManager: configurationManager,
				loggingService:       logging.NewTestService(logging.TypeConsole),
				defaultConfigDirPath: testingInstance.TempDir(),
			}

			command := &cobra.Command{}
			command.SetContext(context.WithValue(context.Background(), contextKeyApplicationResources, resources))

			err := prepareHTTPSContext(command)
			if testCase.expectError {
				if err == nil {
					testingInstance.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				testingInstance.Fatalf("prepare https context: %v", err)
			}

			hostsValue := command.Context().Value(contextKeyHTTPSHosts)
			if hostsValue == nil {
				testingInstance.Fatalf("expected https hosts in context")
			}
			hosts, ok := hostsValue.([]string)
			if !ok {
				testingInstance.Fatalf("expected https hosts slice in context")
			}
			if !reflect.DeepEqual(hosts, testCase.expectedHosts) {
				testingInstance.Fatalf("expected hosts %v, got %v", testCase.expectedHosts, hosts)
			}

			certificateDirectoryValue := command.Context().Value(contextKeyHTTPSCertificateDir)
			if certificateDirectoryValue == nil {
				testingInstance.Fatalf("expected https certificate directory in context")
			}
			certificateDirectoryPath, ok := certificateDirectoryValue.(string)
			if !ok {
				testingInstance.Fatalf("expected https certificate directory string in context")
			}
			absoluteCertificateDirectory, absoluteErr := filepath.Abs(certificateDirectory)
			if absoluteErr != nil {
				testingInstance.Fatalf("resolve certificate directory: %v", absoluteErr)
			}
			if certificateDirectoryPath != absoluteCertificateDirectory {
				testingInstance.Fatalf("expected certificate directory %s, got %s", absoluteCertificateDirectory, certificateDirectoryPath)
			}
		})
	}
}

func TestHTTPSCommandCertificateDirectoryFlagRegistration(testingInstance *testing.T) {
	testCases := []struct {
		name          string
		flagName      string
		expectPresent bool
	}{
		{
			name:          "new flag present",
			flagName:      flagNameCertificateDir,
			expectPresent: true,
		},
		{
			name:          "old flag absent",
			flagName:      "cert-dir",
			expectPresent: false,
		},
	}

	for _, testCase := range testCases {
		testingInstance.Run(testCase.name, func(testingInstance *testing.T) {
			configurationManager := viper.New()
			resources := &applicationResources{
				configurationManager: configurationManager,
				loggingService:       logging.NewTestService(logging.TypeConsole),
				defaultConfigDirPath: testingInstance.TempDir(),
			}

			serveFlags := pflag.NewFlagSet("serve", pflag.ContinueOnError)
			configureServeFlags(serveFlags, resources.configurationManager)
			httpsOptionFlags := pflag.NewFlagSet("serve-https-options", pflag.ContinueOnError)
			configureServeHTTPSOptions(httpsOptionFlags, resources.configurationManager)

			httpsCommand := newHTTPSCommand(resources, serveFlags, httpsOptionFlags)
			lookupResult := httpsCommand.PersistentFlags().Lookup(testCase.flagName)
			if testCase.expectPresent && lookupResult == nil {
				testingInstance.Fatalf("expected flag %s to be registered", testCase.flagName)
			}
			if !testCase.expectPresent && lookupResult != nil {
				testingInstance.Fatalf("expected flag %s to be absent", testCase.flagName)
			}
		})
	}
}

func TestHTTPSCommandBindsCertificateDirectoryFlag(testingInstance *testing.T) {
	configurationManager := viper.New()
	resources := &applicationResources{
		configurationManager: configurationManager,
		loggingService:       logging.NewTestService(logging.TypeConsole),
		defaultConfigDirPath: testingInstance.TempDir(),
	}

	serveFlags := pflag.NewFlagSet("serve", pflag.ContinueOnError)
	configureServeFlags(serveFlags, resources.configurationManager)
	httpsOptionFlags := pflag.NewFlagSet("serve-https-options", pflag.ContinueOnError)
	configureServeHTTPSOptions(httpsOptionFlags, resources.configurationManager)

	httpsCommand := newHTTPSCommand(resources, serveFlags, httpsOptionFlags)
	desiredCertificateDirectory := testingInstance.TempDir()
	parseErr := httpsCommand.ParseFlags([]string{"--" + flagNameCertificateDir, desiredCertificateDirectory})
	if parseErr != nil {
		testingInstance.Fatalf("parse flags: %v", parseErr)
	}

	configuredCertificateDirectory := configurationManager.GetString(configKeyHTTPSCertificateDir)
	if configuredCertificateDirectory != desiredCertificateDirectory {
		testingInstance.Fatalf("expected certificate directory %s, got %s", desiredCertificateDirectory, configuredCertificateDirectory)
	}
}
