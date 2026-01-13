package heimdall

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name           string
		lat1, lng1     float64
		lat2, lng2     float64
		expectedKM     float64
		toleranceKM    float64 // absolute tolerance
		toleranceRatio float64 // relative tolerance (percentage)
	}{
		{
			name:           "same point returns zero",
			lat1:           40.7128,
			lng1:           -74.0060,
			lat2:           40.7128,
			lng2:           -74.0060,
			expectedKM:     0,
			toleranceKM:    0.001,
			toleranceRatio: 0,
		},
		{
			name:           "NYC to London",
			lat1:           40.7128,
			lng1:           -74.0060,
			lat2:           51.5074,
			lng2:           -0.1278,
			expectedKM:     5570,
			toleranceRatio: 0.01, // 1%
		},
		{
			name:           "NYC to Los Angeles",
			lat1:           40.7128,
			lng1:           -74.0060,
			lat2:           34.0522,
			lng2:           -118.2437,
			expectedKM:     3940,
			toleranceRatio: 0.01,
		},
		{
			name:           "Sydney to Tokyo",
			lat1:           -33.8688,
			lng1:           151.2093,
			lat2:           35.6762,
			lng2:           139.6503,
			expectedKM:     7823,
			toleranceRatio: 0.01,
		},
		{
			name:           "North Pole to South Pole (antipodal)",
			lat1:           90,
			lng1:           0,
			lat2:           -90,
			lng2:           0,
			expectedKM:     20015, // half circumference of Earth
			toleranceRatio: 0.01,
		},
		{
			name:           "crossing equator - Quito to Lima",
			lat1:           -0.1807,
			lng1:           -78.4678,
			lat2:           -12.0464,
			lng2:           -77.0428,
			expectedKM:     1320,
			toleranceRatio: 0.02,
		},
		{
			name:           "crossing prime meridian - London to Paris",
			lat1:           51.5074,
			lng1:           -0.1278,
			lat2:           48.8566,
			lng2:           2.3522,
			expectedKM:     344,
			toleranceRatio: 0.02,
		},
		{
			name:           "crossing International Date Line - Tokyo to Honolulu",
			lat1:           35.6762,
			lng1:           139.6503,
			lat2:           21.3069,
			lng2:           -157.8583,
			expectedKM:     6199,
			toleranceRatio: 0.02,
		},
		{
			name:           "short distance - within same city",
			lat1:           40.7484,
			lng1:           -73.9857, // Empire State Building
			lat2:           40.7580,
			lng2:           -73.9855, // Times Square
			expectedKM:     1.07,
			toleranceRatio: 0.05,
		},
		{
			name:           "very short distance - neighboring buildings",
			lat1:           40.7484,
			lng1:           -73.9857,
			lat2:           40.7485,
			lng2:           -73.9856,
			expectedKM:     0.015,
			toleranceRatio: 0.10,
		},
		{
			name:           "southern hemisphere - Cape Town to Buenos Aires",
			lat1:           -33.9249,
			lng1:           18.4241,
			lat2:           -34.6037,
			lng2:           -58.3816,
			expectedKM:     6870,
			toleranceRatio: 0.02,
		},
		{
			name:           "near North Pole",
			lat1:           89.9,
			lng1:           0,
			lat2:           89.9,
			lng2:           180,
			expectedKM:     22.2, // small distance near pole
			toleranceRatio: 0.05,
		},
		{
			name:           "zero coordinates (0,0 to 0,0)",
			lat1:           0,
			lng1:           0,
			lat2:           0,
			lng2:           0,
			expectedKM:     0,
			toleranceKM:    0.001,
			toleranceRatio: 0,
		},
		{
			name:           "origin to arbitrary point",
			lat1:           0,
			lng1:           0,
			lat2:           10,
			lng2:           10,
			expectedKM:     1565,
			toleranceRatio: 0.02,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineDistance(tt.lat1, tt.lng1, tt.lat2, tt.lng2)

			// Calculate tolerance
			tolerance := tt.toleranceKM
			if tt.toleranceRatio > 0 && tt.expectedKM > 0 {
				tolerance = tt.expectedKM * tt.toleranceRatio
			}

			if math.Abs(got-tt.expectedKM) > tolerance {
				t.Errorf("HaversineDistance(%v, %v, %v, %v) = %v km, want ~%v km (tolerance: %v km)",
					tt.lat1, tt.lng1, tt.lat2, tt.lng2, got, tt.expectedKM, tolerance)
			}
		})
	}
}

