package integration

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	globalCoveragePackage                = "./..."
	globalCoverageRequiredPercentageText = "85.0%"
	integrationCommandTimeout            = 20 * time.Second
)

type comprehensiveFixture struct {
	siteDirectory      string
	landingFilePath    string
	unsupportedTxtPath string
}

type fakeSystemTools struct {
	binDirectoryPath        string
	trustOnlyPath           string
	trustAndCertutilPath    string
	trustFailureOnRemoveLog string
}

func TestGlobalIntegrationCoverageGate(t *testing.T) {
	repositoryRoot := getRepositoryRoot(t)
	coverageDirectoryPath := t.TempDir()
	instrumentedCommandBinary := buildInstrumentedBinaryForTarget(t, repositoryRoot, "./cmd/ghttp/main.go", globalCoveragePackage, "ghttp-covered-cmd")
	instrumentedRootBinary := buildInstrumentedBinaryForTarget(t, repositoryRoot, "./main.go", globalCoveragePackage, "ghttp-covered-root")
	fixture := createComprehensiveFixtureDirectory(t)
	tools := createFakeSystemTools(t)

	coverageEnvironment := map[string]string{
		"GOCOVERDIR": coverageDirectoryPath,
	}

	runCommandExpectExitCode(t, repositoryRoot, instrumentedRootBinary, []string{"--help"}, coverageEnvironment, 0)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedRootBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--logging-type", "broken"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(t, repositoryRoot, instrumentedCommandBinary, []string{"--help"}, coverageEnvironment, 0)

	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"invalid-port", "--directory", fixture.siteDirectory},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--protocol", "HTTP/2.0"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--logging-type", "broken"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy", "/api=http://127.0.0.1:1", "--proxy-backend", "http://127.0.0.1:2"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy-path", "/api"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy-backend", "http://127.0.0.1:1"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy", "no-separator"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy", "=http://127.0.0.1:1"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy", "/api=ftp://127.0.0.1:1"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy", "/api=http://"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy", "/api=http://localhost:"},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", fixture.siteDirectory, "--proxy", "/api=http://a", "--proxy", "/api=http://b"},
		coverageEnvironment,
		1,
	)

	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{fixture.unsupportedTxtPath},
		map[string]string{
			"GOCOVERDIR":       coverageDirectoryPath,
			"GHTTP_SERVE_PORT": "18080",
		},
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{fixture.siteDirectory},
		map[string]string{
			"GOCOVERDIR":       coverageDirectoryPath,
			"GHTTP_SERVE_PORT": "18081",
		},
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{filepath.Join(fixture.siteDirectory, "does-not-exist.md")},
		map[string]string{
			"GOCOVERDIR":       coverageDirectoryPath,
			"GHTTP_SERVE_PORT": "18082",
		},
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"70000", "--directory", fixture.siteDirectory},
		coverageEnvironment,
		1,
	)
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"8080", "--directory", filepath.Join(fixture.siteDirectory, "hello.html")},
		coverageEnvironment,
		1,
	)

	invalidConfigurationPath := filepath.Join(t.TempDir(), "invalid.yaml")
	if writeErr := os.WriteFile(invalidConfigurationPath, []byte("serve: ["), 0o644); writeErr != nil {
		t.Fatalf("write invalid config: %v", writeErr)
	}
	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"--config", invalidConfigurationPath},
		coverageEnvironment,
		1,
	)

	runCommandExpectExitCode(
		t,
		repositoryRoot,
		instrumentedCommandBinary,
		[]string{"--help"},
		map[string]string{
			"GOCOVERDIR":       coverageDirectoryPath,
			"HOME":             "",
			"XDG_CONFIG_HOME":  "",
			"GHTTP_SERVE_PORT": "18081",
		},
		1,
	)

	exerciseStandardHTTPServerFlows(t, repositoryRoot, instrumentedCommandBinary, fixture.siteDirectory, coverageDirectoryPath)
	exerciseInitialFileFlow(t, repositoryRoot, instrumentedCommandBinary, fixture.landingFilePath, coverageDirectoryPath)
	exerciseHTTPProxyFlows(t, repositoryRoot, instrumentedCommandBinary, fixture.siteDirectory, coverageDirectoryPath)
	exerciseWebSocketProxyFlows(t, repositoryRoot, instrumentedCommandBinary, coverageDirectoryPath)
	exerciseManualTLSFlows(t, repositoryRoot, instrumentedCommandBinary, fixture.siteDirectory, coverageDirectoryPath)
	exerciseAddressInUseFlow(t, repositoryRoot, instrumentedCommandBinary, fixture.siteDirectory, coverageDirectoryPath)
	exerciseDynamicHTTPSFlows(t, repositoryRoot, instrumentedCommandBinary, fixture.siteDirectory, coverageDirectoryPath, tools)

	coverageProfilePath := filepath.Join(t.TempDir(), "global.coverage.out")
	writeCoverageProfileFromDirectory(t, repositoryRoot, coverageDirectoryPath, coverageProfilePath)
	assertGlobalCoverageAtOneHundredPercent(t, repositoryRoot, coverageProfilePath)
}

func buildInstrumentedBinaryForTarget(testingT *testing.T, repositoryRoot string, targetPath string, coveragePackage string, binaryName string) string {
	testingT.Helper()
	binaryPath := filepath.Join(testingT.TempDir(), binaryName)
	buildCommand := exec.Command(
		"go",
		"build",
		"-cover",
		"-coverpkg="+coveragePackage,
		"-o",
		binaryPath,
		targetPath,
	)
	buildCommand.Dir = repositoryRoot
	buildOutput, buildErr := buildCommand.CombinedOutput()
	if buildErr != nil {
		testingT.Fatalf("build instrumented binary %s: %v\n%s", targetPath, buildErr, string(buildOutput))
	}
	return binaryPath
}

