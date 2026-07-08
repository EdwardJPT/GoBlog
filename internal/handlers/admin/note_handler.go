package admin_handlers

import (
	"database/sql"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	handlers "blog/internal/handlers/public"
	"blog/internal/models"
	admin_services "blog/internal/services/admin"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type NoteHandler struct {
	notes     *models.NoteRepository
	stats     *admin_services.StatsService
	templates *template.Template
}

func NewNoteHandler(db *sql.DB) *NoteHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/notes/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/blog/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &NoteHandler{
		notes:     models.NewNoteRepository(db),
		stats:     admin_services.NewStatsService(db),
		templates: tmpl,
	}
}

type NoteListData struct {
	*admin_services.StatsCardData
	Filter      string
	Status      string
	Notes       []*models.Note
	SectionType string
}

// Notes handlers
func (h *NoteHandler) NotesList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	notes, err := h.notes.GetAll(filter, status)
	if err != nil {
		slog.Error("Failed fetching all notes", "error", err)
		slog.Error("Query parameters", "filter", filter, "status", status)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := NoteListData{
		StatsCardData: statsData,
		Notes:         notes,
		Filter:        filter,
		Status:        status,
		SectionType:   "notes-list",
	}

	err = h.templates.ExecuteTemplate(w, "notes-layout", data)
	if err != nil {
		slog.Error("Template handlers.NotesList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

type NoteListComponent struct {
	Notes []*models.Note
}

func (h *NoteHandler) NotesListComponent(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")

	notes, err := h.notes.GetAll(filter, "published")
	if err != nil {
		slog.Error("Failed fetching all notes", "error", err)
		slog.Error("Query parameters", "filter", filter)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := NoteListComponent{Notes: notes}

	err = h.templates.ExecuteTemplate(w, "notes-list-component", data)
	if err != nil {
		slog.Error("Template handlers.NotesListComponent failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *NoteHandler) NotesForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Note": &models.Note{
			Status: "draft",
		},
		"SectionType": "notes-form",
	}

	err := h.templates.ExecuteTemplate(w, "notes-layout", data)
	if err != nil {
		slog.Error("Template handlers.NotesForm failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *NoteHandler) NotesCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.NotesCreate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	tags := strings.ReplaceAll(r.FormValue("tags"), " ", "")
	status := r.FormValue("status")
	slug := r.FormValue("slug")

	if slug == "" {
		slug = models.GenerateSlug(title)
	}
	nanoId, err := gonanoid.New(10)
	if err != nil {
		slog.Error("Failed to generate nano ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	slug += "-" + nanoId

	note := &models.Note{
		Title:   title,
		Slug:    slug,
		Content: content,
		Tags:    tags,
		Status:  status,
	}

	if err := h.notes.Create(note); err != nil {
		slog.Error("Failed creating note", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/notes", http.StatusFound)
}

func (h *NoteHandler) NotesEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.NotesEdit received invalid note ID")
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := h.notes.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching note by ID", "error", err, "ID", id)
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Note":        note,
		"ID":          note.ID,
		"IsEdit":      true,
		"SectionType": "notes-form",
	}

	err = h.templates.ExecuteTemplate(w, "notes-layout", data)
	if err != nil {
		slog.Error("Template handlers.NotesEdit failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *NoteHandler) NotesPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.NotesPreview received invalid note ID")
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := h.notes.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching note by ID", "error", err, "ID", id)
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	noteContent := handlers.NoteContent{
		Title:       note.Title,
		Content:     utils.RenderMarkdown(note.Content),
		Tags:        note.Tags,
		PublishedAt: note.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_note_layout", noteContent); err != nil {
		slog.Error("Template handlers.NotesPreview failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *NoteHandler) NotesUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.NotesUpdate received invalid note ID")
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.NotesUpdate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	note, err := h.notes.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching note by ID", "error", err, "ID", id)
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	note.Title = r.FormValue("title")
	note.Content = r.FormValue("content")
	note.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	note.Status = r.FormValue("status")
	note.Slug = r.FormValue("slug")

	if note.Slug == "" {
		note.Slug = models.GenerateSlug(note.Title)
		nanoId, err := gonanoid.New(10)
		if err != nil {
			slog.Error("Failed to generate nano ID", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		note.Slug += "-" + nanoId
	}

	if err := h.notes.Update(note); err != nil {
		slog.Error("Failed updating note", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/notes", http.StatusFound)
}

func (h *NoteHandler) NotesDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.NotesDelete received invalid note ID")
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := h.notes.Delete(id); err != nil {
		slog.Error("Failed deleting note by ID", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/notes", http.StatusFound)
}
