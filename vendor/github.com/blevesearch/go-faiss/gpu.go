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

//go:build gpu
// +build gpu

package faiss

/*
#include <stddef.h>
#include <faiss/c_api/gpu/StandardGpuResources_c.h>
#include <faiss/c_api/gpu/GpuAutoTune_c.h>
#include <faiss/c_api/gpu/GpuClonerOptions_c.h>
#include <faiss/c_api/gpu/DeviceUtils_c.h>
#include <faiss/c_api/gpu/GpuIndex_c_ex.h>
#include <faiss/c_api/gpu/GpuIndexIVF_c_ex.h>
#include <faiss/c_api/gpu/GpuMemoryEstimate_c.h>
#include <faiss/c_api/gpu/GpuMemoryPool_c.h>
*/
import "C"
import (
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// memorySpace controls where GPU index data is allocated.
type memorySpace int

const (
	// memorySpaceDevice uses standard GPU memory (cudaMalloc).
	memorySpaceDevice memorySpace = 1
	// memorySpaceUnified uses CUDA managed memory (cudaMallocManaged),
	// allowing the index to exceed GPU memory on Pascal+ (CC 6.0+) GPUs.
	memorySpaceUnified memorySpace = 2
)

const (
	// reserve atleast 10% of total GPU memory for the the memory pool.
	defaultGPUPoolBudget = 0.1
	// reserve 8MB of GPU memory for temporary buffer _per_ GPU index.
	// NOTE: This number must be divisible by 1024.
	// CAUTION: Setting this number too high can result in degraded performance as we we may end up
	// reducing the total number of vector indexes in the GPU as a whole.
	defaultGPUTempMemorySize = 8 * 1024 * 1024
	// use device memory by default since we already do memory estimation and reservation in our GPU snapshot store.
	defaultGPUMemoryMode = memorySpaceDevice
	// disable pinned memory by default to avoid exhausting CPU memory when cloning multiple indexes to GPU.
	defaultGPUPinnedMemory = 0
	// refresh the order in which GPUs are assigned every 500ms.
	defaultGPULoadBalancerInterval = 500 * time.Millisecond
)

var (
	gpuCount        int
	loadBalancer    *gpuLoadBalancer
	snapshotStore   *gpuSnapshotStore
	memoryPoolStore *gpuMemoryPoolStore

	reflectStaticSizeFaissGPUIndex uint64
)

// --------------------------------
// GPU Setup
// --------------------------------

func init() {
	var g faissGPUIndex
	reflectStaticSizeFaissGPUIndex = uint64(reflect.TypeOf(g).Size())

	var err error
	gpuCount, err = numGPUs()
	if err != nil || gpuCount <= 0 {
		gpuCount = 0
	}
	if gpuCount > 0 {
		snapshotStore = newGPUSnapshotStore()
		if gpuCount > 1 {
			loadBalancer = newGPULoadBalancer()
			go loadBalancer.monitor()
		}
		memoryPoolStore = newGPUMemoryPoolStore()
	}
}

// numGPUs returns the number of available GPU devices.
func numGPUs() (int, error) {
	var rv C.int
	c := C.faiss_get_num_gpus(&rv)
	if c != 0 {
		return 0, newFaissError(ErrGPUSetupFailed, getLastError(), int(c))
	}
	return int(rv), nil
}

func getBestGPUDevice() (int, error) {
	if gpuCount == 0 {
		return 0, ErrNoUsableGPUDevices
	}
	if gpuCount == 1 {
		// if there's only one GPU, just return it
		// without going through the load balancer logic.
		return 0, nil
	}
	return loadBalancer.nextDevice(), nil
}

func probeGPUDevice(device int) bool {
	var probeResult C.int
	c := C.faiss_probe_gpu(
		C.int(device),
		&probeResult,
	)
	return c == 0 && probeResult == 0
}

// ---------------------------------
// GPU Snapshot
// ---------------------------------

// gpuSnapshot is a per-device view of GPU state.
type gpuSnapshot struct {
	// GPU device id this snapshot describes.
	device int
	// Total memory in bytes in the GPU.
	totalMem uint64
	// Free memory in bytes in the GPU.
	freeMem atomic.Uint64
}

// newGPUSnapshot creates a new gpuSnapshot for the given device.
// assumed that the total memory available is also the initial free memory.
func newGPUSnapshot(device int, totMemory uint64) *gpuSnapshot {
	s := &gpuSnapshot{device: device, totalMem: totMemory}
	s.setFreeMemory(totMemory)
	return s
}

// reserve attempts to reserve the given size in bytes against the snapshot's free memory.
// it ensures that atleast we remain under the data quota after the reservation, and returns
// ErrGPUOutOfMemory if the reservation cannot be fulfilled.
func (gs *gpuSnapshot) reserveMemory(required uint64) error {
	for {
		cur := gs.freeMemory()
		if required > cur {
			return ErrGPUOutOfMemory
		}
		after := cur - required
		if after < gs.poolQuota() {
			return ErrGPUOutOfMemory
		}
		if gs.freeMem.CompareAndSwap(cur, after) {
			return nil
		}
	}
}

// release adds the given size in bytes back to the snapshot's free memory.
// it ensures that we never exceed the total memory of the GPU when releasing,
// and caps the free memory at totalMem if that happens.
func (gs *gpuSnapshot) releaseMemory(released uint64) {
	for {
		cur := gs.freeMemory()
		after := cur + released
		if after > gs.totalMem {
			after = gs.totalMem
		}
		if gs.freeMem.CompareAndSwap(cur, after) {
			return
		}
	}
}

func (gs *gpuSnapshot) setFreeMemory(freeMem uint64) {
	gs.freeMem.Store(freeMem)
}

func (gs *gpuSnapshot) freeMemory() uint64 {
	return gs.freeMem.Load()
}

func (gs *gpuSnapshot) compare(other *gpuSnapshot) int {
	curFree := gs.freeMemory()
	otherFree := other.freeMemory()
	if curFree == otherFree {
		return 0
	} else if curFree < otherFree {
		return -1
	}
	return 1
}

func (gs *gpuSnapshot) copyTo(other *gpuSnapshot) {
	other.device = gs.device
	other.totalMem = gs.totalMem
	other.setFreeMemory(gs.freeMemory())
}

func (gs *gpuSnapshot) clone() *gpuSnapshot {
	clone := &gpuSnapshot{}
	gs.copyTo(clone)
	return clone
}

func (gs *gpuSnapshot) poolQuota() uint64 {
	return uint64(float64(gs.totalMem) * defaultGPUPoolBudget)
}

func (gs *gpuSnapshot) dataQuota() uint64 {
	return gs.totalMem - gs.poolQuota()
}

// ---------------------------------
// GPU Snapshot Store
// ---------------------------------

// gpuSnapshotStore maintains a mapping of GPU device id to its snapshot,
// providing thread-safe access and updates to GPU state information.
type gpuSnapshotStore struct {
	// device -> snapshot immutable mapping
	// snapshot[i] always describes device i,
	// where i goes from 0 to gpuCount-1.
	snapshots []*gpuSnapshot
}

func newGPUSnapshotStore() *gpuSnapshotStore {
	snapshots := make([]*gpuSnapshot, gpuCount)
	for device := 0; device < gpuCount; device++ {
		totMemory := uint64(0)
		// first probe if the device is healthy and can be used
		if probeGPUDevice(device) {
			var freeBytes C.size_t
			if c := C.faiss_gpu_free_memory(
				C.int(device),
				&freeBytes,
			); c == 0 {
				totMemory = uint64(freeBytes)
			}
		}
		// if we fail to get the free memory for the GPU,
		// we still create a snapshot with 0 total and free memory,
		// which will cause all reservation attempts to fail but won't cause any crashes.
		snapshots[device] = newGPUSnapshot(device, totMemory)
	}
	return &gpuSnapshotStore{snapshots: snapshots}
}

func (gss *gpuSnapshotStore) snapshotForDevice(device int) *gpuSnapshot {
	return gss.snapshots[device]
}

func (gss *gpuSnapshotStore) compare(i, j int) int {
	return gss.snapshots[i].compare(gss.snapshots[j])
}

func (gss *gpuSnapshotStore) copyTo(other *gpuSnapshotStore) {
	for i := 0; i < gpuCount; i++ {
		gss.snapshots[i].copyTo(other.snapshots[i])
	}
}

func (gss *gpuSnapshotStore) clone() *gpuSnapshotStore {
	clone := &gpuSnapshotStore{
		snapshots: make([]*gpuSnapshot, gpuCount),
	}
	for i := 0; i < gpuCount; i++ {
		clone.snapshots[i] = gss.snapshots[i].clone()
	}
	return clone
}

// ---------------------------------
// GPU Load Balancer
// ---------------------------------

// gpuLoadBalancer distributes GPU allotments in a round-robin manner
// in multi-GPU setups, while optimizing to always select the best GPU.
type gpuLoadBalancer struct {
	cursor       atomic.Uint32
	mu           sync.RWMutex
	order        []int
	scratchOrder []int
	scratchStore *gpuSnapshotStore
}

func newGPULoadBalancer() *gpuLoadBalancer {
	lb := &gpuLoadBalancer{
		order:        make([]int, gpuCount),
		scratchOrder: make([]int, gpuCount),
		scratchStore: snapshotStore.clone(),
	}
	for i := 0; i < gpuCount; i++ {
		lb.order[i] = i
		lb.scratchOrder[i] = i
	}
	return lb
}

// monitor periodically refreshes the GPU snapshots.
func (lb *gpuLoadBalancer) monitor() {
	ticker := time.NewTicker(defaultGPULoadBalancerInterval)
	defer ticker.Stop()
	for range ticker.C {
		lb.refresh()
	}
}

// refresh updates the load balancer's GPU snapshots by querying each GPU.
func (lb *gpuLoadBalancer) refresh() {
	// refresh the scratch snapshots with the latest GPU state.
	snapshotStore.copyTo(lb.scratchStore)
	// Sort in descending order
	slices.SortFunc(lb.scratchOrder, func(i, j int) int {
		return lb.scratchStore.compare(j, i)
	})
	// acquire lock to update the real order and reset the round-robin index,
	// ensuring that the next allocation cycle uses the updated order and
	// starts from the most appealing GPU.
	lb.mu.Lock()
	defer lb.mu.Unlock()
	copy(lb.order, lb.scratchOrder)
	lb.cursor.Store(0)
}

// nextDevice returns the next GPU device in round-robin order.
func (lb *gpuLoadBalancer) nextDevice() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	// atomically allocates the GPU.
	// Minus 1 for zero based index and modulo by gpuCount to wrap around.
	idx := (lb.cursor.Add(1) - 1) % uint32(gpuCount)
	// return the device id corresponding to the allocated GPU.
	return lb.order[idx]
}

