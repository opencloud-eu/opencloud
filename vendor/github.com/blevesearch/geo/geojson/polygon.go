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

// Polygon represents the geoJSON polygon type
// and it implements the index.GeoJSON interface.
type Polygon struct {
	Typ      string        `json:"type"`
	Vertices [][][]float64 `json:"coordinates"`
	s2pgn    *s2.Polygon
}

func NewGeoJsonPolygon(points [][][]float64) index.GeoJSON {
	rv := &Polygon{Typ: PolygonType, Vertices: points}
	rv.init()
	return rv
}

func (pg *Polygon) init() {
	if pg.s2pgn == nil {
		pg.s2pgn = s2PolygonFromCoordinates(pg.Vertices)
	}
}

func (pg *Polygon) Type() string {
	return strings.ToLower(pg.Typ)
}

func (pg *Polygon) Value() ([]byte, error) {
	return jsoniter.Marshal(pg)
}

func (pg *Polygon) Marshal() ([]byte, error) {
	pg.init()

	var b bytes.Buffer
	b.Grow(128)
	w := bufio.NewWriter(&b)
	err := pg.s2pgn.Encode(w)
	if err != nil {
		return nil, err
	}

	w.Flush()
	return append([]byte{PolygonTypePrefix}, b.Bytes()...), nil
}

func (pg *Polygon) Intersects(other index.GeoJSON) (bool, error) {
	// lazily build the s2polygon for reuse.
	pg.init()

	return checkPolygonIntersectsShape(pg.s2pgn, pg, other)
}

func (pg *Polygon) Contains(other index.GeoJSON) (bool, error) {
	// lazily build the s2polygon for reuse.
	pg.init()

	return checkMultiPolygonContainsShape([]*s2.Polygon{pg.s2pgn}, pg, other)
}

func (pg *Polygon) Coordinates() [][][]float64 {
	return pg.Vertices
}

// IndexCells returns the polygon's covering partitioned into inner cells
// (fully contained in the polygon, excluding any holes) and cross cells
// (overlapping the boundary). The hole exclusion relies on the s2 polygon
// being built from oriented loops, so ContainsCell is false for cells
// inside interior rings.
func (pg *Polygon) IndexCells() (inner, cross []uint64) {
	pg.init()
	if pg.s2pgn == nil {
		return nil, nil
	}
	return indexCellsFromRegion(pg.s2pgn)
}

// QueryCells returns the polygon's query-time covering, partitioned the same
// way as IndexCells.
func (pg *Polygon) QueryCells() (inner, cross []uint64) {
	pg.init()
	if pg.s2pgn == nil {
		return nil, nil
	}
	return queryCellsFromRegion(pg.s2pgn)
}

func (pg *Polygon) BoundingBox() index.GeoJSON {
	pg.init()
	if pg.s2pgn == nil {
		return envelopeFromRect(s2.EmptyRect())
	}
	return envelopeFromRect(pg.s2pgn.RectBound())
}

func (pg *Polygon) IndexTokens(s *s2.RegionTermIndexer) []string {
	pg.init()
	terms := s.GetIndexTermsForRegion(
		pg.s2pgn.CapBound(), "")
	return StripCoveringTerms(terms)
}

func (pg *Polygon) QueryTokens(s *s2.RegionTermIndexer) []string {
	pg.init()
	terms := s.GetQueryTermsForRegion(
		pg.s2pgn.CapBound(), "")
	return StripCoveringTerms(terms)
}

// --------------------------------------------------------
// MultiPolygon represents the geoJSON multipolygon type
// and it implements the index.GeoJSON interface as well as the
// compositeShape interface.
type MultiPolygon struct {
	Typ      string          `json:"type"`
	Vertices [][][][]float64 `json:"coordinates"`
	s2pgns   []*s2.Polygon
}

func NewGeoJsonMultiPolygon(points [][][][]float64) index.GeoJSON {
	rv := &MultiPolygon{Typ: MultiPolygonType, Vertices: points}
	rv.init()
	return rv
}

func (mp *MultiPolygon) init() {
	if mp.s2pgns == nil {
		mp.s2pgns = make([]*s2.Polygon, len(mp.Vertices))
		for i, vertices := range mp.Vertices {
			pgn := s2PolygonFromCoordinates(vertices)
			mp.s2pgns[i] = pgn
		}
	}
}

func (mp *MultiPolygon) Type() string {
	return strings.ToLower(mp.Typ)
}

func (mp *MultiPolygon) Value() ([]byte, error) {
	return jsoniter.Marshal(mp)
}

