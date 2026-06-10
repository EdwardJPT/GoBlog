package models

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

type BlogItem struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	Tags      string    `json:"tags"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WritingRepository struct {
	db *sql.DB
}

func NewWritingRepository(db *sql.DB) *WritingRepository {
	return &WritingRepository{db: db}
}

func (r *WritingRepository) GetBlogFeed(
	selectedTypes []string,
	encodedCursor string,
	isPrev bool,
	limit int,
) ([]BlogItem, error) {
	allowedBaseQueries := map[string]string{
		"posts": `
			SELECT
				id,
				title,
				slug,
				content,
				'post' AS type,
				COALESCE(tags, '') AS tags,
				'' AS url,
				created_at,
				updated_at
			FROM posts
			WHERE status = 'published'`,
		"bookmarks": `
			SELECT
				id,
				title,
				slug,
				COALESCE(description, '') AS content,
				'bookmark' AS type,
				COALESCE(tags, '') AS tags,
				url,
				created_at,
				updated_at
			FROM bookmarks
			WHERE status = 'published'`,
		"notes": `
			SELECT
				id,
				title,
				slug,
				content,
				'note' AS type,
				COALESCE(tags, '') AS tags,
				'' AS url,
				created_at,
				updated_at
			FROM notes
			WHERE status = 'published'`,
	}

	if len(selectedTypes) == 0 {
		selectedTypes = []string{"posts", "bookmarks", "notes"}
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
			subQuery := fmt.Sprintf(`
				SELECT * FROM (%s AND updated_at %s ? ORDER BY updated_at %s LIMIT ?)`,
				baseQuery,
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
			type,
			tags,
			url,
			created_at,
			updated_at
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

	var feed []BlogItem
	for rows.Next() {
		var item BlogItem
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Slug,
			&item.Content,
			&item.Type,
			&item.Tags,
			&item.URL,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		feed = append(feed, item)
	}

	return feed, nil
}

func (r *WritingRepository) HasPublishedRowsAroundCursor(
	selectedTypes []string,
	firstTimestamp time.Time,
	lastTimestamp time.Time,
) (hasPrev bool, hasNext bool, err error) {
	allowedBaseQueries := map[string]string{
		"posts": `
			SELECT updated_at
			FROM posts
			WHERE status = 'published'`,
		"bookmarks": `
			SELECT updated_at
			FROM bookmarks
			WHERE status = 'published'`,
		"notes": `
			SELECT updated_at
			FROM notes
			WHERE status = 'published'`,
	}

	if len(selectedTypes) == 0 {
		selectedTypes = []string{"posts", "bookmarks", "notes"}
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
				WHERE updated_at > ?
			) AS has_above,
			EXISTS(
				SELECT 1
				FROM combined
				WHERE updated_at < ?
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
