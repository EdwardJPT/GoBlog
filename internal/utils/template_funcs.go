package utils

import (
	"html/template"
	"strings"
	"time"
)

func RegisterTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 at 3:04 PM")
		},
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			return s[:maxLen-3] + "..."
		},
		"mod":   func(i, j int) int { return i % j },
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
	}
}
