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

import (
	"io"
	"sort"
	"sync"
	"time"

	"github.com/tricksterproxy/trickster/pkg/timeseries"
)

// DataSet is the Common Time Series Format that Trickster uses to
// accelerate most of its supported TSDB backends
// DataSet conforms to the Timeseries interface
type DataSet struct {
	// Status is the optional status indicator for the DataSet
	Status string `msg:"status"`
	// ExtentList is the list of Extents (time ranges) represented in the Results
	ExtentList timeseries.ExtentList `msg:"extent_list"`
	// // Timestamps is map of all timestamps in the DataSet
	// Timestamps map[Epoch]bool `msg:"timestamps"`
	// Results is the list of type Result. Each Result represents information about a
	// different statement in the source query for this DataSet
	Results []Result `msg:"results"`
	// UpdateLock is used to synchronize updates to the DataSet
	UpdateLock sync.Mutex `msg:"-"`
	// Error is a container for any DataSet-level Errors
	Error string `msg:"error"`
	// TimeRangeQuery is the trq associated with the Timeseries
	TimeRangeQuery *timeseries.TimeRangeQuery `msg:"trq"`
	// Sorter is the DataSet's Sort function, which defaults to DefaultSort
	Sorter func() `msg:"-"`
	// Merger is the DataSet's Merge function, which defaults to DefaultMerge
	Merger func(sortSeries bool, ts ...timeseries.Timeseries) `msg:"-"`
	// SizeCropper is the DataSet's CropToSize function, whcih defauls to DefautlSizeCropper
	SizeCropper func(int, time.Time, timeseries.Extent) `msg:"-"`
	// RangeCropper is the DataSet's CropToRange function, whcih defauls to DefautlRangeCropper
	RangeCropper func(timeseries.Extent) `msg:"-"`
	// OutputFormat is bit representing the desired output format of the DataSet; it's actual
	// implementation of values is fully federated to the underlying Time Series origin package
	OutputFormat byte `msg:"output_format"`
}

// Marshaler is a function that serializes the provided DataSet into a byte slice
type Marshaler func(*DataSet, io.Writer) error

// Clone returns a new, perfect copy of the DataSet
func (ds *DataSet) Clone() timeseries.Timeseries {
	ds.UpdateLock.Lock()
	defer ds.UpdateLock.Unlock()
	clone := &DataSet{
		Error:        ds.Error,
		Sorter:       ds.Sorter,
		Merger:       ds.Merger,
		SizeCropper:  ds.SizeCropper,
		RangeCropper: ds.RangeCropper,
		OutputFormat: ds.OutputFormat,
		ExtentList:   make(timeseries.ExtentList, len(ds.ExtentList)),
		Results:      make([]Result, len(ds.Results)),
	}
	if ds.TimeRangeQuery != nil {
		clone.TimeRangeQuery = ds.TimeRangeQuery.Clone()
	}
	copy(clone.ExtentList, ds.ExtentList)
	for i := range ds.Results {
		clone.Results[i] = ds.Results[i].Clone()
	}
	return clone
}

// Merge merges the provided Timeseries list into the base DataSet
// (in the order provided) and optionally sorts the merged DataSet
// This implementation ignores any Timeseries that are not of type *DataSet
func (ds *DataSet) Merge(sortSeries bool, collection ...timeseries.Timeseries) {
	if ds.Merger != nil {
		ds.Merger(sortSeries, collection...)
		return
	}
	ds.DefaultMerger(sortSeries, collection...)
}