func runCommandExpectExitCode(testingT *testing.T, repositoryRoot string, binaryPath string, arguments []string, environmentVariables map[string]string, expectedExitCode int) string {
	testingT.Helper()
	commandContext, cancelCommand := context.WithTimeout(context.Background(), integrationCommandTimeout)
	defer cancelCommand()
	command := exec.CommandContext(commandContext, binaryPath, arguments...)
	command.Dir = repositoryRoot
	command.Env = append(os.Environ(), formatEnvironmentVariables(environmentVariables)...)
	outputBytes, runErr := command.CombinedOutput()
	outputText := string(outputBytes)
	exitCode := 0
	if runErr != nil {
		if errors.Is(commandContext.Err(), context.DeadlineExceeded) {
			testingT.Fatalf("command %s %v exceeded timeout %s\n%s", binaryPath, arguments, integrationCommandTimeout, outputText)
		}
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			testingT.Fatalf("run command %s %v: %v\n%s", binaryPath, arguments, runErr, outputText)
		}
	}
	if exitCode != expectedExitCode {
		testingT.Fatalf("command %s %v expected exit %d, got %d\n%s", binaryPath, arguments, expectedExitCode, exitCode, outputText)
	}
	return outputText
}

func startGHTTPProcessWithArguments(
	testingT *testing.T,
	repositoryRoot string,
	serverBinaryPath string,
	arguments []string,
	environmentVariables map[string]string,
	startupProbeURL string,
	insecureStartupTLS bool,
) *startedGHTTPServer {
	testingT.Helper()
	serverCommand := exec.Command(serverBinaryPath, arguments...)
	serverCommand.Dir = repositoryRoot
	serverCommand.Env = append(os.Environ(), formatEnvironmentVariables(environmentVariables)...)
	serverLogBuffer := &bytes.Buffer{}
	serverCommand.Stdout = serverLogBuffer
	serverCommand.Stderr = serverLogBuffer
	if startErr := serverCommand.Start(); startErr != nil {
		testingT.Fatalf("start ghttp process with args %v: %v", arguments, startErr)
	}

	startedServer := &startedGHTTPServer{
		command:   serverCommand,
		logBuffer: serverLogBuffer,
		baseURL:   startupProbeURL,
	}
	testingT.Cleanup(func() {
		stopErr := startedServer.stop()
		if stopErr != nil {
			testingT.Errorf("stop ghttp process: %v\nserver logs:\n%s", stopErr, startedServer.logBuffer.String())
		}
	})

	startupClient := &http.Client{Timeout: browseModeRequestTimeout}
	if insecureStartupTLS {
		startupClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	startDeadline := time.Now().Add(browseModeStartupTimeout)
	for time.Now().Before(startDeadline) {
		request, requestErr := http.NewRequest(http.MethodGet, startupProbeURL, nil)
		if requestErr != nil {
			_ = startedServer.stop()
			testingT.Fatalf("create startup request: %v", requestErr)
		}
		response, responseErr := startupClient.Do(request)
		if responseErr == nil {
			response.Body.Close()
			return startedServer
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = startedServer.stop()
	testingT.Fatalf("ghttp process did not become ready within %s\nargs=%v\nserver logs:\n%s", browseModeStartupTimeout, arguments, startedServer.logBuffer.String())
	return nil
}

func createComprehensiveFixtureDirectory(testingT *testing.T) comprehensiveFixture {
	testingT.Helper()
	siteDirectory := testingT.TempDir()
	directories := []string{
		filepath.Join(siteDirectory, "docs"),
		filepath.Join(siteDirectory, "single"),
		filepath.Join(siteDirectory, "withindex"),
		filepath.Join(siteDirectory, "example"),
	}
	for _, directoryPath := range directories {
		if makeErr := os.MkdirAll(directoryPath, 0o755); makeErr != nil {
			testingT.Fatalf("create directory %s: %v", directoryPath, makeErr)
		}
	}

	landingFilePath := filepath.Join(siteDirectory, "landing.html")
	unsupportedTxtPath := filepath.Join(siteDirectory, "unsupported.txt")
	fileContentByPath := map[string]string{
		filepath.Join(siteDirectory, "index.html"):              "<html><body>ROOT INDEX</body></html>",
		filepath.Join(siteDirectory, "index.htm"):               "<html><body>ROOT INDEX HTM</body></html>",
		filepath.Join(siteDirectory, "hello.html"):              "<html><body>ROOT HELLO</body></html>",
		filepath.Join(siteDirectory, "README.md"):               "# Root Markdown\n",
		filepath.Join(siteDirectory, "landing.html"):            "<html><body>LANDING</body></html>",
		filepath.Join(siteDirectory, "unsupported.txt"):         "not supported\n",
		filepath.Join(siteDirectory, "example", "index.html"):   "<html><body>NESTED INDEX</body></html>",
		filepath.Join(siteDirectory, "example", "README.md"):    "# Nested Markdown\n",
		filepath.Join(siteDirectory, "docs", "README.md"):       "# Docs README\n",
		filepath.Join(siteDirectory, "single", "only.md"):       "# Single Markdown\n",
		filepath.Join(siteDirectory, "withindex", "index.html"): "<html><body>WITH INDEX</body></html>",
		filepath.Join(siteDirectory, "withindex", "README.md"):  "# With Index Markdown\n",
	}
	for filePath, fileContent := range fileContentByPath {
		if writeErr := os.WriteFile(filePath, []byte(fileContent), 0o644); writeErr != nil {
			testingT.Fatalf("write fixture file %s: %v", filePath, writeErr)
		}
	}

	return comprehensiveFixture{
		siteDirectory:      siteDirectory,
		landingFilePath:    landingFilePath,
		unsupportedTxtPath: unsupportedTxtPath,
	}
}

func exerciseStandardHTTPServerFlows(testingT *testing.T, repositoryRoot string, binaryPath string, siteDirectory string, coverageDirectoryPath string) {
	testingT.Helper()

	defaultPort := allocateFreePort(testingT)
	defaultBaseURL := fmt.Sprintf("http://127.0.0.1:%d", defaultPort)
	defaultServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(defaultPort), "--directory", siteDirectory},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		defaultBaseURL+"/",
		false,
	)
	defaultClient := &http.Client{Timeout: browseModeRequestTimeout}

	_, _, rootBody := executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/README.md")
	if !strings.Contains(rootBody, "Root Markdown") {
		testingT.Fatalf("expected markdown rendering for /README.md, body=%s", rootBody)
	}
	_, _, docsBody := executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/docs/")
	if !strings.Contains(docsBody, "Docs README") {
		testingT.Fatalf("expected README markdown rendering in /docs/, body=%s", docsBody)
	}
	_, _, singleBody := executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/single/")
	if !strings.Contains(singleBody, "Single Markdown") {
		testingT.Fatalf("expected single markdown rendering in /single/, body=%s", singleBody)
	}
	docsStatusCode, _, _ := executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/docs")
	if docsStatusCode != http.StatusMovedPermanently && docsStatusCode != http.StatusTemporaryRedirect && docsStatusCode != http.StatusOK {
		testingT.Fatalf("expected redirect-or-ok for /docs, got %d", docsStatusCode)
	}
	_, _, withIndexBody := executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/withindex/")
	if !strings.Contains(withIndexBody, "WITH INDEX") {
		testingT.Fatalf("expected index.html serving in /withindex/, body=%s", withIndexBody)
	}
	executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/hello.html")
	executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/missing")
	exampleStatusCode, _, _ := executeHTTPGet(testingT, defaultClient, defaultBaseURL, "/example")
	if exampleStatusCode != http.StatusMovedPermanently && exampleStatusCode != http.StatusTemporaryRedirect && exampleStatusCode != http.StatusOK {
		testingT.Fatalf("expected redirect-or-ok for /example, got %d", exampleStatusCode)
	}
	if stopErr := defaultServer.stop(); stopErr != nil {
		testingT.Fatalf("stop default server: %v", stopErr)
	}

	browsePort := allocateFreePort(testingT)
	browseBaseURL := fmt.Sprintf("http://127.0.0.1:%d", browsePort)
	browseServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(browsePort), "--directory", siteDirectory, "--browse"},
		map[string]string{
			"GOCOVERDIR":                    coverageDirectoryPath,
			"GHTTPD_DISABLE_DIR_INDEX":      "1",
			"GHTTP_SERVE_PROTOCOL":          "HTTP/1.1",
			"GHTTP_SERVE_LOGGING_TYPE":      "CONSOLE",
			"GHTTP_SERVE_PROXY_PATH_PREFIX": "",
			"GHTTP_SERVE_PROXY_BACKEND":     "",
		},
		browseBaseURL+"/",
		false,
	)
	browseClient := &http.Client{Timeout: browseModeRequestTimeout}
	_, _, browseRootBody := executeHTTPGet(testingT, browseClient, browseBaseURL, "/")
	if strings.Contains(browseRootBody, "ROOT INDEX") {
		testingT.Fatalf("browse root should remain listing, body=%s", browseRootBody)
	}
	_, _, browseIndexBody := executeHTTPGet(testingT, browseClient, browseBaseURL, "/index.html")
	if !strings.Contains(browseIndexBody, "ROOT INDEX") {
		testingT.Fatalf("browse direct /index.html should serve file, body=%s", browseIndexBody)
	}
	executeHTTPGet(testingT, browseClient, browseBaseURL, "/README.md")
	executeHTTPGet(testingT, browseClient, browseBaseURL, "/missing")
	executeHTTPGet(testingT, browseClient, browseBaseURL, "/example")
	if stopErr := browseServer.stop(); stopErr != nil {
		testingT.Fatalf("stop browse server: %v", stopErr)
	}

	disabledPort := allocateFreePort(testingT)
	disabledBaseURL := fmt.Sprintf("http://127.0.0.1:%d", disabledPort)
	disabledServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(disabledPort), "--directory", siteDirectory, "--no-md"},
		map[string]string{
			"GOCOVERDIR":               coverageDirectoryPath,
			"GHTTPD_DISABLE_DIR_INDEX": "1",
		},
		disabledBaseURL+"/hello.html",
		false,
	)
	disabledClient := &http.Client{Timeout: browseModeRequestTimeout}
	rootStatusCode, _, _ := executeHTTPGet(testingT, disabledClient, disabledBaseURL, "/")
	if rootStatusCode != http.StatusForbidden {
		testingT.Fatalf("expected forbidden listing with no markdown and disabled directory listing, got %d", rootStatusCode)
	}
	if stopErr := disabledServer.stop(); stopErr != nil {
		testingT.Fatalf("stop no-markdown server: %v", stopErr)
	}

	jsonPort := allocateFreePort(testingT)
	jsonBaseURL := fmt.Sprintf("http://127.0.0.1:%d", jsonPort)
	jsonServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(jsonPort), "--directory", siteDirectory, "--protocol", "HTTP/1.0", "--logging-type", "JSON"},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		jsonBaseURL+"/hello.html",
		false,
	)
	executeHTTPGet(testingT, &http.Client{Timeout: browseModeRequestTimeout}, jsonBaseURL, "/hello.html")
	if stopErr := jsonServer.stop(); stopErr != nil {
		testingT.Fatalf("stop json/http1.0 server: %v", stopErr)
	}
}

