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
	browseModeStartupTimeout = 20 * time.Second
	browseModeRequestTimeout = 5 * time.Second
	processStopTimeout       = 5 * time.Second
)

func TestBrowseModeServesIndexFilesAsRegularFiles(t *testing.T) {
	repositoryRoot := getRepositoryRoot(t)
	serverBinaryPath := buildGHTTPBinaryForIntegrationTests(t, repositoryRoot)
	siteDirectory := createBrowseModeFixtureDirectory(t)
	serverPort := allocateFreePort(t)
	startedServer := startGHTTPServerProcess(t, repositoryRoot, serverBinaryPath, siteDirectory, serverPort, []string{"--browse"})
	baseURL := startedServer.baseURL

	httpClient := &http.Client{Timeout: browseModeRequestTimeout}

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
		t.Run(directoryListingCase.name, func(testingT *testing.T) {
			statusCode, responseHeaders, responseBody := executeHTTPGet(testingT, httpClient, baseURL, directoryListingCase.requestPath)
			if statusCode != http.StatusOK {
				testingT.Fatalf("request %s expected status %d, got %d", directoryListingCase.requestPath, http.StatusOK, statusCode)
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
		t.Run(directFileCase.name, func(testingT *testing.T) {
			statusCode, responseHeaders, responseBody := executeHTTPGet(testingT, httpClient, baseURL, directFileCase.requestPath)
			if statusCode != http.StatusOK {
				testingT.Fatalf("request %s expected status %d, got %d", directFileCase.requestPath, http.StatusOK, statusCode)
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

func allocateFreePort(testingT *testing.T) int {
	testingT.Helper()
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingT.Fatalf("allocate free tcp port: %v", listenErr)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func startGHTTPServerProcess(testingT *testing.T, repositoryRoot string, serverBinaryPath string, directoryPath string, port int, additionalArguments []string) *startedGHTTPServer {
	testingT.Helper()
	arguments := []string{
		strconv.Itoa(port),
		"--directory", directoryPath,
	}
	arguments = append(arguments, additionalArguments...)

	serverCommand := exec.Command(serverBinaryPath, arguments...)
	serverCommand.Dir = repositoryRoot
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
