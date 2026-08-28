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

package zap

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/RoaringBitmap/roaring/v2"
	index "github.com/blevesearch/bleve_index_api"
	mmap "github.com/blevesearch/mmap-go"
	segment "github.com/blevesearch/scorch_segment_api/v2"
	"github.com/golang/snappy"
)

var reflectStaticSizeSegmentBase int

func init() {
	var sb SegmentBase
	reflectStaticSizeSegmentBase = int(unsafe.Sizeof(sb))
}

// OpenUsing returns a zap impl of a segment which tracks some config values during
// the its lifetime.
func (z *ZapPlugin) OpenUsing(path string, config map[string]interface{}) (segment.Segment, error) {
	return z.open(path, config)
}

// Open returns a zap impl of a segment
func (z *ZapPlugin) Open(path string) (segment.Segment, error) {
	return z.open(path, nil)
}

func (z *ZapPlugin) open(path string, config map[string]interface{}) (segment.Segment, error) {
	zapStats, ok := config[segment.StatsKey].(*segment.Stats)
	if !ok || zapStats == nil {
		zapStats = new(segment.Stats)
	}

	atomic.AddUint64(&zapStats.TotOpenBeg, 1)
	f, err := os.Open(path)
	if err != nil {
		atomic.AddUint64(&zapStats.TotOpenErrors, 1)
		return nil, err
	}
	mm, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		// mmap failed, try to close the file
		_ = f.Close()
		atomic.AddUint64(&zapStats.TotOpenErrors, 1)
		return nil, err
	}

	rv := &Segment{
		SegmentBase: SegmentBase{
			fieldsMap:         make(map[string]uint16),
			fieldsOptions:     make(map[string]index.FieldIndexingOptions),
			invIndexCache:     newInvertedIndexCache(),
			vecIndexCache:     newVectorIndexCache(),
			synIndexCache:     newSynonymIndexCache(),
			geoIndexCache:     newGeoIndexCache(),
			nstIndexCache:     newNestedIndexCache(),
			trainedIndexCache: newTrainedIndexCache(),
			fieldDvReaders:    make([][]*docValueReader, len(segmentSections)),
			config:            config,
			stats:             zapStats,
		},
		f:    f,
		mm:   mm,
		path: path,
		refs: 1,
	}
	rv.SegmentBase.updateSize()

	err = rv.loadConfig()
	if err != nil {
		_ = rv.Close()
		atomic.AddUint64(&zapStats.TotOpenErrors, 1)
		return nil, err
	}

	err = rv.loadFields()
	if err != nil {
		_ = rv.Close()
		atomic.AddUint64(&zapStats.TotOpenErrors, 1)
		return nil, err
	}

	err = rv.loadDvReaders()
	if err != nil {
		_ = rv.Close()
		atomic.AddUint64(&zapStats.TotOpenErrors, 1)
		return nil, err
	}

	// initialize any of the caches if needed
	err = rv.nstIndexCache.initialize(rv.numDocs, rv.getEdgeListOffset(), rv.mem)
	if err != nil {
		_ = rv.Close()
		atomic.AddUint64(&zapStats.TotOpenErrors, 1)
		return nil, err
	}

	atomic.AddUint64(&zapStats.TotOpenEnd, 1)
	return rv, nil
}

// SegmentBase is a memory only, read-only implementation of the
// segment.Segment interface, using zap's data representation.
type SegmentBase struct {
	// atomic access to these variables, moved to top to correct alignment issues on ARM, 386 and 32-bit MIPS.
	bytesRead    uint64
	bytesWritten uint64

	mem                 []byte
	memCRC              uint32
	chunkMode           uint32
	fieldsMap           map[string]uint16                     // fieldName -> fieldID+1
	fieldsOptions       map[string]index.FieldIndexingOptions // fieldName -> fieldOptions
	fieldsInv           []string                              // fieldID -> fieldName
	fieldsSectionsMap   [][]uint64                            // fieldID -> section -> address
	numDocs             uint64
	storedIndexOffset   uint64
	sectionsIndexOffset uint64
	fieldDvReaders      [][]*docValueReader // naive chunk cache per field; section->fieldID->reader
	fieldDvNames        []string            // field names cached in fieldDvReaders
	size                uint64

	// file reader initialised with the writer callback id used by the segment
	fileReader *FileReader

	// index update specific tracking
	updatedFields map[string]*index.UpdateFieldInfo
	config        map[string]interface{} // config for the segment

	// section-specific caches
	invIndexCache     *invertedIndexCache
	vecIndexCache     *vectorIndexCache
	trainedIndexCache *trainedIndexCache
	synIndexCache     *synonymIndexCache
	geoIndexCache     *geoIndexCache
	nstIndexCache     *nestedIndexCache

	// segment level stats that are tracked and reported as part of the segment's lifecycle
	stats *segment.Stats
}