func exerciseInitialFileFlow(testingT *testing.T, repositoryRoot string, binaryPath string, landingFilePath string, coverageDirectoryPath string) {
	testingT.Helper()
	port := allocateFreePort(testingT)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	server := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{landingFilePath},
		map[string]string{
			"GOCOVERDIR":       coverageDirectoryPath,
			"GHTTP_SERVE_PORT": strconv.Itoa(port),
		},
		baseURL+"/",
		false,
	)
	_, _, rootBody := executeHTTPGet(testingT, &http.Client{Timeout: browseModeRequestTimeout}, baseURL, "/")
	if !strings.Contains(rootBody, "LANDING") {
		testingT.Fatalf("expected landing page from initial-file handler, body=%s", rootBody)
	}
	executeHTTPGet(testingT, &http.Client{Timeout: browseModeRequestTimeout}, baseURL, "/landing.html")
	if stopErr := server.stop(); stopErr != nil {
		testingT.Fatalf("stop initial-file server: %v", stopErr)
	}
}

func exerciseHTTPProxyFlows(testingT *testing.T, repositoryRoot string, binaryPath string, siteDirectory string, coverageDirectoryPath string) {
	testingT.Helper()
	backendServer := &http.Server{}
	backendListener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingT.Fatalf("start backend listener: %v", listenErr)
	}
	backendHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		_, _ = responseWriter.Write([]byte("backend:" + request.URL.Path))
	})
	backendServer.Handler = backendHandler
	go func() {
		_ = backendServer.Serve(backendListener)
	}()
	testingT.Cleanup(func() {
		_ = backendServer.Shutdown(context.Background())
	})

	proxyPort := allocateFreePort(testingT)
	proxyBaseURL := fmt.Sprintf("http://127.0.0.1:%d", proxyPort)
	proxyServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{
			strconv.Itoa(proxyPort),
			"--directory", siteDirectory,
			"--proxy", "/api=http://" + backendListener.Addr().String() + ", /svc=http://" + backendListener.Addr().String(),
		},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		proxyBaseURL+"/",
		false,
	)
	_, _, proxiedBody := executeHTTPGet(testingT, &http.Client{Timeout: browseModeRequestTimeout}, proxyBaseURL, "/api/proxy-check")
	if !strings.Contains(proxiedBody, "backend:/api/proxy-check") {
		testingT.Fatalf("expected proxied response body, got %s", proxiedBody)
	}
	_, _, localBody := executeHTTPGet(testingT, &http.Client{Timeout: browseModeRequestTimeout}, proxyBaseURL, "/hello.html")
	if !strings.Contains(localBody, "ROOT HELLO") {
		testingT.Fatalf("expected local file serving for non-proxy route, got %s", localBody)
	}
	if stopErr := proxyServer.stop(); stopErr != nil {
		testingT.Fatalf("stop proxy server: %v", stopErr)
	}

	legacyProxyPort := allocateFreePort(testingT)
	legacyBaseURL := fmt.Sprintf("http://127.0.0.1:%d", legacyProxyPort)
	legacyServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{
			strconv.Itoa(legacyProxyPort),
			"--directory", siteDirectory,
			"--proxy-path", "/legacy",
			"--proxy-backend", "http://" + backendListener.Addr().String(),
		},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		legacyBaseURL+"/",
		false,
	)
	_, _, legacyBody := executeHTTPGet(testingT, &http.Client{Timeout: browseModeRequestTimeout}, legacyBaseURL, "/legacy/proxy-check")
	if !strings.Contains(legacyBody, "backend:/legacy/proxy-check") {
		testingT.Fatalf("expected proxied legacy response body, got %s", legacyBody)
	}
	if stopErr := legacyServer.stop(); stopErr != nil {
		testingT.Fatalf("stop legacy proxy server: %v", stopErr)
	}
}