func TestHaversineDistanceSymmetry(t *testing.T) {
	// Distance from A to B should equal distance from B to A
	testCases := []struct {
		lat1, lng1, lat2, lng2 float64
	}{
		{40.7128, -74.0060, 51.5074, -0.1278},   // NYC to London
		{-33.8688, 151.2093, 35.6762, 139.6503}, // Sydney to Tokyo
		{0, 0, 45, 45},                          // Origin to arbitrary
		{90, 0, -90, 0},                         // Poles
	}

	for _, tc := range testCases {
		d1 := HaversineDistance(tc.lat1, tc.lng1, tc.lat2, tc.lng2)
		d2 := HaversineDistance(tc.lat2, tc.lng2, tc.lat1, tc.lng1)

		if math.Abs(d1-d2) > 0.0001 {
			t.Errorf("Distance not symmetric: (%v,%v)->(%v,%v)=%v but reverse=%v",
				tc.lat1, tc.lng1, tc.lat2, tc.lng2, d1, d2)
		}
	}
}

func TestHaversineDistanceNonNegative(t *testing.T) {
	// Distance should always be non-negative
	testCases := []struct {
		lat1, lng1, lat2, lng2 float64
	}{
		{0, 0, 0, 0},
		{-90, -180, 90, 180},
		{45, -90, -45, 90},
		{0, 180, 0, -180}, // Same point across date line
	}

	for _, tc := range testCases {
		d := HaversineDistance(tc.lat1, tc.lng1, tc.lat2, tc.lng2)
		if d < 0 {
			t.Errorf("HaversineDistance(%v, %v, %v, %v) = %v, want non-negative",
				tc.lat1, tc.lng1, tc.lat2, tc.lng2, d)
		}
	}
}

func TestHaversineDistanceTriangleInequality(t *testing.T) {
	// For any three points A, B, C: dist(A,C) <= dist(A,B) + dist(B,C)
	pointA := struct{ lat, lng float64 }{40.7128, -74.0060} // NYC
	pointB := struct{ lat, lng float64 }{51.5074, -0.1278}  // London
	pointC := struct{ lat, lng float64 }{48.8566, 2.3522}   // Paris

	dAB := HaversineDistance(pointA.lat, pointA.lng, pointB.lat, pointB.lng)
	dBC := HaversineDistance(pointB.lat, pointB.lng, pointC.lat, pointC.lng)
	dAC := HaversineDistance(pointA.lat, pointA.lng, pointC.lat, pointC.lng)

	if dAC > dAB+dBC+0.001 { // small epsilon for floating point
		t.Errorf("Triangle inequality violated: d(A,C)=%v > d(A,B)+d(B,C)=%v",
			dAC, dAB+dBC)
	}
}

