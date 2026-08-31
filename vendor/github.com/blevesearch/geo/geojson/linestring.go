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

// LineString represents the geoJSON linestring type and it
// implements the index.GeoJSON interface.
type LineString struct {
	Typ      string      `json:"type"`
	Vertices [][]float64 `json:"coordinates"`
	pl       *s2.Polyline
}

// NewGeoJsonLinestring instantiates a LineString from the given coordinates.
func NewGeoJsonLinestring(points [][]float64) index.GeoJSON {
	rv := &LineString{Typ: LineStringType, Vertices: points}
	rv.init()
	return rv
}

func (ls *LineString) init() {
	if ls.pl == nil {
		latlngs := make([]s2.LatLng, len(ls.Vertices))
		for i, vertex := range ls.Vertices {
			latlngs[i] = s2.LatLngFromDegrees(vertex[1], vertex[0])
		}
		ls.pl = s2.PolylineFromLatLngs(latlngs)
	}
}

func (ls *LineString) Type() string {
	return strings.ToLower(ls.Typ)
}

func (ls *LineString) Value() ([]byte, error) {
	return jsoniter.Marshal(ls)
}

func (ls *LineString) Marshal() ([]byte, error) {
	ls.init()

	var b bytes.Buffer
	b.Grow(50)
	w := bufio.NewWriter(&b)
	err := ls.pl.Encode(w)
	if err != nil {
		return nil, err
	}

	w.Flush()
	return append([]byte{LineStringTypePrefix}, b.Bytes()...), nil
}

func (ls *LineString) Intersects(other index.GeoJSON) (bool, error) {
	ls.init()

	return checkLineStringsIntersectsShape([]*s2.Polyline{ls.pl}, ls, other)
}

func (ls *LineString) Contains(other index.GeoJSON) (bool, error) {
	ls.init()
	return checkLineStringsContainsShape([]*s2.Polyline{ls.pl}, other)
}

func (ls *LineString) Coordinates() [][]float64 {
	return ls.Vertices
}

// IndexCells returns the linestring's covering: a polyline has no area, so
// every covering cell is a cross cell and inner is always nil.
func (ls *LineString) IndexCells() (inner, cross []uint64) {
	ls.init()
	if ls.pl == nil {
		return nil, nil
	}
	return indexCellsFromRegion(ls.pl)
}

// QueryCells returns the linestring's query-time covering; like IndexCells,
// it can only contain cross cells.
func (ls *LineString) QueryCells() (inner, cross []uint64) {
	ls.init()
	if ls.pl == nil {
		return nil, nil
	}
	return queryCellsFromRegion(ls.pl)
}

func (ls *LineString) BoundingBox() index.GeoJSON {
	ls.init()
	if ls.pl == nil {
		return envelopeFromRect(s2.EmptyRect())
	}
	return envelopeFromRect(ls.pl.RectBound())
}

func (ls *LineString) IndexTokens(s *s2.RegionTermIndexer) []string {
	ls.init()
	return StripCoveringTerms(s.GetIndexTermsForRegion(ls.pl.CapBound(), ""))
}

func (ls *LineString) QueryTokens(s *s2.RegionTermIndexer) []string {
	ls.init()
	return StripCoveringTerms(s.GetQueryTermsForRegion(ls.pl.CapBound(), ""))
}

// --------------------------------------------------------
// MultiLineString represents the geoJSON multilinestring type
// and it implements the index.GeoJSON interface as well as the
// compositeShape interface.
type MultiLineString struct {
	Typ      string        `json:"type"`
	Vertices [][][]float64 `json:"coordinates"`
	pls      []*s2.Polyline
}

// NewGeoJsonMultilinestring instantiates a MultiLineString from the given
// coordinates.
func NewGeoJsonMultilinestring(points [][][]float64) index.GeoJSON {
	rv := &MultiLineString{Typ: MultiLineStringType, Vertices: points}
	rv.init()
	return rv
}

func (mls *MultiLineString) init() {
	if mls.pls == nil {
		mls.pls = s2PolylinesFromCoordinates(mls.Vertices)
	}
}

func (mls *MultiLineString) Type() string {
	return strings.ToLower(mls.Typ)
}

func (mls *MultiLineString) Value() ([]byte, error) {
	return jsoniter.Marshal(mls)
}

func (mls *MultiLineString) Marshal() ([]byte, error) {
	mls.init()

	var b bytes.Buffer
	b.Grow(256)
	w := bufio.NewWriter(&b)

	// first write the number of linestrings.
	count := int32(len(mls.pls))
	err := binary.Write(w, binary.BigEndian, count)
	if err != nil {
		return nil, err
	}
	// write the lines.
	for _, ls := range mls.pls {
		err := ls.Encode(w)
		if err != nil {
			return nil, err
		}
	}

	w.Flush()
	return append([]byte{MultiLineStringTypePrefix}, b.Bytes()...), nil
}

func (mls *MultiLineString) Intersects(other index.GeoJSON) (bool, error) {
	mls.init()
	return checkLineStringsIntersectsShape(mls.pls, mls, other)
}

func (mls *MultiLineString) Contains(other index.GeoJSON) (bool, error) {
	mls.init()
	return checkLineStringsContainsShape(mls.pls, other)
}

func (mls *MultiLineString) Coordinates() [][][]float64 {
	return mls.Vertices
}