func exerciseWebSocketProxyFlows(testingT *testing.T, repositoryRoot string, binaryPath string, coverageDirectoryPath string) {
	testingT.Helper()
	backendListener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingT.Fatalf("start websocket backend listener: %v", listenErr)
	}
	testingT.Cleanup(func() {
		_ = backendListener.Close()
	})

	go func() {
		for {
			connection, acceptErr := backendListener.Accept()
			if acceptErr != nil {
				return
			}
			go func(activeConnection net.Conn) {
				defer activeConnection.Close()
				reader := bufio.NewReader(activeConnection)
				for {
					line, readErr := reader.ReadString('\n')
					if readErr != nil {
						return
					}
					if line == "\r\n" {
						break
					}
				}
				_, _ = io.WriteString(activeConnection, "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
				buffer := make([]byte, 1024)
				for {
					readCount, readErr := activeConnection.Read(buffer)
					if readCount > 0 {
						_, _ = activeConnection.Write(buffer[:readCount])
					}
					if readErr != nil {
						return
					}
				}
			}(connection)
		}
	}()

	proxyPort := allocateFreePort(testingT)
	proxyBaseURL := fmt.Sprintf("http://127.0.0.1:%d", proxyPort)
	proxyServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{
			strconv.Itoa(proxyPort),
			"--directory", testingT.TempDir(),
			"--proxy", "/ws=http://" + backendListener.Addr().String(),
		},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		proxyBaseURL+"/",
		false,
	)
	performWebSocketUpgradeRoundTrip(testingT, fmt.Sprintf("127.0.0.1:%d", proxyPort), "/ws")
	if stopErr := proxyServer.stop(); stopErr != nil {
		testingT.Fatalf("stop websocket proxy server: %v", stopErr)
	}

	tlsWebSocketPort := allocateFreePort(testingT)
	tlsWebSocketBaseURL := fmt.Sprintf("http://127.0.0.1:%d", tlsWebSocketPort)
	tlsWebSocketServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{
			strconv.Itoa(tlsWebSocketPort),
			"--directory", testingT.TempDir(),
			"--proxy", "/wss=https://localhost",
		},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		tlsWebSocketBaseURL+"/",
		false,
	)
	performWebSocketUpgradeExpectedFailure(testingT, fmt.Sprintf("127.0.0.1:%d", tlsWebSocketPort), "/wss")
	if stopErr := tlsWebSocketServer.stop(); stopErr != nil {
		testingT.Fatalf("stop tls websocket proxy server: %v", stopErr)
	}

	tlsBackendListener, tlsBackendListenErr := net.Listen("tcp", "127.0.0.1:0")
	if tlsBackendListenErr != nil {
		testingT.Fatalf("start tls websocket backend listener: %v", tlsBackendListenErr)
	}
	_ = tlsBackendListener.Close()
	tlsWebSocketPortWithPort := allocateFreePort(testingT)
	tlsWebSocketBaseURLWithPort := fmt.Sprintf("http://127.0.0.1:%d", tlsWebSocketPortWithPort)
	tlsWebSocketServerWithPort := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{
			strconv.Itoa(tlsWebSocketPortWithPort),
			"--directory", testingT.TempDir(),
			"--proxy", "/wss=https://" + tlsBackendListener.Addr().String(),
		},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		tlsWebSocketBaseURLWithPort+"/",
		false,
	)
	performWebSocketUpgradeExpectedFailure(testingT, fmt.Sprintf("127.0.0.1:%d", tlsWebSocketPortWithPort), "/wss")
	if stopErr := tlsWebSocketServerWithPort.stop(); stopErr != nil {
		testingT.Fatalf("stop tls websocket proxy server with host:port: %v", stopErr)
	}
}