func (sb *SegmentBase) Size() int {
	return int(sb.size)
}

func (sb *SegmentBase) updateSize() {
	sizeInBytes := reflectStaticSizeSegmentBase +
		cap(sb.mem)

	// fieldsMap
	for k := range sb.fieldsMap {
		sizeInBytes += (len(k) + SizeOfString) + SizeOfUint16
	}

	// fieldsOptions
	for k := range sb.fieldsOptions {
		sizeInBytes += (len(k) + SizeOfString) + SizeOfUint64
	}

	// fieldsInv
	for _, entry := range sb.fieldsInv {
		sizeInBytes += len(entry) + SizeOfString
	}

	// fieldDvReaders
	for _, secDvReaders := range sb.fieldDvReaders {
		for _, v := range secDvReaders {
			sizeInBytes += SizeOfUint16 + SizeOfPtr
			if v != nil {
				sizeInBytes += v.size()
			}
		}
	}

	sb.size = uint64(sizeInBytes)
}

func (sb *SegmentBase) AddRef()             {}
func (sb *SegmentBase) DecRef() (err error) { return nil }
func (sb *SegmentBase) Close() (err error) {
	sb.invIndexCache.Clear()
	sb.vecIndexCache.Clear()
	sb.trainedIndexCache.Clear()
	sb.synIndexCache.Clear()
	sb.nstIndexCache.Clear()
	sb.geoIndexCache.Clear()
	return nil
}

// Segment implements a persisted segment.Segment interface, by
// embedding an mmap()'ed SegmentBase.
type Segment struct {
	SegmentBase

	f       *os.File
	mm      mmap.MMap
	path    string
	version uint32
	crc     uint32

	m    sync.Mutex // Protects the fields that follow.
	refs int64
}

func (s *Segment) Size() int {
	// 8 /* size of file pointer */
	// 4 /* size of version -> uint32 */
	// 4 /* size of crc -> uint32 */
	sizeOfUints := 16

	sizeInBytes := (len(s.path) + SizeOfString) + sizeOfUints

	// mutex, refs -> int64
	sizeInBytes += 16

	// do not include the mmap'ed part
	return sizeInBytes + s.SegmentBase.Size() - cap(s.mem)
}

func (s *Segment) AddRef() {
	s.m.Lock()
	s.refs++
	s.m.Unlock()
}

func (s *Segment) DecRef() (err error) {
	s.m.Lock()
	s.refs--
	if s.refs == 0 {
		err = s.closeActual()
	}
	s.m.Unlock()
	return err
}

