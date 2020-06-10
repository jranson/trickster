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
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/tricksterproxy/trickster/pkg/timeseries"
	"github.com/tricksterproxy/trickster/pkg/timeseries/dataset"
)

// WFDocument the Wire Format Document for the timeseries
type WFDocument struct {
	Results []WFResult `json:"results"`
	Err     string     `json:"error,omitempty"`
}

// WFResult is the Result section of the WFD
type WFResult struct {
	StatementID int        `json:"statement_id"`
	SeriesList  []WFSeries `json:"series,omitempty"`
	Err         string     `json:"error,omitempty"`
}

// WFSeries is the Series section of the WFR
type WFSeries struct {
	Name    string            `json:"name,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
	Columns []string          `json:"columns,omitempty"`
	Values  [][]interface{}   `json:"values,omitempty"`
	Partial bool              `json:"partial,omitempty"`
}

var epochMultipliers = map[byte]int64{
	1: 1,             // nanoseconds
	2: 1000,          // microseconds
	3: 1000000,       // milliseconds
	4: 1000000000,    // seconds
	5: 60000000000,   // minutes
	6: 3600000000000, // hours
}

var marshalers = map[byte]dataset.Marshaler{
	0: marshalTimeseriesJSON,
	1: marshalTimeseriesJSONPretty,
	2: marshalTimeseriesCSV,
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

func marshalTimeseriesJSON(ds *dataset.DataSet, w io.Writer) error {
	if ds == nil || len(ds.ExtentList) != 1 {
		return nil
	}
	var multiplier int64
	var dateWriter func(io.Writer, dataset.Epoch, int64)
	switch ds.TimeRangeQuery.TimeFormat {
	case 0:
		dateWriter = writeRFC3339Time
	default:
		if m, ok := epochMultipliers[ds.TimeRangeQuery.TimeFormat]; ok {
			multiplier = m
		} else {
			multiplier = 1
		}
		dateWriter = writeEpochTime
	}
	w.Write([]byte(`{"results":[`))
	for i := range ds.Results {
		w.Write([]byte(
			fmt.Sprintf(`{"statement_id":%d,"series":[`,
				ds.Results[i].StatementID)))
		for _, s := range ds.Results[i].SeriesList {
			if s == nil {
				continue
			}
			w.Write([]byte(
				fmt.Sprintf(`{"name":"%s","tags":{%s},`, s.Header.Name,
					s.Header.Tags.KVP())))
			fl := len(s.Header.FieldsList)
			l := fl + 1
			cols := make([]string, l)
			for j, f := range s.Header.FieldsList {
				if j == s.Header.TimestampIndex {
					cols[j] = "time"
					j++
				}
				cols[j] = f.Name
			}
			w.Write([]byte(
				`"columns":["` + strings.Join(cols, `","`) + `],"values":[`,
			))
			lp := len(s.Points) - 1
			for j := range s.Points {
				w.Write([]byte("["))
				for n, v := range s.Points[j].Values {
					if n == s.Header.TimestampIndex {
						dateWriter(w, s.Points[j].Epoch, multiplier)
						n++
					}
					writeValue(w, v)
					if n < fl {
						w.Write([]byte(","))
					}
				}
				w.Write([]byte("]"))
				if j < lp {
					w.Write([]byte(","))
				}
			}
			w.Write([]byte("]}"))
		}
		w.Write([]byte("]"))
	}
	w.Write([]byte("}]}"))
	return nil
}

func writeRFC3339Time(w io.Writer, epoch dataset.Epoch, m int64) {
	t := time.Unix(0, int64(epoch))
	w.Write([]byte(`"` + t.Format(time.RFC3339Nano) + `"`))
}

func writeEpochTime(w io.Writer, epoch dataset.Epoch, m int64) {
	w.Write([]byte(strconv.FormatInt(int64(epoch)/m, 10)))
}

func writeValue(w io.Writer, v interface{}) {
	switch t := v.(type) {
	case string:
		w.Write([]byte(`"` + t + `"`))
	case bool:
		w.Write([]byte(strconv.FormatBool(t)))
	case int64:
		w.Write([]byte(strconv.FormatInt(t, 10)))
	case int:
		w.Write([]byte(strconv.FormatInt(int64(t), 10)))
	case float64:
		w.Write([]byte(strconv.FormatFloat(t, 'f', -1, 64)))
	case float32:
		w.Write([]byte(strconv.FormatFloat(float64(t), 'f', -1, 64)))
	}
}

func marshalTimeseriesJSONPretty(ds *dataset.DataSet, w io.Writer) error {
	if ds == nil || len(ds.ExtentList) != 1 {
		return nil
	}
	for _, s := range ds.Results[0].SeriesList {
		if s == nil {
			continue
		}

	}
	return nil
}

func marshalTimeseriesCSV(ds *dataset.DataSet, w io.Writer) error {
	if ds == nil || len(ds.ExtentList) != 1 {
		return nil
	}
	for _, s := range ds.Results[0].SeriesList {
		if s == nil {
			continue
		}

	}
	return nil
}

// UnmarshalTimeseries converts a JSON blob into a Timeseries
func UnmarshalTimeseries(data []byte, trq *timeseries.TimeRangeQuery) (timeseries.Timeseries, error) {
	if trq == nil {
		return nil, timeseries.ErrNoTimerangeQuery
	}
	wfd := &WFDocument{}
	err := json.Unmarshal(data, &wfd)
	if err != nil {
		return nil, err
	}
	ds := &dataset.DataSet{
		Error:          wfd.Err,
		TimeRangeQuery: trq,
		ExtentList:     timeseries.ExtentList{trq.Extent},
	}
	if wfd.Results == nil {
		return nil, timeseries.ErrInvalidBody
	}
	ds.Results = make([]dataset.Result, len(wfd.Results))
	for i := range wfd.Results {
		ds.Results[i].StatementID = wfd.Results[i].StatementID
		ds.Results[i].Error = wfd.Results[i].Err
		if wfd.Results[i].SeriesList == nil {
			return nil, timeseries.ErrInvalidBody
		}
		ds.Results[i].SeriesList = make([]*dataset.Series, len(wfd.Results[i].SeriesList))
		for j := range wfd.Results[i].SeriesList {
			sh := dataset.SeriesHeader{
				Name:           wfd.Results[i].SeriesList[j].Name,
				Tags:           dataset.Tags(wfd.Results[i].SeriesList[j].Tags),
				QueryStatement: trq.Statement,
			}
			if wfd.Results[i].SeriesList[j].Columns == nil ||
				len(wfd.Results[i].SeriesList[j].Columns) < 2 {
				return nil, timeseries.ErrInvalidBody
			}
			var timeFound bool
			cl := len(wfd.Results[i].SeriesList[j].Columns)
			fdl := cl - 1 // -1 excludes time column from list for DataSet format
			sh.FieldsList = make([]dataset.FieldDefinition, fdl)
			for ci, cn := range wfd.Results[i].SeriesList[j].Columns {
				if cn == "time" {
					timeFound = true
					sh.TimestampIndex = ci
					continue
				}
				if ci < fdl {
					sh.FieldsList[ci] = dataset.FieldDefinition{Name: cn}
				}
			}
			if !timeFound || wfd.Results[i].SeriesList[j].Values == nil {
				return nil, timeseries.ErrInvalidBody
			}
			sh.CalculateSize()
			pts := make(dataset.Points, len(wfd.Results[i].SeriesList[j].Values))
			var sz int64
			for vi, v := range wfd.Results[i].SeriesList[j].Values {
				if len(v) != cl {
					return nil, timeseries.ErrInvalidBody
				}

				pt, cols, err := pointFromValues(v, sh.TimestampIndex, trq.TimeFormat)
				if err != nil {
					return nil, err
				}
				if vi == 0 {
					for x := range cols {
						sh.FieldsList[x].DataType = cols[x]
					}
				}
				pts[vi] = pt
				sz += int64(pt.Size)
			}
			s := &dataset.Series{
				Header:    sh,
				Points:    pts,
				PointSize: sz,
			}
			ds.Results[i].SeriesList[j] = s
		}
	}
	return ds, nil
}

func pointFromValues(v []interface{}, tsIndex int, dateFormat byte) (dataset.Point,
	[]dataset.FieldDataType, error) {
	p := dataset.Point{}
	p.Values = append(v[:tsIndex], v[tsIndex+1:]...)
	ns, ok := v[tsIndex].(int64)
	if !ok {
		return p, nil, timeseries.ErrInvalidTimeFormat
	}
	p.Epoch = dataset.Epoch(ns)
	p.Size = 12
	fdts := make([]dataset.FieldDataType, len(v))
	for x := range v {
		switch t := v[x].(type) {
		case string:
			fdts[x] = dataset.String
			p.Size += len(t)
		case bool:
			fdts[x] = dataset.Bool
			p.Size++
		case int64, int:
			fdts[x] = dataset.Int64
			p.Size += 8
		case float64, float32:
			fdts[x] = dataset.Float64
			p.Size += 8
		default:
			return p, nil, timeseries.ErrInvalidTimeFormat
		}
	}
	return p, fdts, nil
}
