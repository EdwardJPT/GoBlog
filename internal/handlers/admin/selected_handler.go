package admin_handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"blog/internal/models"
	admin_services "blog/internal/services/admin"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
)

type SelectedHandler struct {
	selected  *models.SelectedRepository
	stats     *admin_services.StatsService
	templates *template.Template
}

func NewSelectedHandler(db *sql.DB) *SelectedHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/selected-writings/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/selected-works/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &SelectedHandler{
		selected:  models.NewSelectedRepository(db),
		stats:     admin_services.NewStatsService(db),
		templates: tmpl,
	}
}

type WritingsListData struct {
	*admin_services.StatsCardData
	Writings    []*models.SelectedWritingsValue
	Filter      string
	SectionType string
}

func (h *SelectedHandler) WritingsList(w http.ResponseWriter, r *http.Request) {
	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	selectedWritings, err := h.selected.GetAllWritings()
	if err != nil {
		slog.Error("Failed fetching all selected writings", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := WritingsListData{
		StatsCardData: statsData,
		Writings:      selectedWritings,
	}

	err = h.templates.ExecuteTemplate(w, "writings-layout", data)
	if err != nil {
		slog.Error("Template handlers.WritingsList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *SelectedHandler) AddWritings(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.AddWritings received invalid post ID", "error", err)
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	writingsType := chi.URLParam(r, "type")

	err = h.selected.InsertWriting(writingsType, id)
	if err != nil {
		slog.Error("Failed to insert selected writings", "error", err, "ID", id, "type", writingsType)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/selected-writings", http.StatusFound)
}

type WorksListData struct {
	*admin_services.StatsCardData
	Works      []*models.SelectedWorksValue
	TechStacks map[int64][]string
}

func (h *SelectedHandler) WorksList(w http.ResponseWriter, r *http.Request) {
	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	selectedWorks, err := h.selected.GetAllWorks()
	if err != nil {
		slog.Error("Failed fetching all selected works", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	techStacks, err := h.selected.GetAllTechStackLayouts()
	if err != nil {
		slog.Error("Failed to load tech stacks", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := WorksListData{
		StatsCardData: statsData,
		Works:         selectedWorks,
		TechStacks:    techStacks,
	}

	err = h.templates.ExecuteTemplate(w, "works-layout", data)
	if err != nil {
		slog.Error("Template handlers.WorksList failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *SelectedHandler) AddWorks(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Warn("handlers.AddWorks received invalid work ID", "error", err)
		http.Error(w, "Invalid work ID", http.StatusBadRequest)
		return
	}

	worksType := chi.URLParam(r, "type")

	err = h.selected.InsertWork(worksType, id)
	if err != nil {
		slog.Error("Failed to insert selected work", "error", err, "ID", id, "type", worksType)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nimda/selected-works", http.StatusFound)
}

type TechStackPayload struct {
	WorkID       int64      `json:"work_id"`
	Type         string     `json:"type"`
	SkillsLayout [][]string `json:"skills_layout"`
}

func (h *SelectedHandler) UpsertTechStack(w http.ResponseWriter, r *http.Request) {
	var payload TechStackPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		slog.Warn("handlers.UpsertTechStack received invalid JSON payload", "error", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	skillsJSONBytes, err := json.Marshal(payload.SkillsLayout)
	if err != nil {
		slog.Error("Failed to process skills layout", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	skillsJSONString := string(skillsJSONBytes)

	err = h.selected.UpsertTechStackLayout(payload.WorkID, skillsJSONString)
	if err != nil {
		slog.Error("Failed to save tech stack", "error", err, "work_id", payload.WorkID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
