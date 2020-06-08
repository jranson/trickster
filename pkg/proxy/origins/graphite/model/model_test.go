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
	"strconv"
	"testing"

	"github.com/tricksterproxy/trickster/pkg/timeseries"
	"github.com/tricksterproxy/trickster/pkg/util/md5"
)

const testResult1 = `expression1,0,180,60|1,2,3,4
expression2,0,180,60|1,2,3,4
`

const testResultJSON1 = `[{
  "target": "expression1",
  "datapoints": [
    [1, 0],
    [2, 60],
    [3, 120],
    [4, 180]
  ]
},{
  "target": "expression2",
  "datapoints": [
    [1, 0],
    [2, 60],
    [3, 120],
    [4, 180]
  ]
}]
`

func TestUnmarshalTimeseries(t *testing.T) {
	_, err := UnmarshalTimeseries([]byte(testResult1))
	if err != nil {
		t.Error(err)
	}
	_, err = UnmarshalTimeseries(nil)
	if err != timeseries.ErrInvalidBody {
		t.Error("expected error for invalid body")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,0,180,60|1,2,3,4|"))
	if err != timeseries.ErrInvalidBody {
		t.Error("expected error for invalid body")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,0,180,60,|1,2,3,4"))
	if err != timeseries.ErrTableHeader {
		t.Error("expected error for invalid table header")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,a,180,60|1,2,3,4"))
	if err != timeseries.ErrUnmarshalEpoch {
		t.Error("expected error for invalid epoch")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,0,a,60|1,2,3,4"))
	if err != timeseries.ErrUnmarshalEpoch {
		t.Error("expected error for invalid epoch")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,0,180,a|1,2,3,4"))
	if err != timeseries.ErrUnmarshalEpoch {
		t.Error("expected error for invalid epoch")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,180,0,60|1,2,3,4"))
	if err != timeseries.ErrInvalidExtent {
		t.Error("expected error for invalid extent")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,0,180,60|1,2,3,4,5"))
	if err != timeseries.ErrInvalidExtent {
		t.Error("expected error for invalid extent")
	}
	_, err = UnmarshalTimeseries([]byte("expression1,0,180,60|1,2,3,trickster"))
	if err.(*strconv.NumError).Err != strconv.ErrSyntax {
		t.Error("expected error for invalid syntax")
	}
}

func TestMarshalTimeseries(t *testing.T) {

	_, err := MarshalTimeseries(nil)
	if err != timeseries.ErrUnknownFormat {
		t.Error("expected error for unknown format")
	}

	ts, err := UnmarshalTimeseries([]byte(testResult1))
	if err != nil {
		t.Error(err)
	}

	_, err = MarshalTimeseries(ts)
	if err != nil {
		t.Error(err)
	}

	cf := ts.(*timeseries.DataSet)
	cf.OutputFormat = 2

	_, err = MarshalTimeseries(ts)
	if err != timeseries.ErrUnknownFormat {
		t.Error("expected error for unknown format")
	}

}

func TestMarshalTimeseriesRaw(t *testing.T) {
	ts, err := UnmarshalTimeseries([]byte(testResult1))
	if err != nil {
		t.Error(err)
	}

	cf := ts.(*timeseries.DataSet)

	var b []byte
	b, err = marshalTimeseriesRaw(cf)
	if err != nil {
		t.Error(err)
	}

	s := string(b)

	if s != testResult1 {
		t.Errorf("expected [%s] got [%s]", testResult1, s)
	}

	cf.Results = append(cf.Results, &timeseries.Result{})
	b, err = marshalTimeseriesRaw(cf)
	if b != nil {
		t.Error("expected nil")
	}
	if err != nil {
		t.Error(err)
	}

}

func TestMarshalTimeseriesJSON(t *testing.T) {

	b, err := marshalTimeseriesJSON(nil)
	if err != nil {
		t.Error(err)
	}
	if len(b) > 0 {
		t.Error("expected empty slice")
	}

	var ts timeseries.Timeseries
	ts, err = UnmarshalTimeseries([]byte(testResult1))
	if err != nil {
		t.Error(err)
	}

	cf := ts.(*timeseries.DataSet)

	b, err = marshalTimeseriesJSON(cf)
	if err != nil {
		t.Error(err)
	}

	s := string(b)

	if s != testResultJSON1 {
		t.Errorf("expected [%s] got [%s]", md5.Checksum(testResultJSON1), md5.Checksum(s))
	}

}
