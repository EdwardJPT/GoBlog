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

type ContributionsListData struct {
	*admin_services.StatsCardData
	Contributions []*models.Contribution
	Filter        string
	SectionType   string
}

func (h *ContributionsHandler) ContributionsList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	contributions, err := h.contributions.GetAll(filter)
	if err != nil {
		slog.Error("Failed fetching all contributions", "error", err)
		slog.Error("Query parameters", "filter", filter)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := ContributionsListData{
		StatsCardData: statsData,
		Contributions: contributions,
		Filter:        filter,
		SectionType:   "opensource-list",
	}

	err = h.templates.ExecuteTemplate(w, "opensource-layout", data)
	if err != nil {
		slog.Error("Template handlers.ContributionsList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		slog.Error("Failed fetching all contributions", "error", err)
		slog.Error("Query parameters", "filter", filter)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := ContributionListComponent{Contributions: contributions}

	err = h.templates.ExecuteTemplate(w, "os-contributions-list-component", data)
	if err != nil {
		slog.Error("Template handlers.ContributionsListComponent failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		slog.Error("Template handlers.ContributionsForm failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *ContributionsHandler) ContributionsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.ContributionsCreate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	projectName := r.FormValue("project_name")
	description := r.FormValue("description")
	homePageDescription := r.FormValue("home_page_description")
	tags := strings.ReplaceAll(r.FormValue("tags"), " ", "")
	link := r.FormValue("link")
	contributionSummary := r.FormValue("contribution_summary")
	slug := r.FormValue("slug")

	if slug == "" {
		slug = models.GenerateSlug(projectName)
	}
	nanoId, err := gonanoid.New(10)
	if err != nil {
		slog.Error("Failed to generate nano ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	slug += "-" + nanoId

	contribution := &models.Contribution{
		ProjectName:         projectName,
		Slug:                slug,
		Description:         description,
		HomePageDescription: homePageDescription,
		Tags:                tags,
		Link:                link,
		ContributionSummary: contributionSummary,
	}

	if err := h.contributions.Create(contribution); err != nil {
		slog.Error("Failed creating contribution", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}

func (h *ContributionsHandler) ContributionsEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ContributionsEdit received invalid contribution ID")
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	contribution, err := h.contributions.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching contribution by ID", "error", err, "ID", id)
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
		slog.Error("Template handlers.ContributionsEdit failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *ContributionsHandler) ContributionsPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ContributionsPreview received invalid contribution ID")
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	contribution, err := h.contributions.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching contribution by ID", "error", err, "ID", id)
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
		slog.Error("Template handlers.ContributionsPreview failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *ContributionsHandler) ContributionsUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ContributionsUpdate received invalid contribution ID")
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.ContributionsUpdate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	contribution, err := h.contributions.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching contribution by ID", "error", err, "ID", id)
		http.Error(w, "Contribution not found", http.StatusNotFound)
		return
	}

	contribution.ProjectName = r.FormValue("project_name")
	contribution.Description = r.FormValue("description")
	contribution.HomePageDescription = r.FormValue("home_page_description")
	contribution.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	contribution.Link = r.FormValue("link")
	contribution.ContributionSummary = r.FormValue("contribution_summary")
	contribution.Slug = r.FormValue("slug")

	if contribution.Slug == "" {
		contribution.Slug = models.GenerateSlug(contribution.ProjectName)
		nanoId, err := gonanoid.New(10)
		if err != nil {
			slog.Error("Failed to generate nano ID", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		contribution.Slug += "-" + nanoId
	}

	if err := h.contributions.Update(contribution); err != nil {
		slog.Error("Failed updating contribution", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}

func (h *ContributionsHandler) ContributionsDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ContributionsDelete received invalid contribution ID")
		http.Error(w, "Invalid contribution ID", http.StatusBadRequest)
		return
	}

	if err := h.contributions.Delete(id); err != nil {
		slog.Error("Failed deleting contribution by ID", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/os-contributions", http.StatusFound)
}
