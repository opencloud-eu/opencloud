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
	"fmt"
	"strings"

	index "github.com/blevesearch/bleve_index_api"
	"github.com/blevesearch/geo/s2"
)

// Circle represents a custom circle type and it
// implements the index.GeoJSON interface.
type Circle struct {
	Typ            string    `json:"type"`
	Vertices       []float64 `json:"coordinates"`
	Radius         string    `json:"radius"`
	radiusInMeters float64
	s2cap          *s2.Cap
}

func NewGeoCircle(points []float64,
	radius string) index.GeoJSON {
	r, err := ParseDistance(radius)
	if err != nil {
		return nil
	}
	rv := &Circle{
		Typ:            CircleType,
		Vertices:       points,
		Radius:         radius,
		radiusInMeters: r,
	}
	rv.init()

	return rv
}

func (c *Circle) Type() string {
	return strings.ToLower(c.Typ)
}

func (c *Circle) Value() ([]byte, error) {
	return jsoniter.Marshal(c)
}

func (c *Circle) init() {
	if c.s2cap == nil {
		c.s2cap = s2Cap(c.Vertices, c.radiusInMeters)
	}
}

func (c *Circle) Marshal() ([]byte, error) {
	c.init()

	var b bytes.Buffer
	b.Grow(40)
	w := bufio.NewWriter(&b)
	err := c.s2cap.Encode(w)
	if err != nil {
		return nil, err
	}

	w.Flush()
	return append([]byte{CircleTypePrefix}, b.Bytes()...), nil
}

func (c *Circle) Intersects(other index.GeoJSON) (bool, error) {
	c.init()

	return checkCircleIntersectsShape(c.s2cap, c, other)
}

func (c *Circle) Contains(other index.GeoJSON) (bool, error) {
	c.init()
	return checkCircleContainsShape(c.s2cap, c, other)
}

func (c *Circle) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Typ      string    `json:"type"`
		Vertices []float64 `json:"coordinates"`
		Radius   string    `json:"radius"`
	}{}

	err := jsoniter.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	c.Typ = tmp.Typ
	c.Vertices = tmp.Vertices
	c.Radius = tmp.Radius
	if tmp.Radius != "" {
		c.radiusInMeters, err = ParseDistance(tmp.Radius)
	}

	return err
}

// IndexCells returns the circle's covering partitioned into inner cells
// (fully contained in the cap built from the center and radiusInMeters)
// and cross cells (overlapping the cap's boundary).
func (c *Circle) IndexCells() (inner, cross []uint64) {
	c.init()
	if c.s2cap == nil {
		return nil, nil
	}
	return indexCellsFromRegion(*c.s2cap)
}

// QueryCells returns the circle's query-time covering, partitioned the same
// way as IndexCells.
func (c *Circle) QueryCells() (inner, cross []uint64) {
	c.init()
	if c.s2cap == nil {
		return nil, nil
	}
	return queryCellsFromRegion(*c.s2cap)
}

func (c *Circle) BoundingBox() index.GeoJSON {
	c.init()
	if c.s2cap == nil {
		return envelopeFromRect(s2.EmptyRect())
	}
	return envelopeFromRect(c.s2cap.RectBound())
}

func (c *Circle) IndexTokens(s *s2.RegionTermIndexer) []string {
	c.init()
	return StripCoveringTerms(s.GetIndexTermsForRegion(c.s2cap.CapBound(), ""))
}

func (c *Circle) QueryTokens(s *s2.RegionTermIndexer) []string {
	c.init()
	return StripCoveringTerms(s.GetQueryTermsForRegion(c.s2cap.CapBound(), ""))
}

