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
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tricksterproxy/trickster/pkg/timeseries"
	"github.com/tricksterproxy/trickster/pkg/timeseries/dataset"
)

var marshalers = map[byte]dataset.Marshaler{
	0: marshalTimeseriesRaw,
	1: marshalTimeseriesJSON,
}

// MarshalTimeseries converts a Timeseries into a JSON blob
func MarshalTimeseries(ts timeseries.Timeseries) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := MarshalTimeseriesWriter(ts, buf)
	return buf.Bytes(), err
}

// MarshalTimeseriesWriter converts a Timeseries into a JSON blob via an io.Writer
func MarshalTimeseriesWriter(ts timeseries.Timeseries, w io.Writer) error {

	// // MarshalTimeseries converts a Timeseries into a JSON blob
	// func MarshalTimeseries(ts timeseries.Timeseries) ([]byte, error) {
	if ts == nil {
		return timeseries.ErrUnknownFormat
	}
	if ds, ok := ts.(*dataset.DataSet); ok {
		if marshaler, ok2 := marshalers[ds.TimeRangeQuery.OutputFormat]; ok2 {
			return marshaler(ds, w)
		}
	}
	return timeseries.ErrUnknownFormat
}

func marshalTimeseriesRaw(ds *dataset.DataSet, w io.Writer) error {
	if ds == nil || len(ds.Results) != 1 || len(ds.ExtentList) != 1 {
		return nil
	}
	for _, s := range ds.Results[0].SeriesList {
		if s == nil {
			continue
		}
		w.Write([]byte(s.Header.Name + "," +
			strconv.FormatInt(ds.ExtentList[0].Start.Unix(), 10) + "," +
			strconv.FormatInt(ds.ExtentList[0].End.Unix(), 10) + "," +
			strconv.Itoa(int(ds.Step().Seconds())) + "|"))
		sep := ","
		for i, v := range s.Points {
			if i == len(s.Points)-1 {
				sep = ""
			}
			if len(v.Values) == 1 {
				if f, ok := v.Values[0].(float64); ok {
					w.Write([]byte(strconv.FormatFloat(f, 'g', -1, 64) + sep))
				}
			}
		}
		w.Write([]byte("\n"))
	}
	return nil
}

func marshalTimeseriesJSON(ds *dataset.DataSet, w io.Writer) error {
	if ds == nil || len(ds.Results) != 1 || len(ds.ExtentList) != 1 {
		return nil
	}
	w.Write([]byte("["))
	sep2 := ""
	for _, s := range ds.Results[0].SeriesList {
		if s == nil {
			continue
		}
		w.Write([]byte(sep2 + "{\n  \"target\": \"" + s.Header.Name + "\",\n  \"datapoints\": [\n"))
		sep := ","
		for i, v := range s.Points {
			if i == len(s.Points)-1 {
				sep = ""
			}
			if len(v.Values) == 1 {
				if f, ok := v.Values[0].(float64); ok {
					w.Write([]byte("    [" + strconv.FormatFloat(f, 'g', -1, 64) +
						", " + strconv.FormatInt(int64(v.Epoch)/timeseries.Second, 10) + "]" +
						sep + "\n"))
				}
			}
		}
		w.Write([]byte("  ]\n}"))
		sep2 = ","
	}
	w.Write([]byte("]\n"))
	return nil
}

// UnmarshalTimeseries converts a JSON blob into a Timeseries
func UnmarshalTimeseries(data []byte, trq *timeseries.TimeRangeQuery) (timeseries.Timeseries, error) {
	if len(data) == 0 {
		return nil, timeseries.ErrInvalidBody
	}
	var start, end, step int64
	var err error
	ds := &dataset.DataSet{
		Results:        []dataset.Result{{}},
		TimeRangeQuery: trq,
	}
	lines := strings.Split(string(data), "\n")
	sl := make([]*dataset.Series, len(lines))
	//r.SeriesLookup = make(map[timeseries.Hash]*timeseries.Series)
	wg := sync.WaitGroup{}
	for i, line := range lines {
		if line == "" {
			continue
		}
		wg.Add(1)
		go func(l string, k int) {
			defer wg.Done()
			var s, n, p int64
			var e error
			row := strings.Split(string(l), "|")
			if len(row) != 2 {
				err = timeseries.ErrInvalidBody
				return
			}
			headerParts := strings.Split(row[0], ",")
			if len(headerParts) != 4 {
				err = timeseries.ErrTableHeader
				return
			}
			if s, e = strconv.ParseInt(headerParts[1], 10, 64); e != nil {
				err = timeseries.ErrUnmarshalEpoch
				return
			}
			if n, e = strconv.ParseInt(headerParts[2], 10, 64); e != nil {
				err = timeseries.ErrUnmarshalEpoch
				return
			}
			if p, e = strconv.ParseInt(headerParts[3], 10, 64); e != nil {
				err = timeseries.ErrUnmarshalEpoch
				return
			}
			if start == 0 {
				ds.UpdateLock.Lock()
				if start == 0 {
					extent := timeseries.Extent{
						Start: time.Unix(s, 0),
						End:   time.Unix(n, 0),
					}
					ds.TimeRangeQuery = &timeseries.TimeRangeQuery{
						Step: time.Second * time.Duration(step),
					}
					ds.ExtentList = timeseries.ExtentList{extent}
					start = s
					end = n
					step = p
				}
				ds.UpdateLock.Unlock()
			}
			fd := dataset.FieldDefinition{
				Name:     "value",
				DataType: dataset.Float64,
			}
			sh := dataset.SeriesHeader{
				Name:       headerParts[0],
				FieldsList: []dataset.FieldDefinition{fd},
			}
			width := end - start
			if width < 0 {
				err = timeseries.ErrInvalidExtent
				return
			}
			numPoints := int((width / p) + 1)
			values := strings.Split(row[1], ",")
			if len(values) != numPoints {
				err = timeseries.ErrInvalidExtent
				return
			}
			points := make(dataset.Points, numPoints)
			var j int
			for x := s; x <= n && j < numPoints; x += p {
				var v float64
				if v, e = strconv.ParseFloat(values[j], 64); e != nil {
					err = e
					return
				}
				epoch := dataset.Epoch(x * timeseries.Second)
				point := dataset.Point{
					Epoch:  epoch,
					Values: []interface{}{v},
					Size:   20,
				}
				points[j] = point
				j++
			}
			sh.CalculateSize()
			series := &dataset.Series{
				Header:    sh,
				Points:    points,
				PointSize: int64(len(points)) * 20,
			}
			sl[k] = series
		}(line, i)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	ds.Results[0].SeriesList = sl
	if trq != nil {
		ds.ExtentList = timeseries.ExtentList{trq.Extent}
	}

	return ds, nil
}