// --------------------------------
// GPU Index
// --------------------------------

// GPUIndex is the interface for a Faiss index that resides on GPU.
type GPUIndex interface {
	// D returns the dimension of the indexed vectors.
	D() int
	// Add adds vectors to the index.
	Add(x []float32) error
	// Train trains the index on a representative set of vectors.
	Train(x []float32) error
	// Search queries the index with the vectors in x.
	// Returns the IDs of the k nearest neighbors for each query vector and the
	// corresponding distances.
	Search(x []float32, k int64) (distances []float32, labels []int64, err error)
	// Size estimates the memory footprint of the index assuming in bytes,
	// if the underlying faiss index is memory-mapped and not fully loaded into memory.
	Size() uint64
	// Close frees the memory used by the index.
	Close()
	// gPtr returns the underlying C pointer to the FaissGpuIndex.
	gPtr() *C.FaissGpuIndex
}

// faissGPUIndex concrete implementation of GPUIndex.
type faissGPUIndex struct {
	idx *C.FaissGpuIndex
	ctx *gpuContext
}

func (g *faissGPUIndex) D() int {
	return int(C.faiss_GpuIndex_d(g.idx))
}

func (g *faissGPUIndex) Add(x []float32) error {
	n := len(x) / g.D()
	reservedMem, err := g.prepareAdd(n, x)
	if err != nil {
		return err
	}
	if c := C.faiss_GpuIndex_add(
		g.idx,
		C.idx_t(n),
		(*C.float)(&x[0]),
	); c != 0 {
		g.ctx.releaseMemory(reservedMem)
		return newFaissError(ErrAddFailed, getLastError(), int(c))
	}
	return nil
}

