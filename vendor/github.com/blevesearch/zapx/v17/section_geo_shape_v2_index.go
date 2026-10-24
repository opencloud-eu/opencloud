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

package zap

import (
	"container/heap"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync/atomic"

	"github.com/RoaringBitmap/roaring/v2"
	index "github.com/blevesearch/bleve_index_api"
	seg "github.com/blevesearch/scorch_segment_api/v2"
)

func init() {
	registerSegmentSection(SectionGeoShapeV2Index, &geoShapeV2IndexSection{})
	invertedTextIndexSectionExclusionChecks = append(
		invertedTextIndexSectionExclusionChecks,
		func(f index.Field) bool {
			_, ok := f.(index.GeoShapeV2Field)
			return ok
		})
}

type geoShapeV2IndexSection struct {
}

func (g *geoShapeV2IndexSection) Process(opaque map[int]resetable, docNum uint32,
	f index.Field, fieldID uint16) {
	if fieldID == math.MaxUint16 {
		return
	}
	if gsf, ok := f.(index.GeoShapeV2Field); ok {
		gs := g.getGeoShapeV2IndexOpaque(opaque)
		gs.process(gsf, fieldID, docNum)
	}
}

func (g *geoShapeV2IndexSection) Persist(opaque map[int]resetable, w *FileWriter) error {
	gs := g.getGeoShapeV2IndexOpaque(opaque)
	return gs.persist(w)
}

func (g *geoShapeV2IndexSection) AddrForField(opaque map[int]resetable, fieldID int) int {
	gs := g.getGeoShapeV2IndexOpaque(opaque)
	return gs.fieldAddrs[uint16(fieldID)]
}

type geoIndexInfo struct {
	content *geoIndexContent

	// newDocNums maps a document's old segment doc number (the index) to its
	// doc number in the merged segment, with docDropped marking deleted docs.
	newDocNums []uint64
}

func (g *geoShapeV2IndexSection) Merge(opaque map[int]resetable, segments []*SegmentBase,
	drops []*roaring.Bitmap, fieldsInv []string, newDocNumsIn [][]uint64, w *FileWriter,
	closeCh chan struct{}) error {
	gs := g.getGeoShapeV2IndexOpaque(opaque)
	indexInfos := make([]*geoIndexInfo, 0, len(segments))

	for fieldID, fieldName := range fieldsInv {
		// Skip fields that are not indexed
		if !gs.fieldsOptions[fieldName].IsIndexed() {
			continue
		}

		indexInfos = indexInfos[:0]

		for segI, sb := range segments {
			if isClosed(closeCh) {
				return seg.ErrClosed
			}
			// Skip if the field is not present in the segment
			if _, ok := sb.fieldsMap[fieldName]; !ok {
				continue
			}

			// Obtain the field start position
			pos := int(sb.fieldsSectionsMap[sb.fieldsMap[fieldName]-1][SectionGeoShapeV2Index])
			if pos == 0 {
				continue
			}

			// skip doc values, as we don't support them for geo shapes
			_, n := binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
			pos += n
			_, n = binary.Uvarint(sb.mem[pos : pos+binary.MaxVarintLen64])
			pos += n

			// Load the geo index content for the field from the segment
			content, err := loadGeoIndexContent(sb.fileReader, sb.mem[pos:])
			if err != nil {
				return err
			}

			// Append the content and the corresponding doc number remapping for this segment
			indexInfos = append(indexInfos, &geoIndexInfo{
				content:    content,
				newDocNums: newDocNumsIn[segI],
			})
		}

		// Merge the index contents from all segments for this field
		indexContent, err := gs.mergeIndexContents(indexInfos)
		if err != nil {
			return err
		}

		// If the merged index content is nil, it means there are
		// no valid documents for this field across all segments,
		// so we skip writing it.
		if indexContent == nil {
			continue
		}

		// record the starting position of this field's index content
		fieldStart := w.Count()
		tempBuf := gs.grabBuf(binary.MaxVarintLen64)
		// Write two varints for indicating no doc values
		n := binary.PutUvarint(tempBuf, fieldNotUninverted)
		_, err = w.Write(tempBuf[:n])
		if err != nil {
			return err
		}
		n = binary.PutUvarint(tempBuf, fieldNotUninverted)
		_, err = w.Write(tempBuf[:n])
		if err != nil {
			return err
		}

		// Write the merged index content for this field to the file
		err = gs.writeIndexContent(indexContent, w)
		if err != nil {
			return err
		}

		gs.incrementBytesWritten(uint64(w.Count() - fieldStart))
		gs.fieldAddrs[uint16(fieldID)] = fieldStart
	}

	return nil
}