func (mp *MultiPolygon) Marshal() ([]byte, error) {
	mp.init()

	var b bytes.Buffer
	b.Grow(512)
	w := bufio.NewWriter(&b)

	// first write the number of polygons.
	count := int32(len(mp.s2pgns))
	err := binary.Write(w, binary.BigEndian, count)
	if err != nil {
		return nil, err
	}
	// write the polygons.
	for _, pgn := range mp.s2pgns {
		err := pgn.Encode(w)
		if err != nil {
			return nil, err
		}
	}

	w.Flush()
	return append([]byte{MultiPolygonTypePrefix}, b.Bytes()...), nil
}

func (mp *MultiPolygon) Intersects(other index.GeoJSON) (bool, error) {
	mp.init()

	for _, pgn := range mp.s2pgns {
		rv, err := checkPolygonIntersectsShape(pgn, mp, other)
		if rv && err == nil {
			return true, nil
		}
	}

	return false, nil
}

func (mp *MultiPolygon) Contains(other index.GeoJSON) (bool, error) {
	mp.init()

	return checkMultiPolygonContainsShape(mp.s2pgns, mp, other)
}

func (mp *MultiPolygon) Coordinates() [][][][]float64 {
	return mp.Vertices
}

func (mp *MultiPolygon) Members() []index.GeoJSON {
	if len(mp.Vertices) > 0 && len(mp.s2pgns) == 0 {
		polygons := make([]index.GeoJSON, len(mp.Vertices))
		for pos, vertices := range mp.Vertices {
			polygons[pos] = NewGeoJsonPolygon(vertices)
		}
		return polygons
	}

	polygons := make([]index.GeoJSON, len(mp.s2pgns))
	for pos, pgn := range mp.s2pgns {
		polygons[pos] = &Polygon{s2pgn: pgn}
	}
	return polygons
}

// polygonsRegionUnion builds an s2.RegionUnion from the non-nil polygons.
func polygonsRegionUnion(pgns []*s2.Polygon) s2.RegionUnion {
	ru := make(s2.RegionUnion, 0, len(pgns))
	for _, pg := range pgns {
		if pg != nil {
			ru = append(ru, pg)
		}
	}
	return ru
}

// IndexCells returns the multipolygon's covering, computed as a single
// covering of the union of its polygons and partitioned into inner and
// cross cells.
func (mp *MultiPolygon) IndexCells() (inner, cross []uint64) {
	mp.init()
	ru := polygonsRegionUnion(mp.s2pgns)
	if len(ru) == 0 {
		return nil, nil
	}
	return indexCellsFromRegion(ru)
}

// QueryCells returns the multipolygon's query-time covering, partitioned the
// same way as IndexCells.
func (mp *MultiPolygon) QueryCells() (inner, cross []uint64) {
	mp.init()
	ru := polygonsRegionUnion(mp.s2pgns)
	if len(ru) == 0 {
		return nil, nil
	}
	return queryCellsFromRegion(ru)
}

func (mp *MultiPolygon) BoundingBox() index.GeoJSON {
	mp.init()
	r := s2.EmptyRect()
	for _, pg := range mp.s2pgns {
		if pg != nil {
			r = r.Union(pg.RectBound())
		}
	}
	return envelopeFromRect(r)
}

func (mp *MultiPolygon) IndexTokens(s *s2.RegionTermIndexer) []string {
	mp.init()

	var rv []string
	for _, s2pgn := range mp.s2pgns {
		terms := s.GetIndexTermsForRegion(s2pgn.CapBound(), "")
		rv = append(rv, terms...)
	}

	return StripCoveringTerms(rv)
}

func (mp *MultiPolygon) QueryTokens(s *s2.RegionTermIndexer) []string {
	mp.init()

	var rv []string
	for _, s2pgn := range mp.s2pgns {
		terms := s.GetQueryTermsForRegion(s2pgn.CapBound(), "")
		rv = append(rv, terms...)
	}

	return StripCoveringTerms(rv)
}

