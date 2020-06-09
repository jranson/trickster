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

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tricksterproxy/trickster/pkg/sort/times"
	"github.com/tricksterproxy/trickster/pkg/timeseries"
)

// FieldDefinition ...
type FieldDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ResponseValue ...
type ResponseValue map[string]interface{}

// DataSet ...
type DataSet struct {
	Metric map[string]interface{}
	Points []Point
}

// Points ...
type Points []Point

// Point ...
type Point struct {
	Timestamp time.Time
	Value     float64
}

// Response is the JSON responose document structure for ClickHouse query results
type Response struct {
	Meta         []FieldDefinition     `json:"meta"`
	RawData      []ResponseValue       `json:"data"`
	Rows         int                   `json:"rows"`
	Order        []string              `json:"-"`
	StepDuration time.Duration         `json:"step,omitempty"`
	ExtentList   timeseries.ExtentList `json:"extents,omitempty"`
}

// ResultsEnvelope is the ClickHouse document structure optimized for time series manipulation
type ResultsEnvelope struct {
	Meta         []FieldDefinition            `json:"meta"`
	Data         map[string]*DataSet          `json:"data"`
	StepDuration time.Duration                `json:"step,omitempty"`
	ExtentList   timeseries.ExtentList        `json:"extents,omitempty"`
	Serializers  map[string]func(interface{}) `json:"-"`
	SeriesOrder  []string                     `json:"series_order,omitempty"`

	timestamps map[time.Time]bool // tracks unique timestamps in the matrix data
	tslist     times.Times
	isSorted   bool // tracks if the matrix data is currently sorted
	isCounted  bool // tracks if timestamps slice is up-to-date

	timeRangeQuery *timeseries.TimeRangeQuery
}

// Step returns the step for the Timeseries
func (re *ResultsEnvelope) Step() time.Duration {
	if re.timeRangeQuery != nil {
		return re.timeRangeQuery.Step
	}
	return re.StepDuration
}

// SetTimeRangeQuery sets the step for the Timeseries
func (re *ResultsEnvelope) SetTimeRangeQuery(trq *timeseries.TimeRangeQuery) {
	if trq == nil {
		return
	}
	re.StepDuration = trq.Step
	re.timeRangeQuery = trq
}

// Merge merges the provided Timeseries list into the base Timeseries (in the order provided)
// and optionally sorts the merged Timeseries
func (re *ResultsEnvelope) Merge(sort bool, collection ...timeseries.Timeseries) {

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	for _, ts := range collection {
		if ts != nil {
			re2 := ts.(*ResultsEnvelope)
			for k, s := range re2.Data {
				wg.Add(1)
				go func(l string, d *DataSet) {
					mtx.Lock()
					if _, ok := re.Data[l]; !ok {
						re.Data[l] = d
						mtx.Unlock()
						wg.Done()
						return
					}
					re.Data[l].Points = append(re.Data[l].Points, d.Points...)
					mtx.Unlock()
					wg.Done()
				}(k, s)
			}
			wg.Wait()
			re.mergeSeriesOrder(re2.SeriesOrder)
			re.ExtentList = append(re.ExtentList, re2.ExtentList...)
		}
	}

	re.ExtentList = re.ExtentList.Compress(re.Step())
	re.isSorted = false
	re.isCounted = false
	if sort {
		re.Sort()
	}
}

func (re *ResultsEnvelope) mergeSeriesOrder(so2 []string) {

	if len(so2) == 0 {
		return
	}

	if len(re.SeriesOrder) == 0 {
		re.SeriesOrder = so2
		return
	}

	so1 := make([]string, len(re.SeriesOrder), len(re.SeriesOrder)+len(so2))
	copy(so1, re.SeriesOrder)
	adds := make([]string, 0, len(so2))
	added := make(map[string]bool)

	for _, n := range so2 {
		if _, ok := re.Data[n]; !ok {
			if _, ok2 := added[n]; !ok2 {
				adds = append(adds, n)
				added[n] = true
			}
			continue
		}

		if len(adds) > 0 {
			for i, v := range so1 {
				if v == n {
					adds = append(adds, so1[i:]...)
					so1 = append(so1[0:i], adds...)
				}
			}
			adds = adds[:0]
		}
	}

	if len(adds) > 0 {
		so1 = append(so1, adds...)
	}

	re.SeriesOrder = so1

}

