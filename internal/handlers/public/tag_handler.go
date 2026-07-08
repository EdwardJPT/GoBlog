package handlers

import (
	"database/sql"
	"encoding/base64"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"blog/internal/models"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
)

type TagsRepo interface {
	GetAllTags(filter string, selectedTypes []string) ([]models.Tag, error)

	GetContentByTag(
		tag string,
		selectedTypes []string,
		encodedCursor string,
		isPrev bool,
		limit int,
	) ([]*models.TagItem, error)

	HasPublishedRowsAroundCursor(
		tag string,
		selectedTypes []string,
		firstTimestamp time.Time,
		lastTimestamp time.Time,
	) (hasPrev bool, hasNext bool, err error)
}

type TagsHandler struct {
	tags      TagsRepo
	templates *template.Template
}

func NewTagsHandler(db *sql.DB) *TagsHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/tags/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &TagsHandler{
		tags:      models.NewTagRepository(db),
		templates: tmpl,
	}
}

type TagsListData struct{ Tags []models.Tag }

func (h *TagsHandler) TagsList(w http.ResponseWriter, r *http.Request) {
	// Content type request
	r.ParseForm()
	selectedTypes := r.Form["type"]

	// notes tag#001 (<- search this to see the relation in another place)
	// Maybe one day we are going to implement the search result
	tags, err := h.tags.GetAllTags("", selectedTypes)
	if err != nil {
		slog.Error("Failed fetching tags list", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := TagsListData{Tags: tags}

	if err = h.templates.ExecuteTemplate(w, "tags", data); err != nil {
		slog.Error("Template handlers.TagsList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

type TagData struct {
	Tag        string
	Feed       []*models.TagItem
	Pagination CursorPagination
}

func (h *TagsHandler) Tag(w http.ResponseWriter, r *http.Request) {
	// Parsing the pagination request
	cursor := r.URL.Query().Get("next_cursor")
	isPrev := false

	if cursor == "" {
		cursor = r.URL.Query().Get("prev_cursor")
		if cursor != "" {
			isPrev = true
		}
	}

	// The title of the tag
	tag := chi.URLParam(r, "tag")
	// Content type request
	r.ParseForm()
	selectedTypes := r.Form["type"]

	limit := utils.FeedSize

	content, err := h.tags.GetContentByTag(tag, selectedTypes, cursor, isPrev, limit)
	if err != nil {
		slog.Error("Failed fetching contents by tag", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// If we queried going backwards (ASC), we must reverse the slice to maintain DESC order for the UI
	if isPrev {
		for i, j := 0, len(content)-1; i < j; i, j = i+1, j-1 {
			content[i], content[j] = content[j], content[i]
		}
	}

	var nextCursorBase64, prevCursorBase64 string
	if len(content) > 0 {
		firstTimestamp := content[0].UpdatedAt
		lastTimestamp := content[len(content)-1].UpdatedAt

		hasPrev, hasNext, err := h.tags.HasPublishedRowsAroundCursor(
			tag,
			selectedTypes,
			firstTimestamp,
			lastTimestamp,
		)
		if err != nil {
			slog.Error("Failed fetching next and prev cursor of contents by tag", "error", err)
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

	data := TagData{
		Tag:        tag,
		Feed:       content,
		Pagination: pagination,
	}

	if err = h.templates.ExecuteTemplate(w, "tag", data); err != nil {
		slog.Error("Template handlers.Tag failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
