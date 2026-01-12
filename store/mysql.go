package store

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLStore implements SessionStore using MySQL.
type MySQLStore struct {
	db *sql.DB
}

// NewMySQL creates a new MySQL session store.
// The DSN format is: user:password@tcp(host:port)/database
func NewMySQL(db *sql.DB) (*MySQLStore, error) {

	// Create schema
	if err := createMySQLSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &MySQLStore{db: db}, nil
}

// NewMySQLFromDSN creates a new MySQL session store from a DSN.
func NewMySQLFromDSN(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn+"?parseTime=true")
	if err != nil {
		return nil, fmt.Errorf("mysql: failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("mysql: failed to connect: %w", err)
	}

	return NewMySQL(db)
}

func createMySQLSchema(db *sql.DB) error {
	// NOTE: MySQL does not support partial indexes. For PostgreSQL, you could use:
	//   CREATE INDEX idx_active_sessions ON sessions (user_id, expires_at)
	//       WHERE invalidated_at IS NULL AND expires_at > NOW();
	// This would reduce index size by excluding invalidated and expired sessions.
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		session_id     VARCHAR(255) PRIMARY KEY,
		user_id        VARCHAR(255) NOT NULL,
		device_ip      VARCHAR(45),
		device_ua      TEXT,
		browser        VARCHAR(100),
		os             VARCHAR(100),
		device_type    VARCHAR(20),
		loc_city       VARCHAR(100),
		loc_country    VARCHAR(100),
		loc_lat        DECIMAL(10, 8),
		loc_lng        DECIMAL(11, 8),
		ttl_seconds    INT NOT NULL,
		created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at     TIMESTAMP AS (DATE_ADD(created_at, INTERVAL ttl_seconds SECOND)) STORED,
		invalidated_at TIMESTAMP NULL DEFAULT NULL,
		
		INDEX idx_sessions_user_active (user_id, expires_at, invalidated_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("mysql: failed to create schema: %w", err)
	}
	return nil
}

// Save persists a new session.
func (s *MySQLStore) Save(session *Session) error {
	query := `
	INSERT INTO sessions (
		session_id, user_id, device_ip, device_ua, browser, os, device_type,
		loc_city, loc_country, loc_lat, loc_lng, ttl_seconds, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		device_ip = VALUES(device_ip),
		device_ua = VALUES(device_ua),
		browser = VALUES(browser),
		os = VALUES(os),
		device_type = VALUES(device_type),
		loc_city = VALUES(loc_city),
		loc_country = VALUES(loc_country),
		loc_lat = VALUES(loc_lat),
		loc_lng = VALUES(loc_lng),
		ttl_seconds = VALUES(ttl_seconds),
		created_at = VALUES(created_at)
	`

	_, err := s.db.Exec(query,
		session.SessionID,
		session.UserID,
		session.DeviceIP,
		session.DeviceUA,
		session.Browser,
		session.OS,
		session.DeviceType,
		session.LocCity,
		session.LocCountry,
		session.LocLat,
		session.LocLng,
		session.TTLSeconds,
		session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("mysql: failed to save session: %w", err)
	}
	return nil
}

// Delete marks a session as invalidated (soft delete for audit trail).
func (s *MySQLStore) Delete(sessionID string) error {
	_, err := s.db.Exec(
		"UPDATE sessions SET invalidated_at = NOW() WHERE session_id = ?",
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("mysql: failed to invalidate session: %w", err)
	}
	return nil
}

// GetActiveByUser returns all non-expired, non-invalidated sessions for a user.
func (s *MySQLStore) GetActiveByUser(userID string) ([]*Session, error) {
	query := `
	SELECT session_id, user_id, device_ip, device_ua, browser, os, device_type,
		   loc_city, loc_country, loc_lat, loc_lng, ttl_seconds, created_at
	FROM sessions
	WHERE user_id = ? AND expires_at > NOW() AND invalidated_at IS NULL
	ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("mysql: failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session, err := scanMySQLSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: error iterating sessions: %w", err)
	}

	return sessions, nil
}

// Close closes the database connection.
func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func scanMySQLSession(rows *sql.Rows) (*Session, error) {
	var session Session
	err := rows.Scan(
		&session.SessionID,
		&session.UserID,
		&session.DeviceIP,
		&session.DeviceUA,
		&session.Browser,
		&session.OS,
		&session.DeviceType,
		&session.LocCity,
		&session.LocCountry,
		&session.LocLat,
		&session.LocLng,
		&session.TTLSeconds,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("mysql: failed to scan session: %w", err)
	}
	return &session, nil
}