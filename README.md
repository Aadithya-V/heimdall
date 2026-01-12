# Heimdall

Session management SDK for Go. Enforce single-session policies, detect suspicious logins, let users manage their devices.

```go
h, _ := heimdall.New(heimdall.Config{})

// On login
device, location, _ := h.ExtractRequestInfo(r)
result, _ := h.RegisterSession(userID, sessionID, device, location, 3) // max 3 sessions

if result.LimitExceeded {
    // Too many sessions - show user their active devices
}
if result.IsNewLocation {
    // Login from new city - send security alert
}

// On logout
h.InvalidateSession(sessionID)

// In auth middleware
if invalidated, _ := h.IsSessionInvalidated(sessionID); invalidated {
    // Reject request
}

// Show user their sessions
sessions, _ := h.ListSessions(userID)
```

## Install

```bash
go get github.com/aadithya-v/heimdall
```

## What it does

| Feature | Description |
|---------|-------------|
| **Concurrent session limit** | Reject new logins when user has N active sessions |
| **New location detection** | Flag logins from unusual locations (uses IP geolocation) |
| **Session listing** | Let users see and revoke their active sessions |
| **Audit trail** | Sessions soft-deleted, kept for compliance |

## API

```go
New(Config) (*Heimdall, error)
ExtractRequestInfo(*http.Request) (DeviceInfo, LocationInfo, error)
RegisterSession(userID, sessionID string, device, location, limit int) (*RegisterResult, error)
InvalidateSession(sessionID string) error
IsSessionInvalidated(sessionID string) (bool, error)
ListSessions(userID string) ([]*Session, error)
Close() error
```

## Pluggable Storage

Bring your own storage backend. Implement the interface, plug it in.

| Backend | SessionStore | InvalidationCache |
|---------|--------------|-------------------|
| **SQLite** (default) | `store.NewSQLite(path)` | `store.NewSQLiteInvalidationCache(path)` |
| **MySQL** | `store.NewMySQL(dsn)` | — |
| **Redis** | — | `store.NewRedisSimple(addr, pass, db)` |
| **In-Memory** | `store.NewMemorySessionStore()` | `store.NewMemoryCache()` |
| **Custom** | Implement `store.SessionStore` | Implement `store.InvalidationCache` |

```go
// Zero-config (SQLite)
h, _ := heimdall.New(heimdall.Config{})

// Production (MySQL + Redis)
h, _ := heimdall.New(heimdall.Config{
    SessionStore:      store.NewMySQL("user:pass@tcp(localhost:3306)/db"),
    InvalidationCache: store.NewRedisSimple("localhost:6379", "", 0),
})

// Custom backend
h, _ := heimdall.New(heimdall.Config{
    SessionStore:      myPostgresStore,      // implements store.SessionStore
    InvalidationCache: myMemcachedCache,     // implements store.InvalidationCache
})
```

**Interfaces:**
```go
type SessionStore interface {
    Save(session *Session) error
    Delete(sessionID string) error
    GetActiveByUser(userID string) ([]*Session, error)
    Close() error
}

type InvalidationCache interface {
    Set(sessionID string, ttl time.Duration) error
    Exists(sessionID string) (bool, error)
    Close() error
}
```

## Config

```go
heimdall.Config{
    SessionTTL:             24 * time.Hour,  // How long sessions live
    NewLocationThresholdKM: 100,             // Distance to trigger alert
    GeoIPDatabasePath:      "GeoLite2.mmdb", // Optional: MaxMind DB for location
    DatabasePath:           "heimdall.db",   // SQLite path
}
```

## GeoIP (optional)

For city/country detection from IP:

1. Get free database from [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)
2. Set `GeoIPDatabasePath` in config

Without it, `LocationInfo` only contains the IP address.

## License

MIT
