package server

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBrowseHandlerServeDirectFileRequest(t *testing.T) {
	const defaultTimestamp = "Mon, 02 Jan 2006 15:04:05 GMT"
	testCases := []struct {
		name                     string
		path                     string
		fileSystem               http.FileSystem
		expectServed             bool
		expectBodyContains       string
		expectBodyNotContains    string
		expectLastModifiedHeader bool
	}{
		{
			name:         "EmptyPathBypassesDirectServing",
			path:         "",
			fileSystem:   testBrowseFileSystem{openFunction: func(path string) (http.File, error) { return nil, errors.New("unexpected open call") }},
			expectServed: false,
		},
		{
			name:         "TrailingSlashBypassesDirectServing",
			path:         "/docs/",
			fileSystem:   testBrowseFileSystem{openFunction: func(path string) (http.File, error) { return nil, errors.New("unexpected open call") }},
			expectServed: false,
		},
		{
			name: "OpenFailureBypassesDirectServing",
			path: "/missing.html",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) { return nil, errors.New("open failed") },
			},
			expectServed: false,
		},
		{
			name: "StatFailureBypassesDirectServing",
			path: "/broken.html",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						fileInfo:  testBrowseFileInfo{name: "broken.html", isDirectory: false},
						statError: errors.New("stat failed"),
					}, nil
				},
			},
			expectServed: false,
		},
		{
			name: "DirectoryBypassesDirectServing",
			path: "/folder",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						fileInfo: testBrowseFileInfo{name: "folder", isDirectory: true},
					}, nil
				},
			},
			expectServed: false,
		},
		{
			name: "MarkdownBypassesDirectServing",
			path: "/README.md",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						content:  []byte("# README"),
						fileInfo: testBrowseFileInfo{name: "README.md", isDirectory: false},
					}, nil
				},
			},
			expectServed: false,
		},
		{
			name: "DirectHtmlIsServed",
			path: "/index.html",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						content: []byte("<html>Index page</html>"),
						fileInfo: testBrowseFileInfo{
							name:         "index.html",
							isDirectory:  false,
							modifiedTime: time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC),
						},
					}, nil
				},
			},
			expectServed:             true,
			expectBodyContains:       "Index page",
			expectBodyNotContains:    "README",
			expectLastModifiedHeader: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(testingT *testing.T) {
			handler := browseHandler{
				next:       http.NotFoundHandler(),
				fileSystem: testCase.fileSystem,
			}
			request := httptest.NewRequest(http.MethodGet, "/placeholder", nil)
			request.URL.Path = testCase.path
			recorder := httptest.NewRecorder()

			servedDirectly := handler.serveDirectFileRequest(recorder, request)

			if servedDirectly != testCase.expectServed {
				testingT.Fatalf("expected servedDirectly=%v, got %v", testCase.expectServed, servedDirectly)
			}
			responseBody := recorder.Body.String()
			if testCase.expectBodyContains != "" && !strings.Contains(responseBody, testCase.expectBodyContains) {
				testingT.Fatalf("expected body to contain %q, body: %s", testCase.expectBodyContains, responseBody)
			}
			if testCase.expectBodyNotContains != "" && strings.Contains(responseBody, testCase.expectBodyNotContains) {
				testingT.Fatalf("expected body to not contain %q, body: %s", testCase.expectBodyNotContains, responseBody)
			}
			lastModifiedHeader := recorder.Header().Get("Last-Modified")
			if testCase.expectLastModifiedHeader && lastModifiedHeader != defaultTimestamp {
				testingT.Fatalf("expected Last-Modified %q, got %q", defaultTimestamp, lastModifiedHeader)
			}
			if !testCase.expectLastModifiedHeader && lastModifiedHeader != "" {
				testingT.Fatalf("expected no Last-Modified header, got %q", lastModifiedHeader)
			}
		})
	}
}

