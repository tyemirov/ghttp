package integration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	browseModeStartupTimeout                    = 20 * time.Second
	browseModeRequestTimeout                    = 5 * time.Second
	processStopTimeout                          = 5 * time.Second
	browseHandlerCoveragePackage                = "./cmd/ghttp,./internal/server"
	browseHandlerCoverageSourcePath             = "/internal/server/browse_handler.go"
	browseHandlerCoverageRequiredPercentageText = "100.0%"
)

func TestBrowseModeServesIndexFilesAsRegularFiles(t *testing.T) {
	repositoryRoot := getRepositoryRoot(t)
	serverBinaryPath := buildGHTTPBinaryForIntegrationTests(t, repositoryRoot)
	siteDirectory := createBrowseModeFixtureDirectory(t)
	serverPort := allocateFreePort(t)
	startedServer := startGHTTPServerProcess(t, repositoryRoot, serverBinaryPath, siteDirectory, serverPort, []string{"--browse"}, nil)

	httpClient := &http.Client{Timeout: browseModeRequestTimeout}
	runBrowseModeAssertions(t, httpClient, startedServer.baseURL)
}

func TestBrowseModeBrowseHandlerCoverageGate(t *testing.T) {
	repositoryRoot := getRepositoryRoot(t)
	coverageDirectoryPath := t.TempDir()
	coverageBinaryPath := buildInstrumentedGHTTPBinaryForCoverage(t, repositoryRoot, browseHandlerCoveragePackage)
	siteDirectory := createBrowseModeFixtureDirectory(t)
	serverPort := allocateFreePort(t)
	startedServer := startGHTTPServerProcess(
		t,
		repositoryRoot,
		coverageBinaryPath,
		siteDirectory,
		serverPort,
		[]string{"--browse"},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
	)

	httpClient := &http.Client{Timeout: browseModeRequestTimeout}
	runBrowseModeCoverageRequests(t, httpClient, startedServer.baseURL)

	stopErr := startedServer.stop()
	if stopErr != nil {
		t.Fatalf("stop ghttp process before coverage inspection: %v\nserver logs:\n%s", stopErr, startedServer.logBuffer.String())
	}

	coverageProfilePath := filepath.Join(t.TempDir(), "browse_handler.coverage.out")
	writeCoverageProfileFromDirectory(t, repositoryRoot, coverageDirectoryPath, coverageProfilePath)
	assertBrowseHandlerCoverageAtOneHundredPercent(t, repositoryRoot, coverageProfilePath)
}

