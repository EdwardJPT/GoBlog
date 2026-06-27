package utils

import (
	"net"
	"net/http"
	"strings"
)

func GetIP(r *http.Request) string {
	// Get the direct connection IP and strip the port safely
	directIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		directIP = r.RemoteAddr
	}

	// Only trust headers if the request came from Nginx
	if directIP == "127.0.0.1" || directIP == "::1" {
		// Check X-Forwarded-For first
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			ips := strings.Split(xff, ",")
			return strings.TrimSpace(ips[0])
		}

		// Check X-Real-IP second
		xri := r.Header.Get("X-Real-IP")
		if xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	return r.RemoteAddr
}
