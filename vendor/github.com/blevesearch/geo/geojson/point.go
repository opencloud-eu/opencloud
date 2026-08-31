//  Copyright (c) 2026 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package geojson

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	index "github.com/blevesearch/bleve_index_api"
	"github.com/blevesearch/geo/s2"
)

// --------------------------------------------------------
// Point represents the geoJSON point type and it
// implements the index.GeoJSON interface.
type Point struct {
	Typ      string    `json:"type"`
	Vertices []float64 `json:"coordinates"`
	s2point  *s2.Point
}

func NewGeoJsonPoint(v []float64) index.GeoJSON {
	rv := &Point{Typ: PointType, Vertices: v}
	rv.init()
	return rv
}

func (p *Point) Type() string {
	return strings.ToLower(p.Typ)
}

func (p *Point) Value() ([]byte, error) {
	return jsoniter.Marshal(p)
}

func (p *Point) init() {
	if p.s2point == nil {
		s2point := s2.PointFromLatLng(s2.LatLngFromDegrees(
			p.Vertices[1], p.Vertices[0]))
		p.s2point = &s2point
	}
}

func (p *Point) Marshal() ([]byte, error) {
	p.init()

	var b bytes.Buffer
	b.Grow(32)
	w := bufio.NewWriter(&b)
	err := p.s2point.Encode(w)
	if err != nil {
		return nil, err
	}

	w.Flush()
	return append([]byte{PointTypePrefix}, b.Bytes()...), nil
}

func (p *Point) Intersects(other index.GeoJSON) (bool, error) {
	p.init()

	return checkPointIntersectsShape(p.s2point, p, other)
}

func (p *Point) Contains(other index.GeoJSON) (bool, error) {
	p.init()

	return checkPointContainsShape([]*s2.Point{p.s2point}, other)
}

func (p *Point) Coordinates() []float64 {
	return p.Vertices
}

// IndexCells returns the point's covering: a point has no area, so it is a
// single maxCellLevel cross cell and never an inner cell.
func (p *Point) IndexCells() (inner, cross []uint64) {
	p.init()
	if p.s2point == nil {
		return nil, nil
	}
	return nil, []uint64{pointCell(*p.s2point)}
}

// QueryCells delegates to IndexCells: a point maps to exactly one cell, so
// the query-time coverer would produce the same single cross cell as the
// index-time coverer.
func (p *Point) QueryCells() (inner, cross []uint64) {
	return p.IndexCells()
}

func (p *Point) BoundingBox() index.GeoJSON {
	p.init()
	if p.s2point == nil {
		return envelopeFromRect(s2.EmptyRect())
	}
	return envelopeFromRect(p.s2point.RectBound())
}

func (p *Point) IndexTokens(s *s2.RegionTermIndexer) []string {
	p.init()
	terms := s.GetIndexTermsForPoint(*p.s2point, "")
	return StripCoveringTerms(terms)
}

func (p *Point) QueryTokens(s *s2.RegionTermIndexer) []string {
	p.init()
	terms := s.GetQueryTermsForPoint(*p.s2point, "")
	return StripCoveringTerms(terms)
}

// --------------------------------------------------------
// MultiPoint represents the geoJSON multipoint type and it
// implements the index.GeoJSON interface as well as the
// compositeShape interface.
type MultiPoint struct {
	Typ      string      `json:"type"`
	Vertices [][]float64 `json:"coordinates"`
	s2points []*s2.Point
}

func NewGeoJsonMultiPoint(v [][]float64) index.GeoJSON {
	rv := &MultiPoint{Typ: MultiPointType, Vertices: v}
	rv.init()
	return rv
}

func (mp *MultiPoint) init() {
	if mp.s2points == nil {
		mp.s2points = make([]*s2.Point, len(mp.Vertices))
		for i, point := range mp.Vertices {
			s2point := s2.PointFromLatLng(s2.LatLngFromDegrees(
				point[1], point[0]))
			mp.s2points[i] = &s2point
		}
	}
}

func (mp *MultiPoint) Marshal() ([]byte, error) {
	mp.init()

	var b bytes.Buffer
	b.Grow(64)
	w := bufio.NewWriter(&b)

	// first write the number of points.
	count := int32(len(mp.s2points))
	err := binary.Write(w, binary.BigEndian, count)
	if err != nil {
		return nil, err
	}
	// write the points.
	for _, s2point := range mp.s2points {
		err := s2point.Encode(w)
		if err != nil {
			return nil, err
		}
	}

	w.Flush()
	return append([]byte{MultiPointTypePrefix}, b.Bytes()...), nil
}

func (mp *MultiPoint) Type() string {
	return strings.ToLower(mp.Typ)
}

func (mp *MultiPoint) Value() ([]byte, error) {
	return jsoniter.Marshal(mp)
}

func (mp *MultiPoint) Intersects(other index.GeoJSON) (bool, error) {
	mp.init()

	for _, s2point := range mp.s2points {
		rv, err := checkPointIntersectsShape(s2point, mp, other)
		if rv && err == nil {
			return rv, nil
		}
	}

	return false, nil
}

func (mp *MultiPoint) Contains(other index.GeoJSON) (bool, error) {
	mp.init()

	rv, err := checkPointContainsShape(mp.s2points, other)
	if rv && err == nil {
		return rv, nil
	}

	return false, nil
}

