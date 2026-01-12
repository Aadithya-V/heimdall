package store

import (
	"sync"
	"time"
)

// MemoryCache implements InvalidationCache using an in-memory map.
// Expired entries are cleaned up periodically.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]time.Time // sessionID -> expiresAt

	// For periodic cleanup
	stopCleanup chan struct{}
}

// NewMemoryCache creates a new in-memory invalidation cache.
// It starts a background goroutine that periodically cleans up expired entries.
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		entries:     make(map[string]time.Time),
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup every other day
	go cache.cleanupLoop(48 * time.Hour)

	return cache
}

// Set marks a session ID as invalidated with the given TTL.
func (c *MemoryCache) Set(sessionID string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[sessionID] = time.Now().Add(ttl)
	return nil
}

// Exists returns true if the session ID has been invalidated and not expired.
func (c *MemoryCache) Exists(sessionID string) (bool, error) {
	c.mu.RLock()
	_, exists := c.entries[sessionID]
	c.mu.RUnlock()

	if !exists {
		return false, nil
	}

	return true, nil
}

// Close stops the background cleanup goroutine.
func (c *MemoryCache) Close() error {
	close(c.stopCleanup)
	return nil
}

// cleanupLoop periodically removes expired entries.
func (c *MemoryCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes all expired entries.
// todo: do in batches to avoid locking the map for too long
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for sessionID, expiresAt := range c.entries {
		if now.After(expiresAt) {
			delete(c.entries, sessionID)
		}
	}
}

// MemorySessionStore implements SessionStore using an in-memory map.
// This is useful for testing but not recommended for production.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session        // sessionID -> Session
	byUser   map[string]map[string]bool // userID -> set of sessionIDs
}

// NewMemorySessionStore creates a new in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
		byUser:   make(map[string]map[string]bool),
	}
}

// Save persists a new session.
func (s *MemorySessionStore) Save(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store session
	s.sessions[session.SessionID] = session

	// Index by user
	if s.byUser[session.UserID] == nil {
		s.byUser[session.UserID] = make(map[string]bool)
	}
	s.byUser[session.UserID][session.SessionID] = true

	return nil
}

// Delete removes a session by its ID.
func (s *MemorySessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil
	}

	// Remove from user index
	if userSessions, ok := s.byUser[session.UserID]; ok {
		delete(userSessions, sessionID)
		if len(userSessions) == 0 {
			delete(s.byUser, session.UserID)
		}
	}

	// Remove session
	delete(s.sessions, sessionID)
	return nil
}

// GetActiveByUser returns all non-expired sessions for a user.
func (s *MemorySessionStore) GetActiveByUser(userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionIDs, exists := s.byUser[userID]
	if !exists {
		return nil, nil
	}

	var active []*Session
	now := time.Now()

	for sessionID := range sessionIDs {
		session := s.sessions[sessionID]
		if session != nil && now.Before(session.ExpiresAt()) {
			active = append(active, session)
		}
	}

	// Sort by CreatedAt descending
	for i := 0; i < len(active)-1; i++ {
		for j := i + 1; j < len(active); j++ {
			if active[j].CreatedAt.After(active[i].CreatedAt) {
				active[i], active[j] = active[j], active[i]
			}
		}
	}

	return active, nil
}

// Close is a no-op for the memory store.
func (s *MemorySessionStore) Close() error {
	return nil
}
