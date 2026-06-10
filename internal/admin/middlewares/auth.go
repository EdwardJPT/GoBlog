package admin_middlewares

import (
	admin_utils "blog/internal/admin/utils"
	"fmt"
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

func Auth(jwt *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from cookie
			cookie, err := r.Cookie("bearer")
			if err != nil {
				// No cookie → redirect to login
				redirectUrl := fmt.Sprintf("/?redirect=%s", r.URL.Path)
				http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
				return
			}
			// Verify token
			claims, err := jwt.Decode(cookie.Value)
			if err != nil {
				// Invalid token → clear cookie and redirect
				admin_utils.ClearTokenCookie(w)
				redirectUrl := fmt.Sprintf("/?redirect=%s", r.URL.Path)
				http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
				return
			}
			// Add claims to context
			ctx := jwtauth.NewContext(r.Context(), claims, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Middleware that would redirect authenticated users to /nimda/dashboard
func AuthenticatedRedirector(jwt *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from cookie
			cookie, err := r.Cookie("bearer")
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			// Verify token
			_, err = jwt.Decode(cookie.Value)
			// Valid token -> redirect to /nimda/dashboard
			if err == nil {
				http.Redirect(w, r, "/nimda/dashboard", http.StatusSeeOther)
				return
			}
			admin_utils.ClearTokenCookie(w)
			next.ServeHTTP(w, r)
		})
	}
}
