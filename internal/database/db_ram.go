package database

import (
	"database/sql"
	"errors"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLiteRAM() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Println(err)
		return nil, errors.New("Fail to create/open the database")
	}
	db.SetMaxOpenConns(1)

	if err = db.Ping(); err != nil {
		log.Println(err)
		return nil, errors.New("Fail to ping the database")
	}

	return db, nil
}
