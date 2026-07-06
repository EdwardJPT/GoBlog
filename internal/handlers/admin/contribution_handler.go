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

type ContributionsHandler struct {
	contributions *models.ContributionRepository
	stats         *admin_services.StatsService
	templates     *template.Template
}

func NewContributionsHandler(db *sql.DB) *ContributionsHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/os_contributions/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/projects/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &ContributionsHandler{
		contributions: models.NewContributionRepository(db),
		stats:         admin_services.NewStatsService(db),
		templates:     tmpl,
	}
}

type ContribtionsListData struct {
	*admin_services.StatsCardData
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
		log.Printf("Error fetching all contributions: %v", err)
		log.Printf("filter: %s", filter)
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
		log.Printf("Error fetching all contributions: %v", err)
		log.Printf("filter: %s", filter)
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
		log.Printf("Error creating contribution: %v", err)
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
		log.Printf("Error fetching contribution by ID: %v", err)
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

func (h *ContributionsHandler) ContributionsPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	contribution, err := h.contributions.GetByID(id)
	if err != nil {
		log.Printf("Error fetching contribution by ID: %v", err)
		http.Error(w, "Contribution not found", http.StatusNotFound)
		return
	}

	contributionContent := handlers.ContributionContent{
		Title:               contribution.ProjectName,
		Description:         utils.RenderMarkdown(contribution.Description),
		Tags:                contribution.Tags,
		Link:                contribution.Link,
		ContributionSummary: template.HTML(contribution.ContributionSummary),
		PublishedAt:         contribution.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_contribution_layout", contributionContent); err != nil {
		log.Println("handlers.OpenSourceContribution error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		log.Printf("Error fetching contribution by ID: %v", err)
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
		log.Printf("Error updating contribution: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}

func (h *ContributionsHandler) ContributionsDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	if err := h.contributions.Delete(id); err != nil {
		log.Printf("Error deleting contribution by ID: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}