func (g *geoShapeV2IndexSection) InitOpaque(args map[string]interface{}) resetable {
	rv := &geoShapeV2IndexSectionOpaque{
		fieldAddrs:   make(map[uint16]int),
		indexContent: make(map[uint16]*geoIndexContent),
	}
	for k, v := range args {
		rv.Set(k, v)
	}
	return rv
}

func (g *geoShapeV2IndexSection) getGeoShapeV2IndexOpaque(
	opaque map[int]resetable) *geoShapeV2IndexSectionOpaque {
	if _, ok := opaque[SectionGeoShapeV2Index]; !ok {
		opaque[SectionGeoShapeV2Index] = g.InitOpaque(nil)
	}
	return opaque[SectionGeoShapeV2Index].(*geoShapeV2IndexSectionOpaque)
}

type geoShapeV2IndexSectionOpaque struct {
	// indexContent holds the geo index content for each field ID
	indexContent map[uint16]*geoIndexContent
	// fieldAddrs holds the starting address of each field's index content in the file
	fieldAddrs map[uint16]int
	// fieldsOptions holds the indexing options for each field name
	fieldsOptions map[string]index.FieldIndexingOptions

	bytesWritten uint64

	// temporary buffer for reuse
	tmp []byte

	init bool
}

func (g *geoShapeV2IndexSectionOpaque) Reset() error {
	g.tmp = g.tmp[:0]
	g.init = false
	clear(g.indexContent)
	clear(g.fieldAddrs)
	clear(g.fieldsOptions)
	return nil
}

func (g *geoShapeV2IndexSectionOpaque) grabBuf(size int) []byte {
	buf := g.tmp
	if cap(buf) < size {
		buf = make([]byte, size)
		g.tmp = buf
	}
	return buf[:size]
}

func (g *geoShapeV2IndexSectionOpaque) Set(key string, value interface{}) {
	switch key {
	case "fieldsOptions":
		g.fieldsOptions = value.(map[string]index.FieldIndexingOptions)
	}
}

func (g *geoShapeV2IndexSectionOpaque) alloc() {
	g.indexContent = make(map[uint16]*geoIndexContent)
	// The other fields will already be initialized during opaque creation
}

func (g *geoShapeV2IndexSectionOpaque) process(f index.GeoShapeV2Field, fieldID uint16,
	docNum uint32) {
	if !g.init {
		g.alloc()
		g.init = true
	}

	indexContent, ok := g.indexContent[fieldID]
	if !ok {
		indexContent = &geoIndexContent{}
		g.indexContent[fieldID] = indexContent
	}

	indexContent.process(f, docNum)
}

// geoIndexContent holds the geo index content for a specific field ID.
// Geo docIDs and segment document numbers are stored as uint32; cell IDs and
// document scores remain uint64.
type geoIndexContent struct {
	innerCells  []uint64
	innerDocIDs []uint32

	crossCells  []uint64
	crossDocIDs []uint32

	docNums        []uint32
	docScoresInner []uint64
	docScoresCross []uint64

	boundingBoxes [][]byte
	shapes        [][]byte

	init bool
}

func (g *geoIndexContent) alloc() {
	g.innerCells = make([]uint64, 0)
	g.innerDocIDs = make([]uint32, 0)
	g.crossCells = make([]uint64, 0)
	g.crossDocIDs = make([]uint32, 0)

	g.docNums = make([]uint32, 0)
	g.docScoresInner = make([]uint64, 0)
	g.docScoresCross = make([]uint64, 0)

	g.boundingBoxes = make([][]byte, 0)
	g.shapes = make([][]byte, 0)
}

func (g *geoIndexContent) process(f index.GeoShapeV2Field, docNum uint32) {
	if !g.init {
		g.alloc()
		g.init = true
	}

	innerCells := f.InnerCells()
	crossCells := f.CrossCells()

	geoDocID := uint32(len(g.docNums))
	g.docNums = append(g.docNums, docNum)

	// Append inner cells along with their corresponding doc IDs
	g.innerCells = append(g.innerCells, innerCells...)
	for range innerCells {
		g.innerDocIDs = append(g.innerDocIDs, geoDocID)
	}

	// Append cross cells along with their corresponding doc IDs
	g.crossCells = append(g.crossCells, crossCells...)
	for range crossCells {
		g.crossDocIDs = append(g.crossDocIDs, geoDocID)
	}

	// Append bounding box bytes, shape bytes and the document score
	g.boundingBoxes = append(g.boundingBoxes, f.EncodedBoundingBox())
	g.shapes = append(g.shapes, f.EncodedShape())
	innerScore, crossScore := f.Scores()
	g.docScoresInner = append(g.docScoresInner, innerScore)
	g.docScoresCross = append(g.docScoresCross, crossScore)
}