func runBrowseModeAssertions(testingT *testing.T, httpClient *http.Client, baseURL string) {
	testingT.Helper()

	directoryListingCases := []struct {
		name               string
		requestPath        string
		expectedLinks      []string
		unexpectedSnippets []string
	}{
		{
			name:        "root listing remains listing despite index files",
			requestPath: "/",
			expectedLinks: []string{
				`href="/example/"`,
				`href="/hello.html"`,
				`href="/index.htm"`,
				`href="/index.html"`,
				`href="/README.md"`,
			},
			unexpectedSnippets: []string{
				"ROOT INDEX HTML",
				"ROOT INDEX HTM",
				"<h1>Root Markdown</h1>",
			},
		},
		{
			name:        "nested listing remains listing despite index files",
			requestPath: "/example/",
			expectedLinks: []string{
				`href="/example/hello.html"`,
				`href="/example/index.htm"`,
				`href="/example/index.html"`,
				`href="/example/README.md"`,
			},
			unexpectedSnippets: []string{
				"NESTED INDEX HTML",
				"NESTED INDEX HTM",
				"<h1>Nested Markdown</h1>",
			},
		},
	}

	for _, directoryListingCase := range directoryListingCases {
		directoryListingCase := directoryListingCase
		testingT.Run(directoryListingCase.name, func(testingT *testing.T) {
			responseStatusCode, responseHeaders, responseBody := executeHTTPGet(testingT, httpClient, baseURL, directoryListingCase.requestPath)
			if responseStatusCode != http.StatusOK {
				testingT.Fatalf("request %s expected status %d, got %d", directoryListingCase.requestPath, http.StatusOK, responseStatusCode)
			}
			contentTypeHeader := responseHeaders.Get("Content-Type")
			if !strings.Contains(contentTypeHeader, "text/html") {
				testingT.Fatalf("request %s expected html listing content type, got %q", directoryListingCase.requestPath, contentTypeHeader)
			}
			for _, expectedLink := range directoryListingCase.expectedLinks {
				if !strings.Contains(responseBody, expectedLink) {
					testingT.Fatalf("request %s expected listing link %q, body: %s", directoryListingCase.requestPath, expectedLink, responseBody)
				}
			}
			for _, unexpectedSnippet := range directoryListingCase.unexpectedSnippets {
				if strings.Contains(responseBody, unexpectedSnippet) {
					testingT.Fatalf("request %s expected pure listing without %q, body: %s", directoryListingCase.requestPath, unexpectedSnippet, responseBody)
				}
			}
		})
	}

	directFileCases := []struct {
		name              string
		requestPath       string
		expectedSnippet   string
		expectedMimeToken string
	}{
		{
			name:              "root index html direct request",
			requestPath:       "/index.html",
			expectedSnippet:   "ROOT INDEX HTML",
			expectedMimeToken: "text/html",
		},
		{
			name:              "root index htm direct request",
			requestPath:       "/index.htm",
			expectedSnippet:   "ROOT INDEX HTM",
			expectedMimeToken: "text/html",
		},
		{
			name:              "root non index html direct request",
			requestPath:       "/hello.html",
			expectedSnippet:   "ROOT HELLO",
			expectedMimeToken: "text/html",
		},
		{
			name:              "root markdown direct request",
			requestPath:       "/README.md",
			expectedSnippet:   "<h1>Root Markdown</h1>",
			expectedMimeToken: "text/html",
		},
		{
			name:              "nested index html direct request",
			requestPath:       "/example/index.html",
			expectedSnippet:   "NESTED INDEX HTML",
			expectedMimeToken: "text/html",
		},
		{
			name:              "nested index htm direct request",
			requestPath:       "/example/index.htm",
			expectedSnippet:   "NESTED INDEX HTM",
			expectedMimeToken: "text/html",
		},
		{
			name:              "nested non index html direct request",
			requestPath:       "/example/hello.html",
			expectedSnippet:   "NESTED HELLO",
			expectedMimeToken: "text/html",
		},
		{
			name:              "nested markdown direct request",
			requestPath:       "/example/README.md",
			expectedSnippet:   "<h1>Nested Markdown</h1>",
			expectedMimeToken: "text/html",
		},
	}

	for _, directFileCase := range directFileCases {
		directFileCase := directFileCase
		testingT.Run(directFileCase.name, func(testingT *testing.T) {
			responseStatusCode, responseHeaders, responseBody := executeHTTPGet(testingT, httpClient, baseURL, directFileCase.requestPath)
			if responseStatusCode != http.StatusOK {
				testingT.Fatalf("request %s expected status %d, got %d", directFileCase.requestPath, http.StatusOK, responseStatusCode)
			}
			if responseHeaders.Get("Location") != "" {
				testingT.Fatalf("request %s expected no redirect location header, got %q", directFileCase.requestPath, responseHeaders.Get("Location"))
			}
			contentTypeHeader := responseHeaders.Get("Content-Type")
			if !strings.Contains(contentTypeHeader, directFileCase.expectedMimeToken) {
				testingT.Fatalf("request %s expected content type containing %q, got %q", directFileCase.requestPath, directFileCase.expectedMimeToken, contentTypeHeader)
			}
			if !strings.Contains(responseBody, directFileCase.expectedSnippet) {
				testingT.Fatalf("request %s expected body snippet %q, body: %s", directFileCase.requestPath, directFileCase.expectedSnippet, responseBody)
			}
		})
	}
}