func (g *faissGPUIndex) Train(x []float32) error {
	n := len(x) / g.D()
	if c := C.faiss_GpuIndex_train(
		g.idx,
		C.idx_t(n),
		(*C.float)(&x[0]),
	); c != 0 {
		return newFaissError(ErrTrainFailed, getLastError(), int(c))
	}
	return nil
}

func (g *faissGPUIndex) Search(x []float32, k int64) (
	[]float32, []int64, error) {
	n := len(x) / g.D()
	distances := make([]float32, int64(n)*k)
	labels := make([]int64, int64(n)*k)
	if c := C.faiss_GpuIndex_search(
		g.idx,
		C.idx_t(n),
		(*C.float)(&x[0]),
		C.idx_t(k),
		(*C.float)(&distances[0]),
		(*C.idx_t)(&labels[0]),
	); c != 0 {
		return nil, nil, newFaissError(ErrSearchFailed, getLastError(), int(c))
	}
	return distances, labels, nil
}

func (g *faissGPUIndex) Close() {
	if g.idx != nil {
		C.faiss_GpuIndex_free(g.idx)
		g.idx = nil
	}
	if g.ctx != nil {
		g.ctx.delete()
		g.ctx = nil
	}
}

func (g *faissGPUIndex) Size() uint64 {
	return reflectStaticSizeFaissGPUIndex
}

