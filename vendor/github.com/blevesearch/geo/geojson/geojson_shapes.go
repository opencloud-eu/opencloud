//  Copyright (c) 2022 Couchbase, Inc.
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
	index "github.com/blevesearch/bleve_index_api"
)

// s2Serializable is an optional interface for implementations
// supporting custom serialisation of data based out of s2's
// encode method.
type s2Serializable interface {
	// Marshal implementation should encode the shape using the
	// s2's encode methods with appropriate prefix bytes to
	// identify the type of the contents.
	Marshal() ([]byte, error)
}

const (
	PointType              = "point"
	MultiPointType         = "multipoint"
	LineStringType         = "linestring"
	MultiLineStringType    = "multilinestring"
	PolygonType            = "polygon"
	MultiPolygonType       = "multipolygon"
	GeometryCollectionType = "geometrycollection"
	CircleType             = "circle"
	EnvelopeType           = "envelope"
)

// These are the byte prefixes for identifying the
// shape contained within the doc values byte slice
// while decoding the contents during the query
// filtering phase.
const (
	PointTypePrefix              = byte(1)
	MultiPointTypePrefix         = byte(2)
	LineStringTypePrefix         = byte(3)
	MultiLineStringTypePrefix    = byte(4)
	PolygonTypePrefix            = byte(5)
	MultiPolygonTypePrefix       = byte(6)
	GeometryCollectionTypePrefix = byte(7)
	CircleTypePrefix             = byte(8)
	EnvelopeTypePrefix           = byte(9)
)

// compositeShape is an optional interface for the
// composite geoJSON shapes which is composed of
// multiple spatial shapes within it. Composite shapes
// like multipoint, multilinestring, multipolygon and
// geometrycollection shapes are supposed to implement
// this interface.
type compositeShape interface {
	// Members implementation returns the
	// geoJSON shapes composed within the shape.
	Members() []index.GeoJSON
}
