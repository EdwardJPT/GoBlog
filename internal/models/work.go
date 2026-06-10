package models

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

type WorkItem struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	RepoLink  string    `json:"repo_link"`
	DemoLink  string    `json:"demo_link"`
	Tags      string    `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkRepository struct {
	db *sql.DB
}

func NewWorkRepository(db *sql.DB) *WorkRepository {
	return &WorkRepository{db: db}
}

func (r *WorkRepository) GetProjectsFeed(
	selectedTypes []string,
	encodedCursor string,
	isPrev bool,
	limit int,
) ([]WorkItem, error) {
	allowedBaseQueries := map[string]string{
		"projects": `
			SELECT
				id,
				name AS title,
				slug,
				COALESCE(description, '') AS content,
				created_at,
				updated_at,
				'project' AS type,
				COALESCE(repo_link, '') AS repo_link,
				COALESCE(demo_link, '') AS demo_link,
				COALESCE(tags, '') AS tags
			FROM projects
			WHERE status = 'published'`,
		"opensource_contributions": `
			SELECT
				id,
				project_name AS title,
				slug,
				COALESCE(contribution_summary, description, '') AS content,
				created_at,
				updated_at,
				'contribution' AS type,
				COALESCE(link, '') AS repo_link,
				'' AS demo_link,
				COALESCE(tags, '') AS tags
			FROM opensource_contributions`,
	}

	if len(selectedTypes) == 0 {
		selectedTypes = []string{"projects", "opensource_contributions"}
	}

	// Setup default cursors and operators based on direction
	// Default value for cursor, If no cursor is provided, we must be going
	// "Next" from the very beginning
	cursor := "9999-12-31 23:59:59"
	operator := "<"
	order := "DESC"

	if encodedCursor != "" {
		decodedBytes, err := base64.URLEncoding.DecodeString(encodedCursor)
		if err == nil {
			cursor = string(decodedBytes)
		}
	}

	if isPrev {
		operator = ">"
		order = "ASC"
	}

	var activeQueries []string
	var args []any

	// Build the directional subqueries
	for _, t := range selectedTypes {
		if baseQuery, exists := allowedBaseQueries[t]; exists {
			conditionKeyword := "WHERE"
			if strings.Contains(strings.ToUpper(baseQuery), "WHERE") {
				conditionKeyword = "AND"
			}

			subQuery := fmt.Sprintf(`
				SELECT * FROM (
					%s
					%s updated_at %s ?
					ORDER BY updated_at %s
					LIMIT ?
				)`,
				baseQuery,
				conditionKeyword,
				operator,
				order,
			)
			activeQueries = append(activeQueries, subQuery)
			args = append(args, cursor, limit)
		}
	}

	if len(activeQueries) == 0 {
		return nil, errors.New("invalid types requested")
	}

	unionQuery := strings.Join(activeQueries, " UNION ALL ")
	finalQuery := fmt.Sprintf(`
		SELECT
			id,
			title,
			slug,
			content,
			created_at,
			updated_at,
			type,
			repo_link,
			demo_link,
			tags
		FROM (%s) AS combined_data
		ORDER BY updated_at %s
		LIMIT ?
	`, unionQuery, order)

	args = append(args, limit)

	rows, err := r.db.Query(finalQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feed []WorkItem
	for rows.Next() {
		var item WorkItem

		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Slug,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Type,
			&item.RepoLink,
			&item.DemoLink,
			&item.Tags,
		); err != nil {
			return nil, err
		}
		feed = append(feed, item)
	}

	return feed, nil
}

func (r *WorkRepository) HasPublishedRowsAroundCursor(
	selectedTypes []string,
	firstTimestamp time.Time,
	lastTimestamp time.Time,
) (hasPrev bool, hasNext bool, err error) {
	allowedBaseQueries := map[string]string{
		"projects": `
			SELECT updated_at
			FROM projects
			WHERE status = 'published'`,
		"opensource_contributions": `
			SELECT updated_at
			FROM opensource_contributions`,
	}

	if len(selectedTypes) == 0 {
		selectedTypes = []string{"projects", "opensource_contributions"}
	}

	var activeQueries []string

	for _, t := range selectedTypes {
		if baseQuery, exists := allowedBaseQueries[t]; exists {
			activeQueries = append(activeQueries, baseQuery)
		}
	}

	if len(activeQueries) == 0 {
		return false, false, errors.New("invalid types requested")
	}

	unionQuery := strings.Join(activeQueries, " UNION ALL ")

	finalQuery := fmt.Sprintf(`
		WITH combined AS (
			%s
		)
		SELECT
			EXISTS(
				SELECT 1
				FROM combined
				WHERE datetime(updated_at) > datetime(?)
			) AS has_above,
			EXISTS(
				SELECT 1
				FROM combined
				WHERE datetime(updated_at) < datetime(?)
			) AS has_below
	`, unionQuery)

	var hasPrevInt, hasNextInt int
	err = r.db.QueryRow(
		finalQuery,
		firstTimestamp.UTC().Format("2006-01-02 15:04:05"),
		lastTimestamp.UTC().Format("2006-01-02 15:04:05"),
	).Scan(&hasPrevInt, &hasNextInt)

	if err != nil {
		return false, false, err
	}

	return hasPrevInt == 1, hasNextInt == 1, nil
}
