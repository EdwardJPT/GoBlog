package main

import (
	"log"
	"net/http"
	"os"

	admin_handler "blog/internal/admin/handlers"
	admin_middlewares "blog/internal/admin/middlewares"
	"blog/internal/database"
	"blog/internal/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load the environment variable
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env file: %v\n", err)
	}

	// Init the databse connection
	db, err := database.InitSQLite()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}
	defer db.Close()
	log.Printf("Success connecting to the database.\n")
	log.Printf("Foreign Keys %d\n", database.CheckFK(db))
	log.Printf("Journal Mode %s\n", database.CheckJournalMode(db))
	log.Printf("  Cache Size %d\n", database.CheckCacheSize(db))

	// Run the migration
	err = database.RunMigrations(db)
	if err != nil {
		log.Fatalf("DB Migration failed: %v\n", err)
	}
	log.Println("DB Migration success")

	// Init the cache connection
	dbRAM, err := database.InitSQLiteRAM()
	if err != nil {
		log.Fatalf("Failed to connect to RAM database: %v\n", err)
	}
	defer dbRAM.Close()

	// Set up the jwt for auth
	jwtSecret := os.Getenv("JWT_SECRET")
	jwt := jwtauth.New("HS256", []byte(jwtSecret), nil)

	// Init handlers by passing the db and jwt

	// ADMIN HANDLER
	adminHandler := admin_handler.NewAdminHandler(db, jwt)
	postsHandler := admin_handler.NewPostHandler(db)
	notesHandler := admin_handler.NewNoteHandler(db)
	bookmarksHandler := admin_handler.NewBookmarkHandler(db)
	projectsHandler := admin_handler.NewProjectHandler(db)
	contributionsHandler := admin_handler.NewContributionsHandler(db)
	selectedHandler := admin_handler.NewSelectedHandler(db)
	// VISITOR HANDLERS
	homeHandler := handlers.NewHomeHandler(db)
	aboutHandler := handlers.NewAboutHandler()
	writingsHandler := handlers.NewWritingsHandler(db)
	worksHandler := handlers.NewWorksHandler(db)
	notFoundHandler := handlers.NewNotFoundHandler()
	tagsHandler := handlers.NewTagsHandler(db)

	// Init the chi router
	r := chi.NewRouter()

	// Embed middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routers

	// VISITOR ROUTERS
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

	// Login and Dashboard router
	r.Route("/nimda", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Group(func(r chi.Router) {
			// Middleware that would redirect authenticated users to /nimda/dashboard
			r.Use(admin_middlewares.AuthenticatedRedirector(jwt))
			r.Get("/", adminHandler.AdminPage)
		})
		r.Post("/", adminHandler.SignIn)
		r.Get("/logout", adminHandler.SignOut)
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(admin_middlewares.Auth(jwt))
			r.Get("/dashboard", adminHandler.DashboardPage)
		})
	})

	// Content Management router
	r.Route("/nimda/posts", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", postsHandler.PostsList)
		r.Get("/components", postsHandler.PostsListComponent)
		r.Get("/new", postsHandler.PostsForm)
		r.Post("/", postsHandler.PostsCreate)
		r.Get("/{id}/edit", postsHandler.PostsEdit)
		r.Post("/{id}", postsHandler.PostsUpdate)
		r.Post("/{id}/delete", postsHandler.PostsDelete)
	})
	r.Route("/nimda/bookmarks", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", bookmarksHandler.BookmarksList)
		r.Get("/new", bookmarksHandler.BookmarksForm)
		r.Post("/", bookmarksHandler.BookmarksCreate)
		r.Get("/{id}/edit", bookmarksHandler.BookmarksEdit)
		r.Post("/{id}", bookmarksHandler.BookmarksUpdate)
		r.Post("/{id}/delete", bookmarksHandler.BookmarksDelete)
	})
	r.Route("/nimda/notes", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", notesHandler.NotesList)
		r.Get("/components", notesHandler.NotesListComponent)
		r.Get("/new", notesHandler.NotesForm)
		r.Post("/", notesHandler.NotesCreate)
		r.Get("/{id}/edit", notesHandler.NotesEdit)
		r.Post("/{id}", notesHandler.NotesUpdate)
		r.Post("/{id}/delete", notesHandler.NotesDelete)
	})
	r.Route("/nimda/projects", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", projectsHandler.ProjectsList)
		r.Get("/components", projectsHandler.ProjectsListComponent)
		r.Get("/new", projectsHandler.ProjectsForm)
		r.Post("/", projectsHandler.ProjectsCreate)
		r.Get("/{id}/edit", projectsHandler.ProjectsEdit)
		r.Post("/{id}", projectsHandler.ProjectsUpdate)
		r.Post("/{id}/delete", projectsHandler.ProjectsDelete)
	})
	r.Route("/nimda/os-contributions", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", contributionsHandler.ContributionsList)
		r.Get("/components", contributionsHandler.ContributionsListComponent)
		r.Get("/new", contributionsHandler.ContributionsForm)
		r.Post("/", contributionsHandler.ContributionsCreate)
		r.Get("/{id}/edit", contributionsHandler.ContributionsEdit)
		r.Post("/{id}", contributionsHandler.ContributionsUpdate)
		r.Post("/{id}/delete", contributionsHandler.ContributionsDelete)
	})

	// Content management about what shown in the home page and tags router
	r.Route("/nimda/selected-works", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", selectedHandler.WorksList)
		r.Get("/{id}/{type}", selectedHandler.AddWorks)
		r.Post("/tech-stack", selectedHandler.UpsertTechStack)
	})
	r.Route("/nimda/selected-writings", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", selectedHandler.WritingsList)
		r.Get("/{id}/{type}", selectedHandler.AddWritings)
	})
	// TODO:
	// Postponed /nimda/selected-experiences because I think it's better to
	// hard code it now
	r.Route("/nimda/experiences", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", adminHandler.DashboardPage)
	})
	r.Route("/nimda/tags", func(r chi.Router) {
		r.Use(admin_middlewares.IdMiddleware)
		r.Use(admin_middlewares.Auth(jwt))
		r.Get("/", adminHandler.DashboardPage)
		r.Get("/new", adminHandler.DashboardPage)
		r.Post("/", adminHandler.DashboardPage)
		r.Get("/{id}/edit", adminHandler.DashboardPage)
		r.Post("/{id}", adminHandler.DashboardPage)
		r.Post("/{id}/delete", adminHandler.DashboardPage)
		/*
			TODO:
			r.Route("/nimda/tags", func(r chi.Router) {
			»   r.Use(admin_middlewares.IdMiddleware)
			»   r.Use(admin_middlewares.Auth(jwt))
			»   r.Get("/", adminHandler.TagsList)
			»   r.Get("/{tag}", adminHandler.TagsShow)
			»   r.Get("/{tag}/edit", adminHandler.TagsEditForm)
			»   r.Post("/{tag}/edit", adminHandler.TagsUpdate)
			»   r.Post("/{tag}/delete", adminHandler.TagsDelete)
			»   r.Get("/{tag}/{table}", adminHandler.TagsShowByTable)
			})
		*/
	})

	// 404
	r.NotFound(notFoundHandler.NotFoundPage)

	// run the server
	PORT := "4000"
	log.Printf(" Server Port %s\n", PORT)
	http.ListenAndServe(":"+PORT, r)
}
