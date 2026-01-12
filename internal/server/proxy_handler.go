package server

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	headerConnection        = "Connection"
	headerUpgrade           = "Upgrade"
	headerWebSocketProtocol = "Sec-WebSocket-Protocol"
	valueUpgrade            = "upgrade"
	valueWebSocket          = "websocket"
)

type proxyHandler struct {
	next        http.Handler
	backendURL  *url.URL
	pathPrefix  string
	httpProxy   *httputil.ReverseProxy
}

func newProxyHandler(next http.Handler, backendURL string, pathPrefix string) (http.Handler, error) {
	parsedURL, err := url.Parse(backendURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway: "+err.Error(), http.StatusBadGateway)
	}

	return &proxyHandler{
		next:       next,
		backendURL: parsedURL,
		pathPrefix: pathPrefix,
		httpProxy:  proxy,
	}, nil
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.pathPrefix) {
		h.next.ServeHTTP(w, r)
		return
	}

	if h.isWebSocketUpgrade(r) {
		h.handleWebSocket(w, r)
		return
	}

	h.httpProxy.ServeHTTP(w, r)
}

func (h *proxyHandler) isWebSocketUpgrade(r *http.Request) bool {
	connectionHeader := strings.ToLower(r.Header.Get(headerConnection))
	upgradeHeader := strings.ToLower(r.Header.Get(headerUpgrade))
	return strings.Contains(connectionHeader, valueUpgrade) && upgradeHeader == valueWebSocket
}

func (h *proxyHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	backendHost := h.backendURL.Host
	scheme := "ws"
	if h.backendURL.Scheme == "https" {
		scheme = "wss"
	}

	backendConn, err := net.DialTimeout("tcp", backendHost, 10*time.Second)
	if err != nil {
		http.Error(w, "Bad Gateway: failed to connect to backend", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	backendURL := &url.URL{
		Scheme:   scheme,
		Host:     backendHost,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}

	upgradeReq := &http.Request{
		Method:     r.Method,
		URL:        backendURL,
		Proto:      r.Proto,
		ProtoMajor: r.ProtoMajor,
		ProtoMinor: r.ProtoMinor,
		Header:     cloneHeaders(r.Header),
		Host:       backendHost,
	}
	upgradeReq.Header.Set("Host", backendHost)

	if err := upgradeReq.Write(backendConn); err != nil {
		return
	}

	backendBuf := bufio.NewReader(backendConn)
	resp, err := http.ReadResponse(backendBuf, upgradeReq)
	if err != nil {
		return
	}

	if err := resp.Write(clientConn); err != nil {
		return
	}

	done := make(chan struct{}, 2)

	go func() {
		copyWithBuffer(backendConn, clientBuf)
		done <- struct{}{}
	}()

	go func() {
		copyWithBuffer(clientConn, backendBuf)
		done <- struct{}{}
	}()

	<-done
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, vv := range src {
		dst[k] = append([]string(nil), vv...)
	}
	return dst
}

func copyWithBuffer(dst io.Writer, src io.Reader) {
	buf := make([]byte, 32*1024)
	io.CopyBuffer(dst, src, buf)
}
