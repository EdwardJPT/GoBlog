package service

import (
	"blog/internal/models"
	"database/sql"
	"log"
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
		log.Printf("Error fetching post stats: %v", err)
		return &StatsCardData{}, err
	}

	bookmarkStats, err := s.bookmarks.GetStats()
	if err != nil {
		log.Printf("Error fetching bookmark stats: %v", err)
		return &StatsCardData{}, err
	}

	noteStats, err := s.notes.GetStats()
	if err != nil {
		log.Printf("Error fetching note stats: %v", err)
		return &StatsCardData{}, err
	}

	projectStats, err := s.projects.GetStats()
	if err != nil {
		log.Printf("Error fetching project stats: %v", err)
		return &StatsCardData{}, err
	}

	opensourceStats, err := s.contributions.GetStats()
	if err != nil {
		log.Printf("Error fetching open source stats: %v", err)
		return &StatsCardData{}, err
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
