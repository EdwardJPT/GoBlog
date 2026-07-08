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

type ProjectHandler struct {
	projects  *models.ProjectRepository
	stats     *admin_services.StatsService
	templates *template.Template
}

func NewProjectHandler(db *sql.DB) *ProjectHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/projects/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/projects/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &ProjectHandler{
		projects:  models.NewProjectRepository(db),
		stats:     admin_services.NewStatsService(db),
		templates: tmpl,
	}
}

type ProjectListData struct {
	*admin_services.StatsCardData
	Filter      string
	Status      string
	Projects    []*models.Project
	SectionType string
}

// Projects handlers
func (h *ProjectHandler) ProjectsList(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	projects, err := h.projects.GetAll(filter, status)
	if err != nil {
		slog.Error("Failed fetching all projects", "error", err)
		slog.Error("Query parameters", "filter", filter, "status", status)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := ProjectListData{
		StatsCardData: statsData,
		Projects:      projects,
		Filter:        filter,
		Status:        status,
		SectionType:   "projects-list",
	}

	err = h.templates.ExecuteTemplate(w, "projects-layout", data)
	if err != nil {
		slog.Error("Template handlers.ProjectsList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

type ProjectListComponent struct {
	Projects []*models.Project
}

func (h *ProjectHandler) ProjectsListComponent(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")

	projects, err := h.projects.GetAll(filter, "published")
	if err != nil {
		slog.Error("Failed fetching all projects", "error", err)
		slog.Error("Query parameters", "filter", filter)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := ProjectListComponent{Projects: projects}

	err = h.templates.ExecuteTemplate(w, "projects-list-component", data)
	if err != nil {
		slog.Error("Template handlers.ProjectsListComponent failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *ProjectHandler) ProjectsForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Project": &models.Project{
			Status: "draft",
		},
		"SectionType": "projects-form",
	}

	err := h.templates.ExecuteTemplate(w, "projects-layout", data)
	if err != nil {
		slog.Error("Template handlers.ProjectsForm failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *ProjectHandler) ProjectsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.ProjectsCreate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	homePageDescription := r.FormValue("home_page_description")
	repoLink := r.FormValue("repo_link")
	demoLink := r.FormValue("demo_link")
	otherLinks := r.FormValue("other_links")
	tags := strings.ReplaceAll(r.FormValue("tags"), " ", "")
	status := r.FormValue("status")
	slug := r.FormValue("slug")

	if slug == "" {
		slug = models.GenerateSlug(name)
	}
	nanoId, err := gonanoid.New(10)
	if err != nil {
		slog.Error("Failed to generate nano ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	slug += "-" + nanoId

	project := &models.Project{
		Name:                name,
		Slug:                slug,
		Description:         description,
		HomePageDescription: homePageDescription,
		RepoLink:            repoLink,
		DemoLink:            demoLink,
		OtherLinks:          otherLinks,
		Tags:                tags,
		Status:              status,
	}

	if err := h.projects.Create(project); err != nil {
		slog.Error("Failed creating project", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/projects", http.StatusFound)
}

func (h *ProjectHandler) ProjectsEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ProjectsEdit received invalid project ID")
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := h.projects.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching project by ID", "error", err, "ID", id)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Project":     project,
		"ID":          project.ID,
		"IsEdit":      true,
		"SectionType": "projects-form",
	}

	err = h.templates.ExecuteTemplate(w, "projects-layout", data)
	if err != nil {
		slog.Error("Template handlers.ProjectsEdit failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *ProjectHandler) ProjectsPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ProjectsPreview received invalid project ID")
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := h.projects.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching project by ID", "error", err, "ID", id)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	projectContent := handlers.ProjectContent{
		Title:       project.Name,
		Description: utils.RenderMarkdown(project.Description),
		Repo_link:   project.RepoLink,
		Demo_link:   project.DemoLink,
		Other_links: project.OtherLinks,
		Tags:        project.Tags,
		PublishedAt: project.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_project_layout", projectContent); err != nil {
		slog.Error("Template handlers.ProjectsPreview failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *ProjectHandler) ProjectsUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ProjectsUpdate received invalid project ID")
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.ProjectsUpdate received invalid form data")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	project, err := h.projects.GetByID(id)
	if err != nil {
		slog.Error("Failed fetching project by ID", "error", err, "ID", id)
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	project.Name = r.FormValue("name")
	project.Description = r.FormValue("description")
	project.HomePageDescription = r.FormValue("home_page_description")
	project.RepoLink = r.FormValue("repo_link")
	project.DemoLink = r.FormValue("demo_link")
	project.OtherLinks = r.FormValue("other_links")
	project.Tags = strings.ReplaceAll(r.FormValue("tags"), " ", "")
	project.Status = r.FormValue("status")
	project.Slug = r.FormValue("slug")

	if project.Slug == "" {
		project.Slug = models.GenerateSlug(project.Name)
		nanoId, err := gonanoid.New(10)
		if err != nil {
			slog.Error("Failed to generate nano ID", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		project.Slug += "-" + nanoId
	}

	if err := h.projects.Update(project); err != nil {
		slog.Error("Failed updating project", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/projects", http.StatusFound)
}

func (h *ProjectHandler) ProjectsDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.ProjectsDelete received invalid project ID")
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	if err := h.projects.Delete(id); err != nil {
		slog.Error("Failed deleting project by ID", "error", err, "ID", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/projects", http.StatusFound)
}
