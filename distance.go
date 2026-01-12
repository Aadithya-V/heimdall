package heimdall

import "math"

const earthRadiusKM = 6371.0

// HaversineDistance calculates the distance in kilometers between two
// geographic coordinates using the Haversine formula.
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKM * c
}

// IsNewLocation returns true if the distance between two locations
// exceeds the given threshold in kilometers.
func IsNewLocation(prev, curr LocationInfo, thresholdKM float64) bool {
	// If either location has no coordinates, compare by city/country
	if prev.Latitude == 0 && prev.Longitude == 0 {
		return prev.City != curr.City || prev.Country != curr.Country
	}
	if curr.Latitude == 0 && curr.Longitude == 0 {
		return prev.City != curr.City || prev.Country != curr.Country
	}

	distance := HaversineDistance(
		prev.Latitude, prev.Longitude,
		curr.Latitude, curr.Longitude,
	)

	return distance > thresholdKM
}
