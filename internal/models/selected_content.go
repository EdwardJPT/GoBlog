package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SelectedRepository struct {
	db *sql.DB
}

func NewSelectedRepository(db *sql.DB) *SelectedRepository {
	return &SelectedRepository{db: db}
}

type SelectedWritingsRef struct {
	TargetTable string `json:"target_table"`
	TargetID    int64  `json:"target_id"`
}

type SelectedWritingsValue struct {
	Type      string    `json:"type"`
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Tags      string    `json:"tags"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SelectedWorksValue struct {
	Type                string    `json:"type"`
	RowID               int64     `json:"rowid"` // this is the id of selected_works table
	ID                  int64     `json:"id"`    // this is the id of ${Type} table
	Title               string    `json:"title"`
	Slug                string    `json:"slug"`
	Description         string    `json:"description"`
	HomePageDescription string    `json:"home_page_description"`
	Link                string    `json:"link"`
	Tags                string    `json:"tags"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (r *SelectedRepository) GetAllWritings() ([]*SelectedWritingsValue, error) {
	query := `
		SELECT
			target_table,
			target_id
		FROM selected_writings
		ORDER BY rowid DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type selectedRef struct {
		TargetTable string
		TargetID    int64
	}

	var refs []selectedRef

	for rows.Next() {
		var ref selectedRef

		err := rows.Scan(
			&ref.TargetTable,
			&ref.TargetID,
		)
		if err != nil {
			return nil, err
		}

		refs = append(refs, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var writings []*SelectedWritingsValue

	allowedTables := map[string]bool{
		"posts": true,
		"notes": true,
	}

	for _, ref := range refs {
		// Safety check
		if !allowedTables[ref.TargetTable] {
			return nil, fmt.Errorf("invalid target table: %s", ref.TargetTable)
		}

		dynamicQuery := fmt.Sprintf(`
			SELECT
				id,
				title,
				slug,
				content,
				tags,
				updated_at
			FROM %s
			WHERE id = ?
			LIMIT 1
		`, ref.TargetTable)

		var writing SelectedWritingsValue

		err := r.db.QueryRow(dynamicQuery, ref.TargetID).Scan(
			&writing.ID,
			&writing.Title,
			&writing.Slug,
			&writing.Content,
			&writing.Tags,
			&writing.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}

		// Mapping for url in the front end
		writing.Type = map[string]string{
			"posts": "post",
			"notes": "note",
		}[ref.TargetTable]

		writings = append(writings, &writing)
	}

	return writings, nil
}

func (r *SelectedRepository) InsertWriting(targetTable string, targetID int64) error {
	allowedTables := map[string]bool{
		"posts": true,
		"notes": true,
	}

	// Safety check
	if !allowedTables[targetTable] {
		return fmt.Errorf("invalid target table: %s", targetTable)
	}

	query := `
		INSERT OR IGNORE INTO selected_writings (
			target_table,
			target_id
		)
		VALUES (?, ?)
	`

	_, err := r.db.Exec(query, targetTable, targetID)
	if err != nil {
		return err
	}

	return nil
}

func (r *SelectedRepository) GetAllWorks() ([]*SelectedWorksValue, error) {
	query := `
		SELECT
			id,
			target_table,
			target_id
		FROM selected_works
		ORDER BY rowid DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type selectedRef struct {
		RowID       int64
		TargetTable string
		TargetID    int64
	}

	var refs []selectedRef

	for rows.Next() {
		var ref selectedRef

		err := rows.Scan(
			&ref.RowID,
			&ref.TargetTable,
			&ref.TargetID,
		)
		if err != nil {
			return nil, err
		}

		refs = append(refs, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var works []*SelectedWorksValue

	allowedTables := map[string]bool{
		"projects":                 true,
		"opensource_contributions": true,
	}

	for _, ref := range refs {
		// Safety check
		if !allowedTables[ref.TargetTable] {
			return nil, fmt.Errorf("invalid target table: %s", ref.TargetTable)
		}

		var dynamicQuery string
		if ref.TargetTable == "projects" {
			dynamicQuery = `
				SELECT
					id,
					name as title,
					slug,
					description,
					home_page_description,
					repo_link as link,
					tags,
					updated_at
				FROM projects
				WHERE id = ?
				LIMIT 1
			`
		} else {
			dynamicQuery = `
				SELECT
					id,
					project_name as title,
					slug,
					description,
					home_page_description,
					link,
					tags,
					updated_at
				FROM opensource_contributions
				WHERE id = ?
				LIMIT 1
			`
		}

		var work SelectedWorksValue
		var link sql.NullString
		var tags sql.NullString
		var desc sql.NullString

		err := r.db.QueryRow(dynamicQuery, ref.TargetID).Scan(
			&work.ID,
			&work.Title,
			&work.Slug,
			&desc,
			&work.HomePageDescription,
			&link,
			&tags,
			&work.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}

		if link.Valid {
			work.Link = link.String
		}
		if tags.Valid {
			work.Tags = tags.String
		}
		if desc.Valid {
			work.Description = desc.String
		}

		// We need this because of the foreign key relationship we have between
		// the table of selected_works and tech_stack_layouts
		work.RowID = ref.RowID

		// Mapping for url in the front end
		work.Type = map[string]string{
			"projects":                 "project",
			"opensource_contributions": "open-source-contribution",
		}[ref.TargetTable]

		works = append(works, &work)
	}

	return works, nil
}

func (r *SelectedRepository) GetAllTechStackLayouts() (map[int64][]string, error) {
	query := `
        SELECT
            work_id,
            tech_stack_json
        FROM tech_stack_layouts
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Modified map to hold a flat []string instead of [][]string because we
	// split it by "," in the front end
	techStacks := make(map[int64][]string)

	for rows.Next() {
		var workID int64
		var techStackJSON string

		if err := rows.Scan(&workID, &techStackJSON); err != nil {
			return nil, err
		}

		var layout [][]string
		var flatLayout []string // New slice to hold flattened data

		if techStackJSON != "" {
			if err := json.Unmarshal([]byte(techStackJSON), &layout); err != nil {
				parseError := fmt.Sprintf("Failed to parse tech stack for work_id %d: %v", workID, err)
				return nil, errors.New(parseError)
			}

			// Flatten the 2D slice into a 1D slice
			for _, group := range layout {
				flatLayout = append(flatLayout, group...)
			}
		}

		// Assign the flattened slice to the map
		techStacks[workID] = flatLayout
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return techStacks, nil
}

func (r *SelectedRepository) InsertWork(targetTable string, targetID int64) error {
	allowedTables := map[string]bool{
		"projects":                 true,
		"opensource_contributions": true,
	}

	// Safety check
	if !allowedTables[targetTable] {
		return fmt.Errorf("invalid target table: %s", targetTable)
	}

	query := `
		INSERT OR IGNORE INTO selected_works (
			target_table,
			target_id
		)
		VALUES (?, ?)
	`

	_, err := r.db.Exec(query, targetTable, targetID)
	if err != nil {
		return err
	}

	return nil
}

func (r *SelectedRepository) UpsertTechStackLayout(workID int64, skillsJSON string) error {
	query := `
		INSERT INTO tech_stack_layouts (
			work_id,
			tech_stack_json
		)
		VALUES (?, ?)
		ON CONFLICT(work_id) DO UPDATE SET
			tech_stack_json = excluded.tech_stack_json;
	`

	_, err := r.db.Exec(query, workID, skillsJSON)
	if err != nil {
		return fmt.Errorf("failed to upsert tech stack layout: %w", err)
	}

	return nil
}
