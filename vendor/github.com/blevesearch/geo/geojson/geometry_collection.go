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
	"encoding/json"
	"strings"

	index "github.com/blevesearch/bleve_index_api"
	"github.com/blevesearch/geo/s2"
)

// GeometryCollection represents the geoJSON geometryCollection type
// and it implements the index.GeoJSON interface as well as the
// compositeShape interface.
type GeometryCollection struct {
	Typ    string          `json:"type"`
	Shapes []index.GeoJSON `json:"geometries"`
}

func (gc *GeometryCollection) Type() string {
	return strings.ToLower(gc.Typ)
}

func (gc *GeometryCollection) Value() ([]byte, error) {
	return jsoniter.Marshal(gc)
}

func (gc *GeometryCollection) Members() []index.GeoJSON {
	shapes := make([]index.GeoJSON, 0, len(gc.Shapes))
	for _, shape := range gc.Shapes {
		if cs, ok := shape.(compositeShape); ok {
			shapes = append(shapes, cs.Members()...)
		} else {
			shapes = append(shapes, shape)
		}
	}
	return shapes
}

func (gc *GeometryCollection) Marshal() ([]byte, error) {
	var b bytes.Buffer
	b.Grow(512)
	w := bufio.NewWriter(&b)

	// first write the number of shapes.
	count := int32(len(gc.Shapes))
	err := binary.Write(w, binary.BigEndian, count)
	if err != nil {
		return nil, err
	}

	var res []byte
	for _, shape := range gc.Shapes {
		if s, ok := shape.(s2Serializable); ok {
			sb, err := s.Marshal()
			if err != nil {
				return nil, err
			}
			// write the length of each shape.
			err = binary.Write(w, binary.BigEndian, int32(len(sb)))
			if err != nil {
				return nil, err
			}
			// track the shape contents.
			res = append(res, sb...)
		}
	}
	w.Flush()

	return append([]byte{GeometryCollectionTypePrefix}, append(b.Bytes(), res...)...), nil
}

func (gc *GeometryCollection) Intersects(other index.GeoJSON) (bool, error) {
	for _, shape := range gc.Members() {

		intersects, err := shape.Intersects(other)
		if intersects && err == nil {
			return true, nil
		}
	}
	return false, nil
}

func (gc *GeometryCollection) Contains(other index.GeoJSON) (bool, error) {
	// handle composite target shapes explicitly
	if cs, ok := other.(compositeShape); ok {
		otherShapes := cs.Members()
		shapesFoundWithIn := make(map[int]struct{})

	nextShape:
		for pos, shapeInDoc := range otherShapes {
			for _, shape := range gc.Members() {
				within, err := shape.Contains(shapeInDoc)
				if within && err == nil {
					shapesFoundWithIn[pos] = struct{}{}
					continue nextShape
				}
			}
		}

		return len(shapesFoundWithIn) == len(otherShapes), nil
	}

	for _, shape := range gc.Members() {
		within, err := shape.Contains(other)
		if within && err == nil {
			return true, nil
		}
	}

	return false, nil
}

// aggregateCells merges the member shapes' inner/cross cell coverings into a
// single deduplicated pair. A cell reported as inner by any member is fully
// contained in the union of the members, so it stays inner even when another
// member reports the same cell as a cross cell. The results are in no
// particular order.
func (gc *GeometryCollection) aggregateCells(
	cells func(index.GeoJSON) (inner, cross []uint64)) (inner, cross []uint64) {
	innerSet := make(map[uint64]struct{})
	crossSet := make(map[uint64]struct{})
	for _, s := range gc.Shapes {
		if s == nil {
			continue
		}
		in, cr := cells(s)
		for _, cell := range in {
			innerSet[cell] = struct{}{}
		}
		for _, cell := range cr {
			crossSet[cell] = struct{}{}
		}
	}

	inner = make([]uint64, 0, len(innerSet))
	for cell := range innerSet {
		inner = append(inner, cell)
	}
	cross = make([]uint64, 0, len(crossSet))
	for cell := range crossSet {
		// inner wins: the cell is fully contained in some member,
		// hence in the collection as a whole
		if _, ok := innerSet[cell]; ok {
			continue
		}
		cross = append(cross, cell)
	}
	return inner, cross
}

// IndexCells returns the union of the member shapes' coverings, deduplicated
// and reconciled (see aggregateCells).
func (gc *GeometryCollection) IndexCells() (inner, cross []uint64) {
	return gc.aggregateCells(func(s index.GeoJSON) ([]uint64, []uint64) {
		return s.IndexCells()
	})
}

// QueryCells returns the union of the member shapes' query-time coverings,
// deduplicated and reconciled (see aggregateCells).
func (gc *GeometryCollection) QueryCells() (inner, cross []uint64) {
	return gc.aggregateCells(func(s index.GeoJSON) ([]uint64, []uint64) {
		return s.QueryCells()
	})
}

func (gc *GeometryCollection) BoundingBox() index.GeoJSON {
	r := s2.EmptyRect()
	for _, s := range gc.Shapes {
		if s == nil {
			continue
		}
		// every shape in this package returns its bounding box as an
		// *Envelope; members with an empty bounding box contribute an
		// empty rect, which Union ignores
		if env, ok := s.BoundingBox().(*Envelope); ok && env != nil && env.r != nil {
			r = r.Union(*env.r)
		}
	}
	return envelopeFromRect(r)
}

// geometryCollectionIntersectsShape checks whether any member shape of the
// geometrycollection intersects with the given shape.
func geometryCollectionIntersectsShape(gc *GeometryCollection,
	shapeIn index.GeoJSON) bool {
	for _, shape := range gc.Members() {
		intersects, err := shapeIn.Intersects(shape)
		if err == nil && intersects {
			return true
		}
	}
	return false
}

func (gc *GeometryCollection) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Typ    string            `json:"type"`
		Shapes []json.RawMessage `json:"geometries"`
	}{}

	err := jsoniter.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	gc.Typ = tmp.Typ

	for _, shape := range tmp.Shapes {
		var t map[string]interface{}
		err := jsoniter.Unmarshal(shape, &t)
		if err != nil {
			return err
		}

		var typ string

		if val, ok := t["type"]; ok {
			typ = strings.ToLower(val.(string))
		} else {
			continue
		}

		switch typ {
		case PointType:
			var p Point
			err := jsoniter.Unmarshal(shape, &p)
			if err != nil {
				return err
			}
			p.init()
			gc.Shapes = append(gc.Shapes, &p)

		case MultiPointType:
			var mp MultiPoint
			err := jsoniter.Unmarshal(shape, &mp)
			if err != nil {
				return err
			}
			mp.init()
			gc.Shapes = append(gc.Shapes, &mp)

		case LineStringType:
			var ls LineString
			err := jsoniter.Unmarshal(shape, &ls)
			if err != nil {
				return err
			}
			ls.init()
			gc.Shapes = append(gc.Shapes, &ls)

		case MultiLineStringType:
			var mls MultiLineString
			err := jsoniter.Unmarshal(shape, &mls)
			if err != nil {
				return err
			}
			mls.init()
			gc.Shapes = append(gc.Shapes, &mls)

		case PolygonType:
			var pgn Polygon
			err := jsoniter.Unmarshal(shape, &pgn)
			if err != nil {
				return err
			}
			pgn.init()
			gc.Shapes = append(gc.Shapes, &pgn)

		case MultiPolygonType:
			var pgn MultiPolygon
			err := jsoniter.Unmarshal(shape, &pgn)
			if err != nil {
				return err
			}
			pgn.init()
			gc.Shapes = append(gc.Shapes, &pgn)
		case CircleType:
			var cir Circle
			err := jsoniter.Unmarshal(shape, &cir)
			if err != nil {
				return err
			}
			cir.init()
			gc.Shapes = append(gc.Shapes, &cir)
		case EnvelopeType:
			var env Envelope
			err := jsoniter.Unmarshal(shape, &env)
			if err != nil {
				return err
			}
			env.init()
			gc.Shapes = append(gc.Shapes, &env)
		}
	}

	return nil
}
