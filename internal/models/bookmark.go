package models

import (
	"database/sql"
	"time"
)

type Bookmark struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Tags        string    `json:"tags"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BookmarkRepository struct {
	db *sql.DB
}

func NewBookmarkRepository(db *sql.DB) *BookmarkRepository {
	return &BookmarkRepository{db: db}
}

func (r *BookmarkRepository) Create(bookmark *Bookmark) error {
	query := `
		INSERT INTO bookmarks (
			title,
			slug,
			url,
			description,
			tags,
			status
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		bookmark.Title,
		bookmark.Slug,
		bookmark.URL,
		bookmark.Description,
		bookmark.Tags,
		bookmark.Status,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	bookmark.ID = int(id)
	return nil
}

func (r *BookmarkRepository) GetByID(id int64) (*Bookmark, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			url,
			description,
			tags,
			status,
			created_at,
			updated_at
		FROM bookmarks
		WHERE id = ?
	`

	bookmark := &Bookmark{}
	err := r.db.QueryRow(query, id).Scan(
		&bookmark.ID,
		&bookmark.Title,
		&bookmark.Slug,
		&bookmark.URL,
		&bookmark.Description,
		&bookmark.Tags,
		&bookmark.Status,
		&bookmark.CreatedAt,
		&bookmark.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return bookmark, nil
}

func (r *BookmarkRepository) GetAll(filter string, status string) ([]*Bookmark, error) {
	query := `
		SELECT 
			id, 
			title, 
			url, 
			description, 
			tags, 
			status, 
			created_at, 
			updated_at 
		FROM bookmarks 
		WHERE 1=1
	`
	args := []any{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if filter != "" {
		query += " AND (title LIKE ? OR url LIKE ? OR description LIKE ? OR tags LIKE ?)"
		likeFilter := "%" + filter + "%"
		args = append(args, likeFilter, likeFilter, likeFilter, likeFilter)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []*Bookmark
	for rows.Next() {
		bookmark := &Bookmark{}
		err := rows.Scan(
			&bookmark.ID,
			&bookmark.Title,
			&bookmark.URL,
			&bookmark.Description,
			&bookmark.Tags,
			&bookmark.Status,
			&bookmark.CreatedAt,
			&bookmark.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}

func (r *BookmarkRepository) Update(bookmark *Bookmark) error {
	query := `
		UPDATE bookmarks 
		SET 
			title = ?, 
			url = ?, 
			description = ?, 
			tags = ?, 
			status = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(
		query,
		bookmark.Title,
		bookmark.URL,
		bookmark.Description,
		bookmark.Tags,
		bookmark.Status,
		bookmark.ID,
	)
	return err
}

func (r *BookmarkRepository) Delete(id int64) error {
	query := `DELETE FROM bookmarks WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *BookmarkRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM bookmarks`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	return count, err
}

func (r *BookmarkRepository) CountByStatus(status string) (int, error) {
	query := `SELECT COUNT(*) FROM bookmarks WHERE status = ?`

	var count int
	err := r.db.QueryRow(query, status).Scan(&count)
	return count, err
}

func (r *BookmarkRepository) GetRecent(limit int) ([]*Bookmark, error) {
	query := `
		SELECT 
			id, 
			title, 
			url, 
			description, 
			tags, 
			status, 
			created_at, 
			updated_at 
		FROM bookmarks 
		ORDER BY updated_at DESC 
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []*Bookmark
	for rows.Next() {
		bookmark := &Bookmark{}
		err := rows.Scan(
			&bookmark.ID,
			&bookmark.Title,
			&bookmark.URL,
			&bookmark.Description,
			&bookmark.Tags,
			&bookmark.Status,
			&bookmark.CreatedAt,
			&bookmark.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}

type BookmarkStats struct {
	TotalCount     int64
	PublishedCount int64
	DraftCount     int64
}

func (r *BookmarkRepository) GetStats() (*BookmarkStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COALESCE(
				SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0
			) as published_count,
			COALESCE(
				SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0
			)as draft_count
		FROM bookmarks;
	`
	var stats BookmarkStats
	err := r.db.QueryRow(query).Scan(
		&stats.TotalCount,
		&stats.PublishedCount,
		&stats.DraftCount,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *BookmarkRepository) GetRecentPublished(limit int) ([]*Bookmark, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			url,
			description,
			tags,
			status,
			created_at,
			updated_at
		FROM bookmarks
		WHERE status = 'published'
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []*Bookmark
	for rows.Next() {
		bookmark := &Bookmark{}
		err := rows.Scan(
			&bookmark.ID,
			&bookmark.Title,
			&bookmark.Slug,
			&bookmark.URL,
			&bookmark.Description,
			&bookmark.Tags,
			&bookmark.Status,
			&bookmark.CreatedAt,
			&bookmark.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}

func (r *BookmarkRepository) GetBySlug(slug string) (*Bookmark, error) {
	query := `
		SELECT
			id,
			title,
			url,
			description,
			tags,
			status,
			created_at,
			updated_at
		FROM bookmarks
		WHERE slug = ? AND status = 'published'
	`
	bookmark := &Bookmark{}
	err := r.db.QueryRow(query, slug).Scan(
		&bookmark.ID,
		&bookmark.Title,
		&bookmark.URL,
		&bookmark.Description,
		&bookmark.Tags,
		&bookmark.Status,
		&bookmark.CreatedAt,
		&bookmark.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return bookmark, nil
}