func (mls *MultiLineString) Members() []index.GeoJSON {
	if len(mls.Vertices) > 0 && len(mls.pls) == 0 {
		lines := make([]index.GeoJSON, len(mls.Vertices))
		for pos, vertices := range mls.Vertices {
			lines[pos] = NewGeoJsonLinestring(vertices)
		}
		return lines
	}

	lines := make([]index.GeoJSON, len(mls.pls))
	for pos, pl := range mls.pls {
		lines[pos] = &LineString{pl: pl}
	}
	return lines
}

// polylinesRegionUnion builds an s2.RegionUnion from the non-nil polylines.
func polylinesRegionUnion(pls []*s2.Polyline) s2.RegionUnion {
	ru := make(s2.RegionUnion, 0, len(pls))
	for _, pl := range pls {
		if pl != nil {
			ru = append(ru, pl)
		}
	}
	return ru
}

// IndexCells returns the multilinestring's covering, computed as a single
// covering of the union of its polylines. Polylines have no area, so every
// covering cell is a cross cell and inner is always nil.
func (mls *MultiLineString) IndexCells() (inner, cross []uint64) {
	mls.init()
	ru := polylinesRegionUnion(mls.pls)
	if len(ru) == 0 {
		return nil, nil
	}
	return indexCellsFromRegion(ru)
}

// QueryCells returns the multilinestring's query-time covering; like
// IndexCells, it can only contain cross cells.
func (mls *MultiLineString) QueryCells() (inner, cross []uint64) {
	mls.init()
	ru := polylinesRegionUnion(mls.pls)
	if len(ru) == 0 {
		return nil, nil
	}
	return queryCellsFromRegion(ru)
}

func (mls *MultiLineString) BoundingBox() index.GeoJSON {
	mls.init()
	r := s2.EmptyRect()
	for _, pl := range mls.pls {
		if pl != nil {
			r = r.Union(pl.RectBound())
		}
	}
	return envelopeFromRect(r)
}

func (mls *MultiLineString) IndexTokens(s *s2.RegionTermIndexer) []string {
	mls.init()
	var rv []string
	for _, ls := range mls.pls {
		terms := s.GetIndexTermsForRegion(ls.CapBound(), "")
		rv = append(rv, terms...)
	}

	return StripCoveringTerms(rv)
}

func (mls *MultiLineString) QueryTokens(s *s2.RegionTermIndexer) []string {
	mls.init()

	var rv []string
	for _, ls := range mls.pls {
		terms := s.GetQueryTermsForRegion(ls.CapBound(), "")
		rv = append(rv, terms...)
	}

	return StripCoveringTerms(rv)
}

// checkLineStringsIntersectsShape checks whether the given linestrings
// intersects with the shape in the document.
func checkLineStringsIntersectsShape(pls []*s2.Polyline, shapeIn,
	other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if polylineIntersectsPoint(pls, p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check the intersection for any point in the collection.
		for _, point := range p2.s2points {
			if polylineIntersectsPoint(pls, point) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a polygon.
	if p2, ok := other.(*Polygon); ok {
		if polylineIntersectsPolygons(pls, []*s2.Polygon{p2.s2pgn}) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipolygon.
	if p2, ok := other.(*MultiPolygon); ok {
		// check the intersection for any polygon in the collection.
		if polylineIntersectsPolygons(pls, p2.s2pgns) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a linestring.
	if ls, ok := other.(*LineString); ok {
		for _, pl := range pls {
			if ls.pl.Intersects(pl) {
				return true, nil
			}
		}

		return false, nil
	}

	// check if the other shape is a multilinestring.
	if mls, ok := other.(*MultiLineString); ok {
		for _, ls := range pls {
			for _, docLineString := range mls.pls {
				if ls.Intersects(docLineString) {
					return true, nil
				}
			}
		}

		return false, nil
	}

	if gc, ok := other.(*GeometryCollection); ok {
		// check whether the linestring intersects with any of the
		// shapes within the geometrycollection.
		if geometryCollectionIntersectsShape(gc, shapeIn) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a circle.
	if c, ok := other.(*Circle); ok {
		centre := c.s2cap.Center()
		for _, pl := range pls {
			for i := 0; i < pl.NumEdges(); i++ {
				edge := pl.Edge(i)
				distance := s2.DistanceFromSegment(centre, edge.V0, edge.V1)
				if distance <= c.s2cap.Radius() {
					return true, nil
				}
			}
		}

		return false, nil
	}

	// check if the other shape is a envelope.
	if e, ok := other.(*Envelope); ok {
		res := rectangleIntersectsWithLineStrings(e.r, pls)

		return res, nil
	}

	return false, fmt.Errorf("unknown geojson type: %s "+
		"found in document", other.Type())
}

// checkLineStringsContainsShape checks whether the given points or
// multipoints lie on the linestrings (anywhere along their edges,
// not just at the vertices).
func checkLineStringsContainsShape(pls []*s2.Polyline,
	other index.GeoJSON) (bool, error) {
	// check if the other shape is a point.
	if p2, ok := other.(*Point); ok {
		if polylineIntersectsPoint(pls, p2.s2point) {
			return true, nil
		}

		return false, nil
	}

	// check if the other shape is a multipoint.
	if p2, ok := other.(*MultiPoint); ok {
		// check the containment for every point in the collection.
		for _, point := range p2.s2points {
			if !polylineIntersectsPoint(pls, point) {
				return false, nil
			}
		}

		return true, nil
	}

	return false, nil
}
