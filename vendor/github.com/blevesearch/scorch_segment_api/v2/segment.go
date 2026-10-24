//  Copyright (c) 2017 Couchbase, Inc.
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

package segment

import (
	"fmt"

	"github.com/RoaringBitmap/roaring/v2"
	index "github.com/blevesearch/bleve_index_api"
)

var ErrClosed = fmt.Errorf("index closed")

// StoredFieldValueVisitor defines a callback to be visited for each
// stored field value.  The return value determines if the visitor
// should keep going.  Returning true continues visiting, false stops.
type StoredFieldValueVisitor func(field string, typ byte, value []byte, pos []uint64) bool

type Segment interface {
	DiskStatsReporter

	Dictionary(field string) (TermDictionary, error)

	VisitStoredFields(num uint64, visitor StoredFieldValueVisitor) error

	DocID(num uint64) ([]byte, error)

	Count() uint64

	DocNumbers([]string) (*roaring.Bitmap, error)

	Fields() []string

	Close() error

	Size() int

	AddRef()
	DecRef() error
}

type UnpersistedSegment interface {
	Segment
	Persist(path string) error
}

type PersistedSegment interface {
	Segment
	Path() string
}

type UpdatableSegment interface {
	Segment
	GetUpdatedFields() map[string]*index.UpdateFieldInfo
	SetUpdatedFields(fieldInfo map[string]*index.UpdateFieldInfo)
}

type SegmentWithCallbacks interface {
	Segment
	CallbackId() string
}

type TermDictionary interface {
	PostingsList(term []byte, except *roaring.Bitmap, prealloc PostingsList) (PostingsList, error)

	AutomatonIterator(a Automaton,
		startKeyInclusive, endKeyExclusive []byte) DictionaryIterator

	Contains(key []byte) (bool, error)

	// returns total number of terms in the term dictionary
	Cardinality() int
}

type DictionaryIterator interface {
	Next() (*index.DictEntry, error)
}

type PostingsList interface {
	DiskStatsReporter

	Iterator(includeFreq, includeNorm, includeLocations bool, prealloc PostingsIterator) PostingsIterator

	Size() int

	Count() uint64

	// NOTE deferred for future work

	// And(other PostingsList) PostingsList
	// Or(other PostingsList) PostingsList
}

type PostingsIterator interface {
	DiskStatsReporter

	// The caller is responsible for copying whatever it needs from
	// the returned Posting instance before calling Next(), as some
	// implementations may return a shared instance to reduce memory
	// allocations.
	Next() (Posting, error)

	// Advance will return the posting with the specified doc number
	// or if there is no such posting, the next posting.
	// Callers MUST NOT attempt to pass a docNum that is less than or
	// equal to the currently visited posting doc Num.
	Advance(docNum uint64) (Posting, error)

	Size() int
}

type DiskStatsReporter interface {
	// BytesRead returns the bytes read from the disk as
	// part of the current running query.
	BytesRead() uint64

	// ResetBytesRead is used by the parent layer
	// to reset the bytes read value to a consistent
	// value during operations such as merging of segments.
	ResetBytesRead(uint64)

	// BytesWritten returns the bytes written to disk while
	// building an index
	BytesWritten() uint64
}

type OptimizablePostingsIterator interface {
	ActualBitmap() *roaring.Bitmap
	DocNum1Hit() (uint64, bool)
	ReplaceActual(*roaring.Bitmap)
}

type Posting interface {
	Number() uint64

	Frequency() uint64
	Norm() float64

	Locations() []Location

	Size() int
}

type Location interface {
	Field() string
	Start() uint64
	End() uint64
	Pos() uint64
	ArrayPositions() []uint64
	Size() int
}

// DocValueVisitable is implemented by various scorch segment
// implementations with persistence for the un inverting of the
// postings or other indexed values.
type DocValueVisitable interface {
	VisitDocValues(localDocNum uint64, fields []string,
		visitor index.DocValueVisitor, optional DocVisitState) (DocVisitState, error)

	// VisitableDocValueFields implementation should return
	// the list of fields which are document value persisted and
	// therefore visitable by the above VisitDocValues method.
	VisitableDocValueFields() ([]string, error)
}

type DocVisitState interface {
	DiskStatsReporter
}

type StatsReporter interface {
	ReportBytesWritten(bytesWritten uint64)
}

type FieldStatsReporter interface {
	UpdateFieldStats(FieldStats)
}

type VectorFieldStatsReporter interface {
	UpdateVectorFieldStats(FieldStats)
}

type FieldStats interface {
	Store(statName, fieldName string, value uint64)
	Aggregate(stats FieldStats)
	Fetch() map[string]map[string]uint64
}

// ThesaurusSegment provides access to a thesaurus within a specific segment of the index.
type ThesaurusSegment interface {
	Segment
	// Thesaurus returns the Thesaurus with the specified name.
	Thesaurus(name string) (Thesaurus, error)
}

// Thesaurus encapsulates a structured collection of terms and their associated synonyms.
type Thesaurus interface {
	// SynonymsList retrieves a list of synonyms for the specified term. The `except` parameter
	// excludes specific synonyms, such as those originating from deleted documents. The `prealloc`
	// parameter allows the use of preallocated memory to optimize performance.
	SynonymsList(term []byte, except *roaring.Bitmap, prealloc SynonymsList) (SynonymsList, error)

	// AutomatonIterator creates an iterator over the thesaurus keys/terms using the provided automaton.
	// The iteration is constrained by the specified key range [startKeyInclusive, endKeyExclusive).
	// These terms or keys are the ones that have a SynonymsList associated with them, in the thesaurus.
	AutomatonIterator(a Automaton, startKeyInclusive, endKeyExclusive []byte) ThesaurusIterator

	// Contains checks if the given key exists in the thesaurus.
	Contains(key []byte) (bool, error)
}

// ThesaurusIterator iterates over terms in a thesaurus.
type ThesaurusIterator interface {
	// Next returns the next entry in the thesaurus or an error if iteration fails.
	Next() (*index.ThesaurusEntry, error)
}

// SynonymsList represents a list of synonyms for a term.
type SynonymsList interface {
	// Iterator returns an iterator to traverse the list of synonyms.
	// The `prealloc` parameter can be used to reuse existing memory for the iterator.
	Iterator(prealloc SynonymsIterator) SynonymsIterator

	Size() int
}

// SynonymsIterator provides a mechanism to iterate over a list of synonyms.
type SynonymsIterator interface {
	// Next returns the next synonym in the list or an error if iteration fails.
	Next() (Synonym, error)

	Size() int
}

// Synonym represents a single synonym for a term in the thesaurus.
type Synonym interface {
	// Number returns the document number from which the synonym originates.
	Number() uint32
	// Term returns the textual representation of the synonym.
	Term() string

	Size() int
}

// NestedSegment is an optional interface that a Segment may implement
// to provide access to nested document relationships within that segment.
type NestedSegment interface {
	Segment
	// Ancestors returns a slice of ancestor IDs for the given document ID.
	// If the document has no ancestors or if the segment does not support nested documents,
	// a slice containing only the document ID itself is returned.
	Ancestors(docID uint64, prealloc []index.AncestorID) []index.AncestorID

	// CountRoot returns the number of root documents in the segment, excluding any documents
	// that are marked as deleted in the provided bitmap. If the segment does not support nested
	// documents, it returns the total document count minus the count of deleted documents.
	// A root document is defined as a document that is not a child of any other document.
	CountRoot(deleted *roaring.Bitmap) uint64

	// AddNestedDocuments updates the provided bitmap to include all nested documents
	// associated with documents marked as deleted in the bitmap. This ensures that when
	// a parent document is deleted, all its nested child documents are also considered deleted.
	AddNestedDocuments(deleted *roaring.Bitmap) *roaring.Bitmap
}

// GeoShapeV2Segment is an optional interface that a Segment may implement
// to provide access to GeoShapeV2Data within the segment.
type GeoShapeV2Segment interface {
	Segment

	// GeoShapeV2Data returns the geo shape data for the given field,
	// excluding any documents present in the except bitmap.
	GeoShapeV2Data(field string, except *roaring.Bitmap) (GeoShapeV2Data, error)
}

// GeoShapeV2Data provides methods to access separate parts of the GeoShapeV2 data.
// Internally, geo docIDs are sequential from 0 to NumDocs()-1; DocNums() maps each
// geo docID (the slice index) to its segment document number.
type GeoShapeV2Data interface {
	// InnerCells returns all of the shapes' inner cells in ascending order.
	InnerCells() []uint64
	// InnerDocIDs returns the geo docIDs parallel to InnerCells().
	InnerDocIDs() []uint32
	// CrossCells returns all of the shapes' cross cells in ascending order.
	CrossCells() []uint64
	// CrossDocIDs returns the geo docIDs parallel to CrossCells().
	CrossDocIDs() []uint32
	// NumDocs returns the number of documents indexed.
	NumDocs() uint64
	// DocNums returns the mapping from geo docID (the slice index) to
	// segment document number.
	DocNums() []uint32
	// DocScores returns the precomputed inner and cross cell scores (in
	// that order) for the documents indexed, each indexed by geo docID.
	DocScores() (innerScores, crossScores []uint64)
	// BoundingBox returns the bounding box bytes for the given geo docID.
	BoundingBox(geoDocID uint32) ([]byte, error)
	// Shape returns the shape bytes for the given geo docID.
	Shape(geoDocID uint32) ([]byte, error)
	// Excluded returns the bitmap of geo document IDs that are excluded
	// from the index.
	Excluded() *roaring.Bitmap
	// GetScoreMap returns an empty score map, keyed by geo docID, from a
	// segment-level pool.
	GetScoreMap() map[uint32]uint64
	// PutScoreMap clears the score map obtained via GetScoreMap and
	// returns it to the segment-level pool.
	PutScoreMap(scores map[uint32]uint64)
	// Close closes the GeoShapeV2Data and releases any associated resources.
	Close()
}
