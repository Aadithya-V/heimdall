package heimdall

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// GeoIPReader provides IP geolocation using MaxMind GeoLite2 database.
type GeoIPReader struct {
	db   *geoip2.Reader
	path string
}

// NewGeoIPReader opens a MaxMind GeoLite2-City database.
func NewGeoIPReader(dbPath string) (*GeoIPReader, error) {
	if dbPath == "" {
		return nil, ErrGeoIPDatabaseNotConfigured
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("geoip: failed to open database: %w", err)
	}

	return &GeoIPReader{
		db:   db,
		path: dbPath,
	}, nil
}

// Lookup returns location information for an IP address.
func (r *GeoIPReader) Lookup(ip string) (*LocationInfo, error) {
	if r == nil || r.db == nil {
		return nil, ErrGeoIPDatabaseNotConfigured
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidIP, ip)
	}

	record, err := r.db.City(parsed)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGeoIPLookupFailed, err)
	}

	// Extract city name (prefer English, fallback to first available)
	city := ""
	if name, ok := record.City.Names["en"]; ok {
		city = name
	} else {
		for _, name := range record.City.Names {
			city = name
			break
		}
	}

	// Extract country name
	country := ""
	if name, ok := record.Country.Names["en"]; ok {
		country = name
	} else {
		for _, name := range record.Country.Names {
			country = name
			break
		}
	}

	return &LocationInfo{
		IP:        ip,
		City:      city,
		Country:   country,
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
	}, nil
}

// Close closes the GeoIP database.
func (r *GeoIPReader) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// LookupWithFallback attempts IP geolocation, returning a partial result
// with just the IP if lookup fails.
func (r *GeoIPReader) LookupWithFallback(ip string) LocationInfo {
	loc, err := r.Lookup(ip)
	if err != nil || loc == nil {
		return LocationInfo{IP: ip}
	}
	return *loc
}