func (s *Segment) loadConfig() error {
	// read offsets of 32 bit values - crc, ver, chunk
	crcOffset := len(s.mm) - 4
	verOffset := crcOffset - 4
	chunkOffset := verOffset - 4

	// read offsets of 64 bit values - sectionsIndexOffset, storedIndexOffset, numDocsOffset
	sectionsIndexOffset := chunkOffset - 8
	storedIndexOffset := sectionsIndexOffset - 8
	numDocsOffset := storedIndexOffset - 8

	// read offsets for the writer id length
	idLenOffset := numDocsOffset - 4

	// read 32-bit crc
	s.crc = binary.BigEndian.Uint32(s.mm[crcOffset : crcOffset+4])

	// read 32-bit version
	s.version = binary.BigEndian.Uint32(s.mm[verOffset : verOffset+4])
	if s.version != Version {
		return fmt.Errorf("unsupported version %d != %d", s.version, Version)
	}

	// read 32-bit chunk mode
	s.chunkMode = binary.BigEndian.Uint32(s.mm[chunkOffset : chunkOffset+4])

	// read 64-bit sections index offset
	s.sectionsIndexOffset = binary.BigEndian.Uint64(s.mm[sectionsIndexOffset : sectionsIndexOffset+8])

	// read 64-bit stored index offset
	s.storedIndexOffset = binary.BigEndian.Uint64(s.mm[storedIndexOffset : storedIndexOffset+8])

	// read 64-bit num docs
	s.numDocs = binary.BigEndian.Uint64(s.mm[numDocsOffset : numDocsOffset+8])

	// read the length of the id
	idLen := binary.BigEndian.Uint32(s.mm[idLenOffset : idLenOffset+4])
	idOffset := idLenOffset - int(idLen)

	// read the file writer callback id and initialize the file reader with the same id
	fileWriterID := string(s.mm[idOffset : idOffset+int(idLen)])
	var err error
	s.fileReader, err = NewFileReader(fileWriterID, []byte(s.path))
	if err != nil {
		return err
	}

	footerSize := FooterSize + int(idLen)
	s.incrementBytesRead(uint64(footerSize))
	s.SegmentBase.mem = s.mm[:len(s.mm)-footerSize]
	return nil
}

// Implements the segment.DiskStatsReporter interface
// Only the persistedSegment type implments the
// interface, as the intention is to retrieve the bytes
// read from the on-disk segment as part of the current
// query.
func (s *Segment) ResetBytesRead(val uint64) {
	atomic.StoreUint64(&s.SegmentBase.bytesRead, val)
}

func (s *Segment) BytesRead() uint64 {
	return atomic.LoadUint64(&s.bytesRead)
}

func (s *Segment) BytesWritten() uint64 {
	return 0
}

func (s *Segment) incrementBytesRead(val uint64) {
	atomic.AddUint64(&s.bytesRead, val)
}

func (sb *SegmentBase) BytesWritten() uint64 {
	return atomic.LoadUint64(&sb.bytesWritten)
}

func (sb *SegmentBase) setBytesWritten(val uint64) {
	atomic.AddUint64(&sb.bytesWritten, val)
}

func (sb *SegmentBase) BytesRead() uint64 {
	return 0
}

func (sb *SegmentBase) ResetBytesRead(val uint64) {}

func (sb *SegmentBase) incrementBytesRead(val uint64) {
	atomic.AddUint64(&sb.bytesRead, val)
}

func (sb *SegmentBase) loadFields() error {
	pos := sb.sectionsIndexOffset

	if pos == 0 {
		return fmt.Errorf("no sections index present")
	}

	seek := pos + binary.MaxVarintLen64
	if seek > uint64(len(sb.mem)) {
		// handling a buffer overflow case.
		// a rare case where the backing buffer is not large enough to be read directly via
		// a pos+binary.MaxVarintLen64 seek. For eg, this can happen when there is only
		// one field to be indexed in the entire batch of data and while writing out
		// these fields metadata, you write 1 + 8 bytes whereas the MaxVarintLen64 = 10.
		seek = uint64(len(sb.mem))
	}

	// read the number of fields
	numFields, sz := binary.Uvarint(sb.mem[pos:seek])
	// here, the pos is incremented by the valid number bytes read from the buffer
	// so in the edge case pointed out above the numFields = 1, the sz = 1 as well.
	pos += uint64(sz)
	sb.incrementBytesRead(uint64(sz))

	// the following loop will be executed only once in the edge case pointed out above
	// since there is only field's offset store which occupies 8 bytes.
	// the pointer then seeks to a position preceding the sectionsIndexOffset, at
	// which point the responsibility of handling the out-of-bounds cases shifts to
	// the specific section's parsing logic.
	var fieldID uint64
	for fieldID < numFields {
		addr := binary.BigEndian.Uint64(sb.mem[pos : pos+8])
		sb.incrementBytesRead(8)

		fieldSectionMap, err := sb.loadField(uint16(fieldID), addr)
		if err != nil {
			return err
		}

		sb.fieldsSectionsMap = append(sb.fieldsSectionsMap, fieldSectionMap)

		fieldID++
		pos += 8
	}

	return nil
}

