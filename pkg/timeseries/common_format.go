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
	"sync"
	"time"
)

type CommonFormat struct {
	StepDuration  time.Duration
	ExtentList    ExtentList
	TimestampList []Epoch
	PointsLookup  map[Epoch]map[Hash]*Point
	Results       []Result

	UpdateLock          sync.Mutex
	isSorted, isCounted bool
	Error               error

	SortFunc      func()
	MergeFunc     func(...*CommonFormat)
	CropSizeFunc  func(int, time.Time, Extent)
	CropRangeFunc func(Extent)
}

type Result struct {
	StatementID  int
	Error        error
	SeriesList   []*Series
	SeriesLookup map[Hash]*Series
}

type Series struct {
	Header *SeriesHeader
	Points Points
}

type SeriesHeader struct {
	Name           string
	Tags           Tags
	FieldsLookup   map[string]*FieldDefinition
	FieldsList     []*FieldDefinition
	TimestampIndex int
	Hash           Hash
	Size           int
}

func (cf *CommonFormat) Merge(sort bool, collection ...Timeseries) {
}

func (cf *CommonFormat) Clone() Timeseries {
	return nil
}

func (cf *CommonFormat) CropToSize(sz int, t time.Time, lur Extent) {
}

func (cf *CommonFormat) CropToRange(e Extent) {
}

func (cf *CommonFormat) Extents() ExtentList {
	return nil
}

func (cf *CommonFormat) SetExtents(el ExtentList) {

}

func (cf *CommonFormat) SeriesCount() int {
	return 0
}

func (cf *CommonFormat) ValueCount() int {
	return 0
}

func (cf *CommonFormat) SetStep(t time.Duration) {

}

func (cf *CommonFormat) Size() int {
	return 0
}

func (cf *CommonFormat) Sort() {

}

func (cf *CommonFormat) Step() time.Duration {
	return 0
}

func (cf *CommonFormat) TimestampCount() int {
	return 0
}