// DefaultMerger is the default Merger function
func (ds *DataSet) DefaultMerger(sortSeries bool, collection ...timeseries.Timeseries) {
	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}

	ds.UpdateLock.Lock()
	defer ds.UpdateLock.Unlock()

	orderMarkers := make([]Hashes, 0)

	sl := make(SeriesLookup)
	for i := range ds.Results {
		for _, s := range ds.Results[i].SeriesList {
			sl[s.Header.CalculateHash()] = s
		}
	}

	for _, ts := range collection {
		if ts == nil {
			continue
		}
		ds2, ok := ts.(*DataSet)
		if !ok {
			continue
		}
		om := make([]Hashes, 0, ds2.SeriesCount())
		for ri := range ds2.Results {
			if ri >= len(ds.Results) {
				mtx.Lock()
				ds.Results = append(ds.Results, ds2.Results[ri:]...)
				mtx.Unlock()
				break
			}
			for i := range ds2.Results[ri].SeriesList {
				if ds2.Results[ri].SeriesList[i] == nil {
					continue
				}
				wg.Add(1)
				go func(s *Series, rj int) {
					mtx.Lock()
					var es *Series
					h := s.Header.CalculateHash()
					if es, ok = sl[h]; !ok || es == nil {
						om = append(om, ds2.Results[rj].Hashes())
						sl[h] = s
						ds.Results[rj].SeriesList = append(ds.Results[rj].SeriesList, s)
						mtx.Unlock()
						wg.Done()
						return
					}
					mtx.Unlock()
					es.Points = append(es.Points, s.Points...)
					// This will sort and dupe kill the list of points, keeping the newest version
					if sortSeries {
						sort.Sort(es.Points)
						n := len(es.Points)
						if n <= 1 {
							x := make(Points, n)
							copy(x, es.Points[0:n])
							es.Points = x
						} else {
							j := 1
							for i := 1; i < n; i++ {
								if es.Points[i].Epoch != es.Points[i-1].Epoch {
									es.Points[j] = es.Points[i]
									j++
								}
							}
							x := make(Points, len(es.Points[0:j]))
							copy(x, es.Points[0:j])
							es.Points = x
						}
					}
					es.PointSize = es.Points.Size()
					wg.Done()
				}(ds2.Results[ri].SeriesList[i], ri)
			}
			wg.Wait()
			ds.ExtentList = append(ds.ExtentList, ds2.ExtentList...)
			orderMarkers = append(orderMarkers, om...)
		}
	}
	ds.ExtentList = ds.ExtentList.Compress(ds.Step())
	// other housekeeping
	// e.g., if len(orderMarkers) > 0, we potentially got reordering to do.
	// Figure out if this ^^^ needs to be handled by result
}

// CropToSize reduces the number of elements in the Timeseries to the provided count, by evicting elements
// using a least-recently-used methodology. The time parameter limits the upper extent to the provided time,
// in order to support backfill tolerance
func (ds *DataSet) CropToSize(sz int, t time.Time, lur timeseries.Extent) {
	if ds.SizeCropper != nil {
		ds.SizeCropper(sz, t, lur)
		return
	}
	ds.DefaultSizeCropper(sz, t, lur)
}

// DefaultSizeCropper is the default SizeCropper Function
func (ds *DataSet) DefaultSizeCropper(sz int, t time.Time, lur timeseries.Extent) {
	// TODO: Complete this method
}

// CropToRange reduces the DataSet down to timestamps contained within the provided Extents (inclusive).
// CropToRange assumes the base DataSet is already sorted, and will corrupt an unsorted DataSet
func (ds *DataSet) CropToRange(e timeseries.Extent) {
	if ds.RangeCropper != nil {
		ds.RangeCropper(e)
		return
	}
	ds.DefaultRangeCropper(e)
}

// DefaultRangeCropper is the default RangeCropper Function
func (ds *DataSet) DefaultRangeCropper(e timeseries.Extent) {
	x := len(ds.ExtentList)
	// The DataSet has no extents, so no need to do anything
	if x == 0 {
		for i := range ds.Results {
			ds.Results[i].SeriesList = make([]*Series, 0)
		}
		ds.ExtentList = timeseries.ExtentList{}
		return
	}
	// if the extent of the series is entirely outside the extent of the crop
	// range, return empty set and bail
	if ds.ExtentList.OutsideOf(e) {
		for i := range ds.Results {
			ds.Results[i].SeriesList = make([]*Series, 0)
		}
		ds.ExtentList = timeseries.ExtentList{}
		return
	}
	// if the series extent is entirely inside the extent of the crop range,
	// simple adjust down its ExtentList
	if ds.ExtentList.InsideOf(e) {
		if ds.ValueCount() == 0 {
			for i := range ds.Results {
				ds.Results[i].SeriesList = make([]*Series, 0)
			}
		}
		ds.ExtentList = ds.ExtentList.Crop(e)
		return
	}
	startNS := Epoch(e.Start.UnixNano())
	endNS := Epoch(e.End.UnixNano())
	var estDelCnt int
	dsStartNS := Epoch(ds.ExtentList[0].Start.UnixNano())
	if startNS > dsStartNS {
		estDelCnt += int((startNS - dsStartNS) / Epoch(ds.Step()))
	}
	dsEndNS := Epoch(ds.ExtentList[len(ds.ExtentList)-1].End.UnixNano())
	if endNS < dsEndNS {
		estDelCnt += int((dsEndNS - endNS) / Epoch(ds.Step()))
	}
	delPoints := make(map[Epoch]bool)
	for i := range ds.Results {
		if len(ds.Results[i].SeriesList) == 0 {
			ds.ExtentList = ds.ExtentList.Crop(e)
			continue
		}
		deletes := make(map[int]bool)
		for j, s := range ds.Results[i].SeriesList {
			start := -1
			end := -1
			for pi, p := range s.Points {
				if p.Epoch == endNS {
					if pi == 0 || p.Epoch == startNS || start == -1 {
						start = pi
					}
					end = pi + 1
					break
				}
				if p.Epoch > endNS {
					end = pi
					break
				}
				if p.Epoch < startNS {
					continue
				}
				if start == -1 &&
					(p.Epoch == startNS || (endNS > p.Epoch && p.Epoch > startNS)) {
					start = pi
				}
			}
			if start != -1 {
				if end == -1 {
					end = len(s.Points)
				}
				if start > 0 {
					for n := 0; n < start; n++ {
						delPoints[s.Points[n].Epoch] = true
					}
				}
				if end < len(s.Points)-1 {
					for n := len(s.Points) - 1; n >= end; n-- {
						delPoints[s.Points[n].Epoch] = true
					}
				}
				pts := make(Points, len(s.Points[start:end]))
				copy(pts, s.Points[start:end])
				s.Points = pts
				s.PointSize = pts.Size()
			} else {
				deletes[j] = true
			}
		}
		if len(deletes) > 0 {
			list := make([]*Series, len(ds.Results[i].SeriesList))
			for j, s := range ds.Results[i].SeriesList {
				if _, ok := deletes[j]; !ok {
					list = append(list, s)
				}
			}
			ds.Results[i].SeriesList = list
		}
	}
	ds.ExtentList = ds.ExtentList.Crop(e)
}

