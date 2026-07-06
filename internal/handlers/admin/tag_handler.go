package admin_handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strings"

	"blog/internal/models"
	admin_services "blog/internal/services/admin"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
)

type TagsHandler struct {
	tags      *models.TagRepository
	stats     *admin_services.StatsService
	templates *template.Template
}

func NewTagHandler(db *sql.DB) *TagsHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/tags/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &TagsHandler{
		tags:      models.NewTagRepository(db),
		stats:     admin_services.NewStatsService(db),
		templates: tmpl,
	}
}

type TagsListData struct {
	*admin_services.StatsCardData
	Tags        []models.Tag
	SectionType string
}

type TagEditData struct {
	*admin_services.StatsCardData
	TagName     string
	SectionType string
}

func (h *TagsHandler) TagsList(w http.ResponseWriter, r *http.Request) {
	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	filter := r.URL.Query().Get("q")
	r.ParseForm()
	selectedTypes := r.Form["type"]

	tags, err := h.tags.GetAllTags(filter, selectedTypes)
	if err != nil {
		log.Printf("Error fetching tags list: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// notes tag#001 (<- search this to see the relation in another place)
	// We don't filter the returned search result in this admin page because
	// I think I want to see what other tags that also exist with the tag that
	// I searched

	data := TagsListData{
		StatsCardData: statsData,
		Tags:          tags,
		SectionType:   "tags-list", // Inject section type
	}

	if err = h.templates.ExecuteTemplate(w, "tags-layout", data); err != nil {
		log.Println("handlers.TagsList error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *TagsHandler) EditTagPage(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		http.Redirect(w, r, "/nimda/tags", http.StatusSeeOther)
		return
	}

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	data := TagEditData{
		StatsCardData: statsData,
		TagName:       name,
		SectionType:   "tags-form",
	}

	if err := h.templates.ExecuteTemplate(w, "tags-layout", data); err != nil {
		log.Println("handlers.EditTagPage error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *TagsHandler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	oldName := chi.URLParam(r, "name")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	newName := strings.TrimSpace(r.FormValue("new_name"))

	if newName != "" && oldName != newName {
		err := h.tags.RenameTag(oldName, newName)
		if err != nil {
			log.Printf("Error updating tag %s to %s: %v", oldName, newName, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/nimda/tags", http.StatusSeeOther)
}

func (h *TagsHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	err := h.tags.DeleteTag(name)
	if err != nil {
		log.Printf("Error deleting tag %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/tags", http.StatusSeeOther)
}
