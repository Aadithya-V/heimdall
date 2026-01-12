package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aadithya-v/heimdall"
	"github.com/aadithya-v/heimdall/store"
)

var h *heimdall.Heimdall

func main() {
	var err error

	// Option 1: Zero-config (SQLite + in-memory cache)
	// Just works out of the box - creates heimdall.db automatically
	h, err = heimdall.New(heimdall.Config{
		// Optional: path to MaxMind GeoLite2-City.mmdb for IP geolocation
		// Download from: https://dev.maxmind.com/geoip/geolite2-free-geolocation-data
		// GeoIPDatabasePath: "./GeoLite2-City.mmdb",
	})
	if err != nil {
		log.Fatalf("Failed to initialize Heimdall: %v", err)
	}
	defer h.Close()

	// Option 2: Production config (MySQL + Redis)
	// Uncomment to use:
	/*
		mysqlStore, err := store.NewMySQL("user:password@tcp(localhost:3306)/heimdall")
		if err != nil {
			log.Fatalf("Failed to connect to MySQL: %v", err)
		}

		redisCache, err := store.NewRedisSimple("localhost:6379", "", 0)
		if err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}

		h, err = heimdall.New(heimdall.Config{
			SessionStore:       mysqlStore,
			InvalidationCache:  redisCache,
			SessionTTL:         24 * time.Hour,
			InvalidationTTL:    7 * 24 * time.Hour,
			GeoIPDatabasePath:  "./GeoLite2-City.mmdb",
		})
	*/

	// HTTP handlers
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/sessions", sessionsHandler)
	http.HandleFunc("/check", checkSessionHandler)

	fmt.Println("Heimdall example server running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /login?user_id=xxx&session_id=yyy    - Register a session")
	fmt.Println("  POST /logout?session_id=xxx              - Invalidate a session")
	fmt.Println("  GET  /sessions?user_id=xxx               - List active sessions")
	fmt.Println("  GET  /check?session_id=xxx               - Check if session is invalidated")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	sessionID := r.URL.Query().Get("session_id")

	if userID == "" || sessionID == "" {
		http.Error(w, "user_id and session_id required", http.StatusBadRequest)
		return
	}

	// Extract device and location info from request
	device, location, err := h.ExtractRequestInfo(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to extract request info: %v", err), http.StatusInternalServerError)
		return
	}

	// Register the session (limit to 3 concurrent sessions)
	result, err := h.RegisterSession(userID, sessionID, device, location, 3)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to register session: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	response := map[string]interface{}{
		"success": !result.LimitExceeded,
	}

	if result.LimitExceeded {
		response["error"] = "concurrent_session_limit_exceeded"
		response["message"] = "You have too many active sessions. Please log out from one device first."
		response["active_sessions"] = result.ActiveSessions
	} else {
		response["session"] = result.Session

		if result.IsNewLocation {
			response["warning"] = "new_location_detected"
			response["message"] = fmt.Sprintf(
				"Login from new location: %s, %s (previously: %s, %s)",
				location.City, location.Country,
				result.PreviousLocation.City, result.PreviousLocation.Country,
			)
			// In production, you would send a security alert email here
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}

	if err := h.InvalidateSession(sessionID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to invalidate session: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Session invalidated",
	})
}

func sessionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	sessions, err := h.ListSessions(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list sessions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":  userID,
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func checkSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}

	invalidated, err := h.IsSessionInvalidated(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check session: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id":  sessionID,
		"invalidated": invalidated,
	})
}

// Unused import guard
var _ = time.Now
var _ = store.NewMemoryCache