func performWebSocketUpgradeRoundTrip(testingT *testing.T, hostPort string, requestPath string) {
	testingT.Helper()
	connection, dialErr := net.DialTimeout("tcp", hostPort, browseModeRequestTimeout)
	if dialErr != nil {
		testingT.Fatalf("dial websocket proxy %s: %v", hostPort, dialErr)
	}
	defer connection.Close()

	handshakeRequest := strings.Join([]string{
		"GET " + requestPath + " HTTP/1.1",
		"Host: " + hostPort,
		"Connection: Upgrade",
		"Upgrade: websocket",
		"Sec-WebSocket-Version: 13",
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
		"",
		"",
	}, "\r\n")
	if _, writeErr := io.WriteString(connection, handshakeRequest); writeErr != nil {
		testingT.Fatalf("write websocket handshake: %v", writeErr)
	}

	reader := bufio.NewReader(connection)
	statusLine, statusErr := reader.ReadString('\n')
	if statusErr != nil {
		testingT.Fatalf("read websocket status line: %v", statusErr)
	}
	if !strings.Contains(statusLine, "101") {
		testingT.Fatalf("expected websocket 101 response, got %q", statusLine)
	}
	for {
		headerLine, readErr := reader.ReadString('\n')
		if readErr != nil {
			testingT.Fatalf("read websocket header line: %v", readErr)
		}
		if headerLine == "\r\n" {
			break
		}
	}

	payload := "proxy-roundtrip\n"
	if _, writeErr := io.WriteString(connection, payload); writeErr != nil {
		testingT.Fatalf("write websocket payload: %v", writeErr)
	}
	echoBuffer := make([]byte, len(payload))
	if _, readErr := io.ReadFull(reader, echoBuffer); readErr != nil {
		testingT.Fatalf("read websocket echo payload: %v", readErr)
	}
	if string(echoBuffer) != payload {
		testingT.Fatalf("unexpected websocket echo payload: %q", string(echoBuffer))
	}
}

func performWebSocketUpgradeExpectedFailure(testingT *testing.T, hostPort string, requestPath string) {
	testingT.Helper()
	connection, dialErr := net.DialTimeout("tcp", hostPort, browseModeRequestTimeout)
	if dialErr != nil {
		testingT.Fatalf("dial websocket proxy %s: %v", hostPort, dialErr)
	}
	defer connection.Close()

	handshakeRequest := strings.Join([]string{
		"GET " + requestPath + " HTTP/1.1",
		"Host: " + hostPort,
		"Connection: Upgrade",
		"Upgrade: websocket",
		"Sec-WebSocket-Version: 13",
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
		"",
		"",
	}, "\r\n")
	if _, writeErr := io.WriteString(connection, handshakeRequest); writeErr != nil {
		testingT.Fatalf("write websocket handshake: %v", writeErr)
	}
	reader := bufio.NewReader(connection)
	statusLine, statusErr := reader.ReadString('\n')
	if statusErr != nil {
		testingT.Fatalf("read websocket failure status line: %v", statusErr)
	}
	if !strings.Contains(statusLine, "502") {
		testingT.Fatalf("expected websocket 502 response, got %q", statusLine)
	}
}

