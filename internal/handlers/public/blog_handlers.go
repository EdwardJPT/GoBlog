package handlers

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"blog/internal/models"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
)

type WritingsHandler struct {
	posts     *models.PostRepository
	notes     *models.NoteRepository
	bookmarks *models.BookmarkRepository
	writings  *models.WritingRepository
	templates *template.Template
}

func NewWritingsHandler(db *sql.DB) *WritingsHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/blog/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
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
		log.Printf("Error fetching blog feed: %v", err)
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
		log.Println("handlers.BlogPage error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		redirectUrl := fmt.Sprintf("/404?redirect=%s", r.URL.Path)
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return
	}

	postContent := PostContent{
		Title:       post.Title,
		Content:     utils.RenderMarkdown(post.Content),
		Tags:        post.Tags,
		PublishedAt: post.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "post_layout", postContent); err != nil {
		log.Println("handlers.Post error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WritingsHandler) Note(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	note, err := h.notes.GetBySlug(slug)
	if err != nil {
		redirectUrl := fmt.Sprintf("/404?redirect=%s", r.URL.Path)
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return
	}

	noteContent := NoteContent{
		Title:       note.Title,
		Content:     utils.RenderMarkdown(note.Content),
		Tags:        note.Tags,
		PublishedAt: note.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "note_layout", noteContent); err != nil {
		log.Println("handlers.Note error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WritingsHandler) Bookmark(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	bookmark, err := h.bookmarks.GetBySlug(slug)
	if err != nil {
		redirectUrl := fmt.Sprintf("/404?redirect=%s", r.URL.Path)
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return
	}

	bookmarkContent := BookmarkContent{
		Title:       bookmark.Title,
		Content:     utils.RenderMarkdown(bookmark.Description),
		Tags:        bookmark.Tags,
		URL:         bookmark.URL,
		PublishedAt: bookmark.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "bookmark_layout", bookmarkContent); err != nil {
		log.Println("handlers.Bookmark error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
