package database

import (
	"database/sql"
	"errors"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLite() (*sql.DB, error) {
	dbUri := "./sqlite_files/blog.db?_foreign_keys=yes&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dbUri)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Fail to create/open the database")
	}

	if err = db.Ping(); err != nil {
		log.Println(err)
		return nil, errors.New("Fail to ping the database")
	}

	// Set foreign_keys = ON with query too for safety
	sqlStmt := `PRAGMA foreign_keys = ON;`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Fail to set foreign_keys = on")
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

func CheckCacheSize(db *sql.DB) int {
	var cacheSize int
	db.QueryRow("PRAGMA cache_size;").Scan(&cacheSize)
	return cacheSize
}