func (g *geoShapeV2IndexSectionOpaque) persist(w *FileWriter) error {
	tempBuf := g.grabBuf(binary.MaxVarintLen64)
	for fieldID, content := range g.indexContent {
		// Record the starting position of this field's index content
		fieldStart := w.Count()

		// Write two varints for indicating no doc values
		n := binary.PutUvarint(tempBuf, fieldNotUninverted)
		_, err := w.Write(tempBuf[:n])
		if err != nil {
			return err
		}
		n = binary.PutUvarint(tempBuf, fieldNotUninverted)
		_, err = w.Write(tempBuf[:n])
		if err != nil {
			return err
		}

		content.innerCells, content.innerDocIDs = sortArrayPair(content.innerCells, content.innerDocIDs)
		content.crossCells, content.crossDocIDs = sortArrayPair(content.crossCells, content.crossDocIDs)

		// Write the index content for this field to the file
		err = g.writeIndexContent(content, w)
		if err != nil {
			return err
		}

		g.incrementBytesWritten(uint64(w.Count() - fieldStart))
		g.fieldAddrs[fieldID] = fieldStart
	}

	return nil
}

func (g *geoShapeV2IndexSectionOpaque) writeIndexContent(content *geoIndexContent, w *FileWriter) error {
	tempBuf := g.grabBuf(binary.MaxVarintLen64)

	// Write docNums
	numDocs := uint64(len(content.docNums))
	n := binary.PutUvarint(tempBuf, numDocs)
	_, err := w.Write(tempBuf[:n])
	if err != nil {
		return err
	}

	// Write Doc ID to Doc Num mapping
	_, err = w.WriteUint32Array(content.docNums)
	if err != nil {
		return err
	}

	// Write the Document Scores Inner
	_, err = w.WriteUint64Array(content.docScoresInner)
	if err != nil {
		return err
	}

	// Write the Document Scores Cross
	_, err = w.WriteUint64Array(content.docScoresCross)
	if err != nil {
		return err
	}

	// Write Inner Cells
	_, err = w.WriteUint64Array(content.innerCells)
	if err != nil {
		return err
	}

	// Write Inner Cell Doc IDs
	_, err = w.WriteUint32Array(content.innerDocIDs)
	if err != nil {
		return err
	}

	// Write Cross Cells
	_, err = w.WriteUint64Array(content.crossCells)
	if err != nil {
		return err
	}

	// Write Cross Cell Doc IDs
	_, err = w.WriteUint32Array(content.crossDocIDs)
	if err != nil {
		return err
	}

	// Write Bounding Boxes and Offsets
	_, err = w.WriteArrayWithOffsets(content.boundingBoxes)
	if err != nil {
		return err
	}

	// Write Shapes and Offsets
	_, err = w.WriteArrayWithOffsets(content.shapes)
	if err != nil {
		return err
	}

	return nil
}

