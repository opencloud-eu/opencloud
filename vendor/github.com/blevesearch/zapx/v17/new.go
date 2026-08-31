//  Copyright (c) 2018 Couchbase, Inc.
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

package zap

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
	"sync"
	"sync/atomic"

	index "github.com/blevesearch/bleve_index_api"
	segment "github.com/blevesearch/scorch_segment_api/v2"
	"github.com/golang/snappy"
)

var NewSegmentBufferNumResultsBump int = 100
var NewSegmentBufferNumResultsFactor float64 = 1.0
var NewSegmentBufferAvgBytesPerDocFactor float64 = 1.0

// ValidateDocFields can be set by applications to perform additional checks
// on fields in a document being added to a new segment, by default it does
// nothing.
// This API is experimental and may be removed at any time.
var ValidateDocFields = func(field index.Field) error {
	return nil
}

// New creates an in-memory zap-encoded SegmentBase from a set of Documents
func (z *ZapPlugin) New(results []index.Document) (
	segment.Segment, uint64, error) {
	return z.newWithChunkMode(results, DefaultChunkMode, nil)
}

func (z *ZapPlugin) NewUsing(results []index.Document, config map[string]interface{}) (
	segment.Segment, uint64, error) {
	return z.newWithChunkMode(results, DefaultChunkMode, config)
}

func (z *ZapPlugin) newWithChunkMode(results []index.Document,
	chunkMode uint32, config map[string]interface{}) (segment.Segment, uint64, error) {
	s := interimPool.Get().(*interim)

	zapStats, ok := config[segment.StatsKey].(*segment.Stats)
	if !ok || zapStats == nil {
		zapStats = new(segment.Stats)
	}

	s.stats = zapStats
	var br bytes.Buffer
	if s.lastNumDocs > 0 {
		// use previous results to initialize the buf with an estimate
		// size, but note that the interim instance comes from a
		// global interimPool, so multiple scorch instances indexing
		// different docs can lead to low quality estimates
		estimateAvgBytesPerDoc := int(float64(s.lastOutSize/s.lastNumDocs) *
			NewSegmentBufferNumResultsFactor)
		estimateNumResults := int(float64(len(results)+NewSegmentBufferNumResultsBump) *
			NewSegmentBufferAvgBytesPerDocFactor)
		br.Grow(estimateAvgBytesPerDoc * estimateNumResults)
	}

	atomic.AddUint64(&s.stats.TotNewRootDocsProcessed, uint64(len(results)))
	var err error
	s.results, s.edgeList = flattenNestedDocuments(results, s.edgeList)
	s.config = config
	s.chunkMode = chunkMode

	s.w = NewFileWriterEmpty(NewCountHashWriter(&br))

	storedIndexOffset, sectionsIndexOffset, err := s.convert()
	if err != nil {
		return nil, uint64(0), err
	}

	sb, err := InitSegmentBase(br.Bytes(), s.w.Sum32(), chunkMode,
		uint64(len(s.results)), storedIndexOffset, sectionsIndexOffset, config, s.stats)

	// get the bytes written before the interim's reset() call
	// write it to the newly formed segment base.
	totalBytesWritten := s.getBytesWritten()
	if err == nil && s.reset() == nil {
		s.lastNumDocs = len(results)
		s.lastOutSize = len(br.Bytes())
		sb.setBytesWritten(totalBytesWritten)
		interimPool.Put(s)
	}

	return sb, uint64(len(br.Bytes())), err
}

var interimPool = sync.Pool{New: func() interface{} { return &interim{} }}

// interim holds temporary working data used while converting from
// analysis results to a zap-encoded segment
type interim struct {
	// atomic access to this variable, moved to top to correct alignment issues on ARM, 386 and 32-bit MIPS.
	bytesWritten uint64

	results []index.Document

	// edge list for nested documents: child -> parent
	edgeList map[uint64]uint64

	chunkMode uint32

	w *FileWriter

	config map[string]interface{}

	// FieldsMap adds 1 to field id to avoid zero value issues
	//  name -> field id + 1
	FieldsMap map[string]uint16

	// FieldsOptions holds the indexing options for each field
	FieldsOptions map[string]index.FieldIndexingOptions

	// FieldsInv is the inverse of FieldsMap
	//  field id -> name
	FieldsInv []string

	metaBuf bytes.Buffer

	tmp0 []byte
	tmp1 []byte

	lastNumDocs int
	lastOutSize int

	opaque map[int]resetable

	stats *segment.Stats
}

