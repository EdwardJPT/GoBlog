package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"slices"
	"sort"
	"time"

	admin_models "blog/internal/admin/models"
	admin_utils "blog/internal/admin/utils"
	"blog/internal/models"
)

type HomeHandler struct {
	posts         *models.PostRepository
	bookmarks     *models.BookmarkRepository
	notes         *models.NoteRepository
	projects      *models.ProjectRepository
	contributions *models.ContributionRepository
	selected      *admin_models.SelectedRepository
	templates     *template.Template
}

func NewHomeHandler(db *sql.DB) *HomeHandler {
	tmpl := template.New("").Funcs(admin_utils.RegisterTemplateFuncs())
	tmpl = template.Must(tmpl.ParseGlob("web/templates/index.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/components/*.html"))
	return &HomeHandler{
		posts:         models.NewPostRepository(db),
		bookmarks:     models.NewBookmarkRepository(db),
		notes:         models.NewNoteRepository(db),
		projects:      models.NewProjectRepository(db),
		contributions: models.NewContributionRepository(db),
		selected:      admin_models.NewSelectedRepository(db),
		templates:     tmpl,
	}
}

type FeedItem struct {
	Type       string
	ParsedDate time.Time // Used for sorting in Go

	Post         *models.Post
	Project      *models.Project
	Note         *models.Note
	Bookmark     *models.Bookmark
	Contribution *models.Contribution
}

type HomePageData struct {
	Highlighted      []FeedItem // Just Post, Project, and Contribution, not including Note and Bookmark
	Feed             []FeedItem // Include all types, including Note and Bookmark
	SelectedWorks    []*admin_models.SelectedWorksValue
	TechStacks       map[int64][]string
	SelectedWritings []*admin_models.SelectedWritingsValue
}