// Clone returns a perfect copy of the base Timeseries
func (re *ResultsEnvelope) Clone() timeseries.Timeseries {
	re2 := &ResultsEnvelope{
		isCounted:      re.isCounted,
		isSorted:       re.isSorted,
		timeRangeQuery: re.timeRangeQuery.Clone(),
	}

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	if re.SeriesOrder != nil {
		re2.SeriesOrder = make([]string, len(re.SeriesOrder))
		copy(re2.SeriesOrder, re.SeriesOrder)
	}

	if re.ExtentList != nil {
		re2.ExtentList = make(timeseries.ExtentList, len(re.ExtentList))
		copy(re2.ExtentList, re.ExtentList)
	}

	if re.tslist != nil {
		re2.tslist = make(times.Times, len(re.tslist))
		copy(re2.tslist, re.tslist)
	}

	if re.Meta != nil {
		re2.Meta = make([]FieldDefinition, len(re.Meta))
		copy(re2.Meta, re.Meta)
	}

	if re.Serializers != nil {
		re2.Serializers = make(map[string]func(interface{}))
		wg.Add(1)
		go func() {
			for k, s := range re.Serializers {
				re2.Serializers[k] = s
			}
			wg.Done()
		}()
	}

	if re.timestamps != nil {
		re2.timestamps = make(map[time.Time]bool)
		for k, v := range re.timestamps {
			wg.Add(1)
			go func(t time.Time, b bool) {
				mtx.Lock()
				re2.timestamps[t] = b
				mtx.Unlock()
				wg.Done()
			}(k, v)
		}
	}

	if re.Data != nil {
		re2.Data = make(map[string]*DataSet)
		wg.Add(1)
		go func() {
			for k, ds := range re.Data {
				ds2 := &DataSet{Metric: make(map[string]interface{})}
				for l, v := range ds.Metric {
					ds2.Metric[l] = v
				}
				ds2.Points = ds.Points[:]
				re2.Data[k] = ds2
			}
			wg.Done()
		}()
	}

	wg.Wait()

	return re2
}

// CropToSize reduces the number of elements in the Timeseries to the provided count, by evicting elements
// using a least-recently-used methodology. Any timestamps newer than the provided time are removed before
// sizing, in order to support backfill tolerance. The provided extent will be marked as used during crop.
func (re *ResultsEnvelope) CropToSize(sz int, t time.Time, lur timeseries.Extent) {
	re.isCounted = false
	re.isSorted = false
	x := len(re.ExtentList)
	// The Series has no extents, so no need to do anything
	if x < 1 {
		re.Data = make(map[string]*DataSet)
		re.ExtentList = timeseries.ExtentList{}
		return
	}

	// Crop to the Backfill Tolerance Value if needed
	if re.ExtentList[x-1].End.After(t) {
		re.CropToRange(timeseries.Extent{Start: re.ExtentList[0].Start, End: t})
	}

	tc := re.TimestampCount()
	el := timeseries.ExtentListLRU(re.ExtentList).UpdateLastUsed(lur, re.Step())
	sort.Sort(el)
	if len(re.Data) == 0 || tc <= int64(sz) {
		return
	}

	rc := tc - int64(sz) // # of required timestamps we must delete to meet the rentention policy
	removals := make(map[time.Time]bool)
	done := false
	var ok bool

	for _, x := range el {
		for ts := x.Start; !x.End.Before(ts) && !done; ts = ts.Add(re.Step()) {
			if _, ok = re.timestamps[ts]; ok {
				removals[ts] = true
				done = int64(len(removals)) >= rc
			}
		}
		if done {
			break
		}
	}

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	for _, s := range re.Data {
		tmp := s.Points[:0]
		for _, r := range s.Points {
			wg.Add(1)
			go func(p Point) {
				mtx.Lock()
				if _, ok := removals[p.Timestamp]; !ok {
					tmp = append(tmp, p)
				}
				mtx.Unlock()
				wg.Done()
			}(r)
		}
		wg.Wait()
		s.Points = tmp
	}

	tl := times.FromMap(removals)
	sort.Sort(tl)

	for _, t := range tl {
		for i, e := range el {
			if e.StartsAt(t) {
				el[i].Start = e.Start.Add(re.Step())
			}
		}
	}
	wg.Wait()

	re.ExtentList = timeseries.ExtentList(el).Compress(re.Step())
	re.Sort()
}

