package database

import (
	"embed"
	"io/fs"
)

// Embed everything inside migrations/
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

func Migrations() fs.FS {
	return migrationFiles
}