// checkPolygonIntersectsShape checks the intersection between the
// s2 polygon and the other shapes in the documents.
func checkPolygonIntersectsShape(s2pgn *s2.Polygon, shapeIn,
	other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if polygonsIntersectsPoint([]*s2.Polygon{s2pgn}, p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		for _, s2point := range p2.s2points {
			if polygonsIntersectsPoint([]*s2.Polygon{s2pgn}, s2point) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a polygon.
	if p2, ok := other.(*Polygon); ok {
		if s2pgn.Intersects(p2.s2pgn) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipolygon.
	if p2, ok := other.(*MultiPolygon); ok {
		// check the intersection for any polygon in the collection.
		for _, s2pgn1 := range p2.s2pgns {
			if s2pgn.Intersects(s2pgn1) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a linestring.
	if ls, ok := other.(*LineString); ok {
		if polylineIntersectsPolygons([]*s2.Polyline{ls.pl},
			[]*s2.Polygon{s2pgn}) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multilinestring.
	if mls, ok := other.(*MultiLineString); ok {
		if polylineIntersectsPolygons(mls.pls, []*s2.Polygon{s2pgn}) {
			return true, nil
		}

		return false, nil
	}

	if gc, ok := other.(*GeometryCollection); ok {
		// check whether the polygon intersects with any of the
		// member shapes of the geometry collection.
		if geometryCollectionIntersectsShape(gc, shapeIn) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a circle.
	if c, ok := other.(*Circle); ok {
		cp := c.s2cap.Center()
		radius := c.s2cap.Radius()

		projected := s2pgn.Project(&cp)
		distance := projected.Distance(cp)

		return distance <= radius, nil
	}

	// check if the other shape is a envelope.
	if e, ok := other.(*Envelope); ok {
		s2pgnInDoc := s2PolygonFromS2Rectangle(e.r)
		if s2pgn.Intersects(s2pgnInDoc) {
			return true, nil
		}
		return false, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s "+
		" found in document", other.Type())
}

// checkMultiPolygonContainsShape checks whether the given polygons
// collectively contains the shape in the document.
func checkMultiPolygonContainsShape(s2pgns []*s2.Polygon,
	shapeIn, other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if polygonsIntersectsPoint(s2pgns, p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check the containment for every point in the collection.
		idx := s2.NewShapeIndex()
		for _, s2pgn := range s2pgns {
			idx.Add(s2pgn)
		}

		for _, point := range p2.s2points {
			if !s2.NewContainsPointQuery(idx, s2.VertexModelClosed).Contains(*point) {
				return false, nil
			}
		}

		return true, nil
	}

	// check if the other shape is a polygon.
	if p2, ok := other.(*Polygon); ok {
		for _, s2pgn := range s2pgns {
			if s2pgn.Contains(p2.s2pgn) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a multipolygon.
	if p2, ok := other.(*MultiPolygon); ok {
		// check the intersection for every polygon in the collection.
		polygonsWithIn := make(map[int]struct{})
	nextPolygon:
		for pgnIndex, pgn := range p2.s2pgns {
			for _, s2pgn := range s2pgns {
				if s2pgn.Contains(pgn) {
					polygonsWithIn[pgnIndex] = struct{}{}
					continue nextPolygon
				}
			}
		}

		return len(p2.s2pgns) == len(polygonsWithIn), nil
	}

	// check if the other shape is a linestring.
	if ls, ok := other.(*LineString); ok {
		if polygonsContainsLineStrings(s2pgns,
			[]*s2.Polyline{ls.pl}) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multilinestring.
	if mls, ok := other.(*MultiLineString); ok {
		// check whether any of the linestring is inside the polygon.
		if polygonsContainsLineStrings(s2pgns, mls.pls) {
			return true, nil
		}

		return false, nil
	}

	if gc, ok := other.(*GeometryCollection); ok {
		shapesWithIn := make(map[int]struct{})
	nextShape:
		for pos, shape := range gc.Members() {
			for _, s2pgn := range s2pgns {
				contains, err := checkMultiPolygonContainsShape(
					[]*s2.Polygon{s2pgn}, shapeIn, shape)
				if err == nil && contains {
					shapesWithIn[pos] = struct{}{}
					continue nextShape
				}
			}
		}
		return len(shapesWithIn) == len(gc.Members()), nil
	}

	// check if the other shape is a circle.
	if c, ok := other.(*Circle); ok {
		cp := c.s2cap.Center()
		radius := c.s2cap.Radius()

		for _, s2pgn := range s2pgns {
			if s2pgn == nil {
				continue
			}

			// The full polygon covers the sphere, so it contains every circle,
			// but it carries no boundary edges to measure the radius against.
			if s2pgn.IsFull() {
				return true, nil
			}

			// Any other polygon without boundary edges is degenerate: it
			// encloses no area, so it cannot contain a circle, and there is no
			// boundary to project the centre onto.
			if s2pgn.NumEdges() == 0 {
				continue
			}

			if s2pgn.ContainsPoint(cp) {
				projected := s2pgn.ProjectToBoundary(&cp)
				distance := projected.Distance(cp)
				if distance >= radius {
					return true, nil
				}
			}
		}

		return false, nil
	}

	// check if the other shape is a envelope.
	if e, ok := other.(*Envelope); ok {
		// create a polygon from the rectangle and checks the containment.
		s2pgnInDoc := s2PolygonFromS2Rectangle(e.r)
		for _, s2pgn := range s2pgns {
			if s2pgn.Contains(s2pgnInDoc) {
				return true, nil
			}
		}

		return false, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s"+
		" found in document", other.Type())
}