// SeriesCount returns the count of all Series across all Results in the DataSet
func (ds *DataSet) SeriesCount() int {
	var cnt int
	for i := range ds.Results {
		cnt += len(ds.Results[i].SeriesList)
	}
	return cnt
}

// ValueCount returns the count of all values across all Series in the DataSet
func (ds *DataSet) ValueCount() int64 {
	var cnt int64
	for i := range ds.Results {
		if len(ds.Results[i].SeriesList) == 0 {
			continue
		}
		for _, s := range ds.Results[i].SeriesList {
			if s == nil {
				continue
			}
			cnt += int64(len(s.Points))
		}
	}
	return cnt
}

// Size returns the memory utilization in bytes of the DataSet
func (ds *DataSet) Size() int64 {
	c := int64(len(ds.Status) +
		49 + // StepDuration=8 Mutex=8 OutputFormat=1 4xFuncs=32
		(len(ds.ExtentList) * 72) +
		len(ds.Error))
	for i := range ds.Results {
		c += int64(ds.Results[i].Size())
	}
	return c
}

// SetTimeRangeQuery sets the TimeRangeQuery for the DataSet
func (ds *DataSet) SetTimeRangeQuery(trq *timeseries.TimeRangeQuery) {
	ds.TimeRangeQuery = trq
}

// Step returns the step for the DataSet
func (ds *DataSet) Step() time.Duration {
	if ds.TimeRangeQuery != nil {
		return ds.TimeRangeQuery.Step
	}
	return 0
}

// TimestampCount returns the count of unique timestampes across all series in the DataSet
func (ds *DataSet) TimestampCount() int64 {
	return ds.ExtentList.TimestampCount(ds.Step())
}

// Extents returns the DataSet's ExentList
func (ds *DataSet) Extents() timeseries.ExtentList {
	return ds.ExtentList
}

// SetExtents overwrites a DataSet's known extents with the provided extent list
func (ds *DataSet) SetExtents(el timeseries.ExtentList) {
	l := make(timeseries.ExtentList, len(el))
	copy(l, el)
	ds.ExtentList = l
}

// Sort sorts all Values in each Series chronologically by their timestamp
// Sorting is efficiently baked into DataSet.Merge(), therefore this interface function is unused
// unless overriden
func (ds *DataSet) Sort() {
	if ds.Sorter != nil {
		ds.Sorter()
		return
	}
}

// UnmarshalDataSet unmarshals the dataset from a msgpack-formatted byte slice
func UnmarshalDataSet(b []byte) (timeseries.Timeseries, error) {
	ds := &DataSet{}
	_, err := ds.UnmarshalMsg(b)
	if err == nil && ds.TimeRangeQuery != nil {
		ds.TimeRangeQuery.Step = time.Duration(ds.TimeRangeQuery.StepNS)
	}

	return ds, err
}

// MarshalDataSet marshals the dataset into a msgpack-formatted byte slice
func MarshalDataSet(ts timeseries.Timeseries) ([]byte, error) {
	ds, ok := ts.(*DataSet)
	if !ok {
		return nil, timeseries.ErrUnknownFormat
	}
	if ds.TimeRangeQuery != nil {
		ds.TimeRangeQuery.StepNS = ds.TimeRangeQuery.Step.Nanoseconds()
	}
	return ds.MarshalMsg(nil)
}