func exerciseManualTLSFlows(testingT *testing.T, repositoryRoot string, binaryPath string, siteDirectory string, coverageDirectoryPath string) {
	testingT.Helper()
	certificatePath, privateKeyPath := generateSelfSignedCertificatePair(testingT)
	port := allocateFreePort(testingT)
	baseURL := fmt.Sprintf("https://127.0.0.1:%d", port)
	server := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{
			strconv.Itoa(port),
			"--directory", siteDirectory,
			"--tls-cert", certificatePath,
			"--tls-key", privateKeyPath,
		},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		baseURL+"/hello.html",
		true,
	)
	httpsClient := &http.Client{
		Timeout: browseModeRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	_, _, body := executeHTTPGet(testingT, httpsClient, baseURL, "/hello.html")
	if !strings.Contains(body, "ROOT HELLO") {
		testingT.Fatalf("expected root hello via tls cert pair, body=%s", body)
	}
	if stopErr := server.stop(); stopErr != nil {
		testingT.Fatalf("stop manual tls server: %v", stopErr)
	}

	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https", "--tls-cert", certificatePath, "--tls-key", privateKeyPath},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		1,
	)
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--tls-cert", certificatePath},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		1,
	)
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--tls-key", privateKeyPath},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		1,
	)
}

func generateSelfSignedCertificatePair(testingT *testing.T) (string, string) {
	testingT.Helper()
	privateKey, keyErr := rsa.GenerateKey(rand.Reader, 2048)
	if keyErr != nil {
		testingT.Fatalf("generate tls key: %v", keyErr)
	}
	serialNumberUpperBound := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, serialErr := rand.Int(rand.Reader, serialNumberUpperBound)
	if serialErr != nil {
		testingT.Fatalf("generate tls serial: %v", serialErr)
	}
	now := time.Now()
	certificateTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "127.0.0.1",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certificateDERBytes, certificateErr := x509.CreateCertificate(rand.Reader, &certificateTemplate, &certificateTemplate, &privateKey.PublicKey, privateKey)
	if certificateErr != nil {
		testingT.Fatalf("create tls certificate: %v", certificateErr)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDERBytes})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	certificatePath := filepath.Join(testingT.TempDir(), "cert.pem")
	privateKeyPath := filepath.Join(testingT.TempDir(), "key.pem")
	if writeErr := os.WriteFile(certificatePath, certificatePEM, 0o644); writeErr != nil {
		testingT.Fatalf("write tls cert pem: %v", writeErr)
	}
	if writeErr := os.WriteFile(privateKeyPath, privateKeyPEM, 0o600); writeErr != nil {
		testingT.Fatalf("write tls key pem: %v", writeErr)
	}
	return certificatePath, privateKeyPath
}

func exerciseAddressInUseFlow(testingT *testing.T, repositoryRoot string, binaryPath string, siteDirectory string, coverageDirectoryPath string) {
	testingT.Helper()
	occupiedListener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingT.Fatalf("listen for occupied port: %v", listenErr)
	}
	defer occupiedListener.Close()
	occupiedPort := occupiedListener.Addr().(*net.TCPAddr).Port
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(occupiedPort), "--directory", siteDirectory},
		map[string]string{"GOCOVERDIR": coverageDirectoryPath},
		1,
	)
}

func createFakeSystemTools(testingT *testing.T) fakeSystemTools {
	testingT.Helper()
	toolsRootDirectory := testingT.TempDir()
	trustOnlyDirectory := filepath.Join(toolsRootDirectory, "trust-only")
	trustAndCertutilDirectory := filepath.Join(toolsRootDirectory, "trust-and-certutil")
	if makeErr := os.MkdirAll(trustOnlyDirectory, 0o755); makeErr != nil {
		testingT.Fatalf("create trust-only directory: %v", makeErr)
	}
	if makeErr := os.MkdirAll(trustAndCertutilDirectory, 0o755); makeErr != nil {
		testingT.Fatalf("create trust-and-certutil directory: %v", makeErr)
	}

	trustScript := "#!/usr/bin/env bash\nset -euo pipefail\nif [ \"${GHTTP_TEST_TRUST_LOG_FILE:-}\" != \"\" ]; then\n  echo \"$*\" >> \"${GHTTP_TEST_TRUST_LOG_FILE}\"\nfi\nif [ \"${GHTTP_TEST_TRUST_FAIL_ALL:-0}\" = \"1\" ]; then\n  exit 1\nfi\nif [ \"${GHTTP_TEST_TRUST_FAIL_REMOVE:-0}\" = \"1\" ] && [ \"${2:-}\" = \"--remove\" ]; then\n  exit 1\nfi\nexit 0\n"
	certutilScript := "#!/usr/bin/env bash\nset -euo pipefail\nif [ \"${GHTTP_TEST_CERTUTIL_FAIL:-0}\" = \"1\" ]; then\n  exit 1\nfi\nexit 0\n"

	writeExecutableScript(testingT, filepath.Join(trustOnlyDirectory, "trust"), trustScript)
	writeExecutableScript(testingT, filepath.Join(trustAndCertutilDirectory, "trust"), trustScript)
	writeExecutableScript(testingT, filepath.Join(trustAndCertutilDirectory, "certutil"), certutilScript)

	return fakeSystemTools{
		binDirectoryPath:        toolsRootDirectory,
		trustOnlyPath:           trustOnlyDirectory,
		trustAndCertutilPath:    trustAndCertutilDirectory,
		trustFailureOnRemoveLog: filepath.Join(toolsRootDirectory, "trust.log"),
	}
}

