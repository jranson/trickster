/*
 * Copyright 2018 Comcast Cable Communications Management, LLC
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

// Package timeseries defines the interface for managing time seres objects
// and provides time range manipulation capabilities
package timeseries

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tricksterproxy/trickster/pkg/util/fnv"
)

type Hash uint64
type Tags map[string]string

// Hash returns a hash of tag key/value pairs.
func (t Tags) Hash() Hash {
	h := fnv.NewInlineFNV64a()
	keys := t.Keys()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(t[k]))
	}
	return Hash(h.Sum64())
}

func (t Tags) String() string {
	if len(t) == 0 {
		return ""
	}
	pairs := make(sort.StringSlice, len(t))
	var i int
	for k, v := range t {
		pairs[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	sort.Sort(pairs)
	return strings.Join(pairs, ";")
}

func (t Tags) Keys() []string {
	if len(t) == 0 {
		return nil
	}
	keys := make(sort.StringSlice, len(t))
	var i int
	for k := range t {
		keys[i] = k
		i++
	}
	sort.Sort(keys)
	return keys
}
