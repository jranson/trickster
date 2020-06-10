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
	"reflect"
	"testing"
	"time"
)

var testJSON1 = []byte(`{"meta":[{"name":"t","type":"UInt64"},{"name":"cnt","type":"UInt64"},` +
	`{"name":"meta1","type":"UInt16"},{"name":"meta2","type":"String"}],` +
	`"data":[{"t":"1557766080000","cnt":"12648509","meta1":200,"meta2":"value2"},` +
	`{"t":"1557766080000","cnt":"10260032","meta1":200,"meta2":"value3"},` +
	`{"t":"1557766080000","cnt":"1","meta1":206,"meta2":"value3"}],"rows":3}`,
)
var testJSON2 = []byte(`{"meta":[{"name":"t"}],"data":[{"t":"1557766080000","cnt":"12648509",` +
	`"meta1":200,"meta2":"value2"},{"t":"1557766080000","cnt":"10260032","meta1":200,"meta2":"value3"},` +
	`{"t":"1557766080000","cnt":"1","meta1":206,"meta2":"value3"}],"rows":3}`,
) // should generate error

var testRE1 = &ResultsEnvelope{
	Meta: []FieldDefinition{
		{
			Name: "t",
			Type: "UInt64",
		},
		{
			Name: "cnt",
			Type: "UInt64",
		},
		{
			Name: "meta1",
			Type: "UInt16",
		},
		{
			Name: "meta2",
			Type: "String",
		},
	},

	SeriesOrder: []string{"1", "2", "3"},

	Data: map[string]*DataSet{
		"1": {
			Metric: map[string]interface{}{
				"meta1": 200,
				"meta2": "value2",
			},
			Points: []Point{
				{
					Timestamp: time.Unix(1557766080, 0),
					Value:     12648509,
				},
			},
		},
		"2": {
			Metric: map[string]interface{}{
				"meta1": 200,
				"meta2": "value3",
			},
			Points: []Point{
				{
					Timestamp: time.Unix(1557766080, 0),
					Value:     10260032,
				},
			},
		},
		"3": {
			Metric: map[string]interface{}{
				"meta1": 206,
				"meta2": "value3",
			},
			Points: []Point{
				{
					Timestamp: time.Unix(1557766080, 0),
					Value:     1,
				},
			},
		},
	},
}

func TestUnmarshalTimeseries(t *testing.T) {

	ts, err := UnmarshalTimeseries(testJSON1, nil)
	if err != nil {
		t.Error(err)
		return
	}

	re := ts.(*ResultsEnvelope)

	if len(re.Meta) != 4 {
		t.Errorf(`expected 4. got %d`, len(re.Meta))
		return
	}

	if len(re.Data) != 3 {
		t.Errorf(`expected 3. got %d`, len(re.Data))
		return
	}

	_, err = UnmarshalTimeseries(nil, nil)
	if err == nil {
		t.Errorf("expected error: %s", `unexpected end of JSON input`)
		return
	}

	_, err = UnmarshalTimeseries(testJSON2, nil)
	if err == nil {
		t.Errorf("expected error: %s", `Must have at least two fields; only have 1`)
		return
	}

}

func TestMarshalTimeseries(t *testing.T) {
	expectedLen := len(testJSON1)
	bytes, err := MarshalTimeseries(testRE1)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(testJSON1, bytes) {
		t.Errorf("expected %d got %d", expectedLen, len(bytes))
	}
}
