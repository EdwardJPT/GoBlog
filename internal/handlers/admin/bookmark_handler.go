package admin_handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
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

type BookmarkHandler struct {
	bookmarks *models.BookmarkRepository
	stats     *admin_services.StatsService
	templates *template.Template
}

func NewBookmarkHandler(db *sql.DB) *BookmarkHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/bookmarks/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/blog/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &BookmarkHandler{
		bookmarks: models.NewBookmarkRepository(db),
		stats:     admin_services.NewStatsService(db),
		templates: tmpl,
	}
}

type BookmarkListData struct {
	*admin_services.StatsCardData
	Filter      string
	Status      string
	Bookmarks   []*models.Bookmark
	SectionType string
}

// Bookmarks handlers
func (h *BookmarkHandler) BookmarksList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	bookmarks, err := h.bookmarks.GetAll(filter, status)
	if err != nil {
		log.Printf("Error fetching all bookmarks: %v", err)
		log.Printf("filter: %s - status: %s", filter, status)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := BookmarkListData{
		StatsCardData: statsData,
		Bookmarks:     bookmarks,
		Filter:        filter,
		Status:        status,
		SectionType:   "bookmarks-list",
	}

	err = h.templates.ExecuteTemplate(w, "bookmarks-layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *BookmarkHandler) BookmarksForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Bookmark": &models.Bookmark{
			Status: "draft",
		},
		"SectionType": "bookmarks-form",
	}

	err := h.templates.ExecuteTemplate(w, "bookmarks-layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *BookmarkHandler) BookmarksCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	nanoId, err := gonanoid.New(10)
	if err != nil {
		return
	}

	bookmark := &models.Bookmark{
		Title:       r.FormValue("title"),
		Slug:        models.GenerateSlug(r.FormValue("title")) + "-" + nanoId,
		URL:         r.FormValue("url"),
		Description: r.FormValue("description"),
		Tags:        strings.ReplaceAll(r.FormValue("tags"), " ", ""),
		Status:      r.FormValue("status"),
	}

	if err := h.bookmarks.Create(bookmark); err != nil {
		log.Printf("Error creating bookmark: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/bookmarks", http.StatusFound)
}

func (h *BookmarkHandler) BookmarksEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	bookmark, err := h.bookmarks.GetByID(id)
	if err != nil {
		log.Printf("Error fetching bookmark by ID: %v", err)
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Bookmark":    bookmark,
		"ID":          bookmark.ID,
		"IsEdit":      true,
		"SectionType": "bookmarks-form",
	}

	err = h.templates.ExecuteTemplate(w, "bookmarks-layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *BookmarkHandler) BookmarksPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	bookmark, err := h.bookmarks.GetByID(id)
	if err != nil {
		log.Printf("Error fetching bookmark by ID: %v", err)
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	}

	bookmarkContent := handlers.BookmarkContent{
		Title:       bookmark.Title,
		Content:     utils.RenderMarkdown(bookmark.Description),
		Tags:        bookmark.Tags,
		URL:         bookmark.URL,
		PublishedAt: bookmark.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_bookmark_layout", bookmarkContent); err != nil {
		log.Println("handlers.Bookmark error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *BookmarkHandler) BookmarksUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	bookmark, err := h.bookmarks.GetByID(id)
	if err != nil {
		log.Printf("Error fetching bookmark by ID: %v", err)
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	}

	bookmark.Title = r.FormValue("title")
	bookmark.URL = r.FormValue("url")
	bookmark.Description = r.FormValue("description")
	bookmark.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	bookmark.Status = r.FormValue("status")

	if err := h.bookmarks.Update(bookmark); err != nil {
		log.Printf("Error updating bookmark: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/bookmarks", http.StatusFound)
}

func (h *BookmarkHandler) BookmarksDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	if err := h.bookmarks.Delete(id); err != nil {
		log.Printf("Error deleting bookmark by ID: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/bookmarks", http.StatusFound)
}
