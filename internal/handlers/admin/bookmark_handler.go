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
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	bookmarks, err := h.bookmarks.GetAll(filter, status)
	if err != nil {
		slog.Error("Failed fetching all bookmarks", "error", err)
		slog.Error("Query parameters", "filter", filter, "status", status)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		slog.Error("Template handlers.BookmarksList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		slog.Error("Template handlers.BookmarksForm failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *BookmarkHandler) BookmarksCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.BookmarksCreate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	url := r.FormValue("url")
	description := r.FormValue("description")
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

	bookmark := &models.Bookmark{
		Title:       title,
		Slug:        slug,
		URL:         url,
		Description: description,
		Tags:        tags,
		Status:      status,
	}

	if err := h.bookmarks.Create(bookmark); err != nil {
		slog.Error("Failed creating bookmark", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/bookmarks", http.StatusFound)
}

func (h *BookmarkHandler) BookmarksEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.BookmarksEdit received invalid bookmark ID")
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	bookmark, err := h.bookmarks.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching bookmark by ID", "error", err, "ID", id)
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
		slog.Error("Template handlers.BookmarksEdit failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *BookmarkHandler) BookmarksPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.BookmarksPreview received invalid bookmark ID")
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	bookmark, err := h.bookmarks.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching bookmark by ID", "error", err, "ID", id)
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
		slog.Error("Template handlers.BookmarksPreview failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *BookmarkHandler) BookmarksUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.BookmarksUpdate received invalid bookmark ID")
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.BookmarksUpdate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	bookmark, err := h.bookmarks.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching bookmark by ID", "error", err, "ID", id)
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	}

	bookmark.Title = r.FormValue("title")
	bookmark.URL = r.FormValue("url")
	bookmark.Description = r.FormValue("description")
	bookmark.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	bookmark.Status = r.FormValue("status")
	bookmark.Slug = r.FormValue("slug")

	if bookmark.Slug == "" {
		bookmark.Slug = models.GenerateSlug(bookmark.Title)
		nanoId, err := gonanoid.New(10)
		if err != nil {
			slog.Error("Failed to generate nano ID", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		bookmark.Slug += "-" + nanoId
	}

	if err := h.bookmarks.Update(bookmark); err != nil {
		slog.Error("Failed updating bookmark", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/bookmarks", http.StatusFound)
}

func (h *BookmarkHandler) BookmarksDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.BookmarksDelete received invalid bookmark ID")
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	if err := h.bookmarks.Delete(id); err != nil {
		slog.Error("Failed deleting bookmark by ID", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/bookmarks", http.StatusFound)
}
