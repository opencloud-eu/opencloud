//	Copyright (c) 2026 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package segment

import (
	"reflect"
	"sync/atomic"
	"unsafe"
)

const StatsKey = "zap_stats"

type Stats struct {
	TotNewRootDocsProcessed uint64
	TotNewDocsProcessed     uint64
	TotNewDocsIndexed       uint64
	TotNewDocsDropped       uint64
	TotNewVectorsProcessed  uint64

	TotPersistBeg uint64
	TotPersistEnd uint64
	TotPersistErr uint64

	TotMergesBeg          uint64
	TotMergesEnd          uint64
	TotMergesErrors       uint64
	TotMergeInputSegments uint64
	TotMergeOutputDocs    uint64
	TotMergeDroppedDocs   uint64

	TotVecSectionMergesBegin            uint64
	TotVecSectionMergesEnd              uint64
	TotVecSectionMergeErr               uint64
	TotVecSectionMergeTime              uint64
	TotVecSectionVecsReconstructed      uint64
	TotVecSectionIVFIndexesCreated      uint64
	TotVecSectionFlatIndexesCreated     uint64
	TotVecSectionTrainingTime           uint64
	TotVecSectionFastMerges             uint64
	TotVecSectionFastMergeErrs          uint64
	TotVecSectionNaiveMerges            uint64
	TotVecSectionMetadataBytesWritten   uint64
	TotVecSectionFloatIndexBytesWritten uint64
	TotVecSectionVecsDeleted            uint64
	TotVecSectionFieldsIndexed          uint64
	TotVecSectionTrainOps               uint64
	TotVecSectionIndexWriteTime         uint64
	TotVecSectionVecsProcessedTime      uint64

	TotVecSectionTrainingPhaseVecsProcessedTime uint64
	TotVecSectionTrainingPhaseTrainingTime      uint64

	TotOpenBeg    uint64
	TotOpenEnd    uint64
	TotOpenErrors uint64

	TotSegmentsClosed uint64
}

func (s *Stats) StatsMap() map[string]interface{} {
	svet := reflect.TypeOf(s).Elem()
	n := svet.NumField()
	m := make(map[string]interface{}, n)
	base := unsafe.Pointer(s)
	for i := 0; i < n; i++ {
		field := svet.Field(i)

		// use unsafe.Pointer to avoid heap allocs, safe to do this here since all the stats
		// enforced to be uint64
		p := (*uint64)(unsafe.Pointer(uintptr(base) + field.Offset))
		m[field.Name] = atomic.LoadUint64(p)
	}
	return m
}
