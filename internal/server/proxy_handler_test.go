package server

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHostWithoutPort(testingInstance *testing.T) {
	testCases := []struct {
		name     string
		hostPort string
		expected string
	}{
		{
			name:     "hostname with port",
			hostPort: "example.com:443",
			expected: "example.com",
		},
		{
			name:     "ipv4 with port",
			hostPort: "127.0.0.1:8080",
			expected: "127.0.0.1",
		},
		{
			name:     "ipv6 with port",
			hostPort: "[::1]:8443",
			expected: "::1",
		},
		{
			name:     "hostname without port",
			hostPort: "example.com",
			expected: "example.com",
		},
	}

	for _, testCase := range testCases {
		testingInstance.Run(testCase.name, func(testingInstance *testing.T) {
			result := hostWithoutPort(testCase.hostPort)
			if result != testCase.expected {
				testingInstance.Fatalf("expected %s, got %s", testCase.expected, result)
			}
		})
	}
}

func TestProxyHandlerWebSocketRequiresHijacker(testingInstance *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxyRoutes, routeErr := NewProxyRoutes([]string{"/api=" + backend.URL})
	if routeErr != nil {
		testingInstance.Fatalf("proxy routes: %v", routeErr)
	}

	baseHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
	})

	handler := newProxyHandler(baseHandler, proxyRoutes)
	request := httptest.NewRequest(http.MethodGet, "/api/ws", nil)
	request.Header.Set(headerConnection, "Upgrade")
	request.Header.Set(headerUpgrade, valueWebSocket)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusInternalServerError {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusInternalServerError, responseRecorder.Code)
	}
}

func TestProxyRouteHandlerReadResponseFailure(testingInstance *testing.T) {
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingInstance.Fatalf("listen: %v", listenErr)
	}
	defer listener.Close()

	handledConnection := make(chan struct{}, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()
		_ = connection.SetReadDeadline(time.Now().Add(2 * time.Second))
		readBuffer := make([]byte, 64)
		_, _ = connection.Read(readBuffer)
		_, _ = connection.Write([]byte("not-http\r\n\r\n"))
		handledConnection <- struct{}{}
	}()

	backendURL := "http://" + listener.Addr().String()
	route, routeErr := newProxyRoute("/api", backendURL)
	if routeErr != nil {
		testingInstance.Fatalf("route: %v", routeErr)
	}
	routeHandler := newProxyRouteHandler(route)

	clientConnection, clientPeer := net.Pipe()
	defer clientConnection.Close()
	defer clientPeer.Close()
	clientBuffer := bufio.NewReadWriter(bufio.NewReader(strings.NewReader("")), bufio.NewWriter(io.Discard))
	responseWriter := newHijackableResponseWriter(clientConnection, clientBuffer)

	request := httptest.NewRequest(http.MethodGet, "/api/ws", nil)
	request.Header.Set(headerConnection, "Upgrade")
	request.Header.Set(headerUpgrade, valueWebSocket)

	routeHandler.handleWebSocket(responseWriter, request)

	select {
	case <-handledConnection:
	case <-time.After(2 * time.Second):
		testingInstance.Fatalf("backend did not handle connection")
	}
}

func TestProxyRouteHandlerResponseWriteFailure(testingInstance *testing.T) {
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingInstance.Fatalf("listen: %v", listenErr)
	}
	defer listener.Close()

	handledConnection := make(chan struct{}, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()
		backendReader := bufio.NewReader(connection)
		for {
			line, readErr := backendReader.ReadString('\n')
			if readErr != nil {
				break
			}
			if line == "\r\n" {
				break
			}
		}
		_, _ = connection.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n"))
		handledConnection <- struct{}{}
	}()

	backendURL := "http://" + listener.Addr().String()
	route, routeErr := newProxyRoute("/api", backendURL)
	if routeErr != nil {
		testingInstance.Fatalf("route: %v", routeErr)
	}
	routeHandler := newProxyRouteHandler(route)

	clientConnection, clientPeer := net.Pipe()
	defer clientConnection.Close()
	defer clientPeer.Close()
	writeErrorConnection := &writeErrorConn{
		Conn:       clientConnection,
		writeError: errors.New("write failure"),
	}
	clientBuffer := bufio.NewReadWriter(bufio.NewReader(clientPeer), bufio.NewWriter(clientPeer))
	responseWriter := newHijackableResponseWriter(writeErrorConnection, clientBuffer)

	request := httptest.NewRequest(http.MethodGet, "/api/ws", nil)
	request.Header.Set(headerConnection, "Upgrade")
	request.Header.Set(headerUpgrade, valueWebSocket)

	routeHandler.handleWebSocket(responseWriter, request)

	if !writeErrorConnection.writeCalled {
		testingInstance.Fatalf("expected client write to be attempted")
	}

	select {
	case <-handledConnection:
	case <-time.After(2 * time.Second):
		testingInstance.Fatalf("backend did not handle connection")
	}
}

