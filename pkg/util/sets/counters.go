/*
 * Copyright 2018 The Trickster Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sets

import "github.com/trickstercache/trickster/v2/pkg/util/numbers"

// A CounterSet is map with keys of type T and values representing a count.
// The Increment() function is used to change the count value.
// A CounterSet it is not safe for concurrency and its order is not guaranteed.
type CounterSet[T comparable] map[T]int

// Increment increments the CounterSet value for 'key' by 'cnt' and returns the
// new CounterSet value. The bool returns true only if the key was pre-existing.
// Negative cnt values are ok. Increment is not safe for concurrency.
func (s CounterSet[T]) Increment(key T, cnt int) (int, bool) {
	if i, ok := s[key]; ok {
		if j, ok := numbers.SafeAdd(i, cnt); ok {
			s[key] = j
			return j, true
		}
		return i, true
	}
	s[key] = cnt
	return cnt, false
}

// Count returns the value for the provided key, or -1 if the key doesn't exist.
func (s CounterSet[T]) Count(key T) (int, bool) {
	if i, ok := s[key]; ok {
		return i, true
	}
	return -1, false
}

// CounterSet with string keys

// NewStringCounterSet returns a new StringCounterSet
func NewStringCounterSet() CounterSet[string] {
	return make(CounterSet[string])
}

// NewStringCounterSetCap returns a new StringCounterSet with a capacity set.
func NewStringCounterSetCap(capacity int) CounterSet[string] {
	return make(CounterSet[string], capacity)
}
