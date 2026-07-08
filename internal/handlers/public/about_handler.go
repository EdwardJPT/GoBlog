package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
)

type AboutHandler struct {
	templates *template.Template
}

func NewAboutHandler() *AboutHandler {
	tmpl := template.New("")
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/about.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &AboutHandler{templates: tmpl}
}

func (h *AboutHandler) AboutPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "about", nil); err != nil {
		slog.Error("Template handlers.AboutPage failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
