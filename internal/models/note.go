package models

import (
	"database/sql"
	"time"
)

type Note struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Tags      string    `json:"tags"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NoteRepository struct {
	db *sql.DB
}

func NewNoteRepository(db *sql.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) Create(note *Note) error {
	query := `
		INSERT INTO notes (
			title,
			slug,
			content,
			tags,
			status
		) VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		note.Title,
		note.Slug,
		note.Content,
		note.Tags,
		note.Status,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	note.ID = int(id)
	return nil
}

func (r *NoteRepository) GetByID(id int64) (*Note, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at
		FROM notes
		WHERE id = ?
	`

	note := &Note{}
	err := r.db.QueryRow(query, id).Scan(
		&note.ID,
		&note.Title,
		&note.Slug,
		&note.Content,
		&note.Tags,
		&note.Status,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return note, nil
}

func (r *NoteRepository) GetAll(filter string, status string) ([]*Note, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at 
		FROM notes 
		WHERE 1=1
	`
	args := []any{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if filter != "" {
		query += " AND (title LIKE ? OR content LIKE ?)"
		likeFilter := "%" + filter + "%"
		args = append(args, likeFilter, likeFilter)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		note := &Note{}
		err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.Slug,
			&note.Content,
			&note.Tags,
			&note.Status,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, nil
}

func (r *NoteRepository) Update(note *Note) error {
	query := `
		UPDATE notes 
		SET title = ?, content = ?, tags = ?, status = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(
		query,
		note.Title,
		note.Content,
		note.Tags,
		note.Status,
		note.ID,
	)
	return err
}

func (r *NoteRepository) Delete(id int64) error {
	query := `DELETE FROM notes WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *NoteRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM notes`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	return count, err
}

func (r *NoteRepository) CountByStatus(status string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM notes 
		WHERE status = ?
	`

	var count int
	err := r.db.QueryRow(query, status).Scan(&count)
	return count, err
}

func (r *NoteRepository) GetRecent(limit int) ([]*Note, error) {
	query := `
		SELECT
			id,
			title,
			content,
			tags,
			status,
			created_at,
			updated_at 
		FROM notes 
		ORDER BY updated_at DESC 
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		note := &Note{}
		err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.Content,
			&note.Tags,
			&note.Status,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, nil
}

type NoteStats struct {
	TotalCount     int64
	PublishedCount int64
	DraftCount     int64
}

func (r *NoteRepository) GetStats() (*NoteStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COALESCE(
				SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0
			) as published_count,
			COALESCE(
				SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0
			)as draft_count
		FROM notes;
	`

	var stats NoteStats

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

func (r *NoteRepository) GetRecentPublished(limit int) ([]*Note, error) {
	query := `
		SELECT
			id,
			title,
			slug,
			content,
			tags,
			status,
			created_at,
			updated_at
		FROM notes
		WHERE status = 'published'
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		note := &Note{}
		err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.Slug,
			&note.Content,
			&note.Tags,
			&note.Status,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, nil
}

func (r *NoteRepository) GetBySlug(slug string) (*Note, error) {
	query := `
		SELECT
			id,
			title,
			content,
			tags,
			status,
			created_at,
			updated_at
		FROM notes
		WHERE slug = ? AND status = 'published'
	`
	note := &Note{}
	err := r.db.QueryRow(query, slug).Scan(
		&note.ID,
		&note.Title,
		&note.Content,
		&note.Tags,
		&note.Status,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return note, nil
}
