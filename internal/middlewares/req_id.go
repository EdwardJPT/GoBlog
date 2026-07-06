package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func ResponseRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fetch the Request ID from the context (injected by chi)
		reqID := middleware.GetReqID(r.Context())
		// If an ID exists, set it as a response header
		if reqID != "" {
			w.Header().Set("X-Request-Id", reqID)
		}
		// Continue to the next middleware or handler
		next.ServeHTTP(w, r)
	})
}
