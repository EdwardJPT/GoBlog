package main

import (
	"log/slog"
	"net/http"
	"os"

	"blog/internal/database"
	admin_handlers "blog/internal/handlers/admin"
	handlers "blog/internal/handlers/public"
	"blog/internal/logger"
	"blog/internal/middlewares"
	admin_middlewares "blog/internal/middlewares/admin"
	"blog/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Init the logger
	logger.Init()

	// Load the environment variable
	err := godotenv.Load()
	if err != nil {
		slog.Error("Failed to load .env file, shutting down", "error", err)
		os.Exit(1)
	}

	// Init the database connection
	db, err := database.InitSQLite()
	if err != nil {
		slog.Error("Failed to initialize database, shutting down", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Success connecting to the database")
	slog.Info("DB config info", "Foreign Keys", database.CheckFK(db))
	slog.Info("DB config info", "Journal Mode", database.CheckJournalMode(db))
	slog.Info("DB config info", "   Sync Mode", database.CheckSynchronous(db))
	slog.Info("DB config info", "  Cache Size", database.CheckCacheSize(db))

	// Run the migration
	err = database.RunMigrations(db)
	if err != nil {
		slog.Error("DB Migration failed, shutting down", "error", err)
		os.Exit(1)
	}
	slog.Info("DB Migration success")

	// Init the cache connection
	dbRAM, err := database.InitSQLiteRAM()
	if err != nil {
		slog.Error("Failed to initialize RAM database, shutting down", "error", err)
		os.Exit(1)
	}
	defer dbRAM.Close()

	// Set up the jwt for auth
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET environment variable is not set, shutting down")
		os.Exit(1)
	}
	jwt := jwtauth.New("HS256", []byte(jwtSecret), nil)
	// Dummy hash
	hash := utils.GenerateDummyHash()

	// Init handlers by passing the db and jwt
	// PUBLIC HANDLERS
	homeHandler := handlers.NewHomeHandler(db)
	aboutHandler := handlers.NewAboutHandler()
	writingsHandler := handlers.NewWritingsHandler(db)
	worksHandler := handlers.NewWorksHandler(db)
	notFoundHandler := handlers.NewNotFoundHandler()
	tagsHandler := handlers.NewTagsHandler(db)
	// ADMIN HANDLERS
	authHandler := admin_handlers.NewAdminHandler(db, dbRAM, jwt, hash)
	postsHandler := admin_handlers.NewPostHandler(db)
	notesHandler := admin_handlers.NewNoteHandler(db)
	bookmarksHandler := admin_handlers.NewBookmarkHandler(db)
	projectsHandler := admin_handlers.NewProjectHandler(db)
	contributionsHandler := admin_handlers.NewContributionsHandler(db)
	selectedHandler := admin_handlers.NewSelectedHandler(db)
	adminTagsHandler := admin_handlers.NewTagHandler(db)

	// Init the chi router
	r := chi.NewRouter()

	// Embed middlewares
	r.Use(middleware.RequestID)
	r.Use(middlewares.ResponseRequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routers

	// PUBLIC ROUTERS
	r.Get("/", homeHandler.HomePage)
	r.Get("/about", aboutHandler.AboutPage)
	// Blog router
	r.Group(func(r chi.Router) {
		// Parent page
		r.Get("/blog", writingsHandler.BlogPage)
		// Post
		// To handle dummy post url: /post/#, don't delete the trailing "/"
		r.Get("/post/", writingsHandler.BlogPage)
		r.Get("/post/{slug}", writingsHandler.Post)
		// Note
		// To handle dummy post url: /note/#, don't delete the trailing "/"
		r.Get("/note/", writingsHandler.BlogPage)
		r.Get("/note/{slug}", writingsHandler.Note)
		// Bookmark
		// To handle dummy post url: /bookmark/#, don't delete the trailing "/"
		r.Get("/bookmark/", writingsHandler.BlogPage)
		r.Get("/bookmark/{slug}", writingsHandler.Bookmark)
	})
	// Projects router
	r.Group(func(r chi.Router) {
		// Parent page
		r.Get("/projects", worksHandler.ProjectsPage)
		// Project
		// To handle dummy post url: /project/#, don't delete the trailing "/"
		r.Get("/project/", worksHandler.ProjectsPage)
		r.Get("/project/{slug}", worksHandler.Project)
		// Open Source Contribution
		// To handle dummy post url: /open-source-contribution/#, don't delete
		// the trailing "/"
		r.Get("/open-source-contribution/", worksHandler.ProjectsPage)
		r.Get("/open-source-contribution/{slug}", worksHandler.OpenSourceContribution)
	})
	// Tags router
	r.Group(func(r chi.Router) {
		r.Get("/tags", tagsHandler.TagsList)
		r.Get("/tag/{tag}", tagsHandler.Tag)
	})

	// ADMIN ROUTERS
	r.Route("/nimda", func(r chi.Router) {
		// Global admin middleware
		r.Use(admin_middlewares.IdMiddleware)

		// Auth router
		r.Group(func(r chi.Router) {
			// Redirects already authenticated users away from the login page
			r.Use(admin_middlewares.AuthenticatedRedirector(jwt))
			r.Get("/", authHandler.AdminPage)
		})
		r.Post("/", authHandler.SignIn)
		r.Get("/logout", authHandler.SignOut)

		// Protected admin router
		r.Group(func(r chi.Router) {
			r.Use(admin_middlewares.Auth(jwt))

			// Dashboard
			r.Get("/dashboard", authHandler.DashboardPage)

			// Content Management
			r.Route("/posts", func(r chi.Router) {
				r.Get("/", postsHandler.PostsList)
				r.Get("/components", postsHandler.PostsListComponent)
				r.Get("/new", postsHandler.PostsForm)
				r.Post("/", postsHandler.PostsCreate)
				r.Get("/{id}/edit", postsHandler.PostsEdit)
				r.Get("/{id}/preview", postsHandler.PostsPreview)
				r.Post("/{id}", postsHandler.PostsUpdate)
				r.Post("/{id}/delete", postsHandler.PostsDelete)
			})

			r.Route("/bookmarks", func(r chi.Router) {
				r.Get("/", bookmarksHandler.BookmarksList)
				r.Get("/new", bookmarksHandler.BookmarksForm)
				r.Post("/", bookmarksHandler.BookmarksCreate)
				r.Get("/{id}/edit", bookmarksHandler.BookmarksEdit)
				r.Get("/{id}/preview", bookmarksHandler.BookmarksPreview)
				r.Post("/{id}", bookmarksHandler.BookmarksUpdate)
				r.Post("/{id}/delete", bookmarksHandler.BookmarksDelete)
			})

			r.Route("/notes", func(r chi.Router) {
				r.Get("/", notesHandler.NotesList)
				r.Get("/components", notesHandler.NotesListComponent)
				r.Get("/new", notesHandler.NotesForm)
				r.Post("/", notesHandler.NotesCreate)
				r.Get("/{id}/edit", notesHandler.NotesEdit)
				r.Get("/{id}/preview", notesHandler.NotesPreview)
				r.Post("/{id}", notesHandler.NotesUpdate)
				r.Post("/{id}/delete", notesHandler.NotesDelete)
			})

			r.Route("/projects", func(r chi.Router) {
				r.Get("/", projectsHandler.ProjectsList)
				r.Get("/components", projectsHandler.ProjectsListComponent)
				r.Get("/new", projectsHandler.ProjectsForm)
				r.Post("/", projectsHandler.ProjectsCreate)
				r.Get("/{id}/edit", projectsHandler.ProjectsEdit)
				r.Get("/{id}/preview", projectsHandler.ProjectsPreview)
				r.Post("/{id}", projectsHandler.ProjectsUpdate)
				r.Post("/{id}/delete", projectsHandler.ProjectsDelete)
			})

			r.Route("/os-contributions", func(r chi.Router) {
				r.Get("/", contributionsHandler.ContributionsList)
				r.Get("/components", contributionsHandler.ContributionsListComponent)
				r.Get("/new", contributionsHandler.ContributionsForm)
				r.Post("/", contributionsHandler.ContributionsCreate)
				r.Get("/{id}/edit", contributionsHandler.ContributionsEdit)
				r.Get("/{id}/preview", contributionsHandler.ContributionsPreview)
				r.Post("/{id}", contributionsHandler.ContributionsUpdate)
				r.Post("/{id}/delete", contributionsHandler.ContributionsDelete)
			})

			// Selected Works & Writings
			r.Route("/selected-works", func(r chi.Router) {
				r.Get("/", selectedHandler.WorksList)
				r.Get("/{id}/{type}", selectedHandler.AddWorks)
				r.Post("/tech-stack", selectedHandler.UpsertTechStack)
			})

			r.Route("/selected-writings", func(r chi.Router) {
				r.Get("/", selectedHandler.WritingsList)
				r.Get("/{id}/{type}", selectedHandler.AddWritings)
			})

			// TODO: Postponed /nimda/selected-experiences (hardcoded for now)
			r.Route("/experiences", func(r chi.Router) {
				r.Get("/", authHandler.DashboardPage)
			})

			// Tags
			r.Route("/tags", func(r chi.Router) {
				r.Get("/", adminTagsHandler.TagsList)
				r.Get("/{name}/edit", adminTagsHandler.EditTagPage)
				r.Post("/{name}", adminTagsHandler.UpdateTag)
				r.Post("/{name}/delete", adminTagsHandler.DeleteTag)
			})
		})
	})

	// 404
	r.NotFound(notFoundHandler.NotFoundPage)

	// run the server
	PORT := "4000"
	slog.Info("Server started", "port", PORT)
	err = http.ListenAndServe(":"+PORT, r)
	if err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
