package util

import (
	"net/url"
	"strings"
)

// ParseServerURL parses the server URL and returns the protocol (http/https) and host
func ParseServerURL(server string) (protocol, host string) {
	// If no protocol is specified, default to http
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		return "http", server
	}

	// Parse the URL
	parsed, err := url.Parse(server)
	if err != nil {
		// If parsing fails, default to http
		return "http", server
	}

	return parsed.Scheme, parsed.Host
}

// GetWebSocketProtocol returns the WebSocket protocol based on the HTTP protocol
func GetWebSocketProtocol(httpProtocol string) string {
	if httpProtocol == "https" {
		return "wss"
	}
	return "ws"
}
