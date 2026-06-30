package models

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// The shape of content that we returned if visitor asked for contents with a
// certain tag
type TagItem struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	Tags      string    `json:"tags"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RepoLink  string    `json:"repo_link"`
	DemoLink  string    `json:"demo_link"`
}

type TagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

type Tag struct {
	Name  string
	Count int
}

// tagTable pairs a table name with the column that holds its human-readable
// title, and records whether the table has a status column to filter on.
type tagTable struct {
	name       string
	titleField string
	hasStatus  bool // true: AND status = 'published' is added to queries
}

// tagTables is the single source of truth for every content table.
// Update titleField or hasStatus here if the schema ever changes.
var tagTables = []tagTable{
	{"posts", "title", true},
	{"notes", "title", true},
	{"bookmarks", "title", true},
	{"projects", "name", true},
	{"opensource_contributions", "project_name", false}, // no status column
}

// statusClause returns an AND fragment for status filtering when the table
// supports it, or an empty string otherwise. Slots directly into a WHERE
// chain that already has at least one preceding condition.
func statusClause(t tagTable) string {
	if t.hasStatus {
		return "AND status = 'published'"
	}
	return ""
}

func isFilter(filter string) string {
	if filter != "" {
		return "AND (',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')"
	}
	return ""
}

// GetAllTags queries every tag from all content tables, then returns a
// deduplicated, alphabetically-sorted slice of Tag values — each carrying
// the total count of how many rows across all tables reference that tag.
//
// notes tag#001 (<- search this to see the relation in another place)
// Currently the search feature also return another tags that are exist
// in the same content. For example, if there's post a that has tags, go,backend,server
// and if we search for 'go', it will also return backend and server too
//
// Only published rows are counted (tables without a status column are
// included in full). Empty or blank tag segments are silently skipped.
func (r *TagRepository) GetAllTags(filter string, selectedTypes []string) ([]Tag, error) {
	counts := make(map[string]int)

	if len(selectedTypes) == 0 {
		selectedTypes = []string{
			"posts",
			"bookmarks",
			"notes",
			"projects",
			"opensource_contributions",
		}
	}

	for _, table := range tagTables {
		if !slices.Contains(selectedTypes, table.name) {
			continue
		}

		// #nosec G201 — table names come from a hardcoded slice, not user input.
		query := fmt.Sprintf(
			`SELECT tags FROM %s WHERE tags IS NOT NULL AND tags != '' %s %s`,
			table.name,
			isFilter(filter),
			statusClause(table),
		)

		args := []any{}
		if filter != "" {
			args = append(args, filter)
		}

		rows, err := r.db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var raw string
			if err := rows.Scan(&raw); err != nil {
				return nil, err
			}
			for tag := range strings.SplitSeq(raw, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					counts[strings.ToLower(tag)]++
				}
			}
		}
	}

	tags := make([]Tag, 0, len(counts))
	for name, count := range counts {
		tags = append(tags, Tag{Name: name, Count: count})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})
	return tags, nil
}

// GetAllTagNames is a convenience wrapper that returns only the tag name
// strings, sorted alphabetically — useful when counts are not needed.
func (r *TagRepository) GetAllTagNames() ([]string, error) {
	tags, err := r.GetAllTags("", []string{
		"posts",
		"bookmarks",
		"notes",
		"projects",
		"opensource_contributions",
	})
	if err != nil {
		return nil, err
	}

	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names, nil
}

// GetContentByTag returns every content item across all tables whose tags
// column contains the given tag. The match is exact and case-insensitive.
//
// Because tags are stored as a comma-separated string (e.g. "go,backend,sqlite"),
// we wrap both the column and the needle with commas so every position is handled
// uniformly by a single LIKE predicate:
//
//	(',' || LOWER(tags) || ',') LIKE '%,' || LOWER(?) || ',%'
//
// This correctly handles a tag at the start, middle, or end of the list and
// prevents false positives where one tag is a prefix of another (e.g. "go"
// must not match "golang").
//
// Only published rows are returned (tables without a status column are
// always included). Results are grouped by source table in tagTables order.
func (r *TagRepository) GetContentByTag(
	tag string,
	selectedTypes []string,
	encodedCursor string,
	isPrev bool,
	limit int,
) ([]*TagItem, error) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return nil, nil
	}

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
				updated_at,
				'' AS repo_link,
				'' AS demo_link
			FROM posts
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
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
				updated_at,
				'' AS repo_link,
				'' AS demo_link
			FROM bookmarks
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
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
				updated_at,
				'' AS repo_link,
				'' AS demo_link
			FROM notes
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
		"projects": `
			SELECT
				id,
				name AS title,
				slug,
				COALESCE(description, '') AS content,
				'project' AS type,
				COALESCE(tags, '') AS tags,
				'' AS url,
				created_at,
				updated_at,
				COALESCE(repo_link, '') AS repo_link,
				COALESCE(demo_link, '') AS demo_link
			FROM projects
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
		"opensource_contributions": `
			SELECT
				id,
				project_name AS title,
				slug,
				COALESCE(contribution_summary, description, '') AS content,
				'contribution' AS type,
				COALESCE(tags, '') AS tags,
				'' AS url,
				created_at,
				updated_at,
				COALESCE(link, '') AS repo_link,
				'' AS demo_link
			FROM opensource_contributions
			WHERE
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
	}

	if len(selectedTypes) == 0 {
		selectedTypes = []string{
			"posts",
			"bookmarks",
			"notes",
			"projects",
			"opensource_contributions",
		}
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
				SELECT * FROM (
					%s
					AND updated_at %s ?
					ORDER BY updated_at %s
					LIMIT ?
				)`,
				baseQuery,
				operator,
				order,
			)
			activeQueries = append(activeQueries, subQuery)
			args = append(args, tag, cursor, limit)
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
			updated_at,
			repo_link,
			demo_link
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

	var feed []*TagItem
	for rows.Next() {
		var item TagItem
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
			&item.RepoLink,
			&item.DemoLink,
		); err != nil {
			return nil, err
		}
		feed = append(feed, &item)
	}

	return feed, nil
}