func writeExecutableScript(testingT *testing.T, scriptPath string, scriptContent string) {
	testingT.Helper()
	if writeErr := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); writeErr != nil {
		testingT.Fatalf("write script %s: %v", scriptPath, writeErr)
	}
}

func exerciseDynamicHTTPSFlows(testingT *testing.T, repositoryRoot string, binaryPath string, siteDirectory string, coverageDirectoryPath string, tools fakeSystemTools) {
	testingT.Helper()
	homeDirectory := testingT.TempDir()
	firefoxProfileDirectory := filepath.Join(homeDirectory, ".mozilla", "firefox", "profile.default")
	if makeErr := os.MkdirAll(firefoxProfileDirectory, 0o755); makeErr != nil {
		testingT.Fatalf("create firefox profile directory: %v", makeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(firefoxProfileDirectory, "cert9.db"), []byte("db"), 0o644); writeErr != nil {
		testingT.Fatalf("write cert9.db: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(firefoxProfileDirectory, "cert8.db"), []byte("db"), 0o644); writeErr != nil {
		testingT.Fatalf("write cert8.db: %v", writeErr)
	}
	certificateDirectory := filepath.Join(testingT.TempDir(), "dynamic-certs")

	firstPort := allocateFreePort(testingT)
	firstBaseURL := fmt.Sprintf("https://127.0.0.1:%d", firstPort)
	firstRunServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(firstPort), "--directory", siteDirectory, "--https", "--https-host", "localhost", "--https-host", "127.0.0.1", "--logging-type", "JSON"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
			"GHTTP_TEST_TRUST_FAIL_REMOVE":      "1",
			"GHTTP_TEST_TRUST_LOG_FILE":         tools.trustFailureOnRemoveLog,
		},
		firstBaseURL+"/hello.html",
		true,
	)
	httpsClient := &http.Client{
		Timeout: browseModeRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	_, _, firstBody := executeHTTPGet(testingT, httpsClient, firstBaseURL, "/hello.html")
	if !strings.Contains(firstBody, "ROOT HELLO") {
		testingT.Fatalf("expected hello body from dynamic https run, body=%s", firstBody)
	}
	if stopErr := firstRunServer.stop(); stopErr != nil {
		testingT.Fatalf("stop first https server: %v", stopErr)
	}

	secondPort := allocateFreePort(testingT)
	secondBaseURL := fmt.Sprintf("https://127.0.0.1:%d", secondPort)
	secondRunServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(secondPort), "--directory", siteDirectory, "--https", "--https-host", "localhost", "--https-host", "127.0.0.1"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
			"GHTTP_TEST_TRUST_FAIL_REMOVE":      "1",
		},
		secondBaseURL+"/hello.html",
		true,
	)
	executeHTTPGet(testingT, httpsClient, secondBaseURL, "/hello.html")
	if stopErr := secondRunServer.stop(); stopErr != nil {
		testingT.Fatalf("stop second https server: %v", stopErr)
	}

	thirdPort := allocateFreePort(testingT)
	thirdBaseURL := fmt.Sprintf("https://127.0.0.1:%d", thirdPort)
	thirdRunServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(thirdPort), "--directory", siteDirectory, "--https", "--https-host", "localhost", "--https-host", "127.0.0.1", "--https-host", "example.local"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
			"GHTTP_TEST_TRUST_FAIL_REMOVE":      "1",
		},
		thirdBaseURL+"/hello.html",
		true,
	)
	executeHTTPGet(testingT, httpsClient, thirdBaseURL, "/hello.html")
	if stopErr := thirdRunServer.stop(); stopErr != nil {
		testingT.Fatalf("stop third https server: %v", stopErr)
	}

	fourthPort := allocateFreePort(testingT)
	fourthBaseURL := fmt.Sprintf("https://127.0.0.1:%d", fourthPort)
	fourthRunServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(fourthPort), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustOnlyPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
			"GHTTP_TEST_TRUST_FAIL_REMOVE":      "1",
		},
		fourthBaseURL+"/hello.html",
		true,
	)
	executeHTTPGet(testingT, httpsClient, fourthBaseURL, "/hello.html")
	if stopErr := fourthRunServer.stop(); stopErr != nil {
		testingT.Fatalf("stop fourth https server: %v", stopErr)
	}

	fifthPort := allocateFreePort(testingT)
	fifthBaseURL := fmt.Sprintf("https://127.0.0.1:%d", fifthPort)
	fifthRunServer := startGHTTPProcessWithArguments(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(fifthPort), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
			"GHTTP_TEST_TRUST_FAIL_REMOVE":      "0",
		},
		fifthBaseURL+"/hello.html",
		true,
	)
	executeHTTPGet(testingT, httpsClient, fifthBaseURL, "/hello.html")
	if stopErr := fifthRunServer.stop(); stopErr != nil {
		testingT.Fatalf("stop fifth https server: %v", stopErr)
	}

	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
			"GHTTP_TEST_CERTUTIL_FAIL":          "1",
			"GHTTP_TEST_TRUST_FAIL_ALL":         "1",
		},
		1,
	)
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https", "--https-host", "  ", "--https-host", ""},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
		},
		1,
	)

	leafCertificatePath := filepath.Join(certificateDirectory, "localhost.pem")
	leafPrivateKeyPath := filepath.Join(certificateDirectory, "localhost.key")
	if writeErr := os.WriteFile(leafCertificatePath, []byte("-----BEGIN CERTIFICATE-----\ninvalid\n-----END CERTIFICATE-----\n"), 0o600); writeErr != nil {
		testingT.Fatalf("corrupt leaf certificate: %v", writeErr)
	}
	if writeErr := os.WriteFile(leafPrivateKeyPath, []byte("-----BEGIN RSA PRIVATE KEY-----\ninvalid\n-----END RSA PRIVATE KEY-----\n"), 0o600); writeErr != nil {
		testingT.Fatalf("corrupt leaf private key: %v", writeErr)
	}
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
			"GHTTP_TEST_TRUST_FAIL_REMOVE":      "1",
		},
		1,
	)

	rootCertificatePath := filepath.Join(certificateDirectory, "ca.pem")
	rootPrivateKeyPath := filepath.Join(certificateDirectory, "ca.key")
	if writeErr := os.WriteFile(rootCertificatePath, []byte("-----BEGIN PRIVATE KEY-----\nnot-a-cert\n-----END PRIVATE KEY-----\n"), 0o600); writeErr != nil {
		testingT.Fatalf("corrupt root certificate: %v", writeErr)
	}
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
		},
		1,
	)

	if writeErr := os.WriteFile(rootCertificatePath, []byte("-----BEGIN CERTIFICATE-----\ninvalid\n-----END CERTIFICATE-----\n"), 0o600); writeErr != nil {
		testingT.Fatalf("corrupt root certificate contents: %v", writeErr)
	}
	if writeErr := os.WriteFile(rootPrivateKeyPath, []byte("-----BEGIN CERTIFICATE-----\ninvalid\n-----END CERTIFICATE-----\n"), 0o600); writeErr != nil {
		testingT.Fatalf("corrupt root private key: %v", writeErr)
	}
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectory,
		},
		1,
	)

	certificateDirectoryAsFile := filepath.Join(testingT.TempDir(), "certificate-dir-file")
	if writeErr := os.WriteFile(certificateDirectoryAsFile, []byte("not a directory"), 0o644); writeErr != nil {
		testingT.Fatalf("write file replacing certificate directory: %v", writeErr)
	}
	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": certificateDirectoryAsFile,
		},
		1,
	)

	runCommandExpectExitCode(
		testingT,
		repositoryRoot,
		binaryPath,
		[]string{strconv.Itoa(allocateFreePort(testingT)), "--directory", siteDirectory, "--https"},
		map[string]string{
			"GOCOVERDIR":                        coverageDirectoryPath,
			"HOME":                              homeDirectory,
			"PATH":                              tools.trustAndCertutilPath + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GHTTP_HTTPS_CERTIFICATE_DIRECTORY": "   ",
		},
		1,
	)

	if _, statErr := os.Stat(filepath.Join(firefoxProfileDirectory, "user.js")); statErr != nil {
		testingT.Fatalf("expected firefox preference file to be created by fallback integration: %v", statErr)
	}
}

