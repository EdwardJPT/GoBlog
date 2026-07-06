package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

type NotFoundHandler struct {
	templates *template.Template
}

func NewNotFoundHandler() *NotFoundHandler {
	tmpl := template.New("")
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/404.html"))
	return &NotFoundHandler{templates: tmpl}
}

type NotFoundData struct {
	ParentFeature string
}

func (h *NotFoundHandler) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	// Parse where is the 404 came from, and from which sub-url?
	urlSource := r.URL.Query().Get("redirect")
	if urlSource == "" {
		urlSource = r.URL.Path
	}
	urlComponents := strings.Split(urlSource, "/")
	feature := ""
	for _, value := range urlComponents {
		if value != "" {
			feature = value
			break
		}
	}

	data := NotFoundData{}
	// If it's redirected from sub-url that's part of blog or projects,
	// then give the option to go back to /blog or /projects
	if feature != "" {
		featureUrlMap := map[string][]string{
			"blog":     {"post", "note", "bookmark"},
			"projects": {"project", "open-source-contribution"},
		}
		parentFeature := ""
		for key, value := range featureUrlMap {
			if found := slices.Contains(value, feature); found {
				parentFeature = key
				break
			}
		}
		data = NotFoundData{ParentFeature: parentFeature}
	}

	w.WriteHeader(http.StatusNotFound)
	if err := h.templates.ExecuteTemplate(w, "404", data); err != nil {
		slog.Error("Template handlers.NotFoundPage failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
