package osu

import "encoding/json"

// GeoDistanceQuery builds a geo_distance query over a geo_point field.
type GeoDistanceQuery struct {
	field    string
	distance string
	lat, lon float64
}

func NewGeoDistanceQuery(field string) *GeoDistanceQuery {
	return &GeoDistanceQuery{field: field}
}

func (q *GeoDistanceQuery) Distance(v string) *GeoDistanceQuery {
	q.distance = v
	return q
}

func (q *GeoDistanceQuery) Point(lat, lon float64) *GeoDistanceQuery {
	q.lat, q.lon = lat, lon
	return q
}

func (q *GeoDistanceQuery) Map() (map[string]any, error) {
	return map[string]any{
		"geo_distance": map[string]any{
			"distance": q.distance,
			q.field:    map[string]any{"lat": q.lat, "lon": q.lon},
		},
	}, nil
}

func (q *GeoDistanceQuery) MarshalJSON() ([]byte, error) { return marshalMap(q) }

// GeoBoundingBoxQuery builds a geo_bounding_box query over a geo_point field.
type GeoBoundingBoxQuery struct {
	field                          string
	minLat, minLon, maxLat, maxLon float64
}

func NewGeoBoundingBoxQuery(field string) *GeoBoundingBoxQuery {
	return &GeoBoundingBoxQuery{field: field}
}

func (q *GeoBoundingBoxQuery) Box(minLat, minLon, maxLat, maxLon float64) *GeoBoundingBoxQuery {
	q.minLat, q.minLon, q.maxLat, q.maxLon = minLat, minLon, maxLat, maxLon
	return q
}

func (q *GeoBoundingBoxQuery) Map() (map[string]any, error) {
	return map[string]any{
		"geo_bounding_box": map[string]any{
			q.field: map[string]any{
				"top_left":     map[string]any{"lat": q.maxLat, "lon": q.minLon},
				"bottom_right": map[string]any{"lat": q.minLat, "lon": q.maxLon},
			},
		},
	}, nil
}

func (q *GeoBoundingBoxQuery) MarshalJSON() ([]byte, error) { return marshalMap(q) }

// GeoPolygonQuery builds a point-in-polygon query over a geo_point field. It
// emits a geo_shape query (relation intersects), the modern replacement for the
// deprecated geo_polygon; geo_shape queries still run against geo_point fields.
type GeoPolygonQuery struct {
	field  string
	points [][2]float64 // GeoJSON [lon, lat] order
}

func NewGeoPolygonQuery(field string) *GeoPolygonQuery {
	return &GeoPolygonQuery{field: field}
}

func (q *GeoPolygonQuery) Point(lat, lon float64) *GeoPolygonQuery {
	q.points = append(q.points, [2]float64{lon, lat})
	return q
}

func (q *GeoPolygonQuery) Map() (map[string]any, error) {
	ring := make([][2]float64, len(q.points))
	copy(ring, q.points)
	// GeoJSON polygon rings must be closed.
	if n := len(ring); n > 0 && ring[0] != ring[n-1] {
		ring = append(ring, ring[0])
	}
	return map[string]any{
		"geo_shape": map[string]any{
			q.field: map[string]any{
				"shape": map[string]any{
					"type":        "polygon",
					"coordinates": [][][2]float64{ring},
				},
				"relation": "intersects",
			},
		},
	}, nil
}

func (q *GeoPolygonQuery) MarshalJSON() ([]byte, error) { return marshalMap(q) }

func marshalMap(b Builder) ([]byte, error) {
	data, err := b.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}
