package handlers

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"blog/internal/models"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
)

type BlogPostRepo interface {
	GetBySlug(slug string) (*models.Post, error)
}

type BlogNoteRepo interface {
	GetBySlug(slug string) (*models.Note, error)
}

type BlogBookmarkRepo interface {
	GetBySlug(slug string) (*models.Bookmark, error)
}

type BlogWritingRepo interface {
	GetBlogFeed(
		selectedTypes []string,
		encodedCursor string,
		isPrev bool,
		limit int,
	) ([]models.BlogItem, error)

	HasPublishedRowsAroundCursor(
		selectedTypes []string,
		firstTimestamp time.Time,
		lastTimestamp time.Time,
	) (hasPrev bool, hasNext bool, err error)
}

type WritingsHandler struct {
	posts     BlogPostRepo
	notes     BlogNoteRepo
	bookmarks BlogBookmarkRepo
	writings  BlogWritingRepo
	templates *template.Template
}

func NewWritingsHandler(db *sql.DB) *WritingsHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/blog/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/404.html"))
	return &WritingsHandler{
		posts:     models.NewPostRepository(db),
		notes:     models.NewNoteRepository(db),
		bookmarks: models.NewBookmarkRepository(db),
		writings:  models.NewWritingRepository(db),
		templates: tmpl,
	}
}

type CursorPagination struct {
	Limit      int    `json:"limit"`
	PrevCursor string `json:"prev_cursor,omitempty"` // base64
	NextCursor string `json:"next_cursor,omitempty"` // base64
}

type BlogPageData struct {
	BlogFeed   []models.BlogItem
	Pagination CursorPagination
}

func (h *WritingsHandler) BlogPage(w http.ResponseWriter, r *http.Request) {
	// Parsing the pagination request
	cursor := r.URL.Query().Get("next_cursor")
	isPrev := false

	if cursor == "" {
		cursor = r.URL.Query().Get("prev_cursor")
		if cursor != "" {
			isPrev = true
		}
	}

	// Writings type request
	r.ParseForm()
	selectedTypes := r.Form["type"]

	limit := utils.FeedSize

	blogData, err := h.writings.GetBlogFeed(selectedTypes, cursor, isPrev, limit)
	if err != nil {
		slog.Error("Failed fetching blog feed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// If we queried going backwards (ASC), we must reverse the slice to maintain DESC order for the UI
	if isPrev {
		for i, j := 0, len(blogData)-1; i < j; i, j = i+1, j-1 {
			blogData[i], blogData[j] = blogData[j], blogData[i]
		}
	}

	// Calculate Next and Prev cursors
	var nextCursorBase64, prevCursorBase64 string
	if len(blogData) > 0 {
		firstTimestamp := blogData[0].UpdatedAt
		lastTimestamp := blogData[len(blogData)-1].UpdatedAt

		hasPrev, hasNext, err := h.writings.HasPublishedRowsAroundCursor(
			selectedTypes,
			firstTimestamp,
			lastTimestamp,
		)
		if err != nil {
			slog.Error("Failed fetching next and prev cursor of blog", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Logic for Prev Cursor
		if hasPrev {
			prevCursorBase64 = base64.URLEncoding.EncodeToString(
				[]byte(firstTimestamp.UTC().Format(utils.SqliteTimeLayout)),
			)
		}
		// Logic for Next Cursor
		if hasNext {
			nextCursorBase64 = base64.URLEncoding.EncodeToString(
				[]byte(lastTimestamp.UTC().Format(utils.SqliteTimeLayout)),
			)
		}
	}

	pagination := CursorPagination{
		Limit:      limit,
		PrevCursor: prevCursorBase64,
		NextCursor: nextCursorBase64,
	}

	data := BlogPageData{
		BlogFeed:   blogData,
		Pagination: pagination,
	}

	if err := h.templates.ExecuteTemplate(w, "blog", data); err != nil {
		slog.Error("Template handlers.BlogPage failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

type PostContent struct {
	Title       string
	Content     template.HTML
	Tags        string
	PublishedAt time.Time
}

type NoteContent struct {
	Title       string
	Content     template.HTML
	Tags        string
	PublishedAt time.Time
}

type BookmarkContent struct {
	Title       string
	Content     template.HTML
	Tags        string
	URL         string
	PublishedAt time.Time
}

func (h *WritingsHandler) Post(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	post, err := h.posts.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Error("Post not found error", "slug", slug)
			// Set the 404 header explicitly
			w.WriteHeader(http.StatusNotFound)
			// Render 404 directly here
			data := NotFoundData{ParentFeature: "blog"}
			if err := utils.RenderTemplate(w, h.templates, "404", data); err != nil {
				slog.Error("Template 404 failed in blog_handlers", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.Error("Failed fetching post from database", "slug", slug, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	postContent := PostContent{
		Title:       post.Title,
		Content:     utils.RenderMarkdown(post.Content),
		Tags:        post.Tags,
		PublishedAt: post.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_post_layout", postContent); err != nil {
		slog.Error("Template handlers.Post failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *WritingsHandler) Note(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	note, err := h.notes.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Error("Note not found error", "slug", slug)
			// Set the 404 header explicitly
			w.WriteHeader(http.StatusNotFound)
			// Render 404 directly here
			data := NotFoundData{ParentFeature: "blog"}
			if err := utils.RenderTemplate(w, h.templates, "404", data); err != nil {
				slog.Error("Template 404 failed in blog_handlers", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.Error("Failed fetching note from database", "slug", slug, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	noteContent := NoteContent{
		Title:       note.Title,
		Content:     utils.RenderMarkdown(note.Content),
		Tags:        note.Tags,
		PublishedAt: note.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_note_layout", noteContent); err != nil {
		slog.Error("Template handlers.Note failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *WritingsHandler) Bookmark(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	bookmark, err := h.bookmarks.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Error("Bookmark not found error", "slug", slug)
			// Set the 404 header explicitly
			w.WriteHeader(http.StatusNotFound)
			// Render 404 directly here
			data := NotFoundData{ParentFeature: "blog"}
			if err := utils.RenderTemplate(w, h.templates, "404", data); err != nil {
				slog.Error("Template 404 failed in blog_handlers", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.Error("Failed fetching bookmark from database", "slug", slug, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	bookmarkContent := BookmarkContent{
		Title:       bookmark.Title,
		Content:     utils.RenderMarkdown(bookmark.Description),
		Tags:        bookmark.Tags,
		URL:         bookmark.URL,
		PublishedAt: bookmark.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_bookmark_layout", bookmarkContent); err != nil {
		slog.Error("Template handlers.Bookmark failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