func (s *interim) reset() (err error) {
	s.results = nil
	s.chunkMode = 0
	s.w = nil
	clear(s.edgeList)
	clear(s.FieldsMap)
	clear(s.FieldsOptions)
	s.FieldsInv = s.FieldsInv[:0]
	s.metaBuf.Reset()
	s.tmp0 = s.tmp0[:0]
	s.tmp1 = s.tmp1[:0]
	s.lastNumDocs = 0
	s.lastOutSize = 0

	// reset the bytes written stat count
	// to avoid leaking of bytesWritten across reuse cycles.
	s.setBytesWritten(0)

	if s.opaque != nil {
		for _, v := range s.opaque {
			err = v.Reset()
		}
	} else {
		s.opaque = map[int]resetable{}
	}

	return err
}

type interimStoredField struct {
	vals      [][]byte
	typs      []byte
	arrayposs [][]uint64 // array positions
}

type interimFreqNorm struct {
	freq    uint64
	norm    float32
	numLocs int
}

type interimLoc struct {
	fieldID   uint16
	pos       uint64
	start     uint64
	end       uint64
	arrayposs []uint64
}

func (s *interim) convert() (uint64, uint64, error) {
	if s.FieldsMap == nil {
		s.FieldsMap = map[string]uint16{}
	}
	if s.FieldsOptions == nil {
		s.FieldsOptions = map[string]index.FieldIndexingOptions{}
	}

	s.getOrDefineField("_id") // _id field is fieldID 0
	// special case _id field options: the _id is the canonical document identifier and
	// must always be both indexed and stored so that it can be used for lookups/queries
	// and retrieved back from the stored fields, regardless of user-specified field options.
	s.FieldsOptions["_id"] = index.IndexField | index.StoreField

	var fName string
	for _, result := range s.results {
		result.VisitComposite(func(field index.CompositeField) {
			fName = field.Name()
			s.getOrDefineField(fName)
			s.FieldsOptions[fName] |= field.Options()
		})
		result.VisitFields(func(field index.Field) {
			fName = field.Name()
			s.getOrDefineField(fName)
			s.FieldsOptions[fName] |= field.Options()
		})
	}

	sort.Strings(s.FieldsInv[1:]) // keep _id as first field

	for fieldID, fieldName := range s.FieldsInv {
		s.FieldsMap[fieldName] = uint16(fieldID + 1)
	}

	args := map[string]interface{}{
		"results":       s.results,
		"chunkMode":     s.chunkMode,
		"fieldsMap":     s.FieldsMap,
		"fieldsInv":     s.FieldsInv,
		"fieldsOptions": s.FieldsOptions,
		"stats":         s.stats,
	}
	if s.config != nil {
		args["config"] = s.config
	}
	if s.opaque == nil {
		s.opaque = map[int]resetable{}
		for i, x := range segmentSections {
			s.opaque[int(i)] = x.InitOpaque(args)
		}
	} else {
		for k, v := range args {
			for _, op := range s.opaque {
				op.Set(k, v)
			}
		}
	}

	s.processDocuments()

	storedIndexOffset, err := s.writeStoredFields()
	if err != nil {
		return 0, 0, err
	}

	// we can persist the various sections at this point.
	// the rule of thumb here is that each section must persist field wise.
	for _, x := range segmentSections {
		err = x.Persist(s.opaque, s.w)
		if err != nil {
			return 0, 0, err
		}
	}

	// after persisting the sections to the writer, account corresponding
	for _, opaque := range s.opaque {
		opaqueIO, ok := opaque.(segment.DiskStatsReporter)
		if ok {
			s.incrementBytesWritten(opaqueIO.BytesWritten())
		}
	}

	// we can persist a new fields section here
	// this new fields section will point to the various indexes available
	sectionsIndexOffset, err := persistFieldsSection(s.FieldsInv, s.FieldsOptions, s.w, s.opaque)
	if err != nil {
		return 0, 0, err
	}

	return storedIndexOffset, sectionsIndexOffset, nil
}

func (s *interim) getOrDefineField(fieldName string) int {
	fieldIDPlus1, exists := s.FieldsMap[fieldName]
	if !exists {
		fieldIDPlus1 = uint16(len(s.FieldsInv) + 1)
		s.FieldsMap[fieldName] = fieldIDPlus1
		s.FieldsInv = append(s.FieldsInv, fieldName)
	}

	return int(fieldIDPlus1 - 1)
}

