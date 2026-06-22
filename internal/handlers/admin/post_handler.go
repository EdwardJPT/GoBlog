package admin_handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"blog/internal/models"
	admin_service "blog/internal/service/admin"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type PostHandler struct {
	posts     *models.PostRepository
	stats     *admin_service.StatsService
	templates *template.Template
}

func NewPostHandler(db *sql.DB) *PostHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/posts/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &PostHandler{
		posts:     models.NewPostRepository(db),
		stats:     admin_service.NewStatsService(db),
		templates: tmpl,
	}
}

type PostListData struct {
	*admin_service.StatsCardData
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
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	posts, err := h.posts.GetAll(filter, status)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	data := PostListComponent{Posts: posts}

	err = h.templates.ExecuteTemplate(w, "posts-list-component", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) PostsForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Posts": &models.Post{
			Status: "draft",
		},
		"SectionType": "posts-form",
	}

	err := h.templates.ExecuteTemplate(w, "posts-layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) PostsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/posts", http.StatusFound)
}

func (h *PostHandler) PostsEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.posts.GetByID(id)
	if err != nil {
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *PostHandler) PostsUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	post, err := h.posts.GetByID(id)
	if err != nil {
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
	}

	if err := h.posts.Update(post); err != nil {
		http.Error(w, "Bookmark not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/nimda/posts", http.StatusFound)
}

func (h *PostHandler) PostsDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid bookmark ID", http.StatusBadRequest)
		return
	}

	if err := h.posts.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/posts", http.StatusFound)
}
