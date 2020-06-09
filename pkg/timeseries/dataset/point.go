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

//go:generate msgp

package dataset

import "sync/atomic"

// Epoch represents an Epoch timestamp in Nanoseconds and has possible values
// between 1970/1/1 and 2262/4/12
type Epoch uint64

// Epochs is a slice of type Epoch
type Epochs []Epoch

// Point represents a timeseries data point
type Point struct {
	Epoch  Epoch         `msg:"epoch"`
	Size   int           `msg:"size"`
	Values []interface{} `msg:"values"`
}

// Points is a slice of type *Point
type Points []Point

// Clone returns a perfect copy of the Point
func (p *Point) Clone() Point {
	clone := Point{
		Epoch: p.Epoch,
		Size:  p.Size,
	}
	if p.Values != nil {
		clone.Values = make([]interface{}, len(p.Values))
		copy(clone.Values, p.Values)
	}
	return clone
}

// Size returns the memory utilization of the Points in bytes
func (p Points) Size() int64 {
	var c int64
	for _, pt := range p {
		go func(s int64) {
			atomic.AddInt64(&c, s)
		}(int64(pt.Size))
	}
	return c
}

// Clone returns a perfect copy of the Points
func (p Points) Clone() Points {
	clone := make(Points, len(p))
	for i, pt := range p {
		clone[i] = pt.Clone()
	}
	return clone
}

// Len returns the length of a slice of time series data points
func (p Points) Len() int {
	return len(p)
}

// Less returns true if i comes before j
func (p Points) Less(i, j int) bool {
	return p[i].Epoch < (p[j].Epoch)
}

// Swap modifies a slice of time series data points by swapping the values in indexes i and j
func (p Points) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
