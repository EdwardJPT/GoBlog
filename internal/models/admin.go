package models

import (
	"database/sql"
)

type SignInData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type Admin struct {
	ID       int64  `json:"admin_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetByUsername(username string) (*Admin, error) {
	query := `
		SELECT admin_id, username, password
		FROM admin
		WHERE username = ?
	`
	var admin Admin
	err := r.db.QueryRow(query, username).Scan(
		&admin.ID,
		&admin.Username,
		&admin.Password,
	)
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *AdminRepository) UpdateLastLogin(adminId int64) error {
	query := `
		UPDATE admin
		SET login_timestamp = DATETIME()
		WHERE admin_id = ?
	`
	_, err := r.db.Exec(query, adminId)
	return err
}