func assertGlobalCoverageAtOneHundredPercent(testingT *testing.T, repositoryRoot string, coverageProfilePath string) {
	testingT.Helper()
	coverageReportCommand := exec.Command("go", "tool", "cover", "-func="+coverageProfilePath)
	coverageReportCommand.Dir = repositoryRoot
	coverageReportOutput, coverageReportErr := coverageReportCommand.CombinedOutput()
	if coverageReportErr != nil {
		testingT.Fatalf("read global coverage report: %v\n%s", coverageReportErr, string(coverageReportOutput))
	}

	reportLines := strings.Split(string(coverageReportOutput), "\n")
	requiredCoverageValueText := strings.TrimSuffix(globalCoverageRequiredPercentageText, "%")
	requiredCoverageValue, parseRequiredErr := strconv.ParseFloat(requiredCoverageValueText, 64)
	if parseRequiredErr != nil {
		testingT.Fatalf("parse required coverage %s: %v", globalCoverageRequiredPercentageText, parseRequiredErr)
	}
	for _, reportLine := range reportLines {
		normalizedLine := strings.TrimSpace(reportLine)
		if !strings.HasPrefix(normalizedLine, "total:") {
			continue
		}
		segments := strings.Fields(normalizedLine)
		if len(segments) == 0 {
			testingT.Fatalf("unexpected total coverage line format: %s\nfull report:\n%s", normalizedLine, string(coverageReportOutput))
		}
		actualCoverageText := strings.TrimSuffix(segments[len(segments)-1], "%")
		actualCoverageValue, parseActualErr := strconv.ParseFloat(actualCoverageText, 64)
		if parseActualErr != nil {
			testingT.Fatalf("parse actual coverage from line %q: %v\nfull report:\n%s", normalizedLine, parseActualErr, string(coverageReportOutput))
		}
		if actualCoverageValue < requiredCoverageValue {
			testingT.Fatalf("global coverage below required threshold %s: %s\nfull report:\n%s", globalCoverageRequiredPercentageText, normalizedLine, string(coverageReportOutput))
		}
		return
	}
	testingT.Fatalf("coverage report does not contain total line\nreport:\n%s", string(coverageReportOutput))
}