func runBrowseModeCoverageRequests(testingT *testing.T, httpClient *http.Client, baseURL string) {
	testingT.Helper()
	coverageCases := []struct {
		requestPath         string
		expectedStatusCodes []int
	}{
		{requestPath: "/index.html", expectedStatusCodes: []int{http.StatusOK}},
		{requestPath: "/README.md", expectedStatusCodes: []int{http.StatusOK}},
		{requestPath: "/example", expectedStatusCodes: []int{http.StatusOK, http.StatusMovedPermanently, http.StatusTemporaryRedirect}},
		{requestPath: "/missing", expectedStatusCodes: []int{http.StatusNotFound}},
		{requestPath: "/missing/", expectedStatusCodes: []int{http.StatusNotFound}},
		{requestPath: "/", expectedStatusCodes: []int{http.StatusOK}},
		{requestPath: "/example/", expectedStatusCodes: []int{http.StatusOK}},
		{requestPath: "/unreadable/", expectedStatusCodes: []int{http.StatusForbidden, http.StatusNotFound, http.StatusOK}},
	}

	for _, coverageCase := range coverageCases {
		responseStatusCode, _, _ := executeHTTPGet(testingT, httpClient, baseURL, coverageCase.requestPath)
		if !containsStatusCode(coverageCase.expectedStatusCodes, responseStatusCode) {
			testingT.Fatalf("request %s expected one of statuses %v, got %d", coverageCase.requestPath, coverageCase.expectedStatusCodes, responseStatusCode)
		}
	}
}

func buildGHTTPBinaryForIntegrationTests(testingT *testing.T, repositoryRoot string) string {
	testingT.Helper()
	binaryPath := filepath.Join(testingT.TempDir(), "ghttp-integration")
	buildCommand := exec.Command("go", "build", "-o", binaryPath, "./cmd/ghttp/main.go")
	buildCommand.Dir = repositoryRoot
	buildOutput, buildErr := buildCommand.CombinedOutput()
	if buildErr != nil {
		testingT.Fatalf("build integration binary: %v\n%s", buildErr, string(buildOutput))
	}
	return binaryPath
}

func buildInstrumentedGHTTPBinaryForCoverage(testingT *testing.T, repositoryRoot string, coveragePackage string) string {
	testingT.Helper()
	binaryPath := filepath.Join(testingT.TempDir(), "ghttp-integration-covered")
	buildCommand := exec.Command(
		"go",
		"build",
		"-cover",
		"-coverpkg="+coveragePackage,
		"-o",
		binaryPath,
		"./cmd/ghttp/main.go",
	)
	buildCommand.Dir = repositoryRoot
	buildOutput, buildErr := buildCommand.CombinedOutput()
	if buildErr != nil {
		testingT.Fatalf("build instrumented integration binary: %v\n%s", buildErr, string(buildOutput))
	}
	return binaryPath
}

func writeCoverageProfileFromDirectory(testingT *testing.T, repositoryRoot string, coverageDirectoryPath string, coverageProfilePath string) {
	testingT.Helper()
	coverageExportCommand := exec.Command(
		"go",
		"tool",
		"covdata",
		"textfmt",
		"-i="+coverageDirectoryPath,
		"-o="+coverageProfilePath,
	)
	coverageExportCommand.Dir = repositoryRoot
	coverageExportOutput, coverageExportErr := coverageExportCommand.CombinedOutput()
	if coverageExportErr != nil {
		testingT.Fatalf("export coverage profile: %v\n%s", coverageExportErr, string(coverageExportOutput))
	}
}

func assertBrowseHandlerCoverageAtOneHundredPercent(testingT *testing.T, repositoryRoot string, coverageProfilePath string) {
	testingT.Helper()
	coverageReportCommand := exec.Command("go", "tool", "cover", "-func="+coverageProfilePath)
	coverageReportCommand.Dir = repositoryRoot
	coverageReportOutput, coverageReportErr := coverageReportCommand.CombinedOutput()
	if coverageReportErr != nil {
		testingT.Fatalf("read coverage report: %v\n%s", coverageReportErr, string(coverageReportOutput))
	}

	reportLines := strings.Split(string(coverageReportOutput), "\n")
	foundBrowseHandlerLine := false
	for _, reportLine := range reportLines {
		if !strings.Contains(reportLine, browseHandlerCoverageSourcePath) {
			continue
		}
		foundBrowseHandlerLine = true
		normalizedLine := strings.TrimSpace(reportLine)
		if !strings.HasSuffix(normalizedLine, browseHandlerCoverageRequiredPercentageText) {
			testingT.Fatalf("browse handler coverage below required threshold %s: %s", browseHandlerCoverageRequiredPercentageText, normalizedLine)
		}
	}
	if !foundBrowseHandlerLine {
		testingT.Fatalf("coverage report does not contain browse handler path %s\nreport:\n%s", browseHandlerCoverageSourcePath, string(coverageReportOutput))
	}
}

