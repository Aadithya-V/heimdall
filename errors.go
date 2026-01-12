package heimdall

import "errors"

var (
	// ErrSessionNotFound is returned when a session does not exist.
	ErrSessionNotFound = errors.New("heimdall: session not found")

	// ErrSessionLimitExceeded is returned when the concurrent session limit is exceeded.
	ErrSessionLimitExceeded = errors.New("heimdall: concurrent session limit exceeded")

	// ErrSessionInvalidated is returned when attempting to use an invalidated session.
	ErrSessionInvalidated = errors.New("heimdall: session has been invalidated")

	// ErrGeoIPDatabaseNotConfigured is returned when GeoIP lookup is attempted
	// without configuring the GeoIP database path.
	ErrGeoIPDatabaseNotConfigured = errors.New("heimdall: GeoIP database path not configured")

	// ErrGeoIPLookupFailed is returned when IP geolocation lookup fails.
	ErrGeoIPLookupFailed = errors.New("heimdall: GeoIP lookup failed")

	// ErrInvalidIP is returned when an invalid IP address is provided.
	ErrInvalidIP = errors.New("heimdall: invalid IP address")
)