func (g *faissGPUIndex) gPtr() *C.FaissGpuIndex {
	return g.idx
}

// prepareAdd performs the necessary steps to prepare the GPU index for adding new vectors,
// including calculating the required memory for the new vectors based on their assignments
// and reserving that memory in the GPU snapshot.
// It returns the amount of memory reserved if no error occurs, or an error if the reservation fails.
func (g *faissGPUIndex) prepareAdd(n int, x []float32) (uint64, error) {
	// fallback estimate of required memory based on code size,
	// used for non-IVF indexes or if assignment fails.
	requiredMem := uint64(n) * g.ctx.codeSize
	// For IVF Indexes, we follow the following algorithm to
	// calculate the required memory for the new vectors to be added:
	// 1. Get the list assignment for each vector to be added.
	// 2. Count the number of vectors assigned to each list.
	// 3. Calculate the required memory for the new vectors based on their assignments.
	// 4. Reserve the required memory in the GPU snapshot.
	// 5. Actually reserve the memory on the GPU for the new vectors based on their assignments.
	ivfIdx := C.faiss_GpuIndexIVF_cast(g.idx)
	if ivfIdx != nil {
		// 1. list assignment for each vector.
		assign := make([]int64, n)
		if c := C.faiss_GpuIndexIVF_assign(
			ivfIdx,
			C.idx_t(n),
			(*C.float)(&x[0]),
			(*C.idx_t)(&assign[0]),
		); c != 0 {
			if err := g.ctx.reserveMemory(requiredMem); err != nil {
				return 0, err
			}
			return requiredMem, nil
		}
		// 2. find the list count for the assigned vectors.
		nlist := uint64(C.faiss_GpuIndexIVF_nlist(ivfIdx))
		listCount := make([]int64, nlist)
		for _, a := range assign {
			if a >= 0 && a < int64(nlist) {
				listCount[a]++
			}
		}
		// 3. compute required memory for the new vectors based on their assignments.
		var size C.size_t
		if c := C.faiss_GpuIndexIVF_compute_required_memory(
			ivfIdx,
			C.size_t(nlist),
			(*C.idx_t)(&listCount[0]),
			&size,
		); c != 0 {
			if err := g.ctx.reserveMemory(requiredMem); err != nil {
				return 0, err
			}
			return requiredMem, nil
		}
		// 4. reserve on our snapshot.
		requiredMem = uint64(size)
		if err := g.ctx.reserveMemory(requiredMem); err != nil {
			return 0, err
		}
		// 5. reserve on GPU.
		if c := C.faiss_GpuIndexIVF_reserve_assigned_memory(
			ivfIdx,
			C.size_t(nlist),
			(*C.idx_t)(&listCount[0]),
		); c != 0 {
			g.ctx.releaseMemory(requiredMem)
			return 0, newFaissError(ErrGPUOutOfMemory, getLastError(), int(c))
		}
		return requiredMem, nil
	}
	if err := g.ctx.reserveMemory(requiredMem); err != nil {
		return 0, err
	}
	return requiredMem, nil
}

type GPUIndexImpl struct {
	GPUIndex
}

