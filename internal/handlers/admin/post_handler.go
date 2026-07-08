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

type PostHandler struct {
	posts     *models.PostRepository
	stats     *admin_services.StatsService
	templates *template.Template
}

func NewPostHandler(db *sql.DB) *PostHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/posts/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/blog/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &PostHandler{
		posts:     models.NewPostRepository(db),
		stats:     admin_services.NewStatsService(db),
		templates: tmpl,
	}
}

type PostListData struct {
	*admin_services.StatsCardData
	Filter      string
	Status      string
	Posts       []*models.Post
	SectionType string
}

// Blog Posts handlers
func (h *PostHandler) PostsList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	posts, err := h.posts.GetAll(filter, status)
	if err != nil {
		slog.Error("Failed fetching all posts", "error", err)
		slog.Error("Query parameters", "filter", filter, "status", status)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PostListData{
		StatsCardData: statsData,
		Posts:         posts,
		Filter:        filter,
		Status:        status,
		SectionType:   "posts-list",
	}

	err = h.templates.ExecuteTemplate(w, "posts-layout", data)
	if err != nil {
		slog.Error("Template handlers.PostsList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

type PostListComponent struct {
	Posts []*models.Post
}

func (h *PostHandler) PostsListComponent(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")

	posts, err := h.posts.GetAll(filter, "published")
	if err != nil {
		slog.Error("Failed fetching all posts", "error", err)
		slog.Error("Query parameters", "filter", filter)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PostListComponent{Posts: posts}

	err = h.templates.ExecuteTemplate(w, "posts-list-component", data)
	if err != nil {
		slog.Error("Template handlers.PostsListComponent failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) PostsForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Post": &models.Post{
			Status: "draft",
		},
		"SectionType": "posts-form",
	}

	err := h.templates.ExecuteTemplate(w, "posts-layout", data)
	if err != nil {
		slog.Error("Template handlers.PostsForm failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) PostsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.PostsCreate received invalid form data")
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

	post := &models.Post{
		Title:   title,
		Slug:    slug,
		Content: content,
		Tags:    tags,
		Status:  status,
	}

	if err := h.posts.Create(post); err != nil {
		slog.Error("Failed creating post", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/posts", http.StatusFound)
}

func (h *PostHandler) PostsEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.PostsEdit received invalid post ID")
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.posts.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching post by ID", "error", err, "ID", id)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Post":        post,
		"ID":          post.ID,
		"IsEdit":      true,
		"SectionType": "posts-form",
	}

	err = h.templates.ExecuteTemplate(w, "posts-layout", data)
	if err != nil {
		slog.Error("Template handlers.PostsEdit failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) PostsPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.PostsPreview received invalid post ID")
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.posts.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching post by ID", "error", err, "ID", id)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	postContent := handlers.PostContent{
		Title:       post.Title,
		Content:     utils.RenderMarkdown(post.Content),
		Tags:        post.Tags,
		PublishedAt: post.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_post_layout", postContent); err != nil {
		slog.Error("Template handlers.PostsPreview failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) PostsUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.PostsUpdate received invalid post ID")
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.PostsUpdate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	post, err := h.posts.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching post by ID", "error", err, "ID", id)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	post.Title = r.FormValue("title")
	post.Content = r.FormValue("content")
	post.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	post.Status = r.FormValue("status")
	post.Slug = r.FormValue("slug")

	if post.Slug == "" {
		post.Slug = models.GenerateSlug(post.Title)
		nanoId, err := gonanoid.New(10)
		if err != nil {
			slog.Error("Failed to generate nano ID", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		post.Slug += "-" + nanoId
	}

	if err := h.posts.Update(post); err != nil {
		slog.Error("Failed updating post", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/posts", http.StatusFound)
}

func (h *PostHandler) PostsDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.PostsDelete received invalid post ID")
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	if err := h.posts.Delete(id); err != nil {
		slog.Error("Failed deleting post by ID", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/posts", http.StatusFound)
}