// loadField loads the field metadata for the given fieldID at the given position
func (sb *SegmentBase) loadField(fieldID uint16, pos uint64) ([]uint64, error) {
	if pos == 0 {
		// there is no indexing structure present for this field/section
		return nil, nil
	}

	fieldStartPos := pos // to track the number of bytes read
	fieldNameLen, sz := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
	pos += uint64(sz)

	fieldName, err := sb.fileReader.process(sb.mem[pos : pos+fieldNameLen])
	if err != nil {
		return nil, err
	}
	pos += fieldNameLen

	// read field options
	fieldOptions, sz := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
	pos += uint64(sz)

	sb.fieldsInv = append(sb.fieldsInv, string(fieldName))
	sb.fieldsMap[string(fieldName)] = fieldID + 1
	sb.fieldsOptions[string(fieldName)] = index.FieldIndexingOptions(fieldOptions)

	fieldNumSections, sz := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
	pos += uint64(sz)
	// create an address mapping array for each of the segment sections
	// if the field has a valid section index, then the address will be non-zero
	// else it will be zero.
	fieldSectionMap := make([]uint64, NumSections)
	for sectionIdx := uint64(0); sectionIdx < fieldNumSections; sectionIdx++ {
		// read section id
		fieldSectionType := binary.BigEndian.Uint16(sb.mem[pos : pos+2])
		pos += 2
		fieldSectionAddr := binary.BigEndian.Uint64(sb.mem[pos : pos+8])
		pos += 8
		fieldSectionMap[fieldSectionType] = fieldSectionAddr
	}

	// account the bytes read while parsing the sections field index.
	sb.incrementBytesRead((pos - uint64(fieldStartPos)) + fieldNameLen)
	return fieldSectionMap, nil
}

// Dictionary returns the term dictionary for the specified field
func (sb *SegmentBase) Dictionary(field string) (segment.TermDictionary, error) {
	dict, err := sb.dictionary(field)
	if err == nil && dict == nil {
		return emptyDictionary, nil
	}
	return dict, err
}

func (sb *SegmentBase) dictionary(field string) (rv *Dictionary, err error) {
	fieldIDPlus1 := sb.fieldsMap[field]
	if fieldIDPlus1 == 0 {
		return nil, nil
	}
	pos := sb.fieldsSectionsMap[fieldIDPlus1-1][SectionInvertedTextIndex]
	if pos > 0 {
		rv = &Dictionary{
			sb:      sb,
			field:   field,
			fieldID: fieldIDPlus1 - 1,
		}
		// skip the doc value offsets to get to the dictionary portion
		for i := 0; i < 2; i++ {
			_, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
			pos += uint64(n)
		}
		dictLoc, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
		pos += uint64(n)
		fst, bytesRead, err := sb.invIndexCache.loadOrCreate(rv.fieldID, sb.mem[dictLoc:], sb.fileReader)
		if err != nil {
			return nil, fmt.Errorf("dictionary for field %s err: %v", field, err)
		}
		rv.fst = fst
		rv.fstReader, err = rv.fst.Reader()
		if err != nil {
			return nil, fmt.Errorf("dictionary for field %s, vellum reader err: %v", field, err)
		}
		rv.bytesRead += bytesRead
	}

	return rv, nil
}

// Thesaurus returns the thesaurus with the specified name, or an empty thesaurus if not found.
func (sb *SegmentBase) Thesaurus(name string) (segment.Thesaurus, error) {
	thesaurus, err := sb.thesaurus(name)
	if err == nil && thesaurus == nil {
		return emptyThesaurus, nil
	}
	return thesaurus, err
}

