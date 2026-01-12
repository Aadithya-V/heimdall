package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements SessionStore using SQLite.
// It uses the pure Go modernc.org/sqlite driver.
type SQLiteStore struct {
	db *sql.DB
}


// NewSQLite creates a new SQLite session store.
// The database file is created if it doesn't exist.
func NewSQLite(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: failed to enable WAL mode: %w", err)
	}

	// Create sessions table
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		session_id     TEXT PRIMARY KEY,
		user_id        TEXT NOT NULL,
		device_ip      TEXT,
		device_ua      TEXT,
		browser        TEXT,
		os             TEXT,
		device_type    TEXT,
		loc_city       TEXT,
		loc_country    TEXT,
		loc_lat        REAL,
		loc_lng        REAL,
		ttl_seconds    INTEGER NOT NULL,
		created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at     DATETIME NOT NULL,
		invalidated_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_active 
		ON sessions (user_id, expires_at, invalidated_at);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("sqlite: failed to create schema: %w", err)
	}
	return nil
}

// Set marks a session ID as invalidated.
// Note: This is typically already done by SessionStore.Delete(), so this is a no-op
// if the session was already invalidated. The TTL parameter is ignored since
// invalidated sessions are kept permanently for audit.
func (s *SQLiteStore) Set(sessionID string, ttl time.Duration) error {
	// Update invalidated_at only if not already set (Delete already sets it)
	_, err := s.db.Exec(
		"UPDATE sessions SET invalidated_at = datetime('now') WHERE session_id = ? AND invalidated_at IS NULL",
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to set invalidation: %w", err)
	}
	return nil
}

// Exists returns true if the session ID has been invalidated.
// Checks the invalidated_at column in the sessions table.
func (s *SQLiteStore) Exists(sessionID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE session_id = ? AND invalidated_at IS NOT NULL",
		sessionID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("sqlite: failed to check invalidation: %w", err)
	}
	return count > 0, nil
}

// Save persists a new session.
func (s *SQLiteStore) Save(session *Session) error {
	query := `
	INSERT OR REPLACE INTO sessions (
		session_id, user_id, device_ip, device_ua, browser, os, device_type,
		loc_city, loc_country, loc_lat, loc_lng, ttl_seconds, created_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	expiresAt := session.ExpiresAt()

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
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf("sqlite: failed to save session: %w", err)
	}
	return nil
}

// Delete marks a session as invalidated (soft delete for audit trail).
func (s *SQLiteStore) Delete(sessionID string) error {
	_, err := s.db.Exec(
		"UPDATE sessions SET invalidated_at = datetime('now') WHERE session_id = ?",
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to invalidate session: %w", err)
	}
	return nil
}

// GetActiveByUser returns all non-expired, non-invalidated sessions for a user.
func (s *SQLiteStore) GetActiveByUser(userID string) ([]*Session, error) {
	query := `
	SELECT session_id, user_id, device_ip, device_ua, browser, os, device_type,
		   loc_city, loc_country, loc_lat, loc_lng, ttl_seconds, created_at
	FROM sessions
	WHERE user_id = ? AND expires_at > datetime('now') AND invalidated_at IS NULL
	ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: error iterating sessions: %w", err)
	}

	return sessions, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// scanSession scans a session from sql.Rows.
func scanSession(rows *sql.Rows) (*Session, error) {
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
		return nil, fmt.Errorf("sqlite: failed to scan session: %w", err)
	}
	return &session, nil
}
