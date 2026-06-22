package handlers

import (
	"database/sql"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"

	"blog/internal/models"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
)

type TagsHandler struct {
	tags      *models.TagRepository
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

	tags, err := h.tags.GetAllTags(selectedTypes)
	if err != nil {
		log.Printf("Error fetching tags list: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := TagsListData{Tags: tags}

	if err = h.templates.ExecuteTemplate(w, "tags", data); err != nil {
		log.Println("handlers.TagsList error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		log.Printf("Error fetching content by tag: %v", err)
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
		log.Println("handlers.Tag error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
