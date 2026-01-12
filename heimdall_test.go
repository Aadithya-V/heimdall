package heimdall

import (
	"os"
	"testing"
	"time"

	"github.com/aadithya-v/heimdall/store"
)

func TestHeimdallBasicFlow(t *testing.T) {
	// Use in-memory stores for testing
	h, err := newTestHeimdall()
	if err != nil {
		t.Fatalf("Failed to create Heimdall: %v", err)
	}
	defer h.Close()

	userID := "user123"
	sessionID := "session456"

	device := DeviceInfo{
		IP:         "8.8.8.8",
		UserAgent:  "Mozilla/5.0",
		Browser:    "Chrome",
		OS:         "Windows",
		DeviceType: "desktop",
	}

	location := LocationInfo{
		IP:        "8.8.8.8",
		City:      "Mountain View",
		Country:   "United States",
		Latitude:  37.3861,
		Longitude: -122.0839,
	}

	// Register a session
	result, err := h.RegisterSession(userID, sessionID, device, location, 3)
	if err != nil {
		t.Fatalf("Failed to register session: %v", err)
	}

	if result.LimitExceeded {
		t.Error("LimitExceeded should be false for first session")
	}

	if result.Session == nil {
		t.Error("Session should not be nil")
	}

	if result.Session.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, result.Session.SessionID)
	}

	// List sessions
	sessions, err := h.ListSessions(userID)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// Check session is not invalidated
	invalidated, err := h.IsSessionInvalidated(sessionID)
	if err != nil {
		t.Fatalf("Failed to check invalidation: %v", err)
	}

	if invalidated {
		t.Error("Session should not be invalidated")
	}

	// Invalidate session
	if err := h.InvalidateSession(sessionID); err != nil {
		t.Fatalf("Failed to invalidate session: %v", err)
	}

	// Check session is now invalidated
	invalidated, err = h.IsSessionInvalidated(sessionID)
	if err != nil {
		t.Fatalf("Failed to check invalidation: %v", err)
	}

	if !invalidated {
		t.Error("Session should be invalidated")
	}

	// List sessions should be empty now
	sessions, err = h.ListSessions(userID)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after invalidation, got %d", len(sessions))
	}
}

func TestConcurrentSessionLimit(t *testing.T) {
	h, err := newTestHeimdall()
	if err != nil {
		t.Fatalf("Failed to create Heimdall: %v", err)
	}
	defer h.Close()

	userID := "user123"
	device := DeviceInfo{IP: "8.8.8.8", Browser: "Chrome", OS: "Windows"}
	location := LocationInfo{IP: "8.8.8.8", City: "NYC", Country: "US"}

	// Register 2 sessions (limit is 2)
	for i := 1; i <= 2; i++ {
		result, err := h.RegisterSession(userID, "session"+string(rune('0'+i)), device, location, 2)
		if err != nil {
			t.Fatalf("Failed to register session %d: %v", i, err)
		}
		if result.LimitExceeded {
			t.Errorf("Session %d should not exceed limit", i)
		}
	}

	// Third session should be rejected
	result, err := h.RegisterSession(userID, "session3", device, location, 2)
	if err != nil {
		t.Fatalf("Failed to register session 3: %v", err)
	}

	if !result.LimitExceeded {
		t.Error("Third session should exceed limit")
	}

	if result.Session != nil {
		t.Error("Session should be nil when limit exceeded")
	}

	if len(result.ActiveSessions) != 2 {
		t.Errorf("Should return 2 active sessions, got %d", len(result.ActiveSessions))
	}
}

func TestNewLocationDetection(t *testing.T) {
	h, err := newTestHeimdall()
	if err != nil {
		t.Fatalf("Failed to create Heimdall: %v", err)
	}
	defer h.Close()

	userID := "user123"
	device := DeviceInfo{IP: "8.8.8.8", Browser: "Chrome", OS: "Windows"}

	// First session from NYC
	nyc := LocationInfo{
		IP:        "8.8.8.8",
		City:      "New York",
		Country:   "United States",
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	result1, err := h.RegisterSession(userID, "session1", device, nyc, 10)
	if err != nil {
		t.Fatalf("Failed to register first session: %v", err)
	}

	if result1.IsNewLocation {
		t.Error("First session should not be flagged as new location")
	}

	// Second session from London (far away)
	london := LocationInfo{
		IP:        "1.1.1.1",
		City:      "London",
		Country:   "United Kingdom",
		Latitude:  51.5074,
		Longitude: -0.1278,
	}

	result2, err := h.RegisterSession(userID, "session2", device, london, 10)
	if err != nil {
		t.Fatalf("Failed to register second session: %v", err)
	}

	if !result2.IsNewLocation {
		t.Error("Second session from London should be flagged as new location")
	}

	if result2.PreviousLocation == nil {
		t.Error("PreviousLocation should not be nil")
	}

	if result2.PreviousLocation.City != "New York" {
		t.Errorf("Previous location should be New York, got %s", result2.PreviousLocation.City)
	}
}

func TestHaversineDistance(t *testing.T) {
	// NYC to London should be approximately 5,570 km
	nyc := struct{ lat, lng float64 }{40.7128, -74.0060}
	london := struct{ lat, lng float64 }{51.5074, -0.1278}

	distance := HaversineDistance(nyc.lat, nyc.lng, london.lat, london.lng)

	// Allow 1% margin of error
	expected := 5570.0
	margin := expected * 0.01

	if distance < expected-margin || distance > expected+margin {
		t.Errorf("Expected distance ~%f km, got %f km", expected, distance)
	}
}

// newTestHeimdall creates a Heimdall instance with in-memory stores for testing.
func newTestHeimdall() (*Heimdall, error) {
	// Create temp directory for SQLite
	tmpDir, err := os.MkdirTemp("", "heimdall-test-*")
	if err != nil {
		return nil, err
	}

	sqliteStore, err := store.NewSQLite(tmpDir + "/test.db")
	if err != nil {
		return nil, err
	}

	return New(Config{
		SessionStore:           sqliteStore,
		InvalidationCache:      sqliteStore,
		SessionTTL:             1 * time.Hour,
		InvalidationTTL:        24 * time.Hour,
		NewLocationThresholdKM: 100,
	})
}
