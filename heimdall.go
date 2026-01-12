package heimdall

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aadithya-v/heimdall/store"
)

// Heimdall is the main SDK interface for session management.
type Heimdall struct {
	config      Config
	sessions    store.SessionStore
	invalidated store.InvalidationCache
	geoip       *GeoIPReader
}

// New creates a new Heimdall instance with the given configuration.
// If SessionStore or InvalidationCache are not provided, defaults are used:
// - SessionStore: SQLite (creates heimdall.db)
// - InvalidationCache: SQLite (uses sessions table's invalidated_at column)
func New(cfg Config) (*Heimdall, error) {
	cfg.applyDefaults()

	h := &Heimdall{
		config: cfg,
	}

	// Initialize session store (default: SQLite)
	if cfg.SessionStore != nil {
		h.sessions = cfg.SessionStore
	} else {
		sqliteStore, err := store.NewSQLite(cfg.DatabasePath)
		if err != nil {
			return nil, fmt.Errorf("heimdall: failed to initialize SQLite store: %w", err)
		}
		h.sessions = sqliteStore
		h.invalidated = sqliteStore
	}

	// Initialize invalidation cache (default: SQLite using sessions table)
	if cfg.InvalidationCache != nil {
		h.invalidated = cfg.InvalidationCache
	}

	// Initialize GeoIP reader if path is provided
	if cfg.GeoIPDatabasePath != "" {
		geoip, err := NewGeoIPReader(cfg.GeoIPDatabasePath)
		if err != nil {
			return nil, fmt.Errorf("heimdall: failed to initialize GeoIP: %w", err)
		}
		h.geoip = geoip
	}

	return h, nil
}

// Close releases all resources held by Heimdall.
// Should be called when the application shuts down.
func (h *Heimdall) Close() error {
	var errs []error

	if h.sessions != nil {
		if err := h.sessions.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if h.invalidated != nil {
		if err := h.invalidated.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if h.geoip != nil {
		if err := h.geoip.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("heimdall: errors during close: %v", errs)
	}
	return nil
}

// ExtractRequestInfo extracts device and location information from an HTTP request.
// If GeoIP is not configured, location will contain only the IP address.
func (h *Heimdall) ExtractRequestInfo(r *http.Request) (DeviceInfo, LocationInfo, error) {
	device := ExtractDeviceInfo(r)

	if h.geoip != nil {
		loc, err := h.geoip.Lookup(device.IP)
		if err != nil {
			// Return device info with partial location (IP only)
			return device, LocationInfo{IP: device.IP}, nil
		}
		return device, *loc, nil
	}

	// GeoIP not configured, return IP only
	return device, LocationInfo{IP: device.IP}, nil
}

// RegisterSession registers a new session for the user.
//
// concurrentLimit 0 means no limit.
// Otherwise, if the number of active sessions equals or exceeds concurrentLimit,
// the new session is NOT saved and LimitExceeded is set to true.
// The caller should then prompt the user to invalidate an existing session.
//
// If the user is logging in from a new location (distance > NewLocationThresholdKM),
// IsNewLocation is set to true and PreviousLocation contains the last known location.
func (h *Heimdall) RegisterSession(
	userID, sessionID string,
	device DeviceInfo,
	location LocationInfo,
	concurrentLimit int,
) (*RegisterResult, error) {
	result := &RegisterResult{}

	// Get all active sessions for the user
	activeSessions, err := h.sessions.GetActiveByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("heimdall: failed to get active sessions: %w", err)
	}

	// Convert to public Session type
	result.ActiveSessions = make([]*Session, len(activeSessions))
	for i, s := range activeSessions {
		result.ActiveSessions[i] = storeToSession(s)
	}

	// Check concurrent session limit
	if concurrentLimit > 0 && len(activeSessions) >= concurrentLimit {
		result.LimitExceeded = true
		return result, nil
	}

	// Check for new location
	if len(activeSessions) > 0 {
		latestSession := activeSessions[0] // Already sorted by created_at desc
		prevLocation := LocationInfo{
			IP:        latestSession.DeviceIP,
			City:      latestSession.LocCity,
			Country:   latestSession.LocCountry,
			Latitude:  latestSession.LocLat,
			Longitude: latestSession.LocLng,
		}

		if IsNewLocation(prevLocation, location, h.config.NewLocationThresholdKM) {
			result.IsNewLocation = true
			result.PreviousLocation = &prevLocation
		}
	}

	// Create and save the new session
	now := time.Now()
	storeSession := &store.Session{
		SessionID:  sessionID,
		UserID:     userID,
		DeviceIP:   device.IP,
		DeviceUA:   device.UserAgent,
		Browser:    device.Browser,
		OS:         device.OS,
		DeviceType: device.DeviceType,
		LocCity:    location.City,
		LocCountry: location.Country,
		LocLat:     location.Latitude,
		LocLng:     location.Longitude,
		TTLSeconds: int64(h.config.SessionTTL.Seconds()),
		CreatedAt:  now,
	}

	if err := h.sessions.Save(storeSession); err != nil {
		return nil, fmt.Errorf("heimdall: failed to save session: %w", err)
	}

	// Build result session
	result.Session = &Session{
		SessionID:  sessionID,
		UserID:     userID,
		Device:     device,
		Location:   location,
		CreatedAt:  now,
		TTLSeconds: int64(h.config.SessionTTL.Seconds()),
	}

	// Add new session to active sessions list
	result.ActiveSessions = append([]*Session{result.Session}, result.ActiveSessions...)

	return result, nil
}

// InvalidateSession marks a session as invalidated.
// The session ID is stored in the invalidation cache with the configured TTL.
// The session is also deleted from the session store.
func (h *Heimdall) InvalidateSession(sessionID string) error {
	// Delete from session store
	if err := h.sessions.Delete(sessionID); err != nil {
		return fmt.Errorf("heimdall: failed to delete session: %w", err)
	}

	// Add to invalidation cache
	if err := h.invalidated.Set(sessionID, h.config.InvalidationTTL); err != nil {
		return fmt.Errorf("heimdall: failed to set invalidation: %w", err)
	}

	return nil
}

// IsSessionInvalidated checks if a session has been invalidated.
// Returns true if the session ID was explicitly invalidated and the
// invalidation TTL has not expired.
func (h *Heimdall) IsSessionInvalidated(sessionID string) (bool, error) {
	return h.invalidated.Exists(sessionID)
}

// ListSessions returns all active (non-expired) sessions for a user.
// Sessions are ordered by creation time, newest first.
func (h *Heimdall) ListSessions(userID string) ([]*Session, error) {
	storeSessions, err := h.sessions.GetActiveByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("heimdall: failed to list sessions: %w", err)
	}

	sessions := make([]*Session, len(storeSessions))
	for i, s := range storeSessions {
		sessions[i] = storeToSession(s)
	}

	return sessions, nil
}

// storeToSession converts a store.Session to a public Session.
func storeToSession(s *store.Session) *Session {
	return &Session{
		SessionID: s.SessionID,
		UserID:    s.UserID,
		Device: DeviceInfo{
			IP:         s.DeviceIP,
			UserAgent:  s.DeviceUA,
			Browser:    s.Browser,
			OS:         s.OS,
			DeviceType: s.DeviceType,
		},
		Location: LocationInfo{
			IP:        s.DeviceIP,
			City:      s.LocCity,
			Country:   s.LocCountry,
			Latitude:  s.LocLat,
			Longitude: s.LocLng,
		},
		CreatedAt:  s.CreatedAt,
		TTLSeconds: s.TTLSeconds,
	}
}
