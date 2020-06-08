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

import "testing"

func testSeries() *Series {
	sh := testSeriesHeader()
	return &Series{
		Header: sh,
		Points: testPoints(sh),
	}
}

func testSeriesHeader() *SeriesHeader {
	sh := &SeriesHeader{
		Name:           "test",
		QueryStatement: "SELECT TRICKSTER!",
		Tags:           Tags{"test1": "value1"},
		FieldsList: []*FieldDefinition{
			{
				Name:     "Field1",
				DataType: FieldDataType(1),
			},
		},
		TimestampIndex: 37,
		Size:           56,
	}
	sh.FieldsLookup = map[string]*FieldDefinition{"Field1": sh.FieldsList[0]}
	sh.CalculateHash()
	return sh
}

func TestSeriesHeaderCalculateHash(t *testing.T) {
	sh := testSeriesHeader()
	sh.Hash = 0
	sh.CalculateHash()
	if sh.Hash == 0 {
		t.Error("invalid hash value")
	}
}

func TestSeriesHeaderClone(t *testing.T) {
	sh := testSeriesHeader()
	sh2 := sh.Clone()
	if sh2.Size != sh.Size ||
		len(sh2.FieldsList) != 1 || len(sh2.FieldsLookup) != 1 ||
		sh2.FieldsList[0] != sh2.FieldsLookup["Field1"] {
		t.Error("series header clone mismatch")
	}

}

func TestSeriesClone(t *testing.T) {

	s := testSeries()
	s2 := s.Clone()

	if s2.Header.Hash != s.Header.Hash {
		t.Error("series clone mismatch")
	}

	if s2.Header == s.Header {
		t.Error("series clone mismatch")
	}

	if s2.Points[0].Epoch != s.Points[0].Epoch {
		t.Error("series clone mismatch")
	}

}