// checkCircleIntersectsShape checks for intersection of the
// shape in the document with the circle.
func checkCircleIntersectsShape(s2cap *s2.Cap, shapeIn,
	other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if s2cap.ContainsPoint(*p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check the intersection for any point in the collection.
		for _, point := range p2.s2points {
			if s2cap.ContainsPoint(*point) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a polygon.
	if p2, ok := other.(*Polygon); ok {
		centerPoint := s2cap.Center()
		projected := p2.s2pgn.Project(&centerPoint)
		distance := projected.Distance(centerPoint)
		return distance <= s2cap.Radius(), nil
	}

	// check if the other shape is a multipolygon.
	if p2, ok := other.(*MultiPolygon); ok {
		// check the intersection for any polygon in the collection.
		for _, s2pgn := range p2.s2pgns {
			centerPoint := s2cap.Center()
			projected := s2pgn.Project(&centerPoint)
			distance := projected.Distance(centerPoint)
			if distance <= s2cap.Radius() {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a linestring.
	if p2, ok := other.(*LineString); ok {
		projected, _ := p2.pl.Project(s2cap.Center())
		distance := projected.Distance(s2cap.Center())
		return distance <= s2cap.Radius(), nil
	}

	// check if the other shape is a multilinestring.
	if p2, ok := other.(*MultiLineString); ok {
		for _, pl := range p2.pls {
			projected, _ := pl.Project(s2cap.Center())
			distance := projected.Distance(s2cap.Center())
			if distance <= s2cap.Radius() {
				return true, nil
			}
		}

		return false, nil
	}

	if gc, ok := other.(*GeometryCollection); ok {
		// check whether the circle intersects with any of the
		// member shapes within the geometrycollection.
		if geometryCollectionIntersectsShape(gc, shapeIn) {
			return true, nil
		}
		return false, nil
	}

	// check if the other shape is a circle.
	if c, ok := other.(*Circle); ok {
		if s2cap.Intersects(*c.s2cap) {
			return true, nil
		}
		return false, nil
	}

	// check if the other shape is a envelope.
	if e, ok := other.(*Envelope); ok {
		if e.r.ContainsPoint(s2cap.Center()) {
			return true, nil
		}

		latlngs := []s2.LatLng{e.r.Vertex(0), e.r.Vertex(1),
			e.r.Vertex(2), e.r.Vertex(3), e.r.Vertex(0)}
		pl := s2.PolylineFromLatLngs(latlngs)
		projected, _ := pl.Project(s2cap.Center())
		distance := projected.Distance(s2cap.Center())
		if distance <= s2cap.Radius() {
			return true, nil
		}

		return false, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s"+
		" found in document", other.Type())
}

// checkCircleContainsShape checks for containment of the
// shape in the document with the circle.
func checkCircleContainsShape(s2cap *s2.Cap,
	shapeIn, other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if s2cap.ContainsPoint(*p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check the intersection for every point in the collection.
		for _, point := range p2.s2points {
			if !s2cap.ContainsPoint(*point) {
				return false, nil
			}
		}

		return true, nil
	}

	// check if the other shape is a polygon.
	if p2, ok := other.(*Polygon); ok {
		for i := 0; i < p2.s2pgn.NumEdges(); i++ {
			edge := p2.s2pgn.Edge(i)
			if !s2cap.ContainsPoint(edge.V0) ||
				!s2cap.ContainsPoint(edge.V1) {
				return false, nil
			}
		}
		return true, nil
	}

	// check if the other shape is a multipolygon.
	if p2, ok := other.(*MultiPolygon); ok {
		// check the containment for every polygon in the collection.
		for _, s2pgn := range p2.s2pgns {
			for i := 0; i < s2pgn.NumEdges(); i++ {
				edge := s2pgn.Edge(i)
				if !s2cap.ContainsPoint(edge.V0) ||
					!s2cap.ContainsPoint(edge.V1) {
					return false, nil
				}
			}
		}

		return true, nil
	}

	// check if the other shape is a linestring.
	if p2, ok := other.(*LineString); ok {
		for i := 0; i < p2.pl.NumEdges(); i++ {
			edge := p2.pl.Edge(i)
			// check whether both the end vertices are inside the circle.
			if s2cap.ContainsPoint(edge.V0) &&
				s2cap.ContainsPoint(edge.V1) {
				return true, nil
			}
		}
		return false, nil
	}

	// check if the other shape is a multilinestring.
	if p2, ok := other.(*MultiLineString); ok {
		for _, pl := range p2.pls {
			for i := 0; i < pl.NumEdges(); i++ {
				edge := pl.Edge(i)
				// check whether both the end vertices are inside the circle.
				if !(s2cap.ContainsPoint(edge.V0) && s2cap.ContainsPoint(edge.V1)) {
					return false, nil
				}
			}
		}
		return true, nil
	}

	if gc, ok := other.(*GeometryCollection); ok {
		for _, shape := range gc.Members() {
			contains, err := shapeIn.Contains(shape)
			if err == nil && !contains {
				return false, nil
			}
		}
		return true, nil
	}

	// check if the other shape is a circle.
	if c, ok := other.(*Circle); ok {
		if s2cap.Contains(*c.s2cap) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a envelope.
	if e, ok := other.(*Envelope); ok {
		for i := 0; i < 4; i++ {
			if !s2cap.ContainsPoint(
				s2.PointFromLatLng(e.r.Vertex(i))) {
				return false, nil
			}
		}

		return true, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s"+
		" found in document", other.Type())
}