func (s *interim) processDocuments() {
	for docNum, result := range s.results {
		s.processDocument(uint32(docNum), result)
	}
	atomic.AddUint64(&s.stats.TotNewDocsProcessed, uint64(len(s.results)))
}

func (s *interim) processDocument(docNum uint32,
	result index.Document) {
	// this callback is essentially going to be invoked on each field,
	// as part of which preprocessing, cumulation etc. of the doc's data
	// will take place.
	var fieldCount int
	visitField := func(field index.Field) {
		fieldCount++
		fieldID := uint16(s.getOrDefineField(field.Name()))

		// section specific processing of the field
		for _, section := range segmentSections {
			section.Process(s.opaque, docNum, field, fieldID)
		}
	}

	// walk each composite field
	result.VisitComposite(func(field index.CompositeField) {
		visitField(field)
	})

	// walk each field
	result.VisitFields(visitField)

	// given that as part of visiting each field, there may some kind of totalling
	// or accumulation that can be updated, it becomes necessary to commit or
	// put that totalling/accumulation into effect. However, for certain section
	// types this particular step need not be valid, in which case it would be a
	// no-op in the implmentation of the section's process API.
	for _, section := range segmentSections {
		section.Process(s.opaque, docNum, nil, math.MaxUint16)
	}

	if fieldCount > 0 {
		atomic.AddUint64(&s.stats.TotNewDocsIndexed, 1)
	} else {
		atomic.AddUint64(&s.stats.TotNewDocsDropped, 1)
	}
}

func (s *interim) getBytesWritten() uint64 {
	return atomic.LoadUint64(&s.bytesWritten)
}

func (s *interim) incrementBytesWritten(val uint64) {
	atomic.AddUint64(&s.bytesWritten, val)
}

func (s *interim) writeStoredFields() (
	storedIndexOffset uint64, err error) {
	varBuf := make([]byte, binary.MaxVarintLen64)
	metaEncode := func(val uint64) (int, error) {
		wb := binary.PutUvarint(varBuf, val)
		return s.metaBuf.Write(varBuf[:wb])
	}

	data, compressed := s.tmp0[:0], s.tmp1[:0]
	defer func() { s.tmp0, s.tmp1 = data, compressed }()

	// keyed by docNum
	docStoredOffsets := make([]uint64, len(s.results))

	// keyed by fieldID, for the current doc in the loop
	docStoredFields := map[uint16]interimStoredField{}

	for docNum, result := range s.results {
		for fieldID := range docStoredFields { // reset for next doc
			delete(docStoredFields, fieldID)
		}

		var validationErr error
		result.VisitFields(func(field index.Field) {
			fieldID := uint16(s.getOrDefineField(field.Name()))

			if field.Options().IsStored() {
				isf := docStoredFields[fieldID]
				isf.vals = append(isf.vals, field.Value())
				isf.typs = append(isf.typs, field.EncodedFieldType())
				isf.arrayposs = append(isf.arrayposs, field.ArrayPositions())
				docStoredFields[fieldID] = isf
			}

			err := ValidateDocFields(field)
			if err != nil && validationErr == nil {
				validationErr = err
			}
		})
		if validationErr != nil {
			return 0, validationErr
		}

		var curr int

		s.metaBuf.Reset()
		data = data[:0]

		// _id field special case optimizes ExternalID() lookups
		idFieldVal := docStoredFields[uint16(0)].vals[0]
		_, err = metaEncode(uint64(len(idFieldVal)))
		if err != nil {
			return 0, err
		}

		// handle non-"_id" fields
		for fieldID := 1; fieldID < len(s.FieldsInv); fieldID++ {
			isf, exists := docStoredFields[uint16(fieldID)]
			if exists {
				curr, data, err = persistStoredFieldValues(
					fieldID, isf.vals, isf.typs, isf.arrayposs,
					curr, metaEncode, data)
				if err != nil {
					return 0, err
				}
			}
		}

		metaBytes := s.metaBuf.Bytes()

		compressed = snappy.Encode(compressed[:cap(compressed)], data)
		s.incrementBytesWritten(uint64(len(compressed)))
		docStoredOffsets[docNum] = uint64(s.w.Count())

		combined := make([]byte, len(idFieldVal)+len(compressed))
		copy(combined, idFieldVal)
		copy(combined[len(idFieldVal):], compressed)
		bufMeta := s.w.process(metaBytes)
		bufCompressed := s.w.process(combined)

		_, err = writeUvarints(s.w,
			uint64(len(bufMeta)),
			uint64(len(bufCompressed)))
		if err != nil {
			return 0, err
		}

		_, err = s.w.Write(bufMeta)
		if err != nil {
			return 0, err
		}

		_, err = s.w.Write(bufCompressed)
		if err != nil {
			return 0, err
		}
	}

	storedIndexOffset = uint64(s.w.Count())

	for _, docStoredOffset := range docStoredOffsets {
		err = binary.Write(s.w, binary.BigEndian, docStoredOffset)
		if err != nil {
			return 0, err
		}
	}

	// write the number of edges in the child -> parent edge list
	// this will be zero if there are no nested documents
	// and this number also reflects the number of nested documents
	// in the segment
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(len(s.edgeList)))
	_, err = s.w.Write(buf[:n])
	if err != nil {
		return 0, err
	}
	// write the child -> parent edge list
	// child and parent are both flattened doc ids
	for child, parent := range s.edgeList {
		n = binary.PutUvarint(buf, child)
		_, err = s.w.Write(buf[:n])
		if err != nil {
			return 0, err
		}
		n = binary.PutUvarint(buf, parent)
		_, err = s.w.Write(buf[:n])
		if err != nil {
			return 0, err
		}
	}

	return storedIndexOffset, nil
}

