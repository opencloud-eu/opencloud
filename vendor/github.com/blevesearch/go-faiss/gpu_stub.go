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

//go:build !gpu
// +build !gpu

package faiss

// GPUIndexImpl is an opaque type when not built with GPU support.
type GPUIndexImpl struct{}

func (g *GPUIndexImpl) Train(x []float32) error { return ErrNoUsableGPUDevices }
func (g *GPUIndexImpl) Add(x []float32) error   { return ErrNoUsableGPUDevices }
func (g *GPUIndexImpl) Search(x []float32, k int64) ([]float32, []int64, error) {
	return nil, nil, ErrNoUsableGPUDevices
}
func (g *GPUIndexImpl) Close()       {}
func (g *GPUIndexImpl) Size() uint64 { return 0 }

// CloneToGPU is not available without the gpu build tag.
func CloneToGPU(_ *IndexImpl) (*GPUIndexImpl, error) {
	return nil, ErrNoUsableGPUDevices
}

// CloneToCPU is not available without the gpu build tag.
func CloneToCPU(_ *GPUIndexImpl) (*IndexImpl, error) {
	return nil, ErrNoUsableGPUDevices
}
