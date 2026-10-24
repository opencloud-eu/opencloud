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

import "github.com/blevesearch/geo/s2"

// Region coverer configuration for the geo shape v2 index.
const (
	minCellLevel  = 0
	maxCellLevel  = 16
	cellLevelMod  = 1
	maxIndexCells = 200
	maxQueryCells = 1000
)

var regionCovererIndexV2 = &s2.RegionCoverer{
	MinLevel: minCellLevel,
	MaxLevel: maxCellLevel,
	LevelMod: cellLevelMod,
	MaxCells: maxIndexCells,
}

var regionCovererQueryV2 = &s2.RegionCoverer{
	MinLevel: minCellLevel,
	MaxLevel: maxCellLevel,
	LevelMod: cellLevelMod,
	MaxCells: maxQueryCells,
}

// pointCell returns the single maxCellLevel cell that contains a point.
func pointCell(p s2.Point) uint64 {
	return uint64(s2.CellIDFromLatLng(s2.LatLngFromPoint(p)).Parent(maxCellLevel))
}

// envelopeFromRect builds an Envelope GeoJSON from an s2.Rect.
func envelopeFromRect(r s2.Rect) *Envelope {
	lo := r.Lo() // (minLat, minLng)
	hi := r.Hi() // (maxLat, maxLng)
	rc := r
	return &Envelope{
		Typ: EnvelopeType,
		Vertices: [][]float64{
			{lo.Lng.Degrees(), hi.Lat.Degrees()},
			{hi.Lng.Degrees(), lo.Lat.Degrees()},
		},
		r: &rc,
	}
}

// cellsFromRegion covers any s2.Region with the given coverer and partitions
// the single covering into inner cells (region fully contains the cell) and
// cross cells (boundary cells).
//
// Because the partition comes from one covering, the two sets are disjoint and
// exhaustive over the region. For regions without area (Point, Polyline,
// RegionUnion of those) ContainsCell is always false, so every cell is a cross
// cell and inner is nil.
func cellsFromRegion(region s2.Region, coverer *s2.RegionCoverer) (inner, cross []uint64) {
	covering := coverer.Covering(region)
	inner = make([]uint64, 0, len(covering))
	cross = make([]uint64, 0, len(covering))
	for _, id := range covering {
		if region.ContainsCell(s2.CellFromCellID(id)) {
			inner = append(inner, uint64(id))
		} else {
			cross = append(cross, uint64(id))
		}
	}
	return inner, cross
}

// indexCellsFromRegion partitions the index-time covering of the region.
func indexCellsFromRegion(region s2.Region) (inner, cross []uint64) {
	return cellsFromRegion(region, regionCovererIndexV2)
}

// queryCellsFromRegion partitions the query-time covering of the region.
func queryCellsFromRegion(region s2.Region) (inner, cross []uint64) {
	return cellsFromRegion(region, regionCovererQueryV2)
}