func (sb *SegmentBase) thesaurus(name string) (rv *Thesaurus, err error) {
	fieldIDPlus1 := sb.fieldsMap[name]
	if fieldIDPlus1 == 0 {
		return nil, nil
	}
	pos := sb.fieldsSectionsMap[fieldIDPlus1-1][SectionSynonymIndex]
	if pos > 0 {
		rv = &Thesaurus{
			sb:      sb,
			name:    name,
			fieldID: fieldIDPlus1 - 1,
		}
		// skip the doc value offsets as doc values are not supported in thesaurus
		for i := 0; i < 2; i++ {
			_, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
			pos += uint64(n)
		}
		thesLoc, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
		pos += uint64(n)
		fst, synTermMap, bytesRead, err := sb.synIndexCache.loadOrCreate(rv.fieldID, sb.mem[thesLoc:], sb.fileReader)
		if err != nil {
			return nil, fmt.Errorf("thesaurus name %s err: %v", name, err)
		}
		rv.fst = fst
		rv.synIDTermMap = synTermMap
		rv.fstReader, err = rv.fst.Reader()
		if err != nil {
			return nil, fmt.Errorf("thesaurus name %s vellum reader err: %v", name, err)
		}
		rv.bytesRead += bytesRead
	}
	return rv, nil
}

// visitDocumentCtx holds data structures that are reusable across
// multiple VisitDocument() calls to avoid memory allocations
type visitDocumentCtx struct {
	buf      []byte
	reader   bytes.Reader
	arrayPos []uint64
}

var visitDocumentCtxPool = sync.Pool{
	New: func() interface{} {
		reuse := &visitDocumentCtx{}
		return reuse
	},
}

// VisitStoredFields invokes the StoredFieldValueVisitor for each stored field
// for the specified doc number
func (sb *SegmentBase) VisitStoredFields(num uint64, visitor segment.StoredFieldValueVisitor) error {
	vdc := visitDocumentCtxPool.Get().(*visitDocumentCtx)
	defer visitDocumentCtxPool.Put(vdc)
	return sb.visitStoredFields(vdc, num, visitor)
}

func (sb *SegmentBase) visitStoredFields(vdc *visitDocumentCtx, num uint64,
	visitor segment.StoredFieldValueVisitor) error {
	// first make sure this is a valid number in this segment
	if num < sb.numDocs {
		meta, compressed, err := sb.getDocStoredMetaAndCompressed(num)
		if err != nil {
			return err
		}

		vdc.reader.Reset(meta)

		// handle _id field special case
		idFieldValLen, err := binary.ReadUvarint(&vdc.reader)
		if err != nil {
			return err
		}
		idFieldVal := compressed[:idFieldValLen]

		keepGoing := visitor("_id", byte('t'), idFieldVal, nil)
		if !keepGoing {
			visitDocumentCtxPool.Put(vdc)
			return nil
		}

		// handle non-"_id" fields
		compressed = compressed[idFieldValLen:]

		uncompressed, err := snappy.Decode(vdc.buf[:cap(vdc.buf)], compressed)
		if err != nil {
			return err
		}

		for keepGoing {
			field, err := binary.ReadUvarint(&vdc.reader)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			typ, err := binary.ReadUvarint(&vdc.reader)
			if err != nil {
				return err
			}
			offset, err := binary.ReadUvarint(&vdc.reader)
			if err != nil {
				return err
			}
			l, err := binary.ReadUvarint(&vdc.reader)
			if err != nil {
				return err
			}
			numap, err := binary.ReadUvarint(&vdc.reader)
			if err != nil {
				return err
			}
			var arrayPos []uint64
			if numap > 0 {
				if cap(vdc.arrayPos) < int(numap) {
					vdc.arrayPos = make([]uint64, numap)
				}
				arrayPos = vdc.arrayPos[:numap]
				for i := 0; i < int(numap); i++ {
					ap, err := binary.ReadUvarint(&vdc.reader)
					if err != nil {
						return err
					}
					arrayPos[i] = ap
				}
			}
			value := uncompressed[offset : offset+l]
			keepGoing = visitor(sb.fieldsInv[field], byte(typ), value, arrayPos)
		}

		vdc.buf = uncompressed
	}
	return nil
}

// DocID returns the value of the _id field for the given docNum
func (sb *SegmentBase) DocID(num uint64) ([]byte, error) {
	if num >= sb.numDocs {
		return nil, nil
	}

	vdc := visitDocumentCtxPool.Get().(*visitDocumentCtx)

	meta, compressed, err := sb.getDocStoredMetaAndCompressed(num)
	if err != nil {
		return nil, err
	}

	vdc.reader.Reset(meta)

	// handle _id field special case
	idFieldValLen, err := binary.ReadUvarint(&vdc.reader)
	if err != nil {
		return nil, err
	}
	idFieldVal := compressed[:idFieldValLen]

	visitDocumentCtxPool.Put(vdc)

	return idFieldVal, nil
}