// loadGeoIndexContent decodes a field's complete geo index content from mem.
func loadGeoIndexContent(r *FileReader, mem []byte) (*geoIndexContent, error) {
	var pos uint64

	// Load Num Docs
	numDocs, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	pos += uint64(n)
	if numDocs == 0 {
		return nil, fmt.Errorf("no geo docs found")
	}

	// Load Doc ID to Doc Num mapping
	docNums, _, shift, err := r.ReadUint32Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load the Document Scores Inner
	docScoresInner, _, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load the Document Scores Cross
	docScoresCross, _, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Inner Cells
	innerCells, _, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Inner Cell Doc IDs
	innerDocIDs, _, shift, err := r.ReadUint32Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Cross Cells
	crossCells, _, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Cross Cell Doc IDs
	crossDocIDs, _, shift, err := r.ReadUint32Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load BBox Metadata
	bBoxes, shift, err := r.ReadArrayWithOffsets(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Shape Metadata
	shapes, shift, err := r.ReadArrayWithOffsets(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	return &geoIndexContent{
		docNums:        docNums,
		docScoresInner: docScoresInner,
		docScoresCross: docScoresCross,
		innerCells:     innerCells,
		innerDocIDs:    innerDocIDs,
		crossCells:     crossCells,
		crossDocIDs:    crossDocIDs,
		boundingBoxes:  bBoxes,
		shapes:         shapes,
	}, nil
}

// mergeIndexContents combines the geo index contents of multiple segments into
// a single geoIndexContent, dropping deleted documents and remapping doc numbers.
func (g *geoShapeV2IndexSectionOpaque) mergeIndexContents(indexInfos []*geoIndexInfo) (*geoIndexContent, error) {
	var totalDocs, totalInner, totalCross int
	for _, indexInfo := range indexInfos {
		totalDocs += len(indexInfo.content.docNums)
		totalInner += len(indexInfo.content.innerCells)
		totalCross += len(indexInfo.content.crossCells)
	}

	mergedContent := &geoIndexContent{
		docNums:        make([]uint32, 0, totalDocs),
		docScoresInner: make([]uint64, 0, totalDocs),
		docScoresCross: make([]uint64, 0, totalDocs),
		innerCells:     make([]uint64, 0, totalInner),
		innerDocIDs:    make([]uint32, 0, totalInner),
		crossCells:     make([]uint64, 0, totalCross),
		crossDocIDs:    make([]uint32, 0, totalCross),
		boundingBoxes:  make([][]byte, 0, totalDocs),
		shapes:         make([][]byte, 0, totalDocs),
	}

	// Assign merged geo docIDs and build, per segment, a direct
	// oldGeoDocID -> mergedGeoDocID slice
	segRemaps, numGeoDocs := buildGeoDocRemaps(indexInfos, mergedContent)
	if numGeoDocs == 0 {
		return nil, nil
	}

	// Each segment's inner and cross cells are already stored sorted
	// So instead of concatenating every segment's cells and sorting the whole
	// set, we k-way merge the pre-sorted per-segment runs while.
	innerCursors := make([]*geoCellCursor, 0, len(indexInfos))
	crossCursors := make([]*geoCellCursor, 0, len(indexInfos))
	for s, indexInfo := range indexInfos {
		innerCursors = append(innerCursors, &geoCellCursor{
			cells:  indexInfo.content.innerCells,
			docIDs: indexInfo.content.innerDocIDs,
			remap:  segRemaps[s],
		})
		crossCursors = append(crossCursors, &geoCellCursor{
			cells:  indexInfo.content.crossCells,
			docIDs: indexInfo.content.crossDocIDs,
			remap:  segRemaps[s],
		})
	}

	mergedContent.innerCells, mergedContent.innerDocIDs =
		kWayMergeCells(innerCursors, mergedContent.innerCells, mergedContent.innerDocIDs)
	mergedContent.crossCells, mergedContent.crossDocIDs =
		kWayMergeCells(crossCursors, mergedContent.crossCells, mergedContent.crossDocIDs)

	// Bounding boxes, shapes and document scores are stored per document in geo
	// docID order, so they only need to be filtered for dropped documents and
	// concatenated in the same order the merged docNums were assigned above.
	for s, indexInfo := range indexInfos {
		remap := segRemaps[s]
		for i, bbox := range indexInfo.content.boundingBoxes {
			if remap[i] == uint32(math.MaxUint32) {
				continue
			}
			mergedContent.boundingBoxes = append(mergedContent.boundingBoxes, bbox)
		}
		for i, shape := range indexInfo.content.shapes {
			if remap[i] == uint32(math.MaxUint32) {
				continue
			}
			mergedContent.shapes = append(mergedContent.shapes, shape)
		}
		for i, score := range indexInfo.content.docScoresInner {
			if remap[i] == uint32(math.MaxUint32) {
				continue
			}
			mergedContent.docScoresInner = append(mergedContent.docScoresInner, score)
		}
		for i, score := range indexInfo.content.docScoresCross {
			if remap[i] == uint32(math.MaxUint32) {
				continue
			}
			mergedContent.docScoresCross = append(mergedContent.docScoresCross, score)
		}
	}

	return mergedContent, nil
}

// buildGeoDocRemaps assigns the merged segment's geo docIDs and returns, for
// each input segment, a slice mapping that segment's old geo docID to its
// merged geo docID.
func buildGeoDocRemaps(indexInfos []*geoIndexInfo,
	mergedContent *geoIndexContent) (segRemaps [][]uint32, numGeoDocs uint64) {
	segRemaps = make([][]uint32, len(indexInfos))
	for s, indexInfo := range indexInfos {
		remap := make([]uint32, len(indexInfo.content.docNums))
		for geoDocID, oldDocNum := range indexInfo.content.docNums {
			newDocNum := indexInfo.newDocNums[oldDocNum]
			if newDocNum == docDropped {
				remap[geoDocID] = uint32(math.MaxUint32)
				continue
			}
			remap[geoDocID] = uint32(numGeoDocs)
			numGeoDocs++
			mergedContent.docNums = append(mergedContent.docNums, uint32(newDocNum))
		}
		segRemaps[s] = remap
	}
	return segRemaps, numGeoDocs
}

// geoCellCursor iterates one segment's sorted (cell, docID) pairs during a
// k-way merge. It skips cells whose document was dropped and translates each
// surviving cell's geo docID into the merged segment's geo docID space.
type geoCellCursor struct {
	cells  []uint64
	docIDs []uint32 // parallel to cells: the old geo docID of each cell

	// remap maps an old geo docID to its merged geo docID, or math.MaxUint32 if
	// the document was deleted during the merge.
	remap []uint32

	pos      int
	curCell  uint64
	curDocID uint32
}

// next advances to the next non-dropped cell, populating curCell and curDocID
// with the cell value and its merged geo docID. It returns false once the
// cursor is exhausted.
func (c *geoCellCursor) next() bool {
	for c.pos < len(c.cells) {
		cell := c.cells[c.pos]
		newGeoDocID := c.remap[c.docIDs[c.pos]]
		c.pos++

		if newGeoDocID == uint32(math.MaxUint32) {
			continue
		}
		c.curCell = cell
		c.curDocID = newGeoDocID
		return true
	}
	return false
}

// geoCursorHeap is a min-heap of cursors ordered by their current cell value,
// used to drive the k-way merge of the pre-sorted per-segment cell runs.
type geoCursorHeap []*geoCellCursor

func (h geoCursorHeap) Len() int           { return len(h) }
func (h geoCursorHeap) Less(i, j int) bool { return h[i].curCell < h[j].curCell }
func (h geoCursorHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *geoCursorHeap) Push(x interface{}) {
	*h = append(*h, x.(*geoCellCursor))
}

func (h *geoCursorHeap) Pop() interface{} {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return it
}

// kWayMergeCells merges the pre-sorted cell runs held by the given cursors into
// ascending order, appending the merged cells and their parallel merged geo
// docIDs to outCells and outDocIDs respectively and returning the extended
// slices.
func kWayMergeCells(cursors []*geoCellCursor, outCells []uint64, outDocIDs []uint32) ([]uint64, []uint32) {
	h := make(geoCursorHeap, 0, len(cursors))
	for _, c := range cursors {
		if c.next() {
			h = append(h, c)
		}
	}
	heap.Init(&h)

	for h.Len() > 0 {
		c := h[0]
		outCells = append(outCells, c.curCell)
		outDocIDs = append(outDocIDs, c.curDocID)
		if c.next() {
			// current cursor still has cells; restore heap order in place
			heap.Fix(&h, 0)
		} else {
			// cursor exhausted; drop it from the heap
			heap.Pop(&h)
		}
	}

	return outCells, outDocIDs
}

func (g *geoShapeV2IndexSectionOpaque) incrementBytesWritten(val uint64) {
	atomic.AddUint64(&g.bytesWritten, val)
}

func (g *geoShapeV2IndexSectionOpaque) BytesWritten() uint64 {
	return atomic.LoadUint64(&g.bytesWritten)
}

// arrayPair holds references to both slices to swap and sort them in tandem.
// primary holds the sort keys (cell IDs, uint64); secondary holds the parallel
// geo docIDs (uint32) that must move with them.
type arrayPair struct {
	primary   []uint64
	secondary []uint32
}

func (a arrayPair) Len() int {
	return len(a.primary)
}

func (a arrayPair) Less(i, j int) bool {
	return a.primary[i] < a.primary[j]
}

func (a arrayPair) Swap(i, j int) {
	a.primary[i], a.primary[j] = a.primary[j], a.primary[i]
	a.secondary[i], a.secondary[j] = a.secondary[j], a.secondary[i]
}

func sortArrayPair(primary []uint64, secondary []uint32) ([]uint64, []uint32) {
	// Protect against mismatched slice lengths
	if len(primary) != len(secondary) {
		panic("slices must be of equal length")
	}

	sort.Sort(arrayPair{primary: primary, secondary: secondary})
	return primary, secondary
}