// CropToRange reduces the Timeseries down to timestamps contained within the provided Extents (inclusive).
// CropToRange assumes the base Timeseries is already sorted, and will corrupt an unsorted Timeseries
func (re *ResultsEnvelope) CropToRange(e timeseries.Extent) {
	re.isCounted = false
	x := len(re.ExtentList)
	// The Series has no extents, so no need to do anything
	if x < 1 {
		re.Data = make(map[string]*DataSet)
		re.ExtentList = timeseries.ExtentList{}
		return
	}

	// if the extent of the series is entirely outside the extent of the crop range, return empty set and bail
	if re.ExtentList.OutsideOf(e) {
		re.Data = make(map[string]*DataSet)
		re.ExtentList = timeseries.ExtentList{}
		return
	}

	// if the series extent is entirely inside the extent of the crop range, simply adjust down its ExtentList
	if re.ExtentList.InsideOf(e) {
		if re.ValueCount() == 0 {
			re.Data = make(map[string]*DataSet)
		}
		re.ExtentList = re.ExtentList.Crop(e)
		return
	}

	if len(re.Data) == 0 {
		re.ExtentList = re.ExtentList.Crop(e)
		return
	}

	deletes := make(map[string]bool)

	for i, s := range re.Data {
		start := -1
		end := -1
		for j, val := range s.Points {
			t := val.Timestamp
			if t.Equal(e.End) {
				// for cases where the first element is the only qualifying element,
				// start must be incremented or an empty response is returned
				if j == 0 || t.Equal(e.Start) || start == -1 {
					start = j
				}
				end = j + 1
				break
			}
			if t.After(e.End) {
				end = j
				break
			}
			if t.Before(e.Start) {
				continue
			}
			if start == -1 && (t.Equal(e.Start) || (e.End.After(t) && t.After(e.Start))) {
				start = j
			}
		}
		if start != -1 && len(s.Points) > 0 {
			if end == -1 {
				end = len(s.Points)
			}
			re.Data[i].Points = s.Points[start:end]
		} else {
			deletes[i] = true
		}
	}

	for i := range deletes {
		delete(re.Data, i)
	}

	re.ExtentList = re.ExtentList.Crop(e)
}

// Sort sorts all Values in each Series chronologically by their timestamp
func (re *ResultsEnvelope) Sort() {

	if re.isSorted || len(re.Data) == 0 {
		return
	}

	tsm := map[time.Time]bool{}
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	for i, s := range re.Data {
		m := make(map[time.Time]Point)
		keys := make(times.Times, 0, len(s.Points))
		for _, v := range s.Points {
			wg.Add(1)
			go func(sp Point) {
				mtx.Lock()
				if _, ok := m[sp.Timestamp]; !ok {
					keys = append(keys, sp.Timestamp)
					m[sp.Timestamp] = sp
				}
				tsm[sp.Timestamp] = true
				mtx.Unlock()
				wg.Done()
			}(v)
		}
		wg.Wait()
		sort.Sort(keys)
		sm := make(Points, 0, len(keys))
		for _, key := range keys {
			sm = append(sm, m[key])
		}
		re.Data[i].Points = sm
	}

	sort.Sort(re.ExtentList)

	re.timestamps = tsm
	re.tslist = times.FromMap(tsm)
	re.isCounted = true
	re.isSorted = true
}

func (re *ResultsEnvelope) updateTimestamps() {

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	if re.isCounted {
		return
	}
	m := make(map[time.Time]bool)
	for _, s := range re.Data {
		for _, v := range s.Points {
			wg.Add(1)
			go func(t time.Time) {
				mtx.Lock()
				m[t] = true
				mtx.Unlock()
				wg.Done()
			}(v.Timestamp)
		}
	}
	wg.Wait()
	re.timestamps = m
	re.tslist = times.FromMap(m)
	re.isCounted = true
}

// SetExtents overwrites a Timeseries's known extents with the provided extent list
func (re *ResultsEnvelope) SetExtents(extents timeseries.ExtentList) {
	re.isCounted = false
	re.ExtentList = extents
}

// Extents returns the Timeseries's ExentList
func (re *ResultsEnvelope) Extents() timeseries.ExtentList {
	return re.ExtentList
}