func TestIsNewLocation(t *testing.T) {
	tests := []struct {
		name        string
		prev        LocationInfo
		curr        LocationInfo
		thresholdKM float64
		want        bool
	}{
		{
			name: "same location - not new",
			prev: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			curr: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			thresholdKM: 100,
			want:        false,
		},
		{
			name: "within threshold - not new",
			prev: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			curr: LocationInfo{
				City:      "Newark",
				Country:   "United States",
				Latitude:  40.7357,
				Longitude: -74.1724,
			},
			thresholdKM: 100,
			want:        false, // ~15 km apart
		},
		{
			name: "exceeds threshold - is new",
			prev: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			curr: LocationInfo{
				City:      "London",
				Country:   "United Kingdom",
				Latitude:  51.5074,
				Longitude: -0.1278,
			},
			thresholdKM: 100,
			want:        true, // ~5570 km apart
		},
		{
			name: "exactly at threshold boundary - not new (> not >=)",
			prev: LocationInfo{
				Latitude:  10.0, // Non-zero to avoid fallback to city comparison
				Longitude: 10.0,
			},
			curr: LocationInfo{
				Latitude:  10.85, // ~94 km north (staying under 100 km threshold)
				Longitude: 10.0,
			},
			thresholdKM: 100,
			want:        false,
		},
		{
			name: "just over threshold - is new",
			prev: LocationInfo{
				Latitude:  10.0, // Non-zero to avoid fallback to city comparison
				Longitude: 10.0,
			},
			curr: LocationInfo{
				Latitude:  11.0, // ~111 km north
				Longitude: 10.0,
			},
			thresholdKM: 100,
			want:        true,
		},
		{
			name: "zero threshold - any movement is new",
			prev: LocationInfo{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			curr: LocationInfo{
				Latitude:  40.7129,
				Longitude: -74.0060,
			},
			thresholdKM: 0,
			want:        true,
		},
		{
			name: "very large threshold - far locations not new",
			prev: LocationInfo{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			curr: LocationInfo{
				Latitude:  51.5074,
				Longitude: -0.1278,
			},
			thresholdKM: 10000,
			want:        false, // NYC to London is ~5570 km
		},
		{
			name: "previous has no coordinates - same city/country - not new",
			prev: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  0,
				Longitude: 0,
			},
			curr: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			thresholdKM: 100,
			want:        false,
		},
		{
			name: "previous has no coordinates - different city - is new",
			prev: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  0,
				Longitude: 0,
			},
			curr: LocationInfo{
				City:      "Boston",
				Country:   "United States",
				Latitude:  42.3601,
				Longitude: -71.0589,
			},
			thresholdKM: 100,
			want:        true,
		},
		{
			name: "previous has no coordinates - different country - is new",
			prev: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  0,
				Longitude: 0,
			},
			curr: LocationInfo{
				City:      "New York", // same city name
				Country:   "Canada",   // different country (fictional)
				Latitude:  45.0,
				Longitude: -75.0,
			},
			thresholdKM: 100,
			want:        true,
		},
		{
			name: "current has no coordinates - same city/country - not new",
			prev: LocationInfo{
				City:      "London",
				Country:   "United Kingdom",
				Latitude:  51.5074,
				Longitude: -0.1278,
			},
			curr: LocationInfo{
				City:      "London",
				Country:   "United Kingdom",
				Latitude:  0,
				Longitude: 0,
			},
			thresholdKM: 100,
			want:        false,
		},
		{
			name: "current has no coordinates - different city - is new",
			prev: LocationInfo{
				City:      "London",
				Country:   "United Kingdom",
				Latitude:  51.5074,
				Longitude: -0.1278,
			},
			curr: LocationInfo{
				City:      "Manchester",
				Country:   "United Kingdom",
				Latitude:  0,
				Longitude: 0,
			},
			thresholdKM: 100,
			want:        true,
		},
		{
			name: "both have no coordinates - same city/country - not new",
			prev: LocationInfo{
				City:    "Tokyo",
				Country: "Japan",
			},
			curr: LocationInfo{
				City:    "Tokyo",
				Country: "Japan",
			},
			thresholdKM: 100,
			want:        false,
		},
		{
			name: "both have no coordinates - different city - is new",
			prev: LocationInfo{
				City:    "Tokyo",
				Country: "Japan",
			},
			curr: LocationInfo{
				City:    "Osaka",
				Country: "Japan",
			},
			thresholdKM: 100,
			want:        true,
		},
		{
			name:        "empty locations - same - not new",
			prev:        LocationInfo{},
			curr:        LocationInfo{},
			thresholdKM: 100,
			want:        false,
		},
		{
			name: "only IP differs but same city/country without coords - not new",
			prev: LocationInfo{
				IP:      "1.1.1.1",
				City:    "Sydney",
				Country: "Australia",
			},
			curr: LocationInfo{
				IP:      "2.2.2.2",
				City:    "Sydney",
				Country: "Australia",
			},
			thresholdKM: 100,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewLocation(tt.prev, tt.curr, tt.thresholdKM)
			if got != tt.want {
				t.Errorf("IsNewLocation(%+v, %+v, %v) = %v, want %v",
					tt.prev, tt.curr, tt.thresholdKM, got, tt.want)
			}
		})
	}
}

