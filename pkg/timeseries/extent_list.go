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

//go:generate msgp

package timeseries

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/util/sets"
)

// ExtentList is a type of []Extent used for sorting the slice
type ExtentList []Extent

// String returns a string representation of the extentlist
// in the format startEpochSec1-endEpochSec1;startEpochSec2-endEpochSec2
func (el ExtentList) String() string {
	if len(el) == 0 {
		return ""
	}
	lines := make([]string, len(el))
	for i, e := range el {
		lines[i] = e.String()
	}
	return strings.Join(lines, ",")
}

// Encompasses returns true if the provided extent is contained
// completely within boundaries of the subject ExtentList
func (el ExtentList) Encompasses(e Extent) bool {
	x := len(el)
	if x == 0 {
		return false
	}
	return (!e.Start.Before(el[0].Start)) &&
		(!e.End.After(el[x-1].End))
}

// EncompassedBy returns true if the provided extent completely
// surrounds the boundaries of the subject ExtentList
func (el ExtentList) EncompassedBy(e Extent) bool {
	x := len(el)
	if x == 0 {
		return false
	}
	return (!el[0].Start.Before(e.Start)) &&
		(!el[x-1].End.After(e.End))
}

// OutsideOf returns true if the provided extent falls completely
// outside of the boundaries of the subject extent list
func (el ExtentList) OutsideOf(e Extent) bool {
	x := len(el)
	if x == 0 {
		return true
	}
	return e.After(el[x-1].End) || el[0].After(e.End)
}

// Crop reduces the ExtentList to the boundaries defined by the provided Extent
func (el ExtentList) Crop(ex Extent) ExtentList {
	if len(el) == 0 {
		return ExtentList{}
	}
	out := make(ExtentList, len(el))
	var k int
	for _, e := range el {
		if e.End.Before(ex.Start) || e.Start.After(ex.End) {
			continue
		}
		start := e.Start
		end := e.End
		if ex.Start.After(start) && ex.Start.Before(end) {
			start = ex.Start
		} else if ex.Start.Equal(end) {
			start = ex.Start
			end = ex.Start
		}
		if ex.End.Before(end) && ex.End.After(start) {
			end = ex.End
		} else if ex.End.Equal(start) {
			start = ex.End
			end = ex.End
		}
		out[k] = Extent{Start: start, End: end, LastUsed: e.LastUsed}
		k++
	}
	return out[:k]
}

// Compress sorts an ExtentList and merges time-adjacent Extents so that the
// total extent of data is accurately represented in as few Extents as possible
func (el ExtentList) Compress(step time.Duration) ExtentList {
	if len(el) == 0 {
		return ExtentList{}
	}
	sort.Sort(el)
	out := make(ExtentList, len(el))
	var k int
	current := el[0]
	for i := 1; i < len(el); i++ {
		next := el[i]
		if !next.Start.After(current.End.Add(step)) &&
			current.LastUsed.Equal(next.LastUsed) {
			if next.End.After(current.End) {
				current.End = next.End
			}
			continue
		}
		out[k] = current
		k++
		current = next
	}
	out[k] = current
	k++
	return out[:k]
}

// Splice breaks apart extents in the list into smaller, contiguous extents, based on the provided
// splice sizing options, and returns the resulting spliced list.
// Splice assumes el is Compressed (e.g., Compress() was just ran or would be innefectual if ran)
func (el ExtentList) Splice(step, maxRange, spliceStep time.Duration, maxPoints int) ExtentList {
	if len(el) == 0 {
		if el == nil {
			return nil
		}
		return ExtentList{}
	}
	if maxPoints == 0 {
		if spliceStep == 0 {
			return el.spliceByTime(step, maxRange)
		}
		return el.spliceByTimeAligned(step, maxRange, spliceStep)
	}
	return el.spliceByPoints(step, maxPoints)
}

