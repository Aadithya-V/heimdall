package heimdall

import (
	"time"

	"github.com/aadithya-v/heimdall/store"
)

// Config contains configuration options for Heimdall.
type Config struct {
	// SessionTTL is how long sessions remain active.
	// Default: 24 hours.
	SessionTTL time.Duration

	// InvalidationTTL is how long to remember invalidated sessions.
	// This should be at least as long as SessionTTL to prevent
	// invalidated sessions from being reused.
	// Default: 24 hours (Same as SessionTTL).
	InvalidationTTL time.Duration

	// GeoIPDatabasePath is the path to MaxMind GeoLite2-City.mmdb file.
	// Required for IP-based location detection.
	// Download from: https://dev.maxmind.com/geoip/geolite2-free-geolocation-data
	GeoIPDatabasePath string

	// NewLocationThresholdKM is the distance threshold in kilometers
	// for triggering a "new location" alert.
	// Default: 100 km.
	NewLocationThresholdKM float64

	// SessionStore is the storage backend for sessions.
	// Default: SQLite store (creates heimdall.db in current directory).
	SessionStore store.SessionStore

	// InvalidationCache is the cache for invalidated session IDs.
	// Default: in-memory cache.
	InvalidationCache store.InvalidationCache

	// DatabasePath is the path for the default SQLite database.
	// Only used if SessionStore is nil.
	// Default: "heimdall.db".
	DatabasePath string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		SessionTTL:             24 * time.Hour,
		InvalidationTTL:        24 * time.Hour,
		NewLocationThresholdKM: 100,
		DatabasePath:           "heimdall.db",
	}
}

// applyDefaults fills in default values for zero-value fields.
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	if c.SessionTTL <= 0 {
		c.SessionTTL = defaults.SessionTTL
	}
	if c.InvalidationTTL <= 0 {
		c.InvalidationTTL = defaults.SessionTTL
	}
	if c.NewLocationThresholdKM <= 0 {
		c.NewLocationThresholdKM = defaults.NewLocationThresholdKM
	}
	if c.DatabasePath == "" {
		c.DatabasePath = defaults.DatabasePath
	}
}
