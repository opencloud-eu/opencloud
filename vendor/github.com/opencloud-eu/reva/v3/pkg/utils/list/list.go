// Copyright 2018-2023 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package list

// Map returns a list constructed by appling a function f
// to all items in the list l.
func Map[T, V any](l []T, f func(T) V) []V {
	m := make([]V, 0, len(l))
	for _, e := range l {
		m = append(m, f(e))
	}
	return m
}

// Remove removes the element in position i from the list.
// It does not preserve the order of the original slice.
func Remove[T any](l []T, i int) []T {
	l[i] = l[len(l)-1]
	return l[:len(l)-1]
}

// TakeFirst returns the first elemen, if any, that satisfies
// the predicate p.
func TakeFirst[T any](l []T, p func(T) bool) (T, bool) {
	for _, e := range l {
		if p(e) {
			return e, true
		}
	}
	var z T
	return z, false
}

// ToMap returns a map from l where the keys are obtainined applying
// the func k to the elements of l.
func ToMap[K comparable, T any](l []T, k func(T) K) map[K]T {
	m := make(map[K]T, len(l))
	for _, e := range l {
		m[k(e)] = e
	}
	return m
}
