package heimdall

import "time"

// Session represents an active user session.
type Session struct {
	SessionID  string       `json:"session_id"`
	UserID     string       `json:"user_id"`
	Device     DeviceInfo   `json:"device"`
	Location   LocationInfo `json:"location"`
	CreatedAt  time.Time    `json:"created_at"`
	TTLSeconds int64        `json:"ttl_seconds"`
}

// IsExpired returns true if the session has expired based on its TTL.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt())
}

// ExpiresAt returns the time when this session expires.
func (s *Session) ExpiresAt() time.Time {
	return s.CreatedAt.Add(time.Duration(s.TTLSeconds) * time.Second)
}

// DeviceInfo contains device information extracted from the HTTP request.
type DeviceInfo struct {
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
	Browser    string `json:"browser"`
	OS         string `json:"os"`
	DeviceType string `json:"device_type"` // mobile, desktop, tablet
}

// LocationInfo contains geographic location extracted from IP address.
type LocationInfo struct {
	IP        string  `json:"ip"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// RegisterResult is returned from RegisterSession with session info and alerts.
type RegisterResult struct {
	// Session is the newly created session. Nil if LimitExceeded is true.
	Session *Session `json:"session,omitempty"`

	// IsNewLocation is true if the user is logging in from an unusual location.
	IsNewLocation bool `json:"is_new_location"`

	// PreviousLocation is the last known location for comparison.
	// Only set if IsNewLocation is true.
	PreviousLocation *LocationInfo `json:"previous_location,omitempty"`

	// ActiveSessions contains all active sessions for this user.
	ActiveSessions []*Session `json:"active_sessions"`

	// LimitExceeded is true if the concurrent session limit was exceeded.
	// When true, the new session was NOT saved.
	LimitExceeded bool `json:"limit_exceeded"`
}
