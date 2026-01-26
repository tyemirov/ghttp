package app

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
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