func TestProxyRouteHandlerUpgradeWriteFailure(testingInstance *testing.T) {
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingInstance.Fatalf("listen: %v", listenErr)
	}
	defer listener.Close()

	handledConnection := make(chan struct{}, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		if tcpConnection, ok := connection.(*net.TCPConn); ok {
			_ = tcpConnection.SetLinger(0)
		}
		_ = connection.Close()
		handledConnection <- struct{}{}
	}()

	backendURL := "http://" + listener.Addr().String()
	route, routeErr := newProxyRoute("/api", backendURL)
	if routeErr != nil {
		testingInstance.Fatalf("route: %v", routeErr)
	}
	routeHandler := newProxyRouteHandler(route)

	clientConnection, clientPeer := net.Pipe()
	defer clientConnection.Close()
	defer clientPeer.Close()
	clientBuffer := bufio.NewReadWriter(bufio.NewReader(strings.NewReader("")), bufio.NewWriter(io.Discard))
	responseWriter := newHijackableResponseWriter(clientConnection, clientBuffer)

	request := httptest.NewRequest(http.MethodGet, "/api/ws", nil)
	request.Header.Set(headerConnection, "Upgrade")
	request.Header.Set(headerUpgrade, valueWebSocket)

	routeHandler.handleWebSocket(responseWriter, request)

	select {
	case <-handledConnection:
	case <-time.After(2 * time.Second):
		testingInstance.Fatalf("backend did not handle connection")
	}
}

func TestProxyRouteHandlerTLSHandshakeFailure(testingInstance *testing.T) {
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingInstance.Fatalf("listen: %v", listenErr)
	}
	defer listener.Close()

	handledConnection := make(chan struct{}, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		_ = connection.Close()
		handledConnection <- struct{}{}
	}()

	backendURL := &url.URL{
		Scheme: "https",
		Host:   listener.Addr().String(),
	}
	route, routeErr := newProxyRoute("/api", backendURL.String())
	if routeErr != nil {
		testingInstance.Fatalf("route: %v", routeErr)
	}
	routeHandler := newProxyRouteHandler(route)

	request := httptest.NewRequest(http.MethodGet, "/api/ws", nil)
	request.Header.Set(headerConnection, "Upgrade")
	request.Header.Set(headerUpgrade, valueWebSocket)
	responseRecorder := httptest.NewRecorder()

	routeHandler.handleWebSocket(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadGateway {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusBadGateway, responseRecorder.Code)
	}

	select {
	case <-handledConnection:
	case <-time.After(2 * time.Second):
		testingInstance.Fatalf("backend did not handle connection")
	}
}

func TestProxyRouteHandlerHijackFailure(testingInstance *testing.T) {
	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingInstance.Fatalf("listen: %v", listenErr)
	}
	defer listener.Close()

	handledConnection := make(chan struct{}, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		_ = connection.Close()
		handledConnection <- struct{}{}
	}()

	backendURL := "http://" + listener.Addr().String()
	route, routeErr := newProxyRoute("/api", backendURL)
	if routeErr != nil {
		testingInstance.Fatalf("route: %v", routeErr)
	}
	routeHandler := newProxyRouteHandler(route)

	request := httptest.NewRequest(http.MethodGet, "/api/ws", nil)
	request.Header.Set(headerConnection, "Upgrade")
	request.Header.Set(headerUpgrade, valueWebSocket)

	failingWriter := newHijackableResponseWriter(nil, nil)
	failingWriter.hijackFailure = errors.New("hijack failed")

	routeHandler.handleWebSocket(failingWriter, request)

	if failingWriter.statusCode != http.StatusInternalServerError {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusInternalServerError, failingWriter.statusCode)
	}

	select {
	case <-handledConnection:
	case <-time.After(2 * time.Second):
		testingInstance.Fatalf("backend did not handle connection")
	}
}

type hijackableResponseWriter struct {
	header        http.Header
	statusCode    int
	wroteHeader   bool
	hijackConn    net.Conn
	hijackBuffer  *bufio.ReadWriter
	hijackFailure error
}

func newHijackableResponseWriter(connection net.Conn, readWriter *bufio.ReadWriter) *hijackableResponseWriter {
	return &hijackableResponseWriter{
		header:       make(http.Header),
		statusCode:   http.StatusOK,
		hijackConn:   connection,
		hijackBuffer: readWriter,
	}
}

func (writer *hijackableResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *hijackableResponseWriter) Write(data []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return len(data), nil
}

func (writer *hijackableResponseWriter) WriteHeader(statusCode int) {
	writer.statusCode = statusCode
	writer.wroteHeader = true
}

func (writer *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return writer.hijackConn, writer.hijackBuffer, writer.hijackFailure
}

type writeErrorConn struct {
	net.Conn
	writeError  error
	writeCalled bool
}

func (connection *writeErrorConn) Write(data []byte) (int, error) {
	connection.writeCalled = true
	return 0, connection.writeError
}
