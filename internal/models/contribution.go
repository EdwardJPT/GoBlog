package models

import (
	"database/sql"
	"time"
)

type Contribution struct {
	ID                  int       `json:"id"`
	ProjectName         string    `json:"project_name"`
	Slug                string    `json:"slug"`
	Description         string    `json:"description"`
	HomePageDescription string    `json:"home_page_description"`
	Tags                string    `json:"tags"`
	Link                string    `json:"link"`
	ContributionSummary string    `json:"contribution_summary"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ContributionRepository struct {
	db *sql.DB
}

func NewContributionRepository(db *sql.DB) *ContributionRepository {
	return &ContributionRepository{db: db}
}

func (r *ContributionRepository) Create(contribution *Contribution) error {
	query := `
		INSERT INTO opensource_contributions (
			project_name,
			slug,
			description,
			tags,
			link,
			contribution_summary
		) VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		contribution.ProjectName,
		contribution.Slug,
		contribution.Description,
		contribution.Tags,
		contribution.Link,
		contribution.ContributionSummary,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	contribution.ID = int(id)
	return nil
}

func (r *ContributionRepository) GetByID(id int64) (*Contribution, error) {
	query := `
		SELECT
			id,
			project_name,
			description,
			home_page_description,
			tags,
			link,
			contribution_summary,
			created_at,
			updated_at
		FROM opensource_contributions
		WHERE id = ?
	`

	contribution := &Contribution{}
	err := r.db.QueryRow(query, id).Scan(
		&contribution.ID,
		&contribution.ProjectName,
		&contribution.Description,
		&contribution.HomePageDescription,
		&contribution.Tags,
		&contribution.Link,
		&contribution.ContributionSummary,
		&contribution.CreatedAt,
		&contribution.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return contribution, nil
}

func (r *ContributionRepository) GetAll(filter string) ([]*Contribution, error) {
	query := `
		SELECT
			id,
			project_name,
			slug,
			description,
			tags,
			link,
			contribution_summary,
			created_at,
			updated_at
		FROM opensource_contributions
		WHERE 1=1
	`
	args := []any{}

	if filter != "" {
		query += " AND (project_name LIKE ? OR description LIKE ? OR contribution_summary LIKE ?)"
		likeFilter := "%" + filter + "%"
		args = append(args, likeFilter, likeFilter, likeFilter)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contributions []*Contribution
	for rows.Next() {
		contribution := &Contribution{}
		err := rows.Scan(
			&contribution.ID,
			&contribution.ProjectName,
			&contribution.Slug,
			&contribution.Description,
			&contribution.Tags,
			&contribution.Link,
			&contribution.ContributionSummary,
			&contribution.CreatedAt,
			&contribution.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, contribution)
	}

	return contributions, nil
}

func (r *ContributionRepository) Update(contribution *Contribution) error {
	query := `
		UPDATE opensource_contributions
		SET
			project_name = ?,
			description = ?,
			home_page_description = ?,
			link = ?,
			contribution_summary = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(
		query,
		contribution.ProjectName,
		contribution.Description,
		contribution.HomePageDescription,
		contribution.Link,
		contribution.ContributionSummary,
		contribution.ID,
	)
	return err
}

func (r *ContributionRepository) Delete(id int64) error {
	query := `
		DELETE FROM opensource_contributions
		WHERE id = ?
	`

	_, err := r.db.Exec(query, id)
	return err
}

func (r *ContributionRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM opensource_contributions`
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	return count, err
}

func (r *ContributionRepository) GetRecent(limit int) ([]*Contribution, error) {
	query := `
		SELECT
			id,
			project_name,
			description,
			tags,
			link,
			contribution_summary,
			created_at,
			updated_at
		FROM opensource_contributions
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contributions []*Contribution
	for rows.Next() {
		contribution := &Contribution{}
		err := rows.Scan(
			&contribution.ID,
			&contribution.ProjectName,
			&contribution.Description,
			&contribution.Tags,
			&contribution.Link,
			&contribution.ContributionSummary,
			&contribution.CreatedAt,
			&contribution.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, contribution)
	}

	return contributions, nil
}

type OpenSourceStats struct {
	TotalCount int64
}

func (r *ContributionRepository) GetStats() (*OpenSourceStats, error) {
	query := `
		SELECT COUNT(*) as total_count
		FROM opensource_contributions
	`
	var stats OpenSourceStats
	err := r.db.QueryRow(query).Scan(&stats.TotalCount)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *ContributionRepository) GetRecentPublished(limit int) ([]*Contribution, error) {
	query := `
		SELECT
			id,
			project_name,
			slug,
			description,
			tags,
			link,
			contribution_summary,
			created_at,
			updated_at
		FROM opensource_contributions
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contributions []*Contribution
	for rows.Next() {
		contribution := &Contribution{}
		err := rows.Scan(
			&contribution.ID,
			&contribution.ProjectName,
			&contribution.Slug,
			&contribution.Description,
			&contribution.Tags,
			&contribution.Link,
			&contribution.ContributionSummary,
			&contribution.CreatedAt,
			&contribution.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, contribution)
	}

	return contributions, nil
}

func (r *ContributionRepository) GetBySlug(slug string) (*Contribution, error) {
	query := `
		SELECT
			id,
			project_name,
			description,
			tags,
			link,
			contribution_summary,
			created_at,
			updated_at
		FROM opensource_contributions
		WHERE slug = ?
	`
	contribution := &Contribution{}
	err := r.db.QueryRow(query, slug).Scan(
		&contribution.ID,
		&contribution.ProjectName,
		&contribution.Description,
		&contribution.Tags,
		&contribution.Link,
		&contribution.ContributionSummary,
		&contribution.CreatedAt,
		&contribution.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return contribution, nil
}