func TestBrowseHandlerServeHTTP(t *testing.T) {
	type serveHTTPCase struct {
		name                     string
		path                     string
		fileSystem               http.FileSystem
		expectNextStatus         int
		expectBodyContains       string
		expectBodyNotContains    string
		expectContentTypeHeader  string
		expectContentTypePresent bool
	}

	nextStatusCode := http.StatusTeapot
	testCases := []serveHTTPCase{
		{
			name:             "DelegatesNonDirectoryPathToNextHandler",
			path:             "/missing-file",
			fileSystem:       testBrowseFileSystem{openFunction: func(path string) (http.File, error) { return nil, errors.New("open failed") }},
			expectNextStatus: nextStatusCode,
		},
		{
			name:             "DelegatesEmptyPathToNextHandler",
			path:             "",
			fileSystem:       testBrowseFileSystem{openFunction: func(path string) (http.File, error) { return nil, errors.New("open failed") }},
			expectNextStatus: nextStatusCode,
		},
		{
			name: "DelegatesWhenDirectoryOpenFails",
			path: "/",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) { return nil, errors.New("open failed") },
			},
			expectNextStatus: nextStatusCode,
		},
		{
			name: "DelegatesWhenDirectoryStatFails",
			path: "/",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						fileInfo:  testBrowseFileInfo{name: "/", isDirectory: true},
						statError: errors.New("stat failed"),
					}, nil
				},
			},
			expectNextStatus: nextStatusCode,
		},
		{
			name: "DelegatesWhenPathWithSlashIsNotDirectory",
			path: "/",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						fileInfo: testBrowseFileInfo{name: "file.txt", isDirectory: false},
					}, nil
				},
			},
			expectNextStatus: nextStatusCode,
		},
		{
			name: "DelegatesWhenDirectoryReadFails",
			path: "/",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						fileInfo:     testBrowseFileInfo{name: "/", isDirectory: true},
						readdirError: errors.New("readdir failed"),
					}, nil
				},
			},
			expectNextStatus: nextStatusCode,
		},
		{
			name: "ServesDirectFileBeforeDirectoryHandling",
			path: "/index.html",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						content: []byte("<html>Index page</html>"),
						fileInfo: testBrowseFileInfo{
							name:        "index.html",
							isDirectory: false,
						},
					}, nil
				},
			},
			expectBodyContains:    "Index page",
			expectBodyNotContains: "next-handler",
		},
		{
			name: "RendersDirectoryListingWhenDirectoryReadable",
			path: "/",
			fileSystem: testBrowseFileSystem{
				openFunction: func(path string) (http.File, error) {
					return &testBrowseFile{
						fileInfo: testBrowseFileInfo{name: "/", isDirectory: true},
						readdirEntries: []fs.FileInfo{
							testBrowseFileInfo{name: "zeta.txt", isDirectory: false},
							testBrowseFileInfo{name: "alpha", isDirectory: true},
						},
					}, nil
				},
			},
			expectBodyContains:       "href=\"/alpha/\"",
			expectBodyNotContains:    "next-handler",
			expectContentTypeHeader:  "text/html; charset=utf-8",
			expectContentTypePresent: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(testingT *testing.T) {
			nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
				responseWriter.WriteHeader(nextStatusCode)
				_, _ = responseWriter.Write([]byte("next-handler"))
			})
			handler := browseHandler{
				next:       nextHandler,
				fileSystem: testCase.fileSystem,
			}
			request := httptest.NewRequest(http.MethodGet, "/placeholder", nil)
			request.URL.Path = testCase.path
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if testCase.expectNextStatus != 0 && recorder.Code != testCase.expectNextStatus {
				testingT.Fatalf("expected delegated status %d, got %d", testCase.expectNextStatus, recorder.Code)
			}
			responseBody := recorder.Body.String()
			if testCase.expectBodyContains != "" && !strings.Contains(responseBody, testCase.expectBodyContains) {
				testingT.Fatalf("expected body to contain %q, body: %s", testCase.expectBodyContains, responseBody)
			}
			if testCase.expectBodyNotContains != "" && strings.Contains(responseBody, testCase.expectBodyNotContains) {
				testingT.Fatalf("expected body to not contain %q, body: %s", testCase.expectBodyNotContains, responseBody)
			}
			contentTypeHeader := recorder.Header().Get("Content-Type")
			if testCase.expectContentTypePresent && contentTypeHeader != testCase.expectContentTypeHeader {
				testingT.Fatalf("expected content type %q, got %q", testCase.expectContentTypeHeader, contentTypeHeader)
			}
		})
	}
}

type testBrowseFileSystem struct {
	openFunction func(path string) (http.File, error)
}

func (fileSystem testBrowseFileSystem) Open(path string) (http.File, error) {
	return fileSystem.openFunction(path)
}

type testBrowseFile struct {
	content        []byte
	currentOffset  int64
	fileInfo       fs.FileInfo
	statError      error
	readdirEntries []fs.FileInfo
	readdirError   error
}

func (file *testBrowseFile) Close() error {
	return nil
}

func (file *testBrowseFile) Read(buffer []byte) (int, error) {
	if file.currentOffset >= int64(len(file.content)) {
		return 0, io.EOF
	}
	readCount := copy(buffer, file.content[file.currentOffset:])
	file.currentOffset += int64(readCount)
	return readCount, nil
}

func (file *testBrowseFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		file.currentOffset = offset
	case io.SeekCurrent:
		file.currentOffset += offset
	case io.SeekEnd:
		file.currentOffset = int64(len(file.content)) + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if file.currentOffset < 0 {
		return 0, errors.New("negative position")
	}
	return file.currentOffset, nil
}

func (file *testBrowseFile) Readdir(count int) ([]fs.FileInfo, error) {
	if file.readdirError != nil {
		return nil, file.readdirError
	}
	return file.readdirEntries, nil
}

func (file *testBrowseFile) Stat() (fs.FileInfo, error) {
	if file.statError != nil {
		return nil, file.statError
	}
	return file.fileInfo, nil
}

type testBrowseFileInfo struct {
	name         string
	size         int64
	mode         fs.FileMode
	modifiedTime time.Time
	isDirectory  bool
}

func (fileInfo testBrowseFileInfo) Name() string {
	return fileInfo.name
}

func (fileInfo testBrowseFileInfo) Size() int64 {
	return fileInfo.size
}

func (fileInfo testBrowseFileInfo) Mode() fs.FileMode {
	if fileInfo.mode != 0 {
		return fileInfo.mode
	}
	if fileInfo.isDirectory {
		return fs.ModeDir | 0o755
	}
	return 0o644
}

func (fileInfo testBrowseFileInfo) ModTime() time.Time {
	if fileInfo.modifiedTime.IsZero() {
		return time.Unix(0, 0).UTC()
	}
	return fileInfo.modifiedTime
}

func (fileInfo testBrowseFileInfo) IsDir() bool {
	return fileInfo.isDirectory
}

func (fileInfo testBrowseFileInfo) Sys() interface{} {
	return nil
}