func (s *interim) setBytesWritten(val uint64) {
	atomic.StoreUint64(&s.bytesWritten, val)
}

// returns the total # of bytes needed to encode the given uint64's
// into binary.PutUVarint() encoding
func totalUvarintBytes(a, b, c, d, e uint64, more []uint64) (n int) {
	n = numUvarintBytes(a)
	n += numUvarintBytes(b)
	n += numUvarintBytes(c)
	n += numUvarintBytes(d)
	n += numUvarintBytes(e)
	for _, v := range more {
		n += numUvarintBytes(v)
	}
	return n
}

// returns # of bytes needed to encode x in binary.PutUvarint() encoding
func numUvarintBytes(x uint64) (n int) {
	for x >= 0x80 {
		x >>= 7
		n++
	}
	return n + 1
}

// flattenNestedDocuments returns a preorder list of the given documents and
// all their nested documents, along with a map mapping each flattened index
// to its parent index (excluding root docs entirely).
// The edge list is represented as a map[child]parent, where both child and
// parent are flattened document indices.
// Root documents (those without a parent) are not included in the edge list,
// as they have no parent. The order of documents in the returned slice is
// such that parents always appear before their children. A reusable edgeList
// can be provided to avoid allocations across multiple calls.
func flattenNestedDocuments(docs []index.Document, edgeList map[uint64]uint64) (
	[]index.Document, map[uint64]uint64) {
	totalCount := 0
	for _, doc := range docs {
		totalCount += countNestedDocuments(doc)
	}

	if totalCount == len(docs) {
		// no nested documents, return early
		return docs, nil
	}

	flattened := make([]index.Document, 0, totalCount)
	if edgeList == nil {
		edgeList = make(map[uint64]uint64, totalCount-len(docs))
	}

	var traverse func(doc index.Document, hasParent bool, parentIdx uint64)
	traverse = func(d index.Document, hasParent bool, parentIdx uint64) {
		curIdx := uint64(len(flattened))
		flattened = append(flattened, d)

		if hasParent {
			edgeList[curIdx] = parentIdx
		}

		if nestedDoc, ok := d.(index.NestedDocument); ok {
			nestedDoc.VisitNestedDocuments(func(child index.Document) {
				traverse(child, true, curIdx)
			})
		}
	}
	// Top-level docs have no parent
	for _, doc := range docs {
		traverse(doc, false, 0)
	}
	return flattened, edgeList
}

// countNestedDocuments returns the total number of docs in preorder,
// including the parent and all descendants.
func countNestedDocuments(doc index.Document) int {
	count := 1 // include this doc
	if nd, ok := doc.(index.NestedDocument); ok {
		nd.VisitNestedDocuments(func(child index.Document) {
			count += countNestedDocuments(child)
		})
	}
	return count
}