// TimestampCount returns the number of unique timestamps across the timeseries
func (re *ResultsEnvelope) TimestampCount() int64 {
	re.updateTimestamps()
	return int64(len(re.timestamps))
}

// SeriesCount returns the number of individual Series in the Timeseries object
func (re *ResultsEnvelope) SeriesCount() int {
	return len(re.Data)
}

// ValueCount returns the count of all values across all Series in the Timeseries object
func (re *ResultsEnvelope) ValueCount() int64 {
	var c int64
	wg := sync.WaitGroup{}
	for i := range re.Data {
		wg.Add(1)
		go func(j int64) {
			atomic.AddInt64(&c, j)
			wg.Done()
		}(int64(len(re.Data[i].Points)))
	}
	wg.Wait()
	return c
}

// Size returns the approximate memory utilization in bytes of the timeseries
func (re *ResultsEnvelope) Size() int64 {
	wg := sync.WaitGroup{}
	c := int64(24 + // .stepDuration
		(25 * len(re.timestamps)) + // time.Time (24) + bool(1)
		(24 * len(re.tslist)) + // time.Time (24)
		(len(re.ExtentList) * 72) + // time.Time (24) * 3
		2, // .isSorted + .isCounted
	)
	for i := range re.Meta {
		wg.Add(1)
		go func(j int) {
			atomic.AddInt64(&c, int64(len(re.Meta[j].Name)+len(re.Meta[j].Type)))
			wg.Done()
		}(i)
	}
	for _, s := range re.SeriesOrder {
		wg.Add(1)
		go func(t string) {
			atomic.AddInt64(&c, int64(len(t)))
			wg.Done()
		}(s)
	}
	for k, v := range re.Data {
		atomic.AddInt64(&c, int64(len(k)))
		wg.Add(1)
		go func(d *DataSet) {
			atomic.AddInt64(&c, int64(len(d.Points)*32))
			for mk := range d.Metric {
				atomic.AddInt64(&c, int64(len(mk)+8)) // + approx len of value (interface)
			}
			wg.Done()
		}(v)
	}
	wg.Wait()
	return c
}

// Parts ...
func (rv ResponseValue) Parts(timeKey, valKey string) (string, time.Time, float64, ResponseValue) {

	if len(rv) < 3 {
		return noParts()
	}

	labels := make([]string, 0, len(rv)-2)
	var t time.Time
	var val float64
	var err error

	meta := make(ResponseValue)

	for k, v := range rv {
		switch k {
		case timeKey:
			t, err = msToTime(v.(string))
			if err != nil {
				return noParts()
			}
		case valKey:
			if av, ok := v.(float64); ok {
				val = av
				continue
			}
			val, err = strconv.ParseFloat(v.(string), 64)
			if err != nil {
				return noParts()
			}
		default:
			meta[k] = v
			labels = append(labels, fmt.Sprintf("%s=%v", k, v))
		}
	}
	sort.Strings(labels)
	return fmt.Sprintf("{%s}", strings.Join(labels, ";")), t, val, meta
}

func noParts() (string, time.Time, float64, ResponseValue) {
	return "{}", time.Time{}, 0.0, ResponseValue{}
}