// Count returns the number of documents in this segment.
func (sb *SegmentBase) Count() uint64 {
	return sb.numDocs
}

// DocNumbers returns a bitset corresponding to the doc numbers of all the
// provided _id strings
func (sb *SegmentBase) DocNumbers(ids []string) (*roaring.Bitmap, error) {
	rv := roaring.New()

	if len(sb.fieldsMap) > 0 {
		idDict, err := sb.dictionary("_id")
		if err != nil {
			return nil, err
		}

		postingsList := emptyPostingsList

		sMax, err := idDict.fst.GetMaxKey()
		if err != nil {
			return nil, err
		}
		sMaxStr := string(sMax)
		for _, id := range ids {
			if id <= sMaxStr {
				postingsList, err = idDict.postingsList([]byte(id), nil, postingsList)
				if err != nil {
					return nil, err
				}
				postingsList.OrInto(rv)
			}
		}
	}

	return rv, nil
}

// Fields returns the field names used in this segment
func (sb *SegmentBase) Fields() []string {
	return sb.fieldsInv
}

// Path returns the path of this segment on disk
func (s *Segment) Path() string {
	return s.path
}

// Close releases all resources associated with this segment
func (s *Segment) Close() (err error) {
	return s.DecRef()
}

func (s *Segment) closeActual() (err error) {
	// clear contents from all caches before un-mmapping
	s.invIndexCache.Clear()
	s.vecIndexCache.Clear()
	s.synIndexCache.Clear()
	s.nstIndexCache.Clear()
	s.trainedIndexCache.Clear()

	if s.mm != nil {
		err = s.mm.Unmap()
	}
	// try to close file even if unmap failed
	if s.f != nil {
		err2 := s.f.Close()
		if err == nil {
			// try to return first error
			err = err2
		}
	}

	atomic.AddUint64(&s.stats.TotSegmentsClosed, 1)
	return
}

// some helpers i started adding for the command-line utility

// Data returns the underlying mmaped data slice
func (s *Segment) Data() []byte {
	return s.mm
}

// CRC returns the CRC value stored in the file footer
func (s *Segment) CRC() uint32 {
	return s.crc
}

// Version returns the file version in the file footer
func (s *Segment) Version() uint32 {
	return s.version
}

// ChunkFactor returns the chunk factor in the file footer
func (s *Segment) ChunkMode() uint32 {
	return s.chunkMode
}

// SectionsIndexOffset returns the sections index offset in the file footer
func (s *Segment) SectionsIndexOffset() uint64 {
	return s.sectionsIndexOffset
}

// StoredIndexOffset returns the stored value index offset in the file footer
func (s *Segment) StoredIndexOffset() uint64 {
	return s.storedIndexOffset
}

// NumDocs returns the number of documents in the file footer
func (s *Segment) NumDocs() uint64 {
	return s.numDocs
}

// DictAddr is a helper function to compute the file offset where the
// dictionary is stored for the specified field.
func (s *Segment) DictAddr(field string) (uint64, error) {
	fieldIDPlus1, ok := s.fieldsMap[field]
	if !ok {
		return 0, fmt.Errorf("no such field '%s'", field)
	}
	dictStart := s.fieldsSectionsMap[fieldIDPlus1-1][SectionInvertedTextIndex]
	if dictStart == 0 {
		return 0, fmt.Errorf("no dictionary for field '%s'", field)
	}
	for i := 0; i < 2; i++ {
		_, n := binary.Uvarint(s.mem[dictStart : dictStart+binary.MaxVarintLen64])
		dictStart += uint64(n)
	}
	dictLoc, _ := binary.Uvarint(s.mem[dictStart : dictStart+binary.MaxVarintLen64])
	return dictLoc, nil
}