func (h *HomeHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	// To store the result of the query
	groups := map[string][]FeedItem{
		"post":         []FeedItem{},
		"bookmark":     []FeedItem{},
		"note":         []FeedItem{},
		"project":      []FeedItem{},
		"contribution": []FeedItem{},
	}

	// TODO: move this to middleware
	log.Println(r.Header.Get("Referer"))

	// QUERY THE DATA FROM DATABASE
	queryAmount := 8

	postsPublished, err := h.posts.GetRecentPublished(queryAmount)
	if err != nil {
		log.Printf("Error fetching recent posts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lenPublished := len(postsPublished)
	for i := 0; i < lenPublished; i++ {
		p := postsPublished[i]
		groups["post"] = append(
			groups["post"],
			FeedItem{Type: "post", ParsedDate: p.UpdatedAt, Post: p},
		)
	}

	bookmarksPublished, err := h.bookmarks.GetRecentPublished(queryAmount)
	if err != nil {
		log.Printf("Error fetching recent bookmarks: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lenPublished = len(bookmarksPublished)
	for i := 0; i < lenPublished; i++ {
		b := bookmarksPublished[i]
		groups["bookmark"] = append(
			groups["bookmark"],
			FeedItem{Type: "bookmark", ParsedDate: b.UpdatedAt, Bookmark: b},
		)
	}

	notesPublished, err := h.notes.GetRecentPublished(queryAmount)
	if err != nil {
		log.Printf("Error fetching recent notes: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lenPublished = len(notesPublished)
	for i := 0; i < lenPublished; i++ {
		n := notesPublished[i]
		groups["note"] = append(
			groups["note"],
			FeedItem{Type: "note", ParsedDate: n.UpdatedAt, Note: n},
		)
	}

	projectsPublished, err := h.projects.GetRecentPublished(queryAmount)
	if err != nil {
		log.Printf("Error fetching recent project: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lenPublished = len(projectsPublished)
	for i := 0; i < lenPublished; i++ {
		p := projectsPublished[i]
		groups["project"] = append(
			groups["project"],
			FeedItem{Type: "project", ParsedDate: p.UpdatedAt, Project: p},
		)
	}

	opensourcesPublished, err := h.contributions.GetRecentPublished(queryAmount)
	if err != nil {
		log.Printf("Error fetching recent open source: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	lenPublished = len(opensourcesPublished)
	for i := 0; i < lenPublished; i++ {
		o := opensourcesPublished[i]
		groups["contribution"] = append(
			groups["contribution"],
			FeedItem{Type: "contribution", ParsedDate: o.UpdatedAt, Contribution: o},
		)
	}

	// SORTING AND FILTERING THE DATA
	for key := range groups {
		sort.Slice(groups[key], func(i, j int) bool {
			return groups[key][i].ParsedDate.After(groups[key][j].ParsedDate)
		})
	}

	highlighted := make([]FeedItem, 0)
	selected := make([]FeedItem, 0)
	remaining := make([]FeedItem, 0)

	// the order of this slice affecting the order of the category of content too
	// I put contribution and project in the front because I think these 2 more
	// interesting than just my post
	highlightedCategory := []string{"contribution", "project", "post"}
	nonHighlightedCategory := []string{"note", "bookmark"}

	// the amount of highlighted content
	highlightedAmount := 3

	idx := map[string]int{}
	// Make the highlighted slice at least has 3 FeedItem that the content ca-
	// gory is, {"contribution", "project", "post"}
	// If let's say one of the category is empty, then keep filling highlighted
	// with another category that still has elements
	for len(highlighted) < highlightedAmount {
		progress := false
		for _, category := range highlightedCategory {
			if len(highlighted) >= highlightedAmount {
				break
			}
			i := idx[category]
			if i < len(groups[category]) {
				highlighted = append(highlighted, groups[category][i])
				idx[category]++
				progress = true
			}
		}
		if !progress {
			break // all categories exhausted, stop to avoid infinite loop
		}
	}

	// At least show one content of each category to the visitor in the home page
	for category, items := range groups {
		leastAmount := idx[category]
		startIndex := idx[category]
		// If it's non-highlighted category, then the index start from 0 and
		// also the least amount of element in the slice is also at least bigger
		// than 0. It's because if it's highlighted category, we already took
		// some element from the slice at the loop above, so the start index
		// is idx[category], and also at least the array contain more than idx[category]
		// element
		if slices.Contains(nonHighlightedCategory, category) {
			leastAmount = 0
			startIndex = 0
		}

		if len(items) > leastAmount {
			selected = append(selected, items[startIndex])
			// don't use this map anymore but keep increasing it for consistency
			idx[category]++

			if len(items) > leastAmount+1 {
				remaining = append(remaining, items[(startIndex+1):]...)
			}
		}
	}

	nonHighlightedAmount := 9
	highlightedLeft := highlightedAmount - len(highlighted)
	nonHighlightedLeft := nonHighlightedAmount - len(selected)

	// If we already add at least 1 content from each category, then fill the
	// rest of the slice with the remaining that's sorted based on the updated_at
	if len(remaining) > 0 {
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].ParsedDate.After(remaining[j].ParsedDate)
		})

		if len(remaining) < nonHighlightedLeft {
			nonHighlightedLeft = len(remaining)
		}

		selected = append(selected, remaining[:nonHighlightedLeft]...)
	}

	// No need to sort the highlighted so the order is stay:
	// {"contribution", "project", "post"}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].ParsedDate.After(selected[j].ParsedDate)
	})

	// Padding if not full
	if highlightedLeft > 0 {
		highlighted = padFeedItems(highlighted, highlightedAmount)
	}
	if len(selected) < nonHighlightedAmount {
		selected = padFeedItems(selected, nonHighlightedAmount)
	}

	selectedWorks, err := h.selected.GetAllWorks()
	if err != nil {
		log.Printf("Error fetching selected works: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	techStacks, err := h.selected.GetAllTechStackLayouts()
	if err != nil {
		log.Printf("Error fetching tech stacks of selected works: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	selectedWritings, err := h.selected.GetAllWritings()
	if err != nil {
		log.Printf("Error fetching selected writings: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := HomePageData{
		Highlighted:      highlighted,
		Feed:             selected,
		SelectedWorks:    selectedWorks,
		TechStacks:       techStacks,
		SelectedWritings: selectedWritings,
	}

	if err := h.templates.ExecuteTemplate(w, "index", data); err != nil {
		log.Println("handlers.HomePage error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func padFeedItems(items []FeedItem, shownAmount int) []FeedItem {
	if len(items) >= shownAmount {
		return items[:shownAmount]
	}

	out := make([]FeedItem, 0, shownAmount)
	out = append(out, items...)

	i := 0
	feedTypes := []string{"post", "project", "contribution", "note", "bookmark"}
	for len(out) < shownAmount {
		out = append(out, dummyFeedItem(feedTypes[i%len(feedTypes)]))
		i++
	}

	return out
}

func dummyFeedItem(t string) FeedItem {
	now := time.Now()

	item := FeedItem{
		Type:       t,
		ParsedDate: now,
	}

	dummyText := "Edward hasn’t posted anything yet."

	switch t {
	case "post":
		item.Post = &models.Post{
			ID:        0,
			Title:     "Dummy Post",
			Slug:      "#",
			Content:   dummyText,
			Status:    "draft",
			CreatedAt: now,
			UpdatedAt: now,
		}

	case "project":
		item.Project = &models.Project{
			ID:          0,
			Name:        "Dummy Project",
			Slug:        "#",
			Description: dummyText,
			Status:      "draft",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

	case "note":
		item.Note = &models.Note{
			ID:        0,
			Title:     "Dummy Note",
			Slug:      "#",
			Content:   dummyText,
			Status:    "draft",
			CreatedAt: now,
			UpdatedAt: now,
		}

	case "bookmark":
		item.Bookmark = &models.Bookmark{
			ID:          0,
			Title:       "Dummy Bookmark",
			Slug:        "#",
			URL:         "#",
			Description: dummyText,
			Status:      "draft",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

	case "contribution":
		item.Contribution = &models.Contribution{
			ID:                  0,
			ProjectName:         "Dummy Contribution",
			Slug:                "#",
			Description:         dummyText,
			ContributionSummary: dummyText,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
	}

	return item
}
