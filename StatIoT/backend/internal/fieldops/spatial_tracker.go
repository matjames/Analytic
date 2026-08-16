package fieldops

import (
	"context"
	"math"

	"statiot-backend/internal/models"
	"statiot-backend/internal/store"
)

type SpatialTracker struct {
	store store.Store
}

func NewSpatialTracker(st store.Store) *SpatialTracker {
	return &SpatialTracker{store: st}
}

// HaversineDistance calculates the great-circle distance between two points in meters.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(rLat1)*math.Cos(rLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

// EvaluateGeofences checks if a given GPS coordinate is inside any active geofence zones.
func (t *SpatialTracker) EvaluateGeofences(ctx context.Context, tenantID string, lat, lng float64) ([]models.GeofenceZone, error) {
	zones, err := t.store.ListGeofences(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var triggeredZones []models.GeofenceZone
	for _, z := range zones {
		dist := HaversineDistance(lat, lng, z.CenterLat, z.CenterLng)
		if dist <= z.RadiusMeters {
			triggeredZones = append(triggeredZones, z)
		}
	}
	return triggeredZones, nil
}