// VectorAddr is a helper function to compute the file offset where the
// vector index is stored for the specified field.
func (s *Segment) VectorAddr(name string) (uint64, error) {
	fieldIDPlus1, ok := s.fieldsMap[name]
	if !ok {
		return 0, fmt.Errorf("no such field '%s'", name)
	}
	vectorStart := s.fieldsSectionsMap[fieldIDPlus1-1][SectionFaissVectorIndex]
	if vectorStart == 0 {
		return 0, fmt.Errorf("no vector index for field '%s'", name)
	}
	for i := 0; i < 2; i++ {
		_, n := binary.Uvarint(s.mem[vectorStart : vectorStart+binary.MaxVarintLen64])
		vectorStart += uint64(n)
	}
	vectorLoc, _ := binary.Uvarint(s.mem[vectorStart : vectorStart+binary.MaxVarintLen64])
	return vectorLoc, nil
}

// ThesaurusAddr is a helper function to compute the file offset where the
// thesaurus is stored with the specified name.
func (s *Segment) ThesaurusAddr(name string) (uint64, error) {
	fieldIDPlus1, ok := s.fieldsMap[name]
	if !ok {
		return 0, fmt.Errorf("no such thesaurus '%s'", name)
	}
	thesaurusStart := s.fieldsSectionsMap[fieldIDPlus1-1][SectionSynonymIndex]
	if thesaurusStart == 0 {
		return 0, fmt.Errorf("no such thesaurus '%s'", name)
	}
	for i := 0; i < 2; i++ {
		_, n := binary.Uvarint(s.mem[thesaurusStart : thesaurusStart+binary.MaxVarintLen64])
		thesaurusStart += uint64(n)
	}
	thesLoc, _ := binary.Uvarint(s.mem[thesaurusStart : thesaurusStart+binary.MaxVarintLen64])
	return thesLoc, nil
}

// EdgeListAddr is the exported helper function to compute the
// file offset where the edge list is stored.
func (s *Segment) EdgeListAddr() (uint64, error) {
	return s.getEdgeListOffset(), nil
}

func (sb *SegmentBase) loadDvReaders() error {
	if sb.numDocs == 0 {
		return nil
	}
	for fieldID, sections := range sb.fieldsSectionsMap {
		for secID, secOffset := range sections {
			if secOffset > 0 {
				pos := secOffset
				var read uint64
				fieldLocStart, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
				if n <= 0 {
					return fmt.Errorf("loadDvReaders: failed to read the docvalue offset start for field %v", sb.fieldsInv[fieldID])
				}
				pos += uint64(n)
				read += uint64(n)
				fieldLocEnd, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
				if n <= 0 {
					return fmt.Errorf("loadDvReaders: failed to read the docvalue offset end for field %v", sb.fieldsInv[fieldID])
				}
				pos += uint64(n)
				read += uint64(n)

				sb.incrementBytesRead(read)

				fieldDvReader, err := sb.loadFieldDocValueReader(sb.fieldsInv[fieldID], fieldLocStart, fieldLocEnd)
				if err != nil {
					return err
				}
				if fieldDvReader != nil {
					if sb.fieldDvReaders[secID] == nil {
						sb.fieldDvReaders[secID] = make([]*docValueReader, len(sb.fieldsInv))
					}
					sb.fieldDvReaders[secID][uint16(fieldID)] = fieldDvReader
					sb.fieldDvNames = append(sb.fieldDvNames, sb.fieldsInv[fieldID])
				}
			}
		}
	}

	return nil
}

// Getter method to retrieve updateFieldInfo within segment base
func (s *SegmentBase) GetUpdatedFields() map[string]*index.UpdateFieldInfo {
	return s.updatedFields
}

// Setter method to store updateFieldInfo within segment base
func (s *SegmentBase) SetUpdatedFields(updatedFields map[string]*index.UpdateFieldInfo) {
	s.updatedFields = updatedFields
}

// Ancestors returns a slice of document numbers representing the ancestors of the
// specified document (docNum) within the segment. If the document has no ancestors,
// a slice containing only the document number itself is returned. The prealloc
// parameter allows for reusing a preallocated slice to avoid additional allocations.
func (sb *SegmentBase) Ancestors(docNum uint64, prealloc []index.AncestorID) []index.AncestorID {
	return sb.nstIndexCache.ancestry(docNum, prealloc)
}

