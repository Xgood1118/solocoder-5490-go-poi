package utils

import "math"

const earthRadius = 6371000.0

func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	lat1 = toRad(lat1)
	lat2 = toRad(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}

func WalkingDistance(straightDist float64) float64 {
	return straightDist * 1.3
}

func WalkingDuration(distMeters float64) int {
	speed := 5.0 / 3.6
	return int(distMeters / speed)
}