func TestIsNewLocationWithRealWorldScenarios(t *testing.T) {
	// Test realistic session location change scenarios
	tests := []struct {
		name        string
		scenario    string
		prev        LocationInfo
		curr        LocationInfo
		thresholdKM float64
		want        bool
	}{
		{
			name:     "user moves within city - coffee shop to office",
			scenario: "Same city, different location within threshold",
			prev: LocationInfo{
				City:      "San Francisco",
				Country:   "United States",
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			curr: LocationInfo{
				City:      "San Francisco",
				Country:   "United States",
				Latitude:  37.7849,
				Longitude: -122.4094,
			},
			thresholdKM: 50,
			want:        false,
		},
		{
			name:     "user takes domestic flight",
			scenario: "SF to NYC - should trigger new location",
			prev: LocationInfo{
				City:      "San Francisco",
				Country:   "United States",
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			curr: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			thresholdKM: 100,
			want:        true,
		},
		{
			name:     "user travels internationally",
			scenario: "NYC to Tokyo - definitely new location",
			prev: LocationInfo{
				City:      "New York",
				Country:   "United States",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			curr: LocationInfo{
				City:      "Tokyo",
				Country:   "Japan",
				Latitude:  35.6762,
				Longitude: 139.6503,
			},
			thresholdKM: 100,
			want:        true,
		},
		{
			name:     "VPN user appears in different country",
			scenario: "Appears to jump from US to Germany instantly",
			prev: LocationInfo{
				City:      "Chicago",
				Country:   "United States",
				Latitude:  41.8781,
				Longitude: -87.6298,
			},
			curr: LocationInfo{
				City:      "Frankfurt",
				Country:   "Germany",
				Latitude:  50.1109,
				Longitude: 8.6821,
			},
			thresholdKM: 500,
			want:        true,
		},
		{
			name:     "mobile user at country border",
			scenario: "US-Canada border crossing",
			prev: LocationInfo{
				City:      "Detroit",
				Country:   "United States",
				Latitude:  42.3314,
				Longitude: -83.0458,
			},
			curr: LocationInfo{
				City:      "Windsor",
				Country:   "Canada",
				Latitude:  42.3149,
				Longitude: -83.0364,
			},
			thresholdKM: 50,
			want:        false, // Only ~5 km apart
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewLocation(tt.prev, tt.curr, tt.thresholdKM)
			if got != tt.want {
				t.Errorf("Scenario: %s\nIsNewLocation() = %v, want %v",
					tt.scenario, got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkHaversineDistance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HaversineDistance(40.7128, -74.0060, 51.5074, -0.1278)
	}
}

func BenchmarkIsNewLocation(b *testing.B) {
	prev := LocationInfo{
		City:      "New York",
		Country:   "United States",
		Latitude:  40.7128,
		Longitude: -74.0060,
	}
	curr := LocationInfo{
		City:      "London",
		Country:   "United Kingdom",
		Latitude:  51.5074,
		Longitude: -0.1278,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsNewLocation(prev, curr, 100)
	}
}

func BenchmarkIsNewLocationNoCoords(b *testing.B) {
	prev := LocationInfo{
		City:    "New York",
		Country: "United States",
	}
	curr := LocationInfo{
		City:    "London",
		Country: "United Kingdom",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsNewLocation(prev, curr, 100)
	}
}