// CountRoot returns the number of root documents in the segment, excluding any
// documents that are marked as deleted in the provided bitmap. The deleted bitmap
// may contain both root and sub-document numbers, and the method ensures that
// only root documents are counted.
func (sb *SegmentBase) CountRoot(deleted *roaring.Bitmap) uint64 {
	// the formula is as follows:
	// Total Docs (T) = Root Docs (R) + Sub Docs (S)
	// R = T - S
	// Now if we have D deleted docs, some of which may be sub-docs, we need to exclude
	// those from the root doc count. Let D = dR + dS, where dR is the number of deleted
	// root docs and dS is the number of deleted sub docs.
	// dR = D - dS
	// Therefore, the count of root docs excluding deleted ones is:
	// R - dR = (T - S) - (D - dS)
	return sb.countRoot() - sb.countRootDeleted(deleted)
}

// AddNestedDocuments returns a bitmap containing the original root document numbers in drops,
// plus any descendant document numbers for each dropped root document. The drops
// parameter represents a set of root document numbers to be dropped, and the returned
// bitmap includes both the original drops and all their descendants (if any).
// NOTE: This method MODIFIES the drops bitmap in place.
// NOTE: This method EXPECTS that the drops bitmap contains ONLY root document numbers.
func (sb *SegmentBase) AddNestedDocuments(drops *roaring.Bitmap) *roaring.Bitmap {
	// If no drops or no nested documents, nothing to do
	if drops == nil || drops.GetCardinality() == 0 || sb.countNested() == 0 {
		return drops
	}
	ds := sb.descendantStore()
	if ds == nil {
		return drops
	}
	descendantBM := roaring.New()
	dropsItr := drops.Iterator()
	for dropsItr.HasNext() {
		rootDoc := uint64(dropsItr.Next())
		if bm, ok := ds.descendants(rootDoc); ok {
			descendantBM.Or(bm)
		}
	}
	drops.Or(descendantBM)
	return drops
}

// edgeList returns an edgeList interface representing the parent-child relationships between documents in the segment.
// The edgeList interface allows iteration over child-parent document pairs, enabling navigation of document hierarchies.
// The underlying implementation may use a map or a slice, but callers should rely on the interface methods.
func (sb *SegmentBase) edgeList() edgeList {
	return sb.nstIndexCache.edgeList()
}

// descendantStore returns a descendantStore interface that provides access to the descendants of documents in the segment.
// The descendantStore interface allows retrieval of descendant document numbers for a given root document number.
// The underlying implementation may use a map or a slice, but callers should rely on the interface methods.
func (sb *SegmentBase) descendantStore() descendantStore {
	return sb.nstIndexCache.descendantStore()
}

// Utility method to count the number of nested documents in the segment, not exported.
func (sb *SegmentBase) countNested() uint64 {
	return sb.nstIndexCache.countNested()
}

// Utility method to count the number of root documents in the given bitmap, not exported.
func (sb *SegmentBase) countRoot() uint64 {
	return sb.Count() - sb.countNested()
}

// Utility method to count the number of root documents that are marked as deleted in the given bitmap, not exported.
func (sb *SegmentBase) countRootDeleted(deleted *roaring.Bitmap) uint64 {
	return sb.nstIndexCache.countRootDeleted(deleted)
}

func (sb *SegmentBase) CallbackId() string {
	return sb.fileReader.id
}

func (sb *SegmentBase) GeoShapeV2Data(field string, except *roaring.Bitmap) (segment.GeoShapeV2Data, error) {
	fieldIDPlus1 := sb.fieldsMap[field]
	if fieldIDPlus1 == 0 {
		return nil, nil
	}
	pos := sb.fieldsSectionsMap[fieldIDPlus1-1][SectionGeoShapeV2Index]
	if pos > 0 {
		// skip the doc value offsets to get to the geo cell data portion
		for i := 0; i < 2; i++ {
			_, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
			pos += uint64(n)
		}
		return sb.geoIndexCache.loadOrCreate(fieldIDPlus1-1, sb.mem[pos:], except, sb.fileReader)
	}
	return nil, nil
}