func allocateFreePort(testingT *testing.T) int {
	testingT.Helper()
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingT.Fatalf("allocate free tcp port: %v", listenErr)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func startGHTTPServerProcess(
	testingT *testing.T,
	repositoryRoot string,
	serverBinaryPath string,
	directoryPath string,
	port int,
	additionalArguments []string,
	environmentVariables map[string]string,
) *startedGHTTPServer {
	testingT.Helper()
	arguments := []string{
		strconv.Itoa(port),
		"--directory", directoryPath,
	}
	arguments = append(arguments, additionalArguments...)

	serverCommand := exec.Command(serverBinaryPath, arguments...)
	serverCommand.Dir = repositoryRoot
	serverCommand.Env = append(os.Environ(), formatEnvironmentVariables(environmentVariables)...)
	serverLogBuffer := &bytes.Buffer{}
	serverCommand.Stdout = serverLogBuffer
	serverCommand.Stderr = serverLogBuffer
	if startErr := serverCommand.Start(); startErr != nil {
		testingT.Fatalf("start ghttp process: %v", startErr)
	}

	startedServer := &startedGHTTPServer{
		command:   serverCommand,
		logBuffer: serverLogBuffer,
		baseURL:   fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	testingT.Cleanup(func() {
		stopErr := startedServer.stop()
		if stopErr != nil {
			testingT.Errorf("stop ghttp process: %v\nserver logs:\n%s", stopErr, startedServer.logBuffer.String())
		}
	})

	httpClient := &http.Client{Timeout: browseModeRequestTimeout}
	startDeadline := time.Now().Add(browseModeStartupTimeout)
	for time.Now().Before(startDeadline) {
		request, requestErr := http.NewRequest(http.MethodGet, startedServer.baseURL+"/", nil)
		if requestErr != nil {
			stopErr := startedServer.stop()
			if stopErr != nil {
				testingT.Fatalf("create startup probe request: %v\nstop server: %v\nserver logs:\n%s", requestErr, stopErr, startedServer.logBuffer.String())
			}
			testingT.Fatalf("create startup probe request: %v", requestErr)
		}
		response, responseErr := httpClient.Do(request)
		if responseErr == nil {
			response.Body.Close()
			return startedServer
		}
		time.Sleep(100 * time.Millisecond)
	}

	stopErr := startedServer.stop()
	if stopErr != nil {
		testingT.Fatalf("ghttp process did not become ready within %s\nstop server: %v\nserver logs:\n%s", browseModeStartupTimeout, stopErr, startedServer.logBuffer.String())
	}
	testingT.Fatalf("ghttp process did not become ready within %s\nserver logs:\n%s", browseModeStartupTimeout, startedServer.logBuffer.String())
	return nil
}

func formatEnvironmentVariables(environmentVariables map[string]string) []string {
	if len(environmentVariables) == 0 {
		return nil
	}
	formattedVariables := make([]string, 0, len(environmentVariables))
	for variableName, variableValue := range environmentVariables {
		formattedVariables = append(formattedVariables, variableName+"="+variableValue)
	}
	return formattedVariables
}

type startedGHTTPServer struct {
	command   *exec.Cmd
	logBuffer *bytes.Buffer
	baseURL   string
	stopOnce  sync.Once
	stopErr   error
}

func (startedServer *startedGHTTPServer) stop() error {
	startedServer.stopOnce.Do(func() {
		startedServer.stopErr = stopServerProcess(startedServer.command)
	})
	return startedServer.stopErr
}

func stopServerProcess(serverCommand *exec.Cmd) error {
	if serverCommand == nil || serverCommand.Process == nil {
		return nil
	}
	signalErr := serverCommand.Process.Signal(os.Interrupt)
	if signalErr != nil && !errors.Is(signalErr, os.ErrProcessDone) {
		return fmt.Errorf("signal interrupt: %w", signalErr)
	}
	waitErrorChannel := make(chan error, 1)
	go func() {
		waitErrorChannel <- serverCommand.Wait()
	}()
	select {
	case waitErr := <-waitErrorChannel:
		if waitErr == nil {
			return nil
		}
		var processExitError *exec.ExitError
		if errors.As(waitErr, &processExitError) {
			return nil
		}
		return fmt.Errorf("wait process: %w", waitErr)
	case <-time.After(processStopTimeout):
		_ = serverCommand.Process.Kill()
		waitErr := <-waitErrorChannel
		return fmt.Errorf("process did not exit within %s: %w", processStopTimeout, waitErr)
	}
}

func executeHTTPGet(testingT *testing.T, httpClient *http.Client, baseURL string, requestPath string) (int, http.Header, string) {
	testingT.Helper()
	requestURL := baseURL + requestPath
	request, requestErr := http.NewRequest(http.MethodGet, requestURL, nil)
	if requestErr != nil {
		testingT.Fatalf("create request %s: %v", requestURL, requestErr)
	}
	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		testingT.Fatalf("perform request %s: %v", requestURL, responseErr)
	}
	defer response.Body.Close()

	responseBodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		testingT.Fatalf("read body for %s: %v", requestURL, readErr)
	}
	return response.StatusCode, response.Header.Clone(), string(responseBodyBytes)
}

