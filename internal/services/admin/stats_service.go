package admin_services

import (
	"database/sql"
	"log/slog"

	"blog/internal/models"
)

type StatsService struct {
	posts         *models.PostRepository
	bookmarks     *models.BookmarkRepository
	notes         *models.NoteRepository
	projects      *models.ProjectRepository
	contributions *models.ContributionRepository
}

func NewStatsService(db *sql.DB) *StatsService {
	return &StatsService{
		posts:         models.NewPostRepository(db),
		bookmarks:     models.NewBookmarkRepository(db),
		notes:         models.NewNoteRepository(db),
		projects:      models.NewProjectRepository(db),
		contributions: models.NewContributionRepository(db),
	}
}

type StatsCardData struct {
	PostCount          int64
	PublishedPosts     int64
	DraftPosts         int64
	BookmarkCount      int64
	PublishedBookmarks int64
	DraftBookmarks     int64
	NoteCount          int64
	PublishedNotes     int64
	DraftNotes         int64
	ProjectCount       int64
	PublishedProjects  int64
	DraftProjects      int64
	OpenSourceCount    int64
}

type RecentData struct {
	RecentPosts      []*models.Post
	RecentBookmarks  []*models.Bookmark
	RecentNotes      []*models.Note
	RecentProjects   []*models.Project
	RecentOpenSource []*models.Contribution
}

func (s *StatsService) FetchStatsData() (*StatsCardData, error) {
	data := &StatsCardData{}

	postStats, err := s.posts.GetStats()
	if err != nil {
		slog.Error("Failed fetching post stats", "error", err)
		return nil, err
	}

	bookmarkStats, err := s.bookmarks.GetStats()
	if err != nil {
		slog.Error("Failed fetching bookmark stats", "error", err)
		return nil, err
	}

	noteStats, err := s.notes.GetStats()
	if err != nil {
		slog.Error("Failed fetching note stats", "error", err)
		return nil, err
	}

	projectStats, err := s.projects.GetStats()
	if err != nil {
		slog.Error("Failed fetching project stats", "error", err)
		return nil, err
	}

	opensourceStats, err := s.contributions.GetStats()
	if err != nil {
		slog.Error("Failed fetching open source stats", "error", err)
		return nil, err
	}

	data.PostCount = postStats.TotalCount
	data.PublishedPosts = postStats.PublishedCount
	data.DraftPosts = postStats.DraftCount

	data.BookmarkCount = bookmarkStats.TotalCount
	data.PublishedBookmarks = bookmarkStats.PublishedCount
	data.DraftBookmarks = bookmarkStats.DraftCount

	data.NoteCount = noteStats.TotalCount
	data.PublishedNotes = noteStats.PublishedCount
	data.DraftNotes = noteStats.DraftCount

	data.ProjectCount = projectStats.TotalCount
	data.PublishedProjects = projectStats.PublishedCount
	data.DraftProjects = projectStats.DraftCount

	data.OpenSourceCount = opensourceStats.TotalCount

	return data, nil
}