// CloneToGPU clones cpuIndex onto a GPU and returns the resulting index.
func CloneToGPU(cpuIndex *IndexImpl) (*GPUIndexImpl, error) {
	if cpuIndex == nil {
		return nil, ErrIndexNil
	}
	// Use the load balancer to select the best GPU device's current snapshot.
	device, err := getBestGPUDevice()
	if err != nil {
		return nil, err
	}
	// Get the code size of the index to set up the context
	codeSize, err := cpuIndex.CodeSize()
	if err != nil {
		return nil, err
	}
	// Create the GPU context with the selected device.
	ctx, err := newGPUContext(device, codeSize)
	if err != nil {
		return nil, err
	}
	// Estimate the GPU memory the clone will need and reserve it against
	// the GPU snapshot before paying the cost of the actual clone. The
	// estimator uses the same cloner options we will pass to the clone
	// call, so the interleaved-layout / indices-options / coarse-quantizer
	// width all match what faiss will allocate.
	requiredMem := ctx.estimateRequiredMemory(cpuIndex)
	if err := ctx.reserveMemory(requiredMem); err != nil {
		ctx.delete()
		return nil, err
	}
	// Clone the index to GPU
	var gpuIdx *C.FaissGpuIndex
	if c := C.faiss_index_cpu_to_gpu_with_options(
		ctx.resource.cPtr(),
		C.int(device),
		cpuIndex.cPtr(),
		ctx.options.cPtr(),
		&gpuIdx,
	); c != 0 {
		ctx.delete()
		return nil, newFaissError(ErrGPUCloneFailed, getLastError(), int(c))
	}
	idx := &faissGPUIndex{
		idx: gpuIdx,
		ctx: ctx,
	}
	return &GPUIndexImpl{idx}, nil
}

func CloneToCPU(gpuIndex *GPUIndexImpl) (*IndexImpl, error) {
	if gpuIndex == nil {
		return nil, ErrIndexNil
	}
	var cpuIdx *C.FaissIndex
	if c := C.faiss_index_gpu_to_cpu(
		gpuIndex.gPtr(),
		&cpuIdx,
	); c != 0 {
		return nil, newFaissError(ErrGPUCloneFailed, getLastError(), int(c))
	}
	return &IndexImpl{&faissIndex{idx: cpuIdx}}, nil
}

// --------------------------------
// GPU Context
// --------------------------------

// gpuContext provides the context for the GPU clone operation.
type gpuContext struct {
	resource    *gpuResource
	options     *gpuClonerOptions
	device      int
	codeSize    uint64
	memReserved uint64
}

func newGPUContext(device int, codeSize uint64) (*gpuContext, error) {
	res, err := newGPUResource(device)
	if err != nil {
		return nil, err
	}
	clonerOpts, err := newGPUClonerOptions()
	if err != nil {
		res.delete()
		return nil, err
	}
	return &gpuContext{
		resource: res,
		options:  clonerOpts,
		device:   device,
		codeSize: codeSize,
	}, nil
}

func (c *gpuContext) delete() {
	if c.options != nil {
		c.options.delete()
		c.options = nil
	}
	if c.resource != nil {
		c.resource.delete()
		c.resource = nil
	}
	if c.memReserved > 0 {
		c.releaseMemory(c.memReserved)
		c.memReserved = 0
	}
}

func (c *gpuContext) reserveMemory(size uint64) error {
	snapshot := snapshotStore.snapshotForDevice(c.device)
	if err := snapshot.reserveMemory(size); err != nil {
		return err
	}
	c.memReserved += size
	return nil
}

// estimateRequiredMemory predicts the GPU memory the clone of cpuIndex
// will consume on this context's device, given the cloner options. It
// accounts for the interleaved storage layout that faiss uses on the GPU.
// Falls back to a code_size * ntotal estimate if the C API does not
// recognize the index type. Always accounts for the cloned index's temporary memory.
func (c *gpuContext) estimateRequiredMemory(cpuIndex *IndexImpl) uint64 {
	rv := uint64(defaultGPUTempMemorySize)
	numVecs := uint64(cpuIndex.Ntotal())
	if numVecs > 0 {
		var size C.size_t
		if rc := C.faiss_GpuMemoryEstimate_for_cpu_index(
			cpuIndex.cPtr(),
			C.int(c.device),
			c.options.cPtr(),
			&size,
		); rc != 0 {
			rv += numVecs * c.codeSize
		} else {
			rv += uint64(size)
		}
	}
	return rv
}

func (c *gpuContext) releaseMemory(size uint64) {
	snapshot := snapshotStore.snapshotForDevice(c.device)
	snapshot.releaseMemory(size)
	c.memReserved -= size
}

