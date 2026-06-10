package models

import (
	"database/sql"
	"time"
)

type Project struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	Description         string    `json:"description"`
	HomePageDescription string    `json:"home_page_description"`
	RepoLink            string    `json:"repo_link"`
	DemoLink            string    `json:"demo_link"`
	OtherLinks          string    `json:"other_links"`
	Tags                string    `json:"tags"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(project *Project) error {
	query := `
		INSERT INTO projects (
			name,
			slug,
			description,
			repo_link,
			demo_link,
			other_links,
			tags,
			status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		project.Name,
		project.Slug,
		project.Description,
		project.RepoLink,
		project.DemoLink,
		project.OtherLinks,
		project.Tags,
		project.Status,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	project.ID = int(id)
	return nil
}

func (r *ProjectRepository) GetByID(id int64) (*Project, error) {
	query := `
		SELECT
			id,
			name,
			description,
			home_page_description,
			repo_link,
			demo_link,
			other_links,
			tags,
			status,
			created_at,
			updated_at
		FROM projects
		WHERE id = ?
	`

	project := &Project{}
	err := r.db.QueryRow(query, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.HomePageDescription,
		&project.RepoLink,
		&project.DemoLink,
		&project.OtherLinks,
		&project.Tags,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (r *ProjectRepository) GetAll(filter string, status string) ([]*Project, error) {
	query := `
		SELECT
			id,
			name,
			slug,
			description,
			repo_link,
			demo_link,
			other_links,
			tags,
			status,
			created_at,
			updated_at
		FROM projects
		WHERE 1=1
	`
	args := []any{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if filter != "" {
		query += " AND (name LIKE ? OR description LIKE ? OR tags LIKE ?)"
		likeFilter := "%" + filter + "%"
		args = append(args, likeFilter, likeFilter, likeFilter)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Slug,
			&project.Description,
			&project.RepoLink,
			&project.DemoLink,
			&project.OtherLinks,
			&project.Tags,
			&project.Status,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *ProjectRepository) Update(project *Project) error {
	query := `
		UPDATE projects
		SET
			name = ?,
			description = ?,
			home_page_description = ?,
			repo_link = ?,
			demo_link = ?,
			other_links = ?,
			tags = ?,
			status = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(
		query,
		project.Name,
		project.Description,
		project.HomePageDescription,
		project.RepoLink,
		project.DemoLink,
		project.OtherLinks,
		project.Tags,
		project.Status,
		project.ID,
	)
	return err
}

func (r *ProjectRepository) Delete(id int64) error {
	query := `DELETE FROM projects WHERE id = ?`

	_, err := r.db.Exec(query, id)
	return err
}

func (r *ProjectRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM projects`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	return count, err
}

func (r *ProjectRepository) CountByStatus(status string) (int, error) {
	query := `SELECT COUNT(*) FROM projects WHERE status = ?`

	var count int
	err := r.db.QueryRow(query, status).Scan(&count)
	return count, err
}

func (r *ProjectRepository) GetRecent(limit int) ([]*Project, error) {
	query := `
		SELECT
			id,
			name,
			description,
			repo_link,
			demo_link,
			other_links,
			tags,
			status,
			created_at,
			updated_at
		FROM projects
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.RepoLink,
			&project.DemoLink,
			&project.OtherLinks,
			&project.Tags,
			&project.Status,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// ProjectStats holds the counts needed for the dashboard
type ProjectStats struct {
	TotalCount     int64
	PublishedCount int64
	DraftCount     int64
}

func (r *ProjectRepository) GetStats() (*ProjectStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COALESCE(
				SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0
			) as published_count,
			COALESCE(
				SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0
			)as draft_count
		FROM projects
	`
	var stats ProjectStats
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

func (r *ProjectRepository) GetRecentPublished(limit int) ([]*Project, error) {
	query := `
		SELECT
			id,
			name,
			slug,
			description,
			repo_link,
			demo_link,
			other_links,
			tags,
			status,
			created_at,
			updated_at
		FROM projects
		WHERE status = 'published'
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Slug,
			&project.Description,
			&project.RepoLink,
			&project.DemoLink,
			&project.OtherLinks,
			&project.Tags,
			&project.Status,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *ProjectRepository) GetBySlug(slug string) (*Project, error) {
	query := `
		SELECT
			id,
			name,
			description,
			repo_link,
			demo_link,
			other_links,
			tags,
			status,
			created_at,
			updated_at
		FROM projects
		WHERE slug = ? AND status = 'published'
	`
	project := &Project{}
	err := r.db.QueryRow(query, slug).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.RepoLink,
		&project.DemoLink,
		&project.OtherLinks,
		&project.Tags,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return project, nil
}
