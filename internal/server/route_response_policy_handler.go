package server

import (
	"bufio"
	"net"
	"net/http"
)

type routeResponsePolicyHandler struct {
	next            http.Handler
	responseHeaders map[string]string
}

func newRouteResponsePolicyHandler(next http.Handler, policies RouteResponsePolicies) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		resolvedHeaders := policies.HeadersForPath(request.URL.Path)
		if len(resolvedHeaders) == 0 {
			next.ServeHTTP(responseWriter, request)
			return
		}
		policyWriter := &responsePolicyWriter{
			ResponseWriter: responseWriter,
			responseHeaders: func() map[string]string {
				copiedHeaders := make(map[string]string, len(resolvedHeaders))
				for headerName, headerValue := range resolvedHeaders {
					copiedHeaders[headerName] = headerValue
				}
				return copiedHeaders
			}(),
		}
		next.ServeHTTP(policyWriter, request)
	})
}

type responsePolicyWriter struct {
	http.ResponseWriter
	responseHeaders map[string]string
	applied         bool
}

func (writer *responsePolicyWriter) WriteHeader(statusCode int) {
	writer.applyHeaders()
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *responsePolicyWriter) Write(content []byte) (int, error) {
	writer.applyHeaders()
	return writer.ResponseWriter.Write(content)
}

func (writer *responsePolicyWriter) Flush() {
	writer.applyHeaders()
	responseFlusher, supportsFlush := writer.ResponseWriter.(http.Flusher)
	if supportsFlush {
		responseFlusher.Flush()
	}
}

func (writer *responsePolicyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	responseHijacker := writer.ResponseWriter.(http.Hijacker)
	return responseHijacker.Hijack()
}

func (writer *responsePolicyWriter) applyHeaders() {
	if writer.applied {
		return
	}
	for headerName, headerValue := range writer.responseHeaders {
		writer.ResponseWriter.Header().Set(headerName, headerValue)
	}
	writer.applied = true
}
