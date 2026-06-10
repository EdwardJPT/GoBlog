package models

import (
	"database/sql"
	"strings"
	"time"
)

type Post struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Content     string     `json:"content"`
	Tags        string     `json:"tags"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(post *Post) error {
	query := `
		INSERT INTO posts (
			title,
			slug,
			content,
			tags,
			status,
			published_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	var publishedAt *time.Time
	if post.Status == "published" {
		now := time.Now()
		publishedAt = &now
	}

	result, err := r.db.Exec(
		query,
		post.Title,
		post.Slug,
		post.Content,
		post.Tags,
		post.Status,
		publishedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	post.ID = int(id)
	return nil
}

func (r *PostRepository) GetByID(id int64) (*Post, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at,
			published_at
		FROM posts
		WHERE id = ?
	`

	post := &Post{}
	err := r.db.QueryRow(query, id).Scan(
		&post.ID,
		&post.Title,
		&post.Slug,
		&post.Content,
		&post.Tags,
		&post.Status,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.PublishedAt,
	)
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *PostRepository) GetAll(filter string, status string) ([]*Post, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at,
			published_at
		FROM posts 
		WHERE 1=1
	`
	args := []any{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if filter != "" {
		query += " AND (title LIKE ? OR content LIKE ? OR tags LIKE ?)"
		likeFilter := "%" + filter + "%"
		args = append(args, likeFilter, likeFilter, likeFilter)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Slug,
			&post.Content,
			&post.Tags,
			&post.Status,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.PublishedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (r *PostRepository) Update(post *Post) error {
	query := `
		UPDATE posts
		SET
			title = :title,
			slug = :slug,
			content = :content,
			tags = :tags,
			status = :status,
			published_at = CASE
				WHEN :status = 'published' THEN CURRENT_TIMESTAMP
				ELSE published_at
			END
		WHERE id = :id
	`

	_, err := r.db.Exec(
		query,
		// TODO: Look at this feature!
		sql.Named("title", post.Title),
		sql.Named("slug", post.Slug),
		sql.Named("content", post.Content),
		sql.Named("tags", post.Tags),
		sql.Named("status", post.Status),
		sql.Named("id", post.ID),
	)
	return err
}

func (r *PostRepository) Delete(id int64) error {
	query := `DELETE FROM posts WHERE id = ?`

	_, err := r.db.Exec(query, id)
	return err
}

func (r *PostRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM posts`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	return count, err
}

func (r *PostRepository) CountByStatus(status string) (int, error) {
	query := `SELECT COUNT(*) FROM posts WHERE status = ?`

	var count int
	err := r.db.QueryRow(query, status).Scan(&count)
	return count, err
}

func (r *PostRepository) GetRecent(limit int) ([]*Post, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at,
			published_at
		FROM posts
		ORDER BY updated_at DESC 
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Slug,
			&post.Content,
			&post.Tags,
			&post.Status,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.PublishedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// GenerateSlug creates a simple slug from title
func GenerateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove special characters (simplified)
	var result []rune
	for _, char := range slug {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result = append(result, char)
		}
	}
	return string(result)
}

type PostStats struct {
	TotalCount     int64
	PublishedCount int64
	DraftCount     int64
}

func (r *PostRepository) GetStats() (*PostStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COALESCE(
				SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0
			) as published_count,
			COALESCE(
				SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0
			)as draft_count
		FROM posts;
	`
	var stats PostStats
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

func (r *PostRepository) GetRecentPublished(limit int) ([]*Post, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at,
			published_at
		FROM posts
		WHERE status = 'published'
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Slug,
			&post.Content,
			&post.Tags,
			&post.Status,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.PublishedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (r *PostRepository) GetBySlug(slug string) (*Post, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at,
			published_at
		FROM posts
		WHERE slug = ? AND status = 'published'
	`
	post := &Post{}
	err := r.db.QueryRow(query, slug).Scan(
		&post.ID,
		&post.Title,
		&post.Slug,
		&post.Content,
		&post.Tags,
		&post.Status,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.PublishedAt,
	)
	if err != nil {
		return nil, err
	}
	return post, nil
}
