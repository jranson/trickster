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

//go:generate msgp

package timeseries

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tricksterproxy/trickster/pkg/proxy/urls"
)

// TimeRangeQuery represents a timeseries database query parsed from an inbound HTTP request
type TimeRangeQuery struct {
	// Statement is the timeseries database query (with tokenized timeranges where present) requested by the user
	Statement string `msg:"statement"`
	// Extent provides the start and end times for the request from a timeseries database
	Extent Extent `msg:"exent"`
	// Step indicates the amount of time in seconds between each datapoint in a TimeRangeQuery's resulting timeseries
	Step time.Duration `msg:"-"`
	// TimestampFieldName indicates the database field name for the timestamp field
	TimestampFieldName string `msg:"-"`
	// TemplateURL is used by some Origin Types for templatization of url parameters containing timestamps
	TemplateURL *url.URL `msg:"-"`
	// FastForwardDisable indicates whether the Time Range Query result should include fast forward data
	FastForwardDisable bool `msg:"-"`
	// IsOffset is true if the query uses a relative offset modifier
	IsOffset bool `msg:"-"`
	// StepNS is the nanosecond representation for Step
	StepNS int64 `msg:"step"`
}

// Clone returns an exact copy of a TimeRangeQuery
func (trq *TimeRangeQuery) Clone() *TimeRangeQuery {
	t := &TimeRangeQuery{
		Statement:          trq.Statement,
		Step:               trq.Step,
		StepNS:             trq.StepNS,
		Extent:             Extent{Start: trq.Extent.Start, End: trq.Extent.End},
		IsOffset:           trq.IsOffset,
		TimestampFieldName: trq.TimestampFieldName,
		FastForwardDisable: trq.FastForwardDisable,
	}

	if trq.TemplateURL != nil {
		t.TemplateURL = urls.Clone(trq.TemplateURL)
	}

	return t
}

// NormalizeExtent adjusts the Start and End of a TimeRangeQuery's Extent to align against normalized boundaries.
func (trq *TimeRangeQuery) NormalizeExtent() {
	if trq.Step.Seconds() > 0 {
		if !trq.IsOffset && trq.Extent.End.After(time.Now()) {
			trq.Extent.End = time.Now()
		}
		trq.Extent.Start = trq.Extent.Start.Truncate(trq.Step)
		trq.Extent.End = trq.Extent.End.Truncate(trq.Step)
	}
}

func (trq *TimeRangeQuery) String() string {
	return fmt.Sprintf(`{ "statement": "%s", "step": "%s", "extent": "%s" }`,
		strings.Replace(trq.Statement, `"`, `\"`, -1), trq.Step.String(), trq.Extent.String())
}

// GetBackfillTolerance will return the backfill tolerance for the query based on the provided
// default, and any query-specific tolerance directives included in the query comments
func (trq *TimeRangeQuery) GetBackfillTolerance(def time.Duration) time.Duration {
	if x := strings.Index(trq.Statement, "trickster-backfill-tolerance:"); x > 1 {
		x += 29
		y := x
		for ; y < len(trq.Statement); y++ {
			if trq.Statement[y] < 48 || trq.Statement[y] > 57 {
				break
			}
		}
		if i, err := strconv.Atoi(trq.Statement[x:y]); err == nil {
			return time.Second * time.Duration(i)
		}
	}
	return def
}
