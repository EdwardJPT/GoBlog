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

type ProjectsProjectRepo interface {
	GetBySlug(slug string) (*models.Project, error)
}

type ProjectsContributionRepo interface {
	GetBySlug(slug string) (*models.Contribution, error)
}

type ProjectsWorkRepo interface {
	GetProjectsFeed(
		selectedTypes []string,
		encodedCursor string,
		isPrev bool,
		limit int,
	) ([]models.WorkItem, error)

	HasPublishedRowsAroundCursor(
		selectedTypes []string,
		firstTimestamp time.Time,
		lastTimestamp time.Time,
	) (hasPrev bool, hasNext bool, err error)
}

type WorksHandler struct {
	projects      ProjectsProjectRepo
	contributions ProjectsContributionRepo
	works         ProjectsWorkRepo
	templates     *template.Template
}

func NewWorksHandler(db *sql.DB) *WorksHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/public/projects/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/404.html"))
	return &WorksHandler{
		projects:      models.NewProjectRepository(db),
		contributions: models.NewContributionRepository(db),
		works:         models.NewWorkRepository(db),
		templates:     tmpl,
	}
}

type ProjectsPageData struct {
	ProjectsFeed []models.WorkItem
	Pagination   CursorPagination
}

func (h *WorksHandler) ProjectsPage(w http.ResponseWriter, r *http.Request) {
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

	projectsData, err := h.works.GetProjectsFeed(selectedTypes, cursor, isPrev, limit)
	if err != nil {
		slog.Error("Failed fetching projects feed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// If we queried going backwards (ASC), we must reverse the slice to maintain DESC order for the UI
	if isPrev {
		for i, j := 0, len(projectsData)-1; i < j; i, j = i+1, j-1 {
			projectsData[i], projectsData[j] = projectsData[j], projectsData[i]
		}
	}

	// Calculate Next and Prev cursors
	var nextCursorBase64, prevCursorBase64 string
	if len(projectsData) > 0 {
		firstTimestamp := projectsData[0].UpdatedAt
		lastTimestamp := projectsData[len(projectsData)-1].UpdatedAt

		hasPrev, hasNext, err := h.works.HasPublishedRowsAroundCursor(
			selectedTypes,
			firstTimestamp,
			lastTimestamp,
		)
		if err != nil {
			slog.Error("Failed fetching next and prev cursor of projects", "error", err)
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
		NextCursor: nextCursorBase64,
		PrevCursor: prevCursorBase64,
	}

	data := ProjectsPageData{
		ProjectsFeed: projectsData,
		Pagination:   pagination,
	}

	if err := h.templates.ExecuteTemplate(w, "projects", data); err != nil {
		slog.Error("Template handlers.ProjectsPage failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

type ProjectContent struct {
	Title       string
	Description template.HTML
	Repo_link   string
	Demo_link   string
	Other_links string
	Tags        string
	PublishedAt time.Time
}

type ContributionContent struct {
	Title               string
	Description         template.HTML
	Tags                string
	Link                string
	ContributionSummary template.HTML
	PublishedAt         time.Time
}

func (h *WorksHandler) Project(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	project, err := h.projects.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Error("Project not found error", "slug", slug)
			// Set the 404 header explicitly
			w.WriteHeader(http.StatusNotFound)
			// Render 404 directly here
			data := NotFoundData{ParentFeature: "projects"}
			if err := utils.RenderTemplate(w, h.templates, "404", data); err != nil {
				slog.Error("Template 404 failed in projects_handlers", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.Error("Failed fetching project from database", "slug", slug, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	projectContent := ProjectContent{
		Title:       project.Name,
		Description: utils.RenderMarkdown(project.Description),
		Repo_link:   project.RepoLink,
		Demo_link:   project.DemoLink,
		Other_links: project.OtherLinks,
		Tags:        project.Tags,
		PublishedAt: project.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_project_layout", projectContent); err != nil {
		slog.Error("Template handlers.Project failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *WorksHandler) OpenSourceContribution(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	contribution, err := h.contributions.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Error("Contribution not found error", "slug", slug)
			// Set the 404 header explicitly
			w.WriteHeader(http.StatusNotFound)
			// Render 404 directly here
			data := NotFoundData{ParentFeature: "projects"}
			if err := utils.RenderTemplate(w, h.templates, "404", data); err != nil {
				slog.Error("Template 404 failed in projects_handlers", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.Error("Failed fetching contribution from database", "slug", slug, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	contributionContent := ContributionContent{
		Title:               contribution.ProjectName,
		Description:         utils.RenderMarkdown(contribution.Description),
		Tags:                contribution.Tags,
		Link:                contribution.Link,
		ContributionSummary: template.HTML(contribution.ContributionSummary),
		PublishedAt:         contribution.UpdatedAt,
	}

	if err := h.templates.ExecuteTemplate(w, "published_contribution_layout", contributionContent); err != nil {
		slog.Error("Template handlers.OpenSourceContribution failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
