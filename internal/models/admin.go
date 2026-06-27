package models

import (
	"database/sql"
	"log"
	"time"
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
	db    *sql.DB
	dbRAM *sql.DB // in-memory SQLite DB for rate limiting
}

const rateLimitInitSQL = `
		CREATE TABLE IF NOT EXISTS failed_attempts (
			ip TEXT,
			timestamp DATETIME
		);
		CREATE TABLE IF NOT EXISTS ip_blocks (
			ip TEXT PRIMARY KEY,
			blocked_until DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_ip_time ON failed_attempts(ip, timestamp);
	`

func NewAdminRepository(db *sql.DB, dbRAM *sql.DB) *AdminRepository {
	// Bootstrapping the in-memory tables when the repository is created
	// Kinda ugly but whatever
	_, err := dbRAM.Exec(rateLimitInitSQL)
	if err != nil {
		log.Fatalln("Failed to initialize RAM tables:", err)
		return nil
	}
	return &AdminRepository{db: db, dbRAM: dbRAM}
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

// Configuration constants
const (
	MaxFailedAttempts = 5
	WindowDuration    = 5 * time.Minute
	LockoutDuration   = 30 * time.Minute
)

// IsBlocked checks if the IP is currently in the penalty box
func (r *AdminRepository) IsBlocked(ip string) bool {
	query := `SELECT blocked_until FROM ip_blocks WHERE ip = ?`
	var blockedUntil time.Time
	err := r.dbRAM.QueryRow(query, ip).Scan(&blockedUntil)
	if err == sql.ErrNoRows {
		return false
	}
	if time.Now().Before(blockedUntil) {
		return true
	}
	// If the penalty time has expired, remove the block
	r.dbRAM.Exec("DELETE FROM ip_blocks WHERE ip = ?", ip)
	return false
}

// RecordFailure logs a failed attempt and checks if the sliding window
// threshold is breached
func (r *AdminRepository) RecordFailure(ip string) {
	now := time.Now()
	windowStart := now.Add(-WindowDuration)

	tx, err := r.dbRAM.Begin()
	if err != nil {
		log.Println("DB error:", err)
		return
	}
	defer tx.Rollback()

	// Log current specific failure timestamp
	tx.Exec(`INSERT INTO failed_attempts (ip, timestamp) VALUES (?, ?)`, ip, now)

	// Count failures strictly within the sliding window
	query := `
		SELECT COUNT(*) FROM failed_attempts
		WHERE ip = ? AND timestamp >= ?
	`
	var count int
	tx.QueryRow(query, ip, windowStart).Scan(&count)

	// Lock down the IP if it exceeds the limit
	if count >= MaxFailedAttempts {
		log.Printf("IP %s locked out for suspicious activity.\n", ip)
		query = `REPLACE INTO ip_blocks (ip, blocked_until) VALUES (?, ?)`
		tx.Exec(query, ip, now.Add(LockoutDuration))
	}
	tx.Commit()
}

// clearFailures resets the counter upon a successful login
func (r *AdminRepository) ClearFailures(ip string) {
	r.dbRAM.Exec(`DELETE FROM failed_attempts WHERE ip = ?`, ip)
}
