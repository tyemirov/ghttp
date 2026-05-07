package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestBindAddressAcceptsPortQualifiedValues(testingT *testing.T) {
	repositoryRoot := getRepositoryRoot(testingT)
	serverBinaryPath := buildGHTTPBinaryForIntegrationTests(testingT, repositoryRoot)
	siteDirectory := createBrowseModeFixtureDirectory(testingT)

	bindAddressCases := []struct {
		name              string
		formatBindAddress func(serverPort int) string
	}{
		{
			name: "port only bind address",
			formatBindAddress: func(serverPort int) string {
				return ":" + strconv.Itoa(serverPort)
			},
		},
		{
			name: "host and port bind address",
			formatBindAddress: func(serverPort int) string {
				return "127.0.0.1:" + strconv.Itoa(serverPort)
			},
		},
	}

	for _, bindAddressCase := range bindAddressCases {
		bindAddressCase := bindAddressCase
		testingT.Run(bindAddressCase.name, func(caseTestingT *testing.T) {
			serverPort := allocateFreePort(caseTestingT)
			baseURL := fmt.Sprintf("http://127.0.0.1:%d", serverPort)
			startedServer := startGHTTPProcessWithArguments(
				caseTestingT,
				repositoryRoot,
				serverBinaryPath,
				[]string{"--directory", siteDirectory, "--bind", bindAddressCase.formatBindAddress(serverPort)},
				nil,
				baseURL+"/hello.html",
				false,
			)
			_, _, responseBody := executeHTTPGet(caseTestingT, &http.Client{Timeout: browseModeRequestTimeout}, baseURL, "/hello.html")
			if !strings.Contains(responseBody, "ROOT HELLO") {
				caseTestingT.Fatalf("expected served fixture body, got %s", responseBody)
			}
			if stopErr := startedServer.stop(); stopErr != nil {
				caseTestingT.Fatalf("stop bind address server: %v", stopErr)
			}
		})
	}
}

func TestBindAddressRejectsEmptyPortInPortQualifiedValue(testingT *testing.T) {
	repositoryRoot := getRepositoryRoot(testingT)
	serverBinaryPath := buildGHTTPBinaryForIntegrationTests(testingT, repositoryRoot)
	siteDirectory := createBrowseModeFixtureDirectory(testingT)

	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		serverBinaryPath,
		[]string{"--directory", siteDirectory, "--bind", "127.0.0.1:"},
		nil,
		1,
	)
}
