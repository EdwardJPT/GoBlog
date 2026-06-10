package admin_handler

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"blog/internal/admin/service"
	admin_utils "blog/internal/admin/utils"
	"blog/internal/models"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type ContributionsHandler struct {
	contributions *models.ContributionRepository
	stats         *service.StatsService
	templates     *template.Template
}

func NewContributionsHandler(db *sql.DB) *ContributionsHandler {
	tmpl := template.New("").Funcs(admin_utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("internal/admin/templates/os_contributions/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("internal/admin/templates/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/components/*.html"))
	return &ContributionsHandler{
		contributions: models.NewContributionRepository(db),
		stats:         service.NewStatsService(db),
		templates:     tmpl,
	}
}

type ContribtionsListData struct {
	*service.StatsCardData
	Contributions []*models.Contribution
	Filter        string
	SectionType   string
}

func (h *ContributionsHandler) ContributionsList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	contributions, err := h.contributions.GetAll(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := ContribtionsListData{
		StatsCardData: statsData,
		Contributions: contributions,
		Filter:        filter,
		SectionType:   "opensource-list",
	}

	err = h.templates.ExecuteTemplate(w, "opensource-layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

type ContributionListComponent struct {
	Contributions []*models.Contribution
}

func (h *ContributionsHandler) ContributionsListComponent(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")

	contributions, err := h.contributions.GetAll(filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	data := ContributionListComponent{Contributions: contributions}

	err = h.templates.ExecuteTemplate(w, "os-contributions-list-component", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ContributionsHandler) ContributionsForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Contribution": &models.Contribution{},
		"SectionType":  "opensource-form",
	}

	err := h.templates.ExecuteTemplate(w, "opensource-layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ContributionsHandler) ContributionsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	nanoId, err := gonanoid.New(10)
	if err != nil {
		return
	}

	contribution := &models.Contribution{
		ProjectName:         r.FormValue("project_name"),
		Slug:                models.GenerateSlug(r.FormValue("project_name")) + "-" + nanoId,
		Description:         r.FormValue("description"),
		HomePageDescription: r.FormValue("home_page_description"),
		Tags:                strings.ReplaceAll(r.FormValue("tags"), " ", ""),
		Link:                r.FormValue("link"),
		ContributionSummary: r.FormValue("contribution_summary"),
	}

	if err := h.contributions.Create(contribution); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}

func (h *ContributionsHandler) ContributionsEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	contribution, err := h.contributions.GetByID(id)
	if err != nil {
		http.Error(w, "Contribution not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Contribution": contribution,
		"ID":           contribution.ID,
		"IsEdit":       true,
		"SectionType":  "opensource-form",
	}

	err = h.templates.ExecuteTemplate(w, "opensource-layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ContributionsHandler) ContributionsUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	contribution, err := h.contributions.GetByID(id)
	if err != nil {
		http.Error(w, "Contribution not found", http.StatusNotFound)
		return
	}

	contribution.ProjectName = r.FormValue("project_name")
	contribution.Description = r.FormValue("description")
	contribution.HomePageDescription = r.FormValue("home_page_description")
	contribution.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	contribution.Link = r.FormValue("link")
	contribution.ContributionSummary = r.FormValue("contribution_summary")

	if err := h.contributions.Update(contribution); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}

func (h *ContributionsHandler) ContributionsDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := h.contributions.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}
