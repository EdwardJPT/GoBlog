package utils

import (
	"net/http"
	"strings"
)

func GetIP(r *http.Request) string {
	// X-Forwarded-For may contain multiple IPs
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Fallback
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Last fallback
	return r.RemoteAddr
}