// gpuResource wraps a FAISS standard GPU resources handle.
type gpuResource struct {
	res *C.FaissStandardGpuResources
}

// newGPUResource creates a new GPU resource handle for the given device,
// configuring it with the shared memory overflow pool, per-index temporary
// buffers, and pinned memory for CPU-GPU transfers.
func newGPUResource(device int) (*gpuResource, error) {
	var res *C.FaissStandardGpuResources
	if c := C.faiss_StandardGpuResources_new(&res); c != 0 {
		return nil, newFaissError(ErrGPUContextFailed, getLastError(), int(c))
	}
	pool := memoryPoolStore.poolForDevice(device)
	if pool != nil {
		if c := C.faiss_StandardGpuResources_setTempMemoryOverflowPool(res, pool.cPtr()); c != 0 {
			C.faiss_StandardGpuResources_free(res)
			return nil, newFaissError(ErrGPUContextFailed, getLastError(), int(c))
		}
	}
	if c := C.faiss_StandardGpuResources_setTempMemory(res, C.size_t(defaultGPUTempMemorySize)); c != 0 {
		C.faiss_StandardGpuResources_free(res)
		return nil, newFaissError(ErrGPUContextFailed, getLastError(), int(c))
	}
	if c := C.faiss_StandardGpuResources_setPinnedMemory(res, C.size_t(defaultGPUPinnedMemory)); c != 0 {
		C.faiss_StandardGpuResources_free(res)
		return nil, newFaissError(ErrGPUContextFailed, getLastError(), int(c))
	}
	return &gpuResource{res: res}, nil
}

func (r *gpuResource) cPtr() *C.FaissStandardGpuResources {
	return r.res
}

func (r *gpuResource) delete() {
	if r.res != nil {
		C.faiss_StandardGpuResources_free(r.res)
		r.res = nil
	}
}

// gpuClonerOptions wraps a FAISS GPU cloner options handle.
type gpuClonerOptions struct {
	opts *C.FaissGpuClonerOptions
}

func newGPUClonerOptions() (*gpuClonerOptions, error) {
	var opts *C.FaissGpuClonerOptions
	if c := C.faiss_GpuClonerOptions_new(&opts); c != 0 {
		return nil, newFaissError(ErrGPUContextFailed, getLastError(), int(c))
	}
	C.faiss_GpuClonerOptions_set_memorySpace(opts, C.int(defaultGPUMemoryMode))
	return &gpuClonerOptions{opts: opts}, nil
}

func (c *gpuClonerOptions) cPtr() *C.FaissGpuClonerOptions {
	return c.opts
}

func (c *gpuClonerOptions) delete() {
	if c.opts != nil {
		C.faiss_GpuClonerOptions_free(c.opts)
		c.opts = nil
	}
}

// --------------------------------
// GPU Memory Pool Store
// --------------------------------

// gpuMemoryPoolStore indicates a per-device reusable pool of GPU memory
// that grows dynamically up to a threshold.
type gpuMemoryPoolStore struct {
	devicePool []*gpuMemoryPool
}

func newGPUMemoryPoolStore() *gpuMemoryPoolStore {
	pool := make([]*gpuMemoryPool, gpuCount)
	for i := 0; i < gpuCount; i++ {
		device := snapshotStore.snapshotForDevice(i)
		pq := device.poolQuota()
		if pq > 0 {
			pool[i] = newGPUMemoryPool(i, pq)
		}
	}
	return &gpuMemoryPoolStore{devicePool: pool}
}

func (mps *gpuMemoryPoolStore) poolForDevice(device int) *gpuMemoryPool {
	return mps.devicePool[device]
}

// gpuMemoryPool wraps a FAISS GPU memory pool handle.
type gpuMemoryPool struct {
	pool *C.FaissGpuMemoryPool
}

func newGPUMemoryPool(device int, cap uint64) *gpuMemoryPool {
	var pool *C.FaissGpuMemoryPool
	if c := C.faiss_GpuMemoryPool_new(
		C.int(device),
		C.size_t(cap),
		&pool,
	); c == 0 {
		return &gpuMemoryPool{pool: pool}
	}
	return nil
}

func (mp *gpuMemoryPool) cPtr() *C.FaissGpuMemoryPool {
	return mp.pool
}

func (mp *gpuMemoryPool) delete() {
	if mp.pool != nil {
		C.faiss_GpuMemoryPool_free(mp.pool)
		mp.pool = nil
	}
}
