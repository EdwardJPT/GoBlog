package database

import (
	"database/sql"
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	iofs "github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations runs all embedded SQL migrations.
func RunMigrations(db *sql.DB) error {
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		log.Println(err)
		return errors.New("SQLite driver error")
	}

	src, err := iofs.New(Migrations(), "migrations")
	if err != nil {
		log.Println(err)
		return errors.New("iofs source error")
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		log.Println(err)
		return errors.New("Migrate init error")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Println(err)
		return errors.New("Migration error")
	}

	return nil
}
