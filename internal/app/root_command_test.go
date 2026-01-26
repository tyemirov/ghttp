package app

import (
	"context"
	"testing"

	"github.com/spf13/viper"

	"github.com/tyemirov/ghttp/pkg/logging"
)

func TestNewRootCommandProvidesHTTPSFlagOnce(t *testing.T) {
	configurationManager := viper.New()
	resources := &applicationResources{
		configurationManager: configurationManager,
		loggingService:       logging.NewTestService(logging.TypeConsole),
		defaultConfigDirPath: t.TempDir(),
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("newRootCommand panicked: %v", recovered)
		}
	}()

	rootCommand := newRootCommand(resources)
	if rootCommand.Flags().Lookup(flagNameHTTPSHosts) == nil {
		t.Fatalf("expected https-host flag to be registered")
	}
}

func TestRootCommandDoesNotRegisterHTTPSSubcommand(t *testing.T) {
	configurationManager := viper.New()
	resources := &applicationResources{
		configurationManager: configurationManager,
		loggingService:       logging.NewTestService(logging.TypeConsole),
		defaultConfigDirPath: t.TempDir(),
	}

	rootCommand := newRootCommand(resources)
	for _, subCommand := range rootCommand.Commands() {
		if subCommand.Name() == "https" {
			t.Fatalf("unexpected https subcommand registered")
		}
	}
}

func TestNewRootCommandProvidesProxyFlag(testingInstance *testing.T) {
	configurationManager := viper.New()
	resources := &applicationResources{
		configurationManager: configurationManager,
		loggingService:       logging.NewTestService(logging.TypeConsole),
		defaultConfigDirPath: testingInstance.TempDir(),
	}

	rootCommand := newRootCommand(resources)
	if rootCommand.Flags().Lookup(flagNameProxy) == nil {
		testingInstance.Fatalf("expected proxy flag to be registered")
	}
}

func TestRootCommandBindsBrowseFlag(t *testing.T) {
	configurationManager := viper.New()
	resources := &applicationResources{
		configurationManager: configurationManager,
		loggingService:       logging.NewTestService(logging.TypeConsole),
		defaultConfigDirPath: t.TempDir(),
	}

	rootCommand := newRootCommand(resources)
	rootCommand.SetArgs([]string{"--browse"})
	rootCommand.SetContext(context.WithValue(context.Background(), contextKeyApplicationResources, resources))

	parseErr := rootCommand.ParseFlags([]string{"--browse"})
	if parseErr != nil {
		t.Fatalf("parse flags: %v", parseErr)
	}

	if !configurationManager.GetBool(configKeyServeBrowse) {
		t.Fatalf("expected browse flag to bind configuration")
	}
}

func TestRootCommandBindsConfigFlag(t *testing.T) {
	configurationManager := viper.New()
	resources := &applicationResources{
		configurationManager: configurationManager,
		loggingService:       logging.NewTestService(logging.TypeConsole),
		defaultConfigDirPath: t.TempDir(),
	}

	rootCommand := newRootCommand(resources)
	rootCommand.SetArgs([]string{"--config", "/tmp/ghttp.yaml"})
	rootCommand.SetContext(context.WithValue(context.Background(), contextKeyApplicationResources, resources))

	parseErr := rootCommand.ParseFlags([]string{"--config", "/tmp/ghttp.yaml"})
	if parseErr != nil {
		t.Fatalf("parse flags: %v", parseErr)
	}

	if configurationManager.GetString(configKeyConfigFile) != "/tmp/ghttp.yaml" {
		t.Fatalf("expected config flag to bind configuration")
	}
}