// spliceByTimeAligned handles extents that must be spliced at a precise cadence divisible by
// the epoch. step indicates the timeseries step, and spliceStep indicates the splicing interval
// for aligning to the epoch. maxRange is the maximum width of a splice, and must be
// a multiple of spliceStep or the results will be unpredictable
func (el ExtentList) spliceByTimeAligned(step, maxRange, spliceStep time.Duration) ExtentList {
	if step == 0 || maxRange == 0 || spliceStep == 0 {
		return el.Clone()
	}
	// reserve enough capacity that 50% of extents could be spliced without having to re-allocate
	out := make(ExtentList, 0, len(el)+(len(el)/2))
	for _, e := range el {
		// if the size of the extent is smaller than the max splice size, and
		// the extent doesn't cross spliceSteps, pass through and continue
		if e.End.Sub(e.Start) <= maxRange &&
			e.End.Truncate(spliceStep).Equal(e.Start.Truncate(spliceStep)) {
			out = append(out, e)
			continue
		}
		// otherwise the extent must be spliced.
		t1 := e.Start.Truncate(spliceStep)
		// if t1 == e.Start, then the extent falls perfectly on the spliceStep. If it does not,
		// e will be spliced on the first spliceStep interval after e.Start
		//
		// this handles the left-side splice, when required
		if t1.Before(e.Start) {
			t1 = t1.Add(spliceStep)
			// this calculates the splice and adds it to the output
			t2 := t1.Truncate(step)
			// This ensures that if spliceStep is not a multiple of step, then the left-side
			// splice's end time retreats to the step just prior to the start of the next splice
			if !t2.Before(t1) {
				t2 = t2.Add(-step)
			}
			out = append(out, Extent{Start: e.Start, End: t2})
			// this advances e.Start to the first timeseries step on or after the new left boundary
			e.Start = t2.Add(step)
		}
		// now that left-side splicing is done, this splices the rest of e on the epoch.
		//
		// this re-checks if e is shallower than maxRange, since it might've just been reduced
		if e.End.Sub(e.Start) <= maxRange &&
			e.End.Truncate(spliceStep).Equal(e.Start.Truncate(spliceStep)) {
			out = append(out, e)
			continue
		}
		// otherwise, we still need to further splice e
		for i := e.Start; !i.After(e.End); {
			// this creates a new splice to add to the output
			e2 := Extent{Start: i, End: i.Add(maxRange - step).Truncate(step)}
			if e2.End.Before(e2.Start) {
				e2.End = e2.Start
			}
			// the final iteration may be partial/smaller than the splice size, so this clamps it
			if e2.End.After(e.End) {
				e2.End = e.End
			}
			out = append(out, e2)
			// this advances i forward a step beyond the current splice end, to start the next one
			i = e2.End.Add(step)
		}
	}
	return out
}

// spliceByTime splices extents that are not aligned to any particular epoch cadence
func (el ExtentList) spliceByTime(step, maxRange time.Duration) ExtentList {
	if step == 0 || maxRange == 0 {
		return el.Clone()
	}
	// reserve enough capacity that 50% of extents could be spliced without having to re-allocate
	out := make(ExtentList, 0, len(el)+(len(el)/2))
	for _, e := range el {
		// if the size of the extent is smaller than the max splice size, pass through and continue
		if e.End.Sub(e.Start) <= maxRange {
			out = append(out, e)
			continue
		}
		// otherwise, we still need to further splice e
		for i := e.Start; !i.After(e.End); {
			// this creates a new splice to add to the output
			e2 := Extent{Start: i, End: i.Add(maxRange - step).Truncate(step)}
			// the final iteration may be partial/smaller than the splice size, so this clamps it
			if e2.End.Before(e2.Start) {
				e2.End = e2.Start
			}
			if e2.End.After(e.End) {
				e2.End = e.End
			}
			out = append(out, e2)
			// this advances i forward a step beyond the current splice end, to start the next one
			i = e2.End.Add(step)
		}
	}
	return out
}

// spliceByTime splices by a given number of contiguous timestamps (points) per splice
func (el ExtentList) spliceByPoints(step time.Duration, maxPoints int) ExtentList {
	if maxPoints == 0 || step == 0 {
		return el.Clone()
	}
	out := make(ExtentList, 0, len(el)+(len(el)/2))
	for _, e := range el {
		// this determines the number of timestamps in the extent
		s := int(e.End.Sub(e.Start) / step)
		// if the splice size is larger than the timestamp count, the extent can be passed through
		if maxPoints > s || e.Start.IsZero() || e.End.IsZero() {
			out = append(out, e)
			continue
		}
		// otherwise, this extent is larger than the splice size, so it needs to be spliced
		for i := e.Start; !i.After(e.End); {
			// this creates a new splice to add to the output
			e2 := Extent{Start: i, End: i.Add(step * time.Duration(maxPoints-1))}
			// the final iteration may be partial/smaller than the splice size, so this clamps it
			if e2.End.Before(e2.Start) {
				e2.End = e2.Start
			}
			if e2.End.After(e.End) {
				e2.End = e.End
			}
			out = append(out, e2)
			// this advances i forward a step beyond the current splice end, to start the next one
			i = e2.End.Add(step)
		}
	}
	return out
}

