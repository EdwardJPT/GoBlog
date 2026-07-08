package admin_handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	admin_middlewares "blog/internal/middlewares/admin"
	"blog/internal/models"
	admin_services "blog/internal/services/admin"
	"blog/internal/utils"

	"github.com/go-chi/jwtauth/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	jwt           *jwtauth.JWTAuth
	dummyHash     []byte
	auth          *models.AdminRepository
	posts         *models.PostRepository
	bookmarks     *models.BookmarkRepository
	notes         *models.NoteRepository
	projects      *models.ProjectRepository
	contributions *models.ContributionRepository
	stats         *admin_services.StatsService
	templates     *template.Template
}

func NewAdminHandler(
	db *sql.DB,
	dbRAM *sql.DB,
	jwt *jwtauth.JWTAuth,
	dummyHash []byte,
) *AdminHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/admin_login.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/admin_dashboard_overview.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &AdminHandler{
		jwt:           jwt,
		dummyHash:     dummyHash,
		auth:          models.NewAdminRepository(db, dbRAM),
		posts:         models.NewPostRepository(db),
		bookmarks:     models.NewBookmarkRepository(db),
		notes:         models.NewNoteRepository(db),
		projects:      models.NewProjectRepository(db),
		contributions: models.NewContributionRepository(db),
		stats:         admin_services.NewStatsService(db),
		templates:     tmpl,
	}
}

type AuthResponse struct{ Message string }

func (h *AdminHandler) AdminPage(w http.ResponseWriter, r *http.Request) {
	slog.Info("=== LoginPage handler called ===")
	id := r.Context().Value(admin_middlewares.IdKey).(admin_middlewares.RequestId)
	slog.Info("Admin Info", "IP", id.IP, "User-Agent", id.UserAgent)

	if h.auth.IsBlocked(id.IP) {
		slog.Error("Access Denied: IP Temporarily Locked", "IP", id.IP)
		http.Error(w, "Access Denied: IP Temporarily Locked", http.StatusTooManyRequests)
		return
	}

	err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{Message: ""})
	if err != nil {
		slog.Error("Template handlers.AdminPage failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *AdminHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	slog.Info("=== Admin trying to Sign In ===")
	id := r.Context().Value(admin_middlewares.IdKey).(admin_middlewares.RequestId)
	slog.Info("Admin Info", "IP", id.IP, "User-Agent", id.UserAgent)

	if h.auth.IsBlocked(id.IP) {
		slog.Error("Access Denied: IP Temporarily Locked", "IP", id.IP)
		http.Error(w, "Access Denied: IP Temporarily Locked", http.StatusTooManyRequests)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		slog.Warn("handlers.SignIn received invalid form data", "error", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Construct the form data into a struct
	formData := models.SignInData{
		Username: strings.ToLower(r.FormValue("username")),
		Password: r.FormValue("password"),
		Remember: r.FormValue("remember") == "on",
	}

	// Check if username valid
	admin, err := h.auth.GetByUsername(formData.Username)
	if err != nil {
		slog.Error("Failed query admin username", "error", err, "username", formData.Username)

		h.auth.RecordFailure(id.IP)

		// Still run the bcrypt math against dummy hash so there would be no
		// difference of delay time between invalid username and invalid password.
		// Without it, invalid username would take 2ms, and invalid password
		// would take 100ms
		bcrypt.CompareHashAndPassword(h.dummyHash, []byte(formData.Password))

		if err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{
			Message: "Invalid credentials",
		}); err != nil {
			slog.Error("Template handlers.SignIn failed", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Check the password
	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(formData.Password))
	if err != nil {
		slog.Error("Bcrypt compare failed", "error", err)

		h.auth.RecordFailure(id.IP)

		if err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{
			Message: "Invalid credentials",
		}); err != nil {
			slog.Error("Template handlers.SignIn failed", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// On Success, clear previous failed attempts to reset their window
	h.auth.ClearFailures(id.IP)

	// Set the cookie for auth
	expDays := 1 // the expired time of the jwt
	if formData.Remember {
		expDays = 30
	}

	claims := map[string]any{
		"username": admin.Username,
		"admin_id": fmt.Sprintf("%d", admin.ID),
		"exp":      time.Now().Add(time.Duration(expDays) * 24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	_, tokenString, err := h.jwt.Encode(claims)
	if err != nil {
		slog.Error("Failed encoding JWT", "error", err)
		if err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{
			Message: "Invalid credentials",
		}); err != nil {
			slog.Error("Template handlers.SignIn failed", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Update the last login in database
	if err = h.auth.UpdateLastLogin(admin.ID); err != nil {
		slog.Error("Failed Update last login", "error", err, "ID", admin.ID)
	}

	// Set the cookie to the user and redirect
	utils.SetTokenCookie(w, tokenString, formData.Remember)
	http.Redirect(w, r, "/nimda/dashboard", http.StatusSeeOther)
}

func (h *AdminHandler) SignOut(w http.ResponseWriter, r *http.Request) {
	utils.ClearTokenCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type DashboardData struct {
	*admin_services.StatsCardData
	admin_services.RecentData
}

func (h *AdminHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	slog.Info("=== DashboardPage handler called ===")
	id := r.Context().Value(admin_middlewares.IdKey).(admin_middlewares.RequestId)
	slog.Info("Admin Info", "IP", id.IP, "User-Agent", id.UserAgent)

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		slog.Error("Failed to load dashboard data", "error", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	recentAmount := 4

	postRecent, err := h.posts.GetRecent(recentAmount)
	if err != nil {
		slog.Error("Failed fetching recent posts", "error", err)
		http.Error(w, "Failed to load recent posts", http.StatusInternalServerError)
		return
	}

	bookmarkRecent, err := h.bookmarks.GetRecent(recentAmount)
	if err != nil {
		slog.Error("Failed fetching recent bookmarks", "error", err)
		http.Error(w, "Failed to load recent bookmarks", http.StatusInternalServerError)
		return
	}

	noteRecent, err := h.notes.GetRecent(recentAmount)
	if err != nil {
		slog.Error("Failed fetching recent notes", "error", err)
		http.Error(w, "Failed to load recent notes", http.StatusInternalServerError)
		return
	}

	projectRecent, err := h.projects.GetRecent(recentAmount)
	if err != nil {
		slog.Error("Failed fetching recent projects", "error", err)
		http.Error(w, "Failed to load recent project", http.StatusInternalServerError)
		return
	}

	opensourceRecent, err := h.contributions.GetRecent(recentAmount)
	if err != nil {
		slog.Error("Failed fetching recent open source contributions", "error", err)
		http.Error(w, "Failed to load recent open source", http.StatusInternalServerError)
		return
	}

	data := DashboardData{
		// Statistics
		StatsCardData: statsData,
		// Recent Posts
		RecentData: admin_services.RecentData{
			RecentPosts: postRecent,
			// Recent Bookmarks
			RecentBookmarks: bookmarkRecent,
			// Recent Notes
			RecentNotes: noteRecent,
			// Recent Projects
			RecentProjects: projectRecent,
			// Recent Open Source Contributions
			RecentOpenSource: opensourceRecent,
		},
	}

	err = h.templates.ExecuteTemplate(w, "admin_dashboard_overview", data)
	if err != nil {
		slog.Error("Template handlers.DashboardPage failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
