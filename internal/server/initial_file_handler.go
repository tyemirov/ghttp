package server

import (
	"net/http"
	pathpkg "path"
	"strings"
)

const (
	initialFileRootRequestPath = "/"
)

type initialFileHandler struct {
	next               http.Handler
	initialRequestPath string
}

func newInitialFileHandler(next http.Handler, initialFileRelativePath string) http.Handler {
	cleanPath := pathpkg.Clean(pathpkg.Join(initialFileRootRequestPath, strings.ReplaceAll(initialFileRelativePath, "\\", "/")))
	return initialFileHandler{
		next:               next,
		initialRequestPath: cleanPath,
	}
}

func (handler initialFileHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	if request.URL.Path != initialFileRootRequestPath {
		handler.next.ServeHTTP(responseWriter, request)
		return
	}
	handler.next.ServeHTTP(responseWriter, rewriteRequestPath(request, handler.initialRequestPath))
}

func rewriteRequestPath(request *http.Request, requestPath string) *http.Request {
	clonedRequest := request.Clone(request.Context())
	urlCopy := *request.URL
	urlCopy.Path = requestPath
	urlCopy.RawPath = requestPath
	clonedRequest.URL = &urlCopy
	clonedRequest.RequestURI = requestPath
	return clonedRequest
}
