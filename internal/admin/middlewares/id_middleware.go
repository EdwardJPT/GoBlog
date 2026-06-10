package admin_middlewares

import (
	admin_utils "blog/internal/admin/utils"
	"context"
	"net/http"
)

type RequestId struct {
	IP        string
	UserAgent string
}

type contextKey string

const IdKey contextKey = "meta"

func IdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := RequestId{
			IP:        admin_utils.GetIP(r),
			UserAgent: r.UserAgent(),
		}
		ctx := context.WithValue(r.Context(), IdKey, meta)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
