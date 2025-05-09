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

package flux

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/timeseries"
)

const testFluxResponseCSV1 = `
#datatype,string,long,dateTime:RFC3339,dateTime:RFC3339,dateTime:RFC3339,double,string,string,string,string
#group,false,false,true,true,false,false,true,true,true,true
#default,_result,,,,,,,,,
,result,table,_start,_stop,_time,_value,_field,_measurement,cpu,host
,,0,2025-05-04T22:28:15Z,2025-05-04T22:43:15Z,2025-05-04T22:28:30Z,0.30229527991953403,usage_system,cpu,cpu-total,9793408bbc62
,,0,2025-05-04T22:28:15Z,2025-05-04T22:43:15Z,2025-05-04T22:28:45Z,0.3023433778023357,usage_system,cpu,cpu-total,9793408bbc62`

func testFluxResponseHeaders() [][]string {
	reader := bytes.NewReader([]byte(testFluxResponseCSV1))
	out, _ := csv.NewReader(reader).ReadAll()
	return out[:4]
}

func TestTypeToFieldDataType(t *testing.T) {
	tests := []struct {
		input    string
		expected timeseries.FieldDataType
	}{
		{input: "string", expected: timeseries.String},
		{input: "long", expected: timeseries.Int64},
		{input: "unsignedLong", expected: timeseries.Uint64},
		{input: "double", expected: timeseries.Float64},
		{input: "bool", expected: timeseries.Bool},
		{input: "duration", expected: timeseries.String},
		{input: "dateTime:RFC3339", expected: timeseries.DateTimeRFC3339},
		{input: "dateTime:RFC3339Nano", expected: timeseries.DateTimeRFC3339Nano},
		{input: "time", expected: timeseries.DateTimeRFC3339Nano},
		{input: "null", expected: timeseries.Null},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			out := typeToFieldDataType(test.input)
			if out != test.expected {
				t.Errorf("expected %d got %d", test.expected, out)
			}
		})
	}
}

func TestLoadFieldDefinition(t *testing.T) {
	fd := loadFieldDef("test", "long", "false", "a")
	if fd.Name != "test" || fd.DataType != 1 {
		t.Error("failed to parse fields")
	}
	fd = loadFieldDef("_time", "long", "true", "a'")
	if fd.Name != "_time" || fd.DataType != 1 {
		t.Error("failed to parse fields")
	}
}

func TestBuildFieldDefinitions(t *testing.T) {
	sf, err := buildFieldDefinitions(testFluxResponseHeaders())
	if err != nil {
		t.Error(err)
	}
	const expected = 4
	if len(sf.Tags) != expected {
		t.Errorf("expected %d got %d", expected, len(sf.Tags))
	}
	if sf.Timestamp.OutputPosition != 5 {
		t.Errorf("expected %d got %d", 5, sf.Timestamp.OutputPosition)
	}
}

func TestUnmarshalTimeseries(t *testing.T) {
	b := []byte(testFluxResponseCSV1)
	_, err := UnmarshalTimeseries(b, &timeseries.TimeRangeQuery{})
	if err != nil {
		t.Error(err)
	}
}
