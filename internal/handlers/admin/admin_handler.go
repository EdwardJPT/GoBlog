package admin_handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	admin_middlewares "blog/internal/middlewares/admin"
	"blog/internal/models"
	admin_service "blog/internal/service/admin"
	"blog/internal/utils"

	"github.com/go-chi/jwtauth/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	jwt           *jwtauth.JWTAuth
	auth          *models.AdminRepository
	posts         *models.PostRepository
	bookmarks     *models.BookmarkRepository
	notes         *models.NoteRepository
	projects      *models.ProjectRepository
	contributions *models.ContributionRepository
	stats         *admin_service.StatsService
	templates     *template.Template
}

func NewAdminHandler(db *sql.DB, jwt *jwtauth.JWTAuth) *AdminHandler {
	tmpl := template.New("").Funcs(utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/admin_login.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/admin_dashboard_overview.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/admin/components/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/shared/components/*.html"))
	return &AdminHandler{
		jwt:           jwt,
		auth:          models.NewAdminRepository(db),
		posts:         models.NewPostRepository(db),
		bookmarks:     models.NewBookmarkRepository(db),
		notes:         models.NewNoteRepository(db),
		projects:      models.NewProjectRepository(db),
		contributions: models.NewContributionRepository(db),
		stats:         admin_service.NewStatsService(db),
		templates:     tmpl,
	}
}

type AuthResponse struct{ Message string }

func (h *AdminHandler) AdminPage(w http.ResponseWriter, r *http.Request) {
	log.Println("=== LoginPage handler called ===")
	id := r.Context().Value(admin_middlewares.IdKey).(admin_middlewares.RequestId)
	log.Println("IP		:", id.IP)
	log.Println("User-Agent	:", id.UserAgent)

	err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{Message: ""})
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AdminHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	log.Println("=== Admin trying to Sign In ===")
	id := r.Context().Value(admin_middlewares.IdKey).(admin_middlewares.RequestId)
	log.Println("IP		:", id.IP)
	log.Println("User-Agent	:", id.UserAgent)

	// Parse form data
	r.ParseForm()
	// Construct the form data into a struct
	formData := models.SignInData{
		Username: strings.ToLower(r.FormValue("username")),
		Password: r.FormValue("password"),
		Remember: r.FormValue("remember") == "on",
	}

	// Check if username valid
	admin, err := h.auth.GetByUsername(formData.Username)
	if err != nil {
		log.Println("Query admin username error:", err)
		err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{Message: "Invalid credentials"})
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Check the password
	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(formData.Password))
	if err != nil {
		log.Println("Bcrypt compare error:", err)
		err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{Message: "Invalid credentials"})
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

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
		log.Println("JWT encode error:", err)
		err := h.templates.ExecuteTemplate(w, "admin_login", AuthResponse{Message: "Invalid credentials"})
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Update the last login in database
	if err = h.auth.UpdateLastLogin(admin.ID); err != nil {
		log.Println("Update last login error:", admin.ID, err)
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
	*admin_service.StatsCardData
	admin_service.RecentData
}

func (h *AdminHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	log.Println("=== DashboardPage handler called ===")
	id := r.Context().Value(admin_middlewares.IdKey).(admin_middlewares.RequestId)
	log.Println("IP		:", id.IP)
	log.Println("User-Agent	:", id.UserAgent)

	statsData, err := h.stats.FetchStatsData()
	if err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	recentAmount := 4

	postRecent, err := h.posts.GetRecent(recentAmount)
	if err != nil {
		log.Printf("Error fetching recent posts: %v", err)
		http.Error(w, "Failed to load recent posts", http.StatusInternalServerError)
		return
	}

	bookmarkRecent, err := h.bookmarks.GetRecent(recentAmount)
	if err != nil {
		log.Printf("Error fetching recent bookmarks: %v", err)
		http.Error(w, "Failed to load recent bookmarks", http.StatusInternalServerError)
		return
	}

	noteRecent, err := h.notes.GetRecent(recentAmount)
	if err != nil {
		log.Printf("Error fetching recent notes: %v", err)
		http.Error(w, "Failed to load recent notes", http.StatusInternalServerError)
		return
	}

	projectRecent, err := h.projects.GetRecent(recentAmount)
	if err != nil {
		log.Printf("Error fetching recent project: %v", err)
		http.Error(w, "Failed to load recent project", http.StatusInternalServerError)
		return
	}

	opensourceRecent, err := h.contributions.GetRecent(recentAmount)
	if err != nil {
		log.Printf("Error fetching recent open source: %v", err)
		http.Error(w, "Failed to load recent open source", http.StatusInternalServerError)
		return
	}

	data := DashboardData{
		// Statistics
		StatsCardData: statsData,
		// Recent Posts
		RecentData: admin_service.RecentData{
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
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
