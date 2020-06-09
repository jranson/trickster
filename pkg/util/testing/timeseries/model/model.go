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

// Package model is a test utility package that serializes between a *DataSet
// and the  Prometheus HTTP API JSON Format
package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/tricksterproxy/trickster/pkg/timeseries"
)

// WFDocument the Wire Format Document for the timeseries
type WFDocument struct {
	Status string `json:"status"`
	Data   WFData `json:"data"`
}

// WFData is the data section of the WFD
type WFData struct {
	ResultType string     `json:"resultType"`
	Results    []WFResult `json:"result"`
}

// WFResult is the Result section of the WFD
type WFResult struct {
	Metric timeseries.Tags `json:"metric"`
	Values [][]interface{} `json:"values"`
	Value  []interface{}   `json:"value"`
}

func Modeler() *timeseries.Modeler {
	return &timeseries.Modeler{
		WireMarshalWriter: MarshalTimeseriesWriter,
		WireMarshaler:     MarshalTimeseries,
		WireUnmarshaler:   UnmarshalTimeseries,
		CacheMarshaler:    timeseries.MarshalDataSet,
		CacheUnmarshaler:  timeseries.UnmarshalDataSet,
	}
}

// UnmarshalTimeseries converts a JSON blob into a Timeseries
func UnmarshalTimeseries(data []byte) (timeseries.Timeseries, error) {
	wfd := &WFDocument{}
	err := json.Unmarshal(data, &wfd)
	if err != nil {
		return nil, err
	}
	ds := &timeseries.DataSet{
		Status:  wfd.Status,
		Results: []timeseries.Result{{}},
	}
	ds.Results[0].SeriesList = make([]*timeseries.Series, len(wfd.Data.Results))

	for i, pr := range wfd.Data.Results {
		sh := timeseries.SeriesHeader{
			Tags: pr.Metric,
		}
		if n, ok := pr.Metric["__name__"]; ok {
			sh.Name = n
		}
		fd := timeseries.FieldDefinition{
			Name:     "value",
			DataType: timeseries.String,
		}
		sh.FieldsList = []timeseries.FieldDefinition{fd}
		//sh.FieldsLookup = map[string]*timeseries.FieldDefinition{"value": fd}
		sh.CalculateHash()
		var pts timeseries.Points
		l := len(pr.Values)
		if wfd.Data.ResultType == "matrix" && l > 0 {
			pts = make(timeseries.Points, l)
			for j, v := range pr.Values {
				pt := pointFromValues(v)
				pts[j] = pt
			}
			// hello extentlist??? TODO
		} else if wfd.Data.ResultType == "vector" && len(pr.Value) == 2 {
			pts = make(timeseries.Points, 1)
			pt := pointFromValues(pr.Value)
			pts[0] = pt
			t := time.Unix(0, int64(pt.Epoch))
			ds.ExtentList = timeseries.ExtentList{timeseries.Extent{Start: t, End: t}}
		}
		s := &timeseries.Series{
			Header: sh,
			Points: pts,
		}
		ds.Results[0].SeriesList[i] = s
	}
	return ds, nil
}

func pointFromValues(v []interface{}) timeseries.Point {
	if len(v) != 2 {
		return timeseries.Point{}
	}
	var f1 float64
	var s string
	var ok bool
	if f1, ok = v[0].(float64); !ok {
		return timeseries.Point{}
	}
	if s, ok = v[1].(string); !ok {
		return timeseries.Point{}
	}
	return timeseries.Point{
		Epoch:  timeseries.Epoch(f1) * 1000000000,
		Size:   len(s) + 16,
		Values: []interface{}{s},
	}
}

// MarshalTimeseries converts a Timeseries into a JSON blob
func MarshalTimeseries(ts timeseries.Timeseries) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := MarshalTimeseriesWriter(ts, buf)
	return buf.Bytes(), err
}

// MarshalTimeseriesWriter converts a Timeseries into a JSON blob via an io.Writer
func MarshalTimeseriesWriter(ts timeseries.Timeseries, w io.Writer) error {

	ds, ok := ts.(*timeseries.DataSet)
	if !ok {
		return timeseries.ErrUnknownFormat
	}
	// With Prometheus we presume only one Result per Dataset
	if len(ds.Results) != 1 {
		return timeseries.ErrUnknownFormat
	}

	w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[`)) // todo: always "success" ?

	seriesSep := ""
	for _, s := range ds.Results[0].SeriesList {
		if s == nil {
			continue
		}
		w.Write([]byte(seriesSep + `{"metric":{`))
		sep := ""
		for _, k := range s.Header.Tags.Keys() {
			w.Write([]byte(fmt.Sprintf(`%s"%s":"%s"`, sep, k, s.Header.Tags[k])))
			sep = ","
		}
		w.Write([]byte(`},"values":[`))
		sep = ""
		sort.Sort(s.Points)
		for _, p := range s.Points {
			w.Write([]byte(fmt.Sprintf(`%s[%s,"%s"]`,
				sep,
				strconv.FormatFloat(float64(p.Epoch)/1000000000, 'f', -1, 64),
				p.Values[0]),
			))
			sep = ","
		}
		w.Write([]byte("]}"))
		seriesSep = ","
	}
	w.Write([]byte("]}}"))
	return nil
}
