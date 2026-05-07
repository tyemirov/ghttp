package app

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var errInvalidServeBinding = errors.New("serve.binding.invalid")

type serveBinding struct {
	bindAddress string
	port        string
}

func newServeBinding(rawBindAddress string, rawPort string) (serveBinding, error) {
	bindAddress := strings.TrimSpace(rawBindAddress)
	port := strings.TrimSpace(rawPort)
	if bindAddress == "" {
		return serveBinding{bindAddress: bindAddress, port: port}, nil
	}

	host, splitPort, splitErr := net.SplitHostPort(bindAddress)
	if splitErr == nil {
		normalizedPort := strings.TrimSpace(splitPort)
		if normalizedPort == "" {
			return serveBinding{}, fmt.Errorf("%w: bind address port is empty", errInvalidServeBinding)
		}
		return serveBinding{bindAddress: strings.TrimSpace(host), port: normalizedPort}, nil
	}

	bracketedHost, bracketedHostErr := normalizeBracketedBindHost(bindAddress)
	if bracketedHostErr != nil {
		return serveBinding{}, bracketedHostErr
	}
	if bracketedHost != "" {
		return serveBinding{bindAddress: bracketedHost, port: port}, nil
	}

	return serveBinding{bindAddress: bindAddress, port: port}, nil
}

func normalizeBracketedBindHost(bindAddress string) (string, error) {
	if !strings.HasPrefix(bindAddress, "[") || !strings.HasSuffix(bindAddress, "]") {
		return "", nil
	}
	host := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(bindAddress, "["), "]"))
	if host == "" {
		return "", fmt.Errorf("%w: bind address host is empty", errInvalidServeBinding)
	}
	return host, nil
}
