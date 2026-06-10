package database

import (
	"database/sql"
	"errors"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLiteRAM() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Println(err)
		return nil, errors.New("Fail to create/open the database")
	}

	if err = db.Ping(); err != nil {
		log.Println(err)
		return nil, errors.New("Fail to ping the database")
	}

	return db, nil
}