func createBrowseModeFixtureDirectory(testingT *testing.T) string {
	testingT.Helper()
	siteDirectory := testingT.TempDir()
	nestedDirectory := filepath.Join(siteDirectory, "example")
	if makeDirectoryErr := os.MkdirAll(nestedDirectory, 0o755); makeDirectoryErr != nil {
		testingT.Fatalf("create nested directory: %v", makeDirectoryErr)
	}
	unreadableDirectory := filepath.Join(siteDirectory, "unreadable")
	if makeDirectoryErr := os.MkdirAll(unreadableDirectory, 0o755); makeDirectoryErr != nil {
		testingT.Fatalf("create unreadable directory: %v", makeDirectoryErr)
	}
	if chmodErr := os.Chmod(unreadableDirectory, 0o111); chmodErr != nil {
		testingT.Fatalf("chmod unreadable directory: %v", chmodErr)
	}

	fileContentByPath := map[string]string{
		filepath.Join(siteDirectory, "index.html"):   "<html><body>ROOT INDEX HTML</body></html>",
		filepath.Join(siteDirectory, "index.htm"):    "<html><body>ROOT INDEX HTM</body></html>",
		filepath.Join(siteDirectory, "hello.html"):   "<html><body>ROOT HELLO</body></html>",
		filepath.Join(siteDirectory, "README.md"):    "# Root Markdown\n",
		filepath.Join(nestedDirectory, "index.html"): "<html><body>NESTED INDEX HTML</body></html>",
		filepath.Join(nestedDirectory, "index.htm"):  "<html><body>NESTED INDEX HTM</body></html>",
		filepath.Join(nestedDirectory, "hello.html"): "<html><body>NESTED HELLO</body></html>",
		filepath.Join(nestedDirectory, "README.md"):  "# Nested Markdown\n",
	}
	for filePath, fileContent := range fileContentByPath {
		if writeErr := os.WriteFile(filePath, []byte(fileContent), 0o644); writeErr != nil {
			testingT.Fatalf("write fixture file %s: %v", filePath, writeErr)
		}
	}

	return siteDirectory
}

func containsStatusCode(expectedStatusCodes []int, statusCode int) bool {
	for _, expectedStatusCode := range expectedStatusCodes {
		if expectedStatusCode == statusCode {
			return true
		}
	}
	return false
}

func getRepositoryRoot(testingT *testing.T) string {
	testingT.Helper()

	currentDirectory, directoryError := os.Getwd()
	if directoryError != nil {
		testingT.Fatalf("resolve working directory: %v", directoryError)
	}

	for {
		dockerfilePath := filepath.Join(currentDirectory, "Dockerfile")
		if _, statError := os.Stat(dockerfilePath); statError == nil {
			return currentDirectory
		}

		parentDirectory := filepath.Dir(currentDirectory)
		if parentDirectory == currentDirectory {
			testingT.Fatal("could not locate repository root")
		}
		currentDirectory = parentDirectory
	}
}
