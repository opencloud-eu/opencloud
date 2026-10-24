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
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"sync/atomic"

	"github.com/RoaringBitmap/roaring/v2"
	index "github.com/blevesearch/bleve_index_api"
	segment "github.com/blevesearch/scorch_segment_api/v2"
)

const Version uint32 = 17

const Type string = "zap"

const fieldNotUninverted uint64 = math.MaxUint64

func (sb *SegmentBase) Persist(path string) error {
	atomic.AddUint64(&sb.stats.TotPersistBeg, 1)
	var err error
	defer func() {
		atomic.AddUint64(&sb.stats.TotPersistEnd, 1)
		if err != nil {
			atomic.AddUint64(&sb.stats.TotPersistErr, 1)
		}
	}()
	err = PersistSegmentBase(sb, path)
	if err != nil {
		return err
	}
	return nil
}

// WriteTo is an implementation of io.WriterTo interface.
func (sb *SegmentBase) WriteTo(w io.Writer) (int64, error) {
	if w == nil {
		return 0, fmt.Errorf("invalid writer found")
	}

	n, err := persistSegmentBaseToWriter(sb, w)
	return int64(n), err
}

// PersistSegmentBase persists SegmentBase in the zap file format.
func PersistSegmentBase(sb *SegmentBase, path string) error {
	// since in-memory data is not processed by any writer callback,
	// check with the latest writer to see if data needs to be processed
	writer, err := NewFileWriter(nil, []byte(path))
	if err != nil {
		return err
	}
	if writer.id != sb.fileReader.id {
		// rewrite the segment base with the latest writer callback;
		// the rewrite will persist the segment to the given path (upon
		// success), so we should return early to avoid overwriting again.
		return rewriteSegmentBase(sb, path)
	}

	flag := os.O_RDWR | os.O_CREATE

	f, err := os.OpenFile(path, flag, 0600)
	if err != nil {
		return err
	}

	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(path)
	}

	_, err = persistSegmentBaseToWriter(sb, f)
	if err != nil {
		cleanup()
		return err
	}

	err = f.Sync()
	if err != nil {
		cleanup()
		return err
	}

	err = f.Close()
	if err != nil {
		cleanup()
		return err
	}

	return err
}

// rewrites the segment base with the latest writer callback by leveraging
// the merge path
func rewriteSegmentBase(sb *SegmentBase, path string) error {
	closeCh := make(chan struct{})
	defer close(closeCh)
	_, _, err := mergeSegmentBases([]*SegmentBase{sb}, []*roaring.Bitmap{nil},
		path, DefaultChunkMode, closeCh, nil, nil, sb.stats)
	if err != nil {
		return err
	}
	return nil
}

type bufWriter struct {
	w *bufio.Writer
	n int
}

func (br *bufWriter) Write(in []byte) (int, error) {
	n, err := br.w.Write(in)
	br.n += n
	return n, err
}

func persistSegmentBaseToWriter(sb *SegmentBase, w io.Writer) (int, error) {
	br := &bufWriter{w: bufio.NewWriter(w)}

	_, err := br.Write(sb.mem)
	if err != nil {
		return 0, err
	}

	err = persistFooter(sb.numDocs, sb.storedIndexOffset, sb.sectionsIndexOffset,
		sb.chunkMode, sb.memCRC, br, sb.fileReader.id)
	if err != nil {
		return 0, err
	}

	err = br.w.Flush()
	if err != nil {
		return 0, err
	}

	return br.n, nil
}

func persistStoredFieldValues(fieldID int,
	storedFieldValues [][]byte, stf []byte, spf [][]uint64,
	curr int, metaEncode varintEncoder, data []byte) (
	int, []byte, error) {
	for i := 0; i < len(storedFieldValues); i++ {
		// encode field
		_, err := metaEncode(uint64(fieldID))
		if err != nil {
			return 0, nil, err
		}
		// encode type
		_, err = metaEncode(uint64(stf[i]))
		if err != nil {
			return 0, nil, err
		}
		// encode start offset
		_, err = metaEncode(uint64(curr))
		if err != nil {
			return 0, nil, err
		}
		// end len
		_, err = metaEncode(uint64(len(storedFieldValues[i])))
		if err != nil {
			return 0, nil, err
		}
		// encode number of array pos
		_, err = metaEncode(uint64(len(spf[i])))
		if err != nil {
			return 0, nil, err
		}
		// encode all array positions
		for _, pos := range spf[i] {
			_, err = metaEncode(pos)
			if err != nil {
				return 0, nil, err
			}
		}

		data = append(data, storedFieldValues[i]...)
		curr += len(storedFieldValues[i])
	}

	return curr, data, nil
}

func InitSegmentBase(mem []byte, memCRC uint32, chunkMode uint32, numDocs uint64,
	storedIndexOffset uint64, sectionsIndexOffset uint64,
	config map[string]interface{}, stats *segment.Stats) (*SegmentBase, error) {
	sb := &SegmentBase{
		mem:                 mem,
		memCRC:              memCRC,
		chunkMode:           chunkMode,
		numDocs:             numDocs,
		storedIndexOffset:   storedIndexOffset,
		sectionsIndexOffset: sectionsIndexOffset,
		fieldDvReaders:      make([][]*docValueReader, len(segmentSections)),
		updatedFields:       make(map[string]*index.UpdateFieldInfo),
		invIndexCache:       newInvertedIndexCache(),
		vecIndexCache:       newVectorIndexCache(),
		synIndexCache:       newSynonymIndexCache(),
		geoIndexCache:       newGeoIndexCache(),
		nstIndexCache:       newNestedIndexCache(),
		trainedIndexCache:   newTrainedIndexCache(),
		// following fields gets populated by loadFields
		fieldsMap:     make(map[string]uint16),
		fieldsOptions: make(map[string]index.FieldIndexingOptions),
		fieldsInv:     make([]string, 0),
		config:        config,
		stats:         stats,
	}
	sb.updateSize()

	// initialize the file reader with an empty callback
	// since the data is not yet persisted, the data has also
	// not been processed by any writer callback
	fileReader, err := NewFileReader("", nil)
	if err != nil {
		return nil, err
	}
	sb.fileReader = fileReader

	// load the data/section starting offsets for each field
	// by via the sectionsIndexOffset as starting point.
	err = sb.loadFields()
	if err != nil {
		return nil, err
	}

	err = sb.loadDvReaders()
	if err != nil {
		return nil, err
	}

	// initialize any of the caches if needed
	err = sb.nstIndexCache.initialize(sb.numDocs, sb.getEdgeListOffset(), sb.mem)
	if err != nil {
		return nil, err
	}

	return sb, nil
}