func (r *TagRepository) HasPublishedRowsAroundCursor(
	tag string,
	selectedTypes []string,
	firstTimestamp time.Time,
	lastTimestamp time.Time,
) (hasPrev bool, hasNext bool, err error) {
	allowedBaseQueries := map[string]string{
		"posts": `
			SELECT updated_at
			FROM posts
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
		"bookmarks": `
			SELECT updated_at
			FROM bookmarks
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
		"notes": `
			SELECT updated_at
			FROM notes
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
		"projects": `
			SELECT updated_at
			FROM projects
			WHERE
				status = 'published'
				AND
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
		"opensource_contributions": `
			SELECT updated_at
			FROM opensource_contributions
			WHERE
				tags IS NOT NULL AND tags != ''
				AND
				(',' || LOWER(tags) || ',') LIKE ('%%,' || LOWER(?) || ',%%')
		`,
	}

	if len(selectedTypes) == 0 {
		selectedTypes = []string{
			"posts",
			"bookmarks",
			"notes",
			"projects",
			"opensource_contributions",
		}
	}

	var activeQueries []string
	var args []any

	for _, t := range selectedTypes {
		if baseQuery, exists := allowedBaseQueries[t]; exists {
			activeQueries = append(activeQueries, baseQuery)
			args = append(args, tag)
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

	args = append(
		args,
		firstTimestamp.UTC().Format("2006-01-02 15:04:05"),
		lastTimestamp.UTC().Format("2006-01-02 15:04:05"),
	)

	var hasPrevInt, hasNextInt int
	if err = r.db.QueryRow(finalQuery, args...).Scan(
		&hasPrevInt,
		&hasNextInt,
	); err != nil {
		return false, false, err
	}

	return hasPrevInt == 1, hasNextInt == 1, nil
}

func replaceTagInString(tagsStr, oldTag, newTag string) string {
	parts := strings.Split(tagsStr, ",")
	var newParts []string
	for _, p := range parts {
		clean := strings.TrimSpace(p)
		if clean == "" {
			continue
		}
		if clean == oldTag {
			if newTag != "" { // If not deleting
				newParts = append(newParts, newTag)
			}
		} else {
			newParts = append(newParts, clean)
		}
	}
	return strings.Join(newParts, ",")
}

// RenameTag updates a tag's name across all 5 tables
func (r *TagRepository) RenameTag(oldName, newName string) error {
	oldName = strings.TrimSpace(strings.ToLower(oldName))
	newName = strings.TrimSpace(strings.ToLower(newName))

	if oldName == "" || newName == "" || oldName == newName {
		return nil
	}

	for _, table := range tagTables {
		if err := r.replaceTagInTable(table.name, oldName, newName); err != nil {
			return err
		}
	}
	return nil
}

// DeleteTag completely removes a tag from all 5 tables
func (r *TagRepository) DeleteTag(name string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil
	}

	for _, table := range tagTables {
		// Passing an empty string as newName instructs the helper to remove the tag
		if err := r.replaceTagInTable(table.name, name, ""); err != nil {
			return err
		}
	}
	return nil
}

// replaceTagInTable fetches rows containing the tag, manipulates the string safely, and updates the row
func (r *TagRepository) replaceTagInTable(tableName, oldTag, newTag string) error {
	// Use LIKE to narrow down rows, minimizing the amount of data we process in Go
	query := fmt.Sprintf(`SELECT id, tags FROM %s WHERE tags LIKE ?`, tableName)
	likePattern := "%" + oldTag + "%"

	rows, err := r.db.Query(query, likePattern)
	if err != nil {
		return err
	}
	defer rows.Close()

	type updateData struct {
		id      int64
		newTags string
	}
	var updates []updateData

	for rows.Next() {
		var id int64
		var currentTags string
		if err := rows.Scan(&id, &currentTags); err != nil {
			return err
		}

		updatedTagsStr, changed := processTagsString(currentTags, oldTag, newTag)
		if changed {
			updates = append(updates, updateData{id: id, newTags: updatedTagsStr})
		}
	}

	// Perform the updates for rows that actually contained the exact tag
	updateQuery := fmt.Sprintf(`UPDATE %s SET tags = ? WHERE id = ?`, tableName)
	for _, u := range updates {
		if _, err := r.db.Exec(updateQuery, u.newTags, u.id); err != nil {
			return err
		}
	}

	return nil
}

// processTagsString handles the comma-separated manipulation safely (prevents substring matching bugs)
func processTagsString(tagsStr, oldTag, newTag string) (string, bool) {
	parts := strings.Split(tagsStr, ",")
	changed := false
	var updated []string

	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}

		if strings.EqualFold(t, oldTag) {
			changed = true
			if newTag != "" {
				// Avoid duplicate tags if renaming to a tag that already exists on this item
				exists := false
				for _, existing := range updated {
					if strings.EqualFold(existing, newTag) {
						exists = true
						break
					}
				}
				if !exists {
					updated = append(updated, newTag)
				}
			}
		} else {
			updated = append(updated, t)
		}
	}

	return strings.Join(updated, ","), changed
}
