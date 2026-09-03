package mapping

import (
	"strings"
)

// VectorIndexSuffix is appended to a TypeVector field's name to produce the
// bleve sibling field that faiss actually indexes (e.g. "imageVector" ->
// "imageVector_faiss"), mirroring the "location" / "location_geopoint" split.
// bleve hands vector fields to faiss and cannot return them as stored fields
// (verified: even Store:true yields nothing), so the original field is mapped
// stored-only instead and round-trips like any other field, keeping vectors
// alive across Move / Delete / Restore. OpenSearch keeps vectors in _source
// and indexes the original field directly, no sibling there.
const VectorIndexSuffix = "_faiss"

// AddVectorIndexSiblings writes, for each TypeVector override present in m,
// the vector again under the suffixed sibling key. bleve vectors-build upsert
// path only: without the vectors tag the sibling has no mapping and bleve's
// dynamic mapping would index the raw floats.
func AddVectorIndexSiblings(m map[string]any, overrides map[string]FieldOpts) {
	for key, opts := range overrides {
		if opts.Type != TypeVector {
			continue
		}
		parts := strings.Split(key, ".")
		parent := m
		for _, p := range parts[:len(parts)-1] {
			next, ok := parent[p].(map[string]any)
			if !ok {
				parent = nil
				break
			}
			parent = next
		}
		if parent == nil {
			continue
		}
		leaf := parts[len(parts)-1]
		if vec, ok := parent[leaf].([]any); ok && len(vec) > 0 {
			parent[leaf+VectorIndexSuffix] = vec
		}
	}
}