// MarshalJSON ...
func (re ResultsEnvelope) MarshalJSON() ([]byte, error) {

	if len(re.Meta) < 2 {
		return nil, ErrNotEnoughFields(len(re.Meta))
	}

	var mpl, fl int
	for _, v := range re.Data {
		lp := len(v.Points)
		fl += lp
		if mpl < lp {
			mpl = lp
		}
	}

	rsp := &Response{
		Meta:         re.Meta,
		RawData:      make([]ResponseValue, 0, fl),
		Rows:         int(re.ValueCount()),
		StepDuration: re.Step(),
		ExtentList:   re.ExtentList,
	}

	rsp.Order = make([]string, 0, len(re.Meta))
	for _, k := range re.Meta {
		rsp.Order = append(rsp.Order, k.Name)
	}

	// Assume the first item in the meta array is the time, and the second is the value
	timestampFieldName := rsp.Order[0]
	valueFieldName := rsp.Order[1]

	tm := make(map[time.Time][]ResponseValue)
	tl := make(times.Times, 0, mpl)

	l := len(re.Data)

	prepareMarshalledPoints := func(ds *DataSet) {

		var ok bool
		var t []ResponseValue

		for _, p := range ds.Points {

			t, ok = tm[p.Timestamp]
			if !ok {
				tl = append(tl, p.Timestamp)
				t = make([]ResponseValue, 0, l)
			}

			r := ResponseValue{
				timestampFieldName: strconv.FormatInt(p.Timestamp.UnixNano()/int64(time.Millisecond), 10),
				valueFieldName:     strconv.FormatFloat(p.Value, 'f', -1, 64),
			}
			for k2, v2 := range ds.Metric {
				r[k2] = v2
			}

			t = append(t, r)
			tm[p.Timestamp] = t
		}
	}

	for _, key := range re.SeriesOrder {
		if ds, ok := re.Data[key]; ok {
			prepareMarshalledPoints(ds)
		}
	}

	sort.Sort(tl)

	for _, t := range tl {
		rsp.RawData = append(rsp.RawData, tm[t]...)
	}

	bytes, err := json.Marshal(rsp)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// MarshalJSON ...
func (rsp *Response) MarshalJSON() ([]byte, error) {

	buf := &bytes.Buffer{}
	buf.WriteString(`{"meta":`)
	meta, _ := json.Marshal(rsp.Meta)
	buf.Write(meta)
	buf.WriteString(`,"data":[`)
	d := make([]string, 0, len(rsp.RawData))
	for _, rd := range rsp.RawData {
		d = append(d, string(rd.ToJSON(rsp.Order)))
	}
	buf.WriteString(strings.Join(d, ",") + "]")
	buf.WriteString(fmt.Sprintf(`,"rows": %d`, rsp.Rows))

	if rsp.ExtentList != nil && len(rsp.ExtentList) > 0 {
		el, _ := json.Marshal(rsp.ExtentList)
		buf.WriteString(fmt.Sprintf(`,"extents": %s`, string(el)))
	}

	buf.WriteString("}")

	b := buf.Bytes()

	return b, nil
}

// ToJSON ...
func (rv ResponseValue) ToJSON(order []string) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString("{")
	lines := make([]string, 0, len(rv))
	for _, k := range order {
		if v, ok := rv[k]; ok {

			// cleanup here
			j, err := json.Marshal(v)
			if err != nil {
				continue
			}
			lines = append(lines, fmt.Sprintf(`"%s":%s`, k, string(j)))
		}
	}
	buf.WriteString(strings.Join(lines, ",") + "}")
	return buf.Bytes()
}

// UnmarshalJSON ...
func (re *ResultsEnvelope) UnmarshalJSON(b []byte) error {

	response := Response{}
	err := json.Unmarshal(b, &response)
	if err != nil {
		return err
	}

	if len(response.Meta) < 2 {
		return ErrNotEnoughFields(len(response.Meta))
	}

	re.Meta = response.Meta
	re.ExtentList = response.ExtentList
	re.timeRangeQuery = &timeseries.TimeRangeQuery{Step: response.StepDuration}
	re.SeriesOrder = make([]string, 0)

	// Assume the first item in the meta array is the time field, and the second is the value field
	timestampFieldName := response.Meta[0].Name
	valueFieldName := response.Meta[1].Name

	registeredMetrics := make(map[string]bool)

	re.Data = make(map[string]*DataSet)
	l := len(response.RawData)
	for _, v := range response.RawData {
		metric, ts, val, meta := v.Parts(timestampFieldName, valueFieldName)
		if _, ok := registeredMetrics[metric]; !ok {
			registeredMetrics[metric] = true
			re.SeriesOrder = append(re.SeriesOrder, metric)
		}
		if !ts.IsZero() {
			a, ok := re.Data[metric]
			if !ok {
				a = &DataSet{Metric: meta, Points: make([]Point, 0, l)}
			}
			a.Points = append(a.Points, Point{Timestamp: ts, Value: val})
			re.Data[metric] = a
		}
	}

	return nil
}

// Len returns the length of a slice of time series data points
func (p Points) Len() int {
	return len(p)
}

// Less returns true if i comes before j
func (p Points) Less(i, j int) bool {
	return p[i].Timestamp.Before(p[j].Timestamp)
}

// Swap modifies a slice of time series data points by swapping the values in indexes i and j
func (p Points) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
