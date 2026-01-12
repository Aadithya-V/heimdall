package store

import "time"

// Session represents a user session for storage.
// This is a copy of the main Session type to avoid circular imports.
type Session struct {
	SessionID  string
	UserID     string
	DeviceIP   string
	DeviceUA   string
	Browser    string
	OS         string
	DeviceType string
	LocCity    string
	LocCountry string
	LocLat     float64
	LocLng     float64
	TTLSeconds int64
	CreatedAt  time.Time
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt())
}

// ExpiresAt returns the expiration time of the session.
func (s *Session) ExpiresAt() time.Time {
	return s.CreatedAt.Add(time.Duration(s.TTLSeconds) * time.Second)
}

// SessionStore defines the interface for session storage backends.
// Implementations must be safe for concurrent use.
type SessionStore interface {
	// Save persists a new session. If a session with the same ID exists,
	// it will be overwritten.
	Save(session *Session) error

	// Delete marks a session as invalidated (soft delete or hard delete based on your implementation).
	// The built-in provider implementations soft delete.
	// The session is kept for audit purposes but excluded from active queries.
	Delete(sessionID string) error

	// GetActiveByUser returns all non-expired, non-invalidated sessions for a user.
	// Sessions are ordered by CreatedAt descending (newest first).
	// Use [0] to get the latest session.
	GetActiveByUser(userID string) ([]*Session, error)

	// Close releases any resources held by the store.
	Close() error
}

// InvalidationCache defines the interface for tracking invalidated session IDs.
// Implementations must be safe for concurrent use.
type InvalidationCache interface {
	// Set marks a session ID as invalidated with the given TTL.
	// After TTL expires, the entry is automatically removed.
	Set(sessionID string, ttl time.Duration) error

	// Exists returns true if the session ID has been invalidated
	// and the TTL has not expired.
	Exists(sessionID string) (bool, error)

	// Close releases any resources held by the cache.
	Close() error
}
