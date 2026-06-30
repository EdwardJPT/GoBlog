package admin_handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"blog/internal/models"
	admin_service "blog/internal/service/admin"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type NoteHandler struct {
	notes     *models.NoteRepository
	stats     *admin_service.StatsService
	templates *template.Template
}

func NewNoteHandler(db *sql.DB) *NoteHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/notes/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &NoteHandler{
		notes:     models.NewNoteRepository(db),
		stats:     admin_service.NewStatsService(db),
		templates: tmpl,
	}
}

type NoteListData struct {
	*admin_service.StatsCardData
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
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	notes, err := h.notes.GetAll(filter, status)
	if err != nil {
		log.Printf("Error fetching all notes: %v", err)
		log.Printf("filter: %s - status: %s", filter, status)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
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
		log.Printf("Error fetching all notes: %v", err)
		log.Printf("filter: %s", filter)
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	data := NoteListComponent{Notes: notes}

	err = h.templates.ExecuteTemplate(w, "notes-list-component", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *NoteHandler) NotesCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	nanoId, err := gonanoid.New(10)
	if err != nil {
		return
	}

	note := &models.Note{
		Title:   r.FormValue("title"),
		Slug:    models.GenerateSlug(r.FormValue("title")) + "-" + nanoId,
		Content: r.FormValue("content"),
		Tags:    strings.ReplaceAll(r.FormValue("tags"), " ", ""),
		Status:  r.FormValue("status"),
	}

	if err := h.notes.Create(note); err != nil {
		log.Printf("Error creating note: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/notes", http.StatusFound)
}

func (h *NoteHandler) NotesEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := h.notes.GetByID(id)
	if err != nil {
		log.Printf("Error fetching note by ID: %v", err)
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *NoteHandler) NotesUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	note, err := h.notes.GetByID(id)
	if err != nil {
		log.Printf("Error fetching note by ID: %v", err)
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	note.Title = r.FormValue("title")
	note.Content = r.FormValue("content")
	note.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	note.Status = r.FormValue("status")

	if err := h.notes.Update(note); err != nil {
		log.Printf("Error updating note: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/notes", http.StatusFound)
}

func (h *NoteHandler) NotesDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := h.notes.Delete(id); err != nil {
		log.Printf("Error deleting note by ID: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/notes", http.StatusFound)
}
