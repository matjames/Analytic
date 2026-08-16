package store

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// ─── Spatial primitives (pure functions, dependency-free) ───────────────────

// HaversineMeters returns the great-circle distance between two coordinates.
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// ParseCentroid parses "lat,lng" into floats.
func ParseCentroid(c string) (lat, lng float64, ok bool) {
	parts := strings.Split(c, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return lat, lng, true
}

// ContainsPoint reports whether a GeoJSON polygon feature contains (lat, lng).
// GeoJSON coordinates are [lng, lat]; ray casting uses point x=lng, y=lat.
func ContainsPoint(geometry string, lat, lng float64) bool {
	var geom struct {
		Type        string      `json:"type"`
		Coordinates interface{} `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(geometry), &geom); err != nil || geom.Type != "Polygon" {
		return false
	}
	raw, _ := json.Marshal(geom.Coordinates)
	var rings [][][2]float64
	if err := json.Unmarshal(raw, &rings); err != nil || len(rings) == 0 {
		return false
	}
	return pointInPolygon(lng, lat, rings)
}

// pointInPolygon ray-casts over multiple rings (outer + holes).
func pointInPolygon(x, y float64, coords [][][2]float64) bool {
	var outer *[][2]float64
	for i := range coords {
		if pointInRing(x, y, coords[i]) {
			if i == 0 {
				outer = &coords[0]
			} else {
				return false // inside a hole
			}
		}
	}
	return outer != nil
}

// pointInRing ray-casts a point (x=lng, y=lat) against a GeoJSON ring.
func pointInRing(x, y float64, ring [][2]float64) bool {
	inside := false
	n := len(ring)
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := ring[i][0], ring[i][1] // xi=lng_i, yi=lat_i
		xj, yj := ring[j][0], ring[j][1]
		if (yi > y) != (yj > y) {
			xint := xj + (xi-xj)*(y-yj)/(yi-yj)
			if x < xint {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

// DeriveCentroid computes "lat,lng" for a GeoJSON Point or Polygon (approx: bbox
// centre). Returns "" when the geometry is unsupported.
func DeriveCentroid(geometry string) string {
	var geom struct {
		Type        string      `json:"type"`
		Coordinates interface{} `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(geometry), &geom); err != nil {
		return ""
	}
	switch geom.Type {
	case "Point":
		raw, _ := json.Marshal(geom.Coordinates)
		var p [2]float64
		if json.Unmarshal(raw, &p) == nil {
			return strconv.FormatFloat(p[1], 'f', 6, 64) + "," + strconv.FormatFloat(p[0], 'f', 6, 64)
		}
	case "Polygon":
		raw, _ := json.Marshal(geom.Coordinates)
		var rings [][][2]float64
		if json.Unmarshal(raw, &rings) != nil || len(rings) == 0 || len(rings[0]) == 0 {
			return ""
		}
		var minX, maxX, minY, maxY float64
		first := rings[0][0]
		minX, maxX, minY, maxY = first[0], first[0], first[1], first[1]
		for _, c := range rings[0] {
			if c[0] < minX {
				minX = c[0]
			}
			if c[0] > maxX {
				maxX = c[0]
			}
			if c[1] < minY {
				minY = c[1]
			}
			if c[1] > maxY {
				maxY = c[1]
			}
		}
		lat := (minY + maxY) / 2
		lng := (minX + maxX) / 2
		return strconv.FormatFloat(lat, 'f', 6, 64) + "," + strconv.FormatFloat(lng, 'f', 6, 64)
	}
	return ""
}