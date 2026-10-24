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

//go:build vectors
// +build vectors

package zap

import (
	"errors"
	"reflect"
	"sync"
)

var (
	errBatcherStopped error = errors.New("batcher has been stopped")
)

var (
	reflectStaticSizeRequestBatcher uint64
	reflectStaticSizeCoalesceQueue  uint64
)

func init() {
	var b requestBatcher
	reflectStaticSizeRequestBatcher = uint64(reflect.TypeOf(b).Size())
	var cq coalesceQueue
	reflectStaticSizeCoalesceQueue = uint64(reflect.TypeOf(cq).Size())
}

// The requestBatcher is responsible for batching search requests to a Faiss index.
// It will accumulate incoming search requests and execute them in batches to improve performance.
// The batcher will use the provided Faiss index to perform the searches, and it
// will manage the batching logic, including timing and concurrency control.
type requestBatcher struct {
	// the coalesce queue that manages the batching of incoming search requests.
	cq *coalesceQueue
}

func newRequestBatcher(idx faissQueryBatch) *requestBatcher {
	b := &requestBatcher{
		cq: newCoalesceQueue(idx),
	}
	return b
}

// search performs a search on the Faiss index using the provided query vector and k value.
// NOTE: it must be ensured that every query vector passed to this method has the same dimensionality
// as the vectors in the Faiss index, this is considered as an invariant to be upheld by the caller,
// and is not checked within this method for performance reasons.
func (b *requestBatcher) search(qVector *vectorSet, k int64) ([]float32, []int64, error) {
	// create a new batch request for this search query.
	req, respCh := newBatchRequest(qVector, k)
	// check if the batcher has been stopped before processing the search request.
	select {
	case b.cq.enqueueCh <- req:
	case <-b.cq.stopCh:
		return nil, nil, errBatcherStopped
	}
	// wait for the search results to be sent back through the response channel,
	// and return those results to the caller.
	resp := <-respCh
	return resp.distances, resp.ids, resp.err
}

func (b *requestBatcher) stop() {
	b.cq.stop()
}

func (b *requestBatcher) size() uint64 {
	return reflectStaticSizeRequestBatcher +
		b.cq.size()
}

// --------------------------------------------------
// batch request
// --------------------------------------------------

type batchRequest struct {
	qVector *vectorSet
	k       int64
	respCh  []chan *batchResponse
}

func newBatchRequest(qVector *vectorSet, k int64) (*batchRequest, chan *batchResponse) {
	// response channel for sending the search results back to the requester.
	respChan := make(chan *batchResponse, 1)
	return &batchRequest{
		qVector: qVector,
		k:       k,
		respCh:  []chan *batchResponse{respChan},
	}, respChan
}

// canMerge checks if this batch request can be merged with another request.
// For now, we can only merge requests that have the same k value.
func (r *batchRequest) canMerge(other *batchRequest) bool {
	// for now, we can only merge requests that have the same k value,
	// since the Faiss search API requires a single k value for each search.
	return r.k == other.k
}

// mergeWith combines another batch request into this one by concatenating their query vectors and response channels.
// NOTE: must only be called after veryfing that canMerge() returns true for these two requests.
func (r *batchRequest) mergeWith(other *batchRequest) {
	// merge the query vectors of the two requests by concatenating them together.
	r.qVector.mergeWith(other.qVector)
	// append the response channels from the other request to this request, so that when the search results are ready,
	// we can send the results back to all requesters that were merged into this batch.
	r.respCh = append(r.respCh, other.respCh...)
}

func (r *batchRequest) sendResponse(distances []float32, ids []int64, err error) {
	// we may have multiple batches merged together, so we need to segregate the results for each original request
	// and send them back to the appropriate response channels.
	if err != nil {
		// if there was an error during the search, send the error back to all requesters in this batch.
		for _, respCh := range r.respCh {
			respCh <- newBatchResponse(nil, nil, err)
			close(respCh)
		}
		return
	}
	// if the search was successful, we need to split the combined results back into individual responses for each original request.
	for i, respCh := range r.respCh {
		offset := int64(i) * r.k
		// calculate the start and end indices for the results corresponding to this response channel.
		curDistances := distances[offset : offset+r.k]
		curIDs := ids[offset : offset+r.k]
		// send the results back to the requester through the response channel.
		respCh <- newBatchResponse(curDistances, curIDs, nil)
		// close the response channel to signal that the response has been sent and there will be no more data.
		close(respCh)
	}
}

// --------------------------------------------------
// batch response
// --------------------------------------------------

type batchResponse struct {
	distances []float32
	ids       []int64
	err       error
}

func newBatchResponse(distances []float32, ids []int64, err error) *batchResponse {
	return &batchResponse{
		distances: distances,
		ids:       ids,
		err:       err,
	}
}

// ---------------------------------------------------
// batch manager
// ---------------------------------------------------
type batchManager struct {
	batchPool sync.Pool
}

func newBatchManager() *batchManager {
	return &batchManager{
		batchPool: sync.Pool{
			New: func() any {
				return make([]*batchRequest, 0, 16)
			},
		},
	}
}

func (m *batchManager) getBatch() []*batchRequest {
	return m.batchPool.Get().([]*batchRequest)[:0]
}

func (m *batchManager) putBatch(batch []*batchRequest) {
	clear(batch)
	m.batchPool.Put(batch[:0])
}

// --------------------------------------------------
// coalesceQueue
// --------------------------------------------------
// Implements Nagle's algorithm for coalescing search requests:
//   - The coalesce goroutine continuously receives and coalesces incoming requests.
//   - When the flusher is idle, the coalesce goroutine hands off the coalesced batch.
//   - While the flusher is busy executing a batch, the coalesce goroutine keeps coalescing new requests.
//   - Once the flusher completes, the coalesce goroutine hands off any accumulated requests right away.
type coalesceQueue struct {
	// the Faiss index that this coalesce queue will execute search requests against.
	idx faissQueryBatch
	// channel for enqueuing new batch requests into the queue.
	enqueueCh chan *batchRequest
	// channel for handing off coalesced batches to the flusher goroutine for execution.
	flushCh chan []*batchRequest
	// safeguard to ensure that the stop() method is thread-safe and can only be called once,
	// preventing multiple close operations on the stopCh.
	stopOnce sync.Once
	// channel for signaling the batcher to stop processing requests and shut down.
	stopCh chan struct{}
	// closed when filler goroutine has exited after receiving a stop signal.
	fillerDoneCh chan struct{}
	// closed when flusher goroutine has exited after receiving a stop signal.
	flusherDoneCh chan struct{}
	// a sync.Pool for reusing batch slices to reduce allocations and GC overhead.
	batchManager *batchManager
}

func newCoalesceQueue(idx faissQueryBatch) *coalesceQueue {
	q := &coalesceQueue{
		idx:           idx,
		enqueueCh:     make(chan *batchRequest),
		flushCh:       make(chan []*batchRequest),
		stopCh:        make(chan struct{}),
		fillerDoneCh:  make(chan struct{}),
		flusherDoneCh: make(chan struct{}),
		batchManager:  newBatchManager(),
	}
	go q.filler()
	go q.flusher()
	return q
}

func (q *coalesceQueue) stop() {
	q.stopOnce.Do(func() {
		close(q.stopCh)
	})
	// wait for all goroutines to exit
	<-q.fillerDoneCh
	<-q.flusherDoneCh
}

// filler is the enqueuer goroutine. It receives incoming search requests,
// coalesces them into batches, and hands them off to the flusher when it is idle.
func (q *coalesceQueue) filler() {
	defer close(q.fillerDoneCh)
	var pendingBatch []*batchRequest
	for {
		if len(pendingBatch) > 0 {
			select {
			case req := <-q.enqueueCh:
				pendingBatch = q.coalesce(pendingBatch, req)
			case q.flushCh <- pendingBatch:
				pendingBatch = nil
			case <-q.stopCh:
				q.flushCh <- pendingBatch
				return
			}
		} else {
			select {
			case req := <-q.enqueueCh:
				pendingBatch = q.coalesce(pendingBatch, req)
			case <-q.stopCh:
				return
			}
		}
	}
}

// flusher is the background goroutine that executes batches handed off by the monitor.
func (q *coalesceQueue) flusher() {
	defer close(q.flusherDoneCh)
	for {
		select {
		case batch := <-q.flushCh:
			q.executeBatch(batch)
		case <-q.fillerDoneCh:
			return
		}
	}
}

// coalesce merges req into the queue, either by finding a compatible pending
// request to merge with or by appending a new entry.
func (q *coalesceQueue) coalesce(queue []*batchRequest, req *batchRequest) []*batchRequest {
	for _, pendingReq := range queue {
		if pendingReq.canMerge(req) {
			pendingReq.mergeWith(req)
			return queue
		}
	}
	// No compatible request found; clone the query vector so that future
	// merges into this entry do not mutate the caller's data.
	req.qVector = req.qVector.clone()
	if queue == nil {
		queue = q.batchManager.getBatch()
	}
	return append(queue, req)
}

// executeBatch runs all coalesced requests against the Faiss index and delivers results.
func (q *coalesceQueue) executeBatch(batch []*batchRequest) {
	for _, req := range batch {
		distances, ids, err := q.idx.batchSearch(req.qVector, req.k)
		req.sendResponse(distances, ids, err)
	}
	// recycle the batch slice back into the pool
	q.batchManager.putBatch(batch)
}

func (q *coalesceQueue) size() uint64 {
	return reflectStaticSizeCoalesceQueue
}
