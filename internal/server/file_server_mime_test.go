package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/temirov/ghttp/internal/server"
	"github.com/temirov/ghttp/internal/serverdetails"
	"github.com/temirov/ghttp/pkg/logging"
)

const (
	assetDirectoryName        = "assets"
	cssFileName               = "styles.css"
	cssFileContent            = "body { color: black; }\n"
	cssMimeSubstring          = "text/css"
	javascriptFileName        = "application.js"
	javascriptFileContent     = "console.log('ready');\n"
	javascriptMimeSubstring   = "text/javascript"
	httpHeaderContentType     = "Content-Type"
	requestURLFormat          = "%s%s"
	cssFileRequestPath        = "/" + assetDirectoryName + "/" + cssFileName
	javascriptFileRequestPath = "/" + assetDirectoryName + "/" + javascriptFileName
)

func TestFileServerServesStaticAssetsWithCorrectMimeTypes(t *testing.T) {
	rootDirectory := t.TempDir()
	assetDirectoryPath := filepath.Join(rootDirectory, assetDirectoryName)
	createDirectory(t, assetDirectoryPath)

	createFile(t, filepath.Join(assetDirectoryPath, cssFileName), cssFileContent)
	createFile(t, filepath.Join(assetDirectoryPath, javascriptFileName), javascriptFileContent)

	loggingService := logging.NewTestService(logging.TypeConsole)
	addressFormatter := serverdetails.NewServingAddressFormatter()
	fileServerInstance := server.NewFileServer(loggingService, addressFormatter)
	handler := fileServerInstance.Handler(server.FileServerConfiguration{DirectoryPath: rootDirectory})
	testServer := httptest.NewServer(handler)
	t.Cleanup(testServer.Close)

	testCases := []struct {
		name                  string
		relativeRequestPath   string
		expectedMimeSubstring string
	}{
		{
			name:                  "CSS asset served with CSS MIME type",
			relativeRequestPath:   cssFileRequestPath,
			expectedMimeSubstring: cssMimeSubstring,
		},
		{
			name:                  "JavaScript asset served with JavaScript MIME type",
			relativeRequestPath:   javascriptFileRequestPath,
			expectedMimeSubstring: javascriptMimeSubstring,
		},
	}

	for testCaseIndex := range testCases {
		currentTestCase := testCases[testCaseIndex]
		t.Run(currentTestCase.name, func(t *testing.T) {
			response, requestErr := http.Get(fmt.Sprintf(requestURLFormat, testServer.URL, currentTestCase.relativeRequestPath))
			if requestErr != nil {
				t.Fatalf("request asset: %v", requestErr)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				t.Fatalf("unexpected status code: %d", response.StatusCode)
			}

			contentType := response.Header.Get(httpHeaderContentType)
			if !strings.Contains(contentType, currentTestCase.expectedMimeSubstring) {
				t.Fatalf("unexpected content type %s", contentType)
			}
		})
	}
}

func createFile(t *testing.T, filePath string, content string) {
	t.Helper()
	writeErr := os.WriteFile(filePath, []byte(content), 0o600)
	if writeErr != nil {
		t.Fatalf("write file %s: %v", filePath, writeErr)
	}
}

func createDirectory(t *testing.T, directoryPath string) {
	t.Helper()
	makeErr := os.MkdirAll(directoryPath, 0o755)
	if makeErr != nil {
		t.Fatalf("mkdir %s: %v", directoryPath, makeErr)
	}
}