// Len returns the length of a slice of type ExtentList
func (el ExtentList) Len() int {
	return len(el)
}

// Less returns true if element i in the ExtentList comes before j
func (el ExtentList) Less(i, j int) bool {
	return el[i].Start.Before(el[j].Start)
}

// Swap modifies an ExtentList by swapping the values in indexes i and j
func (el ExtentList) Swap(i, j int) {
	el[i], el[j] = el[j], el[i]
}

// Clone returns a true copy of the ExtentList
func (el ExtentList) Clone() ExtentList {
	c := make(ExtentList, len(el))
	// this is safe because all fields in an Extent are by value
	copy(c, el)
	return c
}

// CloneRange returns a perfect copy of the ExtentList, cloning only the
// Extents in the provided index range (upper-bound exclusive)
func (el ExtentList) CloneRange(start, end int) ExtentList {
	if end < start || start < 0 || end < 0 {
		return nil
	}
	size := end - start
	if size > len(el) {
		return nil
	}
	c := make(ExtentList, size)
	j := start
	for i := 0; i < size; i++ {
		c[i].Start = el[j].Start
		c[i].End = el[j].End
		c[i].LastUsed = el[j].LastUsed
		j++
	}
	return c
}

// Equal returns true if the provided extent list is identical to the subject list
func (el ExtentList) Equal(el2 ExtentList) bool {
	if el2 == nil {
		return false
	}

	l := len(el)
	l2 := len(el2)
	if l != l2 {
		return false
	}

	for i := range el {
		if el2[i] != el[i] {
			return false
		}
	}
	return true
}

// Remove removes the provided extent list ranges from the subject extent list
func (el ExtentList) Remove(r ExtentList, step time.Duration) ExtentList {
	if len(r) == 0 {
		return el
	}
	if len(el) == 0 {
		return r
	}

	splices := sets.NewIntSet()
	spliceIns := make(map[int]Extent)
	c := el.Clone()
	for _, rem := range r {
		for i, ex := range c {

			if rem.End.Before(ex.Start) || rem.Start.After(ex.End) {
				// removal range is not relevant to this extent
				continue
			}

			if rem.StartsAtOrBefore(ex.Start) && rem.EndsAtOrAfter(ex.End) {
				// removal range end is >= extent.End, and start is <= extent.Start
				// so the entire range will be spliced out of the list
				splices.Add(i)
				continue
			}

			// the removal is fully inside of the extent, it must be split into two
			if rem.Start.After(ex.Start) && rem.End.Before(ex.End) {
				// the first piece will be inserted back in later
				spliceIns[i] = Extent{Start: ex.Start, End: rem.Start.Add(-step)}
				// the existing piece will be adjusted in place
				c[i].Start = rem.End.Add(step)
				continue
			}

			// The removal is attached to only one side of the extent, so the
			// boundaries can be adjusted
			if rem.Start.After(ex.Start) {
				c[i].End = rem.Start.Add(-step)
			} else if rem.End.Before(ex.End) {
				c[i].Start = rem.End.Add(step)
			}

		}
	}

	// if the clone is final, return it now
	if len(splices) == 0 && len(spliceIns) == 0 {
		return c
	}

	// otherwise, make a version of the does not include the splice out indexes
	// and includes any splice-in indexes
	r = make(ExtentList, len(c)*2)
	var k int
	for i, ex := range c {
		if ex2, ok := spliceIns[i]; ok {
			r[k] = ex2
			k++
		}
		if _, ok := splices[i]; !ok {
			r[k] = ex
			k++
		}
	}
	return r[:k]
}

// TimestampCount returns the calculated number of timestamps based on the extents
// in the list and the provided duration
func (el ExtentList) TimestampCount(d time.Duration) int64 {
	var c int64
	for i := range el {
		if el[i].Start.IsZero() || el[i].End.IsZero() {
			continue
		}
		c += ((el[i].End.UnixNano() - el[i].Start.UnixNano()) / d.Nanoseconds()) + 1
	}
	return c
}

