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

import (
	"testing"
	"time"
)

func testDataSet() *DataSet {
	ds := &DataSet{
		StepDuration: time.Duration(5 * Second),
		Results:      []*Result{testResult()},
		ExtentList:   ExtentList{Extent{Start: time.Unix(5, 0), End: time.Unix(10, 0)}},
	}
	ds.PointsLookup = PointsLookup{
		ds.Results[0].SeriesList[0].Points[0].Epoch: map[Hash]*Point{
			ds.Results[0].SeriesList[0].Header.Hash: ds.Results[0].SeriesList[0].Points[0],
		},
		ds.Results[0].SeriesList[0].Points[1].Epoch: map[Hash]*Point{
			ds.Results[0].SeriesList[0].Header.Hash: ds.Results[0].SeriesList[0].Points[1],
		},
	}
	ds.Merger = ds.DefaultMerger
	ds.SizeCropper = ds.DefaultSizeCropper
	ds.RangeCropper = ds.DefaultRangeCropper
	ds.Sorter = func() {}
	return ds
}

func testDataSet2() *DataSet {

	sh1 := testSeriesHeader()
	sh1.Name = "test1"
	sh1.CalculateHash()

	sh2 := testSeriesHeader()
	sh2.Name = "test2"
	sh2.CalculateHash()

	sh3 := testSeriesHeader()
	sh3.Name = "test3"
	sh3.CalculateHash()

	sh4 := testSeriesHeader()
	sh4.Name = "test4"
	sh4.CalculateHash()

	newPoints := func(sh *SeriesHeader) Points {
		return Points{
			&Point{
				Epoch:  Epoch(5 * Second),
				Size:   27,
				Values: []interface{}{1},
				Header: sh,
			},
			&Point{
				Epoch:  Epoch(10 * Second),
				Size:   27,
				Values: []interface{}{1},
				Header: sh,
			},
			&Point{
				Epoch:  Epoch(15 * Second),
				Size:   27,
				Values: []interface{}{1},
				Header: sh,
			},
			&Point{
				Epoch:  Epoch(20 * Second),
				Size:   27,
				Values: []interface{}{1},
				Header: sh,
			},
			&Point{
				Epoch:  Epoch(25 * Second),
				Size:   27,
				Values: []interface{}{1},
				Header: sh,
			},
			&Point{
				Epoch:  Epoch(30 * Second),
				Size:   27,
				Values: []interface{}{1},
				Header: sh,
			},
		}
	}

	// r1 s1
	r1 := &Result{
		StatementID: 0,
		SeriesList: []*Series{
			{sh1, newPoints(sh1)},
		},
	}
	r1.SeriesLookup = map[Hash]*Series{
		sh1.Hash: r1.SeriesList[0],
	}
	r2 := &Result{
		StatementID: 1,
		SeriesList: []*Series{
			{sh2, newPoints(sh2)},
			{sh3, newPoints(sh3)},
			{sh4, newPoints(sh4)},
		},
	}
	r2.SeriesLookup = map[Hash]*Series{
		sh2.Hash: r2.SeriesList[0],
		sh3.Hash: r2.SeriesList[1],
		sh4.Hash: r2.SeriesList[2],
	}

	ds := &DataSet{
		StepDuration: time.Duration(5 * Second),
		Results:      []*Result{r1, r2},
		ExtentList:   ExtentList{Extent{Start: time.Unix(5, 0), End: time.Unix(30, 0)}},
	}

	buildPointsLookup := func(ds *DataSet) PointsLookup {
		pl := make(PointsLookup)
		for _, r := range ds.Results {
			for _, s := range r.SeriesList {
				for _, p := range s.Points {
					var m map[Hash]*Point
					var ok bool
					if m, ok = pl[p.Epoch]; !ok {
						m = map[Hash]*Point{s.Header.Hash: p}
						pl[p.Epoch] = m
					}
					m[s.Header.Hash] = p
				}
			}
		}
		return pl
	}

	ds.PointsLookup = buildPointsLookup(ds)
	ds.Merger = ds.DefaultMerger
	ds.SizeCropper = ds.DefaultSizeCropper
	ds.RangeCropper = ds.DefaultRangeCropper
	ds.Sorter = func() {}
	return ds
}

func TestDataSetClone(t *testing.T) {
	ds := testDataSet()
	ts := ds.Clone()
	ds2 := ts.(*DataSet)
	if len(ds2.ExtentList) != len(ds.ExtentList) {
		t.Error("dataset clone mismatch")
	}
}

func TestSort(t *testing.T) {
	var x int
	testFunc := func() {
		x = 20
	}
	ds := &DataSet{Sorter: testFunc}
	ds.Sort()
	if x != 20 {
		t.Error("sortfunc error")
	}
}

func TestSetExtents(t *testing.T) {
	ds := &DataSet{}
	ex := ExtentList{Extent{Start: time.Time{}, End: time.Time{}}}
	ds.SetExtents(ex)
	if len(ds.Extents()) != 1 {
		t.Errorf(`expected 1. got %d`, len(ds.ExtentList))
	}
}

func TestTimestampCount(t *testing.T) {
	ds := testDataSet()
	if ds.TimestampCount() != 2 {
		t.Errorf("expected 2 got %d", ds.TimestampCount())
	}
}

func TestStepDuration(t *testing.T) {
	ds := testDataSet()
	if int(ds.Step().Seconds()) != 5 {
		t.Errorf("expected 5 got %d", int(ds.Step().Seconds()))
	}
}

func TestSetStep(t *testing.T) {
	ds := testDataSet()
	ds.SetStep(1 * time.Second)
	if int(ds.Step().Seconds()) != 1 {
		t.Errorf("expected 1 got %d", int(ds.Step().Seconds()))
	}
}

func TestValueCount(t *testing.T) {
	ds := testDataSet()
	if ds.ValueCount() != 2 {
		t.Errorf("expected 2 got %d", ds.ValueCount())
	}
	ds.Results[0] = nil
	if ds.ValueCount() != 0 {
		t.Errorf("expected 0 got %d", ds.ValueCount())
	}
}

func TestSeriesCount(t *testing.T) {
	ds := testDataSet()
	if ds.SeriesCount() != 1 {
		t.Errorf("expected 1 got %d", ds.ValueCount())
	}
	ds.Results[0] = nil
	if ds.SeriesCount() != 0 {
		t.Errorf("expected 0 got %d", ds.ValueCount())
	}
}

func TestMerge(t *testing.T) {
	ds := &DataSet{}
	ds.Merge(false, nil)
	if len(ds.Results) > 0 {
		t.Error("dataset merge error")
	}

	ds = testDataSet2()
	ds2 := testDataSet2()
	ds.Results = ds.Results[:1]

	ds.Merge(false, ds2)

	if ds.SeriesCount() != 4 {
		t.Errorf("expected %d got %d", 4, ds.SeriesCount())
	}

}
