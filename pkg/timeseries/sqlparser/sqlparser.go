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

// Package sqlparser provides customizations to the base sql parser that are
// specific to timeseries (for example, parsing trickster directives from comments)
package sqlparser

import (
	"context"
	"strconv"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/parsing"
	lsql "github.com/trickstercache/trickster/v2/pkg/parsing/lex/sql"
	"github.com/trickstercache/trickster/v2/pkg/parsing/sql"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"
	"github.com/trickstercache/trickster/v2/pkg/timeseries/epoch"
)

// Parser is a basic, extendable SQL Parser
type Parser struct {
	*sql.Parser
}

// NewRunContext returns a context with the Time Range Query and Request Options attached
func NewRunContext(trq *timeseries.TimeRangeQuery,
	ro *timeseries.RequestOptions) context.Context {
	return context.WithValue(
		context.WithValue(context.Background(), timeseries.TimeRangeQueryCtx, trq),
		timeseries.RequestOptionsCtx, ro)
}

// ArtifactsFromContext returns the Time Range Query and Request Options from the context, if present
func ArtifactsFromContext(ctx context.Context) (*timeseries.TimeRangeQuery, *timeseries.RequestOptions) {
	v := ctx.Value(timeseries.TimeRangeQueryCtx)
	trq, _ := v.(*timeseries.TimeRangeQuery)
	v = ctx.Value(timeseries.RequestOptionsCtx)
	ro, _ := v.(*timeseries.RequestOptions)
	return trq, ro
}

// New returns a new Time Series SQL Parser
func New(po *parsing.Options) parsing.Parser {
	po = po.WithDecisions("FindVerb",
		parsing.DecisionSet{
			lsql.TokenComment: ParseFVComment,
		},
	)
	p := &Parser{
		Parser: sql.New(po).(*sql.Parser),
	}
	return p
}

// ParseComment will parse the comment for Trickster time range query directives
// such as Fast Forward Disable or a Backfill Tolerance value. It assumes the
// RunState is currently on the Comment Token
func ParseComment(rs *parsing.RunState) {
	i := rs.Current()
	trq, ro := ArtifactsFromContext(rs.Context())
	if trq != nil {
		// TimeRangeQuery extractions here
		trq.ExtractBackfillTolerance(i.Val)
	}
	if ro != nil {
		// RequestOption extractions here
		ro.ExtractFastForwardDisabled(i.Val)
	}
}

// ParseFVComment will parse the comment and return FindVerb to be invoked
func ParseFVComment(_, _ parsing.Parser, rs *parsing.RunState) parsing.StateFn {
	ParseComment(rs)
	return rs.GetReturnFunc(sql.FindVerb, nil, true)
}

const billion epoch.Epoch = 1000000000
const million epoch.Epoch = 1000000

// ParseEpoch accepts the following formats, and returns an epoch.Epoch (NS since 1-1-1970):
// typ 0 - 10-digit epoch in seconds:     "1577836800"
// typ 1 - 13-digit epoch in millisecons: "1577836800000"
// typ 2 - SQL DateTime:                   "2020-01-01 00:00:00"
// typ 3 - SQL Date:                       "2020-01-01"
func ParseEpoch(input string) (epoch.Epoch, byte, error) {
	var ts epoch.Epoch
	var i int64
	li := len(input)
	if isAllDigits(input) {
		i, _ = strconv.ParseInt(input, 10, 64)
		switch li {
		case 10:
			return billion * epoch.Epoch(i), 0, nil
		case 13:
			return million * epoch.Epoch(i), 1, nil
		}
	}

	if (li == 10 || li == 19) && (input[4] == '-' && input[7] == '-') {
		var typ byte = 3
		year, err := strconv.Atoi(input[0:4])
		if err != nil {
			return 0, 0, err
		}
		month, err := strconv.Atoi(input[5:7])
		if err != nil {
			return 0, 0, err
		}
		day, err := strconv.Atoi(input[8:10])
		if err != nil {
			return 0, 0, err
		}
		var hour, minute, second int
		if (li == 19) && (input[13] == ':' && input[16] == ':') {
			typ = 2
			hour, err = strconv.Atoi(input[11:13])
			if err != nil {
				return 0, 0, err
			}
			minute, err = strconv.Atoi(input[14:16])
			if err != nil {
				return 0, 0, err
			}
			second, err = strconv.Atoi(input[17:19])
			if err != nil {
				return 0, 0, err
			}
		}
		t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
		return epoch.Epoch(t.UnixNano()), typ, nil
	}
	return ts, 0, timeseries.ErrInvalidTimeFormat
}

func isAllDigits(input string) bool {
	for _, c := range input {
		if c < 48 || c > 57 {
			return false
		}
	}
	return true
}
