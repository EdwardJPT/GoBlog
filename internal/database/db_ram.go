package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLiteRAM() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("create/open sqlite path: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ping the database: %w", err)
	}

	return db, nil
}
