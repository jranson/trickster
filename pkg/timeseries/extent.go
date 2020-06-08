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

package timeseries

import (
	"fmt"
	"time"
)

// Extent describes the start and end times for a given range of data
type Extent struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	LastUsed time.Time `json:"-"`
}

// Includes returns true if the Extent includes the provided Time
func (e *Extent) Includes(t time.Time) bool {
	return !t.Before(e.Start) && !t.After(e.End)
}

// StartsAt returns true if t is equal to the Extent's start time
func (e *Extent) StartsAt(t time.Time) bool {
	return t.Equal(e.Start)
}

// EndsAt returns true if t is equal to the Extent's end time
func (e *Extent) EndsAt(t time.Time) bool {
	return t.Equal(e.End)
}

// After returns true if the range of the Extent is completely after the provided time
func (e *Extent) After(t time.Time) bool {
	return t.Before(e.Start)
}

// After returns true if the range of the Extent is completely after the provided time
func (e Extent) String() string {
	return fmt.Sprintf("%d-%d", e.Start.Unix(), e.End.Unix())
}

// CalculateDeltas provides a list of extents that are not in a cached timeseries,
// when provided a list of extents that are cached.
func CalculateDeltas(have ExtentList, want Extent, step time.Duration) ExtentList {
	if len(have) == 0 {
		return ExtentList{want}
	}
	misCap := want.End.Sub(want.Start) / step
	if misCap < 0 {
		misCap = 0
	}
	misses := make([]time.Time, 0, misCap)
	for i := want.Start; !want.End.Before(i); i = i.Add(step) {
		found := false
		for j := range have {
			if j == 0 && i.Before(have[j].Start) {
				// our earliest datapoint in cache is after the first point the user wants
				break
			}
			if i.Equal(have[j].Start) || i.Equal(have[j].End) || (i.After(have[j].Start) && have[j].End.After(i)) {
				found = true
				break
			}
		}
		if !found {
			misses = append(misses, i)
		}
	}
	// Find the fill and gap ranges
	ins := ExtentList{}
	var inStart = time.Time{}
	l := len(misses)
	for i := range misses {
		if inStart.IsZero() {
			inStart = misses[i]
		}
		if i+1 == l || !misses[i+1].Equal(misses[i].Add(step)) {
			ins = append(ins, Extent{Start: inStart, End: misses[i]})
			inStart = time.Time{}
		}
	}
	return ins
}