func (mp *MultiPoint) Coordinates() [][]float64 {
	return mp.Vertices
}

func (mp *MultiPoint) Members() []index.GeoJSON {
	if len(mp.Vertices) > 0 && len(mp.s2points) == 0 {
		points := make([]index.GeoJSON, len(mp.Vertices))
		for pos, vertices := range mp.Vertices {
			points[pos] = NewGeoJsonPoint(vertices)
		}
		return points
	}

	points := make([]index.GeoJSON, len(mp.s2points))
	for pos, point := range mp.s2points {
		points[pos] = &Point{s2point: point}
	}
	return points
}

// IndexCells returns the multipoint's covering: points have no area, so the
// result is one maxCellLevel cross cell per distinct point cell and never any
// inner cells. Cells are deduplicated, since points can share a cell.
func (mp *MultiPoint) IndexCells() (inner, cross []uint64) {
	mp.init()
	seen := make(map[uint64]struct{}, len(mp.s2points))
	cross = make([]uint64, 0, len(mp.s2points))
	for _, pt := range mp.s2points {
		if pt == nil {
			continue
		}
		cell := pointCell(*pt)
		if _, ok := seen[cell]; ok {
			continue
		}
		seen[cell] = struct{}{}
		cross = append(cross, cell)
	}
	return nil, cross
}

// QueryCells delegates to IndexCells: each point maps to exactly one cell, so
// the query-time coverer would produce the same cross cells as the index-time
// coverer.
func (mp *MultiPoint) QueryCells() (inner, cross []uint64) {
	return mp.IndexCells()
}

func (mp *MultiPoint) BoundingBox() index.GeoJSON {
	mp.init()
	r := s2.EmptyRect()
	for _, pt := range mp.s2points {
		if pt == nil {
			continue
		}
		r = r.Union(pt.RectBound())
	}
	return envelopeFromRect(r)
}

func (mp *MultiPoint) IndexTokens(s *s2.RegionTermIndexer) []string {
	mp.init()
	var rv []string
	for _, s2point := range mp.s2points {
		terms := s.GetIndexTermsForPoint(*s2point, "")
		rv = append(rv, terms...)
	}
	return StripCoveringTerms(rv)
}

func (mp *MultiPoint) QueryTokens(s *s2.RegionTermIndexer) []string {
	mp.init()
	var rv []string
	for _, s2point := range mp.s2points {
		terms := s.GetQueryTermsForPoint(*s2point, "")
		rv = append(rv, terms...)
	}

	return StripCoveringTerms(rv)
}

// checkPointIntersectsShape checks for intersection between
// the point and the shape in the document.
func checkPointIntersectsShape(point *s2.Point, shapeIn, other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		// check if the points are equal
		if point.ApproxEqual(*p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check if any of the points are equal
		for _, p := range p2.s2points {
			if point.ApproxEqual(*p) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a polygon.
	if p2, ok := other.(*Polygon); ok {
		// check if the point is contained within the polygon.
		if polygonsIntersectsPoint([]*s2.Polygon{p2.s2pgn}, point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipolygon.
	if p2, ok := other.(*MultiPolygon); ok {
		// check if the point is contained within any of the polygons
		if polygonsIntersectsPoint(p2.s2pgns, point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a linestring.
	if p2, ok := other.(*LineString); ok {
		// project the point to the linestring and check if
		// the projection is equal to the point.
		if polylineIntersectsPoint([]*s2.Polyline{p2.pl}, point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multilinestring.
	if p2, ok := other.(*MultiLineString); ok {
		// check the intersection for any linestring in the array.
		if polylineIntersectsPoint(p2.pls, point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a geometrycollection.
	if gc, ok := other.(*GeometryCollection); ok {
		// check for intersection across every member shape.
		if geometryCollectionIntersectsShape(gc, shapeIn) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a circle.
	if c, ok := other.(*Circle); ok {
		// check if the point is contained within the circle
		// by calculating the distance between the point and the
		// center of the circle.
		if c.s2cap.ContainsPoint(*point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is an envelope.
	if e, ok := other.(*Envelope); ok {
		// check if the point is contained by the envelope
		// by checking if the point is within its bounds
		if e.r.ContainsPoint(*point) {
			return true, nil
		}

		return false, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s "+
		" found in document", other.Type())
}

// checkPointContainsShape checks whether the given shape in
// in the document is approximately contained with the point.
func checkPointContainsShape(points []*s2.Point,
	other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		for _, point := range points {
			if point.ApproxEqual(*p2.s2point) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a multipoint, if so containment is
	// checked for every point in the multipoint with every given point.
	if p2, ok := other.(*MultiPoint); ok {
		// check the containment for every point in the collection.
		lookup := make(map[int]struct{})
		for _, qpoint := range points {
			for pos, dpoint := range p2.s2points {
				if _, done := lookup[pos]; done {
					continue
				}
				// already processed all the points in the multipoint.
				if len(lookup) == len(p2.s2points) {
					return true, nil
				}

				if qpoint.ApproxEqual(*dpoint) {
					lookup[pos] = struct{}{}
				}
			}
		}

		return len(lookup) == len(p2.s2points), nil
	}

	// as point is a non closed shape, containment isn't feasible
	// for other higher dimensions.
	return false, nil
}
