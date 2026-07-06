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
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	projects, err := h.projects.GetAll(filter, status)
	if err != nil {
		log.Printf("Error fetching all projects: %v", err)
		log.Printf("filter: %s - status: %s", filter, status)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
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
		log.Printf("Error fetching all projects: %v", err)
		log.Printf("filter: %s", filter)
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	data := ProjectListComponent{Projects: projects}

	err = h.templates.ExecuteTemplate(w, "projects-list-component", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ProjectHandler) ProjectsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	nanoId, err := gonanoid.New(10)
	if err != nil {
		return
	}

	project := &models.Project{
		Name:                r.FormValue("name"),
		Slug:                models.GenerateSlug(r.FormValue("name")) + "-" + nanoId,
		Description:         r.FormValue("description"),
		HomePageDescription: r.FormValue("home_page_description"),
		RepoLink:            r.FormValue("repo_link"),
		DemoLink:            r.FormValue("demo_link"),
		OtherLinks:          r.FormValue("other_links"),
		Tags:                strings.ReplaceAll(r.FormValue("tags"), " ", ""),
		Status:              r.FormValue("status"),
	}

	if err := h.projects.Create(project); err != nil {
		log.Printf("Error creating project: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/projects", http.StatusFound)
}

func (h *ProjectHandler) ProjectsEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	project, err := h.projects.GetByID(id)
	if err != nil {
		log.Printf("Error fetching project by ID: %v", err)
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
		http.Error(w, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *ProjectHandler) ProjectsPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	project, err := h.projects.GetByID(id)
	if err != nil {
		log.Printf("Error fetching project by ID: %v", err)
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
		log.Println("handlers.Project error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *ProjectHandler) ProjectsUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	project, err := h.projects.GetByID(id)
	if err != nil {
		log.Printf("Error fetching project by ID: %v", err)
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

	if err := h.projects.Update(project); err != nil {
		log.Printf("Error updating project: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/projects", http.StatusFound)
}

func (h *ProjectHandler) ProjectsDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	if err := h.projects.Delete(id); err != nil {
		log.Printf("Error deleting project by ID: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/projects", http.StatusFound)
}
