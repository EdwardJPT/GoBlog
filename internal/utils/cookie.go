package utils

import (
	"net/http"
	"time"
)

func SetTokenCookie(w http.ResponseWriter, token string, remember bool) {
	maxAge := int(24 * time.Hour.Seconds()) // 24 hours
	if remember {
		maxAge = int(30 * 24 * time.Hour.Seconds()) // 30 days
	}
	cookie := &http.Cookie{
		Name:     "bearer",
		Value:    token,
		Path:     "/",
		HttpOnly: true, // Not accessible via JavaScript (XSS protection)
		Secure:   true, // Only sent over HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
	http.SetCookie(w, cookie)
}

func ClearTokenCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "bearer",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // match your set cookie
		MaxAge:   -1,   // delete
	}
	http.SetCookie(w, cookie)
}
