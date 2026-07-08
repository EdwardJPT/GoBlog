package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLite() (*sql.DB, error) {
	// Ensure the directory exists before attempting to open the database file
	if err := os.MkdirAll("./sqlite_files", 0755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	dbUri := "./sqlite_files/blog.db?_foreign_keys=yes&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dbUri)
	if err != nil {
		return nil, fmt.Errorf("create/open sqlite path: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ping the database: %w", err)
	}

	// Set foreign_keys = ON with query too for safety
	sqlStmt := `PRAGMA foreign_keys = ON;`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("set the foreign_keys = on: %w", err)
	}

	return db, nil
}

func CheckFK(db *sql.DB) int {
	var fkEnabled int
	db.QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled)
	return fkEnabled
}

func CheckJournalMode(db *sql.DB) string {
	var journalMode string
	db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	return journalMode
}

func CheckSynchronous(db *sql.DB) string {
	var synchronous int
	syncMode := map[int]string{0: "OFF", 1: "NORMAL", 2: "FULL", 3: "EXTRA"}
	db.QueryRow("PRAGMA synchronous;").Scan(&synchronous)
	return syncMode[synchronous]
}

func CheckCacheSize(db *sql.DB) int {
	var cacheSize int
	db.QueryRow("PRAGMA cache_size;").Scan(&cacheSize)
	return cacheSize
}