// CalculateDeltas provides a list of extents that are not in el based on the
// needed extent. step is used to determine which absolute timestamps in need
// will be checked in el.
func (el ExtentList) CalculateDeltas(need Extent, step time.Duration) ExtentList {
	if step <= 0 || !need.End.After(need.Start) {
		return ExtentList{}
	}
	if len(el) == 0 {
		return ExtentList{need}
	}
	sort.Sort(el)
	out := make(ExtentList, len(el)+1)
	var missStart time.Time
	var j, k int
	for ts := need.Start; !ts.After(need.End); ts = ts.Add(step) {
		// this advances to j to the point in el where ts would be if it were
		// present in el (whether it currently is or not)
		for j < len(el) && ts.After(el[j].End) {
			j++
		}
		inExisting := false
		if j < len(el) {
			s := el[j].Start
			e := el[j].End
			inExisting = !ts.Before(s) && !ts.After(e)
		}
		if !inExisting {
			if missStart.IsZero() {
				missStart = ts
			}
		} else if !missStart.IsZero() {
			out[k] = Extent{
				Start: missStart,
				End:   ts.Add(-step),
			}
			k++
			missStart = time.Time{}
		}
	}
	if !missStart.IsZero() {
		out[k] = Extent{
			Start: missStart,
			End:   need.End,
		}
		k++
	}
	return out[:k]
}

// Size returns the approximate memory utilization in bytes of the timeseries
func (el ExtentList) Size() int {
	return len(el) * 72
}

// ExtentListLRU is a type of []Extent used for sorting the slice by LRU
type ExtentListLRU []Extent

// Len returns the length of an slice of type ExtentListLRU
func (el ExtentListLRU) Len() int {
	return len(el)
}

// Less returns true if element i in the ExtentListLRU comes before j
func (el ExtentListLRU) Less(i, j int) bool {
	return el[i].LastUsed.Before(el[j].LastUsed)
}

// Swap modifies an ExtentListLRU by swapping the values in indexes i and j
func (el ExtentListLRU) Swap(i, j int) {
	el[i], el[j] = el[j], el[i]
}

// Clone returns a true copy of the ExtentListLRU
func (el ExtentListLRU) Clone() ExtentListLRU {
	c := make(ExtentListLRU, len(el))
	for i := range el {
		c[i].Start = el[i].Start
		c[i].End = el[i].End
		c[i].LastUsed = el[i].LastUsed
	}
	return c
}

func (el ExtentListLRU) String() string {
	if len(el) == 0 {
		return ""
	}
	lines := make([]string, 0, len(el))
	for _, e := range el {
		lines = append(lines, fmt.Sprintf("%d-%d:%d", e.Start.Unix(), e.End.Unix(), e.LastUsed.Unix()))
	}
	return strings.Join(lines, ",")
}

// UpdateLastUsed updates the ExtentListLRU's LastUsed field for the provided extent.
// The step is required in order to properly split extents.
func (el ExtentListLRU) UpdateLastUsed(lur Extent, step time.Duration) ExtentListLRU {

	if el == nil {
		return nil
	}

	if len(el) == 0 {
		return ExtentListLRU{}
	}

	now := time.Now().Truncate(time.Second)
	el2 := make(ExtentList, 0, len(el))

	for _, x := range el {

		// This case captures when extent x is sandwiched between the
		// extents in the list containing lur.Start and lur.End
		// So we'll mark its Last Used and move on without splitting.
		if !lur.Start.After(x.Start) && !lur.End.Before(x.End) {
			x.LastUsed = now
			el2 = append(el2, x)
			continue
		}

		// The LastUsed extent is before or after this entire extent
		// so we don't do anything
		if x.Start.After(lur.End) || x.End.Before(lur.Start) {
			el2 = append(el2, x)
			continue
		}

		// The Last Used Range starts in this extent, but not on the starting edge
		// So we'll break it up into two pieces on that start point
		if lur.Start.After(x.Start) && !lur.Start.After(x.End) {
			// v will serve as the left portion of x that we broke off
			// it is outside of the Last Used Range so LU is untouched
			v := Extent{Start: x.Start, End: lur.Start.Add(-step), LastUsed: x.LastUsed}
			x.Start = lur.Start
			el2 = append(el2, v)

			// The right portion may be fully enclosed by the LUR, if so
			// go ahead an mark the usage time, append to our new ExtentList and move on
			if !lur.End.Before(x.End) {
				x.LastUsed = now
				el2 = append(el2, x)
				continue
			}
		}

		// If we got here, the LUR covers a left portion of this extent, break it up and append
		if lur.End.Before(x.End) && !lur.End.Before(x.Start) {
			y := Extent{Start: lur.End.Add(step), End: x.End, LastUsed: x.LastUsed}
			x.End = lur.End
			x.LastUsed = now
			el2 = append(el2, x, y)
			continue
		}
	}
	return ExtentListLRU(el2.Compress(step))
}
