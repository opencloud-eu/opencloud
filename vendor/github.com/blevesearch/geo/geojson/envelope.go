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

// Envelope represents the envelope/bounding box type and it
// implements the index.GeoJSON interface.
type Envelope struct {
	Typ      string      `json:"type"`
	Vertices [][]float64 `json:"coordinates"`
	r        *s2.Rect
}

func NewGeoEnvelope(points [][]float64) index.GeoJSON {
	rv := &Envelope{Vertices: points, Typ: EnvelopeType}
	rv.init()

	return rv
}

func (e *Envelope) Type() string {
	return strings.ToLower(e.Typ)
}

func (e *Envelope) Value() ([]byte, error) {
	return jsoniter.Marshal(e)
}

func (e *Envelope) init() {
	if e.r == nil {
		e.r = s2RectFromBounds(e.Vertices[0], e.Vertices[1])
	}
}

func (e *Envelope) Marshal() ([]byte, error) {
	e.init()

	var b bytes.Buffer
	b.Grow(50)
	w := bufio.NewWriter(&b)
	err := e.r.Encode(w)
	if err != nil {
		return nil, err
	}

	w.Flush()
	return append([]byte{EnvelopeTypePrefix}, b.Bytes()...), nil
}

func (e *Envelope) Intersects(other index.GeoJSON) (bool, error) {
	e.init()

	return checkEnvelopeIntersectsShape(e.r, e, other)
}

func (e *Envelope) Contains(other index.GeoJSON) (bool, error) {
	e.init()

	return checkEnvelopeContainsShape(e.r, e, other)
}

// IndexCells returns the envelope's covering partitioned into inner cells
// (fully contained in the rectangle) and cross cells (overlapping its
// boundary).
func (e *Envelope) IndexCells() (inner, cross []uint64) {
	e.init()
	if e.r == nil {
		return nil, nil
	}
	return indexCellsFromRegion(*e.r)
}

// QueryCells returns the envelope's query-time covering, partitioned the
// same way as IndexCells.
func (e *Envelope) QueryCells() (inner, cross []uint64) {
	e.init()
	if e.r == nil {
		return nil, nil
	}
	return queryCellsFromRegion(*e.r)
}

func (e *Envelope) BoundingBox() index.GeoJSON {
	e.init()
	if e.r == nil {
		return envelopeFromRect(s2.EmptyRect())
	}
	return e
}

func (e *Envelope) IndexTokens(s *s2.RegionTermIndexer) []string {
	e.init()
	return StripCoveringTerms(s.GetIndexTermsForRegion(e.r.CapBound(), ""))
}

func (e *Envelope) QueryTokens(s *s2.RegionTermIndexer) []string {
	e.init()
	return StripCoveringTerms(s.GetQueryTermsForRegion(e.r.CapBound(), ""))
}

// checkEnvelopeIntersectsShape checks whether the given shape in
// the document is intersecting with the envelope/rectangle.
func checkEnvelopeIntersectsShape(s2rect *s2.Rect, shapeIn,
	other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if s2rect.ContainsPoint(*p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check the intersection for any point in the collection.
		for _, point := range p2.s2points {
			if s2rect.ContainsPoint(*point) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a polygon.
	if pgn, ok := other.(*Polygon); ok {
		if rectangleIntersectsWithPolygons(s2rect,
			[]*s2.Polygon{pgn.s2pgn}) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipolygon.
	if mpgn, ok := other.(*MultiPolygon); ok {
		// check the intersection for any polygon in the collection.
		if rectangleIntersectsWithPolygons(s2rect, mpgn.s2pgns) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a linestring.
	if ls, ok := other.(*LineString); ok {
		if rectangleIntersectsWithLineStrings(s2rect,
			[]*s2.Polyline{ls.pl}) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multilinestring.
	if mls, ok := other.(*MultiLineString); ok {
		if rectangleIntersectsWithLineStrings(s2rect, mls.pls) {
			return true, nil
		}

		return false, nil
	}

	if gc, ok := other.(*GeometryCollection); ok {
		// check for the intersection of every member shape
		// within the geometrycollection.
		if geometryCollectionIntersectsShape(gc, shapeIn) {
			return true, nil
		}
		return false, nil
	}

	// check if the other shape is a circle.
	if c, ok := other.(*Circle); ok {
		// check if the distance of the center of the circle from the
		// rectangle is less than the radius of the circle.
		if s2rect.DistanceToLatLng(s2.LatLngFromPoint(c.s2cap.Center())) <=
			c.s2cap.Radius() {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a envelope.
	if e, ok := other.(*Envelope); ok {
		if s2rect.Intersects(*e.r) {
			return true, nil
		}

		return false, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s"+
		" found in document", other.Type())
}

// checkEnvelopeContainsShape checks whether the given shape in
// the document is contained within the envelope/rectangle.
func checkEnvelopeContainsShape(s2rect *s2.Rect, shapeIn,
	other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if s2rect.ContainsPoint(*p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check the intersection for any point in the collection.
		for _, point := range p2.s2points {
			if !s2rect.ContainsPoint(*point) {
				return false, nil
			}
		}

		return true, nil
	}

	// check if the other shape is a polygon.
	if p2, ok := other.(*Polygon); ok {
		return s2rect.Contains(p2.s2pgn.RectBound()), nil
	}

	// check if the other shape is a multipolygon.
	if p2, ok := other.(*MultiPolygon); ok {
		// check the containment for every polygon in the collection.
		for _, s2pgn := range p2.s2pgns {
			if !s2rect.Contains(s2pgn.RectBound()) {
				return false, nil
			}
		}

		return true, nil
	}

	// check if the other shape is a linestring.
	if p2, ok := other.(*LineString); ok {
		return s2rect.Contains(p2.pl.RectBound()), nil
	}

	// check if the other shape is a multilinestring.
	if p2, ok := other.(*MultiLineString); ok {
		for _, pl := range p2.pls {
			if !s2rect.Contains(pl.RectBound()) {
				return false, nil
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
		if s2rect.Contains(c.s2cap.RectBound()) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a envelope.
	if e, ok := other.(*Envelope); ok {
		if s2rect.Contains(*e.r) {
			return true, nil
		}

		return false, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s"+
		" found in document", other.Type())
}
