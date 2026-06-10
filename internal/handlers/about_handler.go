package handlers

import (
	"html/template"
	"log"
	"net/http"
)

type AboutHandler struct {
	templates *template.Template
}

func NewAboutHandler() *AboutHandler {
	tmpl := template.New("")
	tmpl = template.Must(tmpl.ParseGlob("web/templates/about.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/components/*.html"))
	return &AboutHandler{templates: tmpl}
}

func (h *AboutHandler) AboutPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "about", nil); err != nil {
		log.Println("handlers.AboutPage error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
