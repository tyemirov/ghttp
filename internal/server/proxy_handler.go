package server

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	headerConnection = "Connection"
	headerHost       = "Host"
	headerUpgrade    = "Upgrade"
	valueUpgrade     = "upgrade"
	valueWebSocket   = "websocket"
)

type proxyHandler struct {
	next   http.Handler
	routes []proxyRouteHandler
}

type proxyRouteHandler struct {
	pathPrefix string
	backendURL *url.URL
	httpProxy  *httputil.ReverseProxy
}

func newProxyHandler(next http.Handler, proxyRoutes ProxyRoutes) http.Handler {
	routeHandlers := make([]proxyRouteHandler, 0, len(proxyRoutes.routes))
	for _, route := range proxyRoutes.routes {
		routeHandlers = append(routeHandlers, newProxyRouteHandler(route))
	}
	return &proxyHandler{
		next:   next,
		routes: routeHandlers,
	}
}

func newProxyRouteHandler(route proxyRoute) proxyRouteHandler {
	reverseProxy := httputil.NewSingleHostReverseProxy(route.backendURL)
	reverseProxy.ErrorHandler = func(responseWriter http.ResponseWriter, request *http.Request, err error) {
		http.Error(responseWriter, "Bad Gateway: "+err.Error(), http.StatusBadGateway)
	}
	return proxyRouteHandler{
		pathPrefix: route.pathPrefix,
		backendURL: route.backendURL,
		httpProxy:  reverseProxy,
	}
}

func (handler *proxyHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	routeHandler, matched := handler.matchRoute(request.URL.Path)
	if !matched {
		handler.next.ServeHTTP(responseWriter, request)
		return
	}

	if routeHandler.isWebSocketUpgrade(request) {
		routeHandler.handleWebSocket(responseWriter, request)
		return
	}

	routeHandler.httpProxy.ServeHTTP(responseWriter, request)
}

func (handler *proxyHandler) matchRoute(requestPath string) (*proxyRouteHandler, bool) {
	for routeIndex := range handler.routes {
		routeHandler := &handler.routes[routeIndex]
		if strings.HasPrefix(requestPath, routeHandler.pathPrefix) {
			return routeHandler, true
		}
	}
	return nil, false
}

func (routeHandler *proxyRouteHandler) isWebSocketUpgrade(request *http.Request) bool {
	connectionHeader := strings.ToLower(request.Header.Get(headerConnection))
	upgradeHeader := strings.ToLower(request.Header.Get(headerUpgrade))
	return strings.Contains(connectionHeader, valueUpgrade) && upgradeHeader == valueWebSocket
}

func (routeHandler *proxyRouteHandler) handleWebSocket(responseWriter http.ResponseWriter, request *http.Request) {
	backendHost := routeHandler.backendURL.Host
	scheme := "ws"
	useTLS := strings.EqualFold(routeHandler.backendURL.Scheme, proxySchemeHTTPS)
	if useTLS {
		scheme = "wss"
	}

	var backendConnection net.Conn
	var dialErr error

	if useTLS {
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		backendConnection, dialErr = tls.DialWithDialer(dialer, "tcp", backendHost, &tls.Config{
			ServerName: hostWithoutPort(backendHost),
		})
	} else {
		backendConnection, dialErr = net.DialTimeout("tcp", backendHost, 10*time.Second)
	}

	if dialErr != nil {
		http.Error(responseWriter, "Bad Gateway: failed to connect to backend", http.StatusBadGateway)
		return
	}
	defer backendConnection.Close()

	hijacker, supportsHijacker := responseWriter.(http.Hijacker)
	if !supportsHijacker {
		http.Error(responseWriter, "WebSocket hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConnection, clientBuffer, hijackErr := hijacker.Hijack()
	if hijackErr != nil {
		http.Error(responseWriter, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConnection.Close()

	backendURL := &url.URL{
		Scheme:   scheme,
		Host:     backendHost,
		Path:     request.URL.Path,
		RawQuery: request.URL.RawQuery,
	}

	upgradeRequest := &http.Request{
		Method:     request.Method,
		URL:        backendURL,
		Proto:      request.Proto,
		ProtoMajor: request.ProtoMajor,
		ProtoMinor: request.ProtoMinor,
		Header:     cloneHeaders(request.Header),
		Host:       backendHost,
	}
	upgradeRequest.Header.Set(headerConnection, headerUpgrade)
	upgradeRequest.Header.Set(headerUpgrade, valueWebSocket)
	upgradeRequest.Header.Set(headerHost, backendHost)

	if err := upgradeRequest.Write(backendConnection); err != nil {
		return
	}

	backendReader := bufio.NewReader(backendConnection)
	backendResponse, readErr := http.ReadResponse(backendReader, upgradeRequest)
	if readErr != nil {
		return
	}

	if err := backendResponse.Write(clientConnection); err != nil {
		return
	}

	completionSignals := make(chan struct{}, 2)

	go func() {
		copyWithBuffer(backendConnection, clientBuffer)
		completionSignals <- struct{}{}
	}()

	go func() {
		copyWithBuffer(clientConnection, backendReader)
		completionSignals <- struct{}{}
	}()

	<-completionSignals
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for key, values := range src {
		dst[key] = append([]string(nil), values...)
	}
	return dst
}

func copyWithBuffer(dst io.Writer, src io.Reader) {
	buf := make([]byte, 32*1024)
	io.CopyBuffer(dst, src, buf)
}

func hostWithoutPort(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}
