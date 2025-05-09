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

package clickhouse

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/trickstercache/trickster/v2/pkg/parsing/lex/sql"
	"github.com/trickstercache/trickster/v2/pkg/proxy/methods"
	"github.com/trickstercache/trickster/v2/pkg/proxy/request"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"
)

// Common URL Parameter Names
const (
	upQuery = "query"
)

// SetExtent will change the upstream request query to use the provided Extent
func (c *Client) SetExtent(r *http.Request, trq *timeseries.TimeRangeQuery,
	extent *timeseries.Extent) {
	if extent == nil || r == nil || trq == nil {
		return
	}
	qi := r.URL.Query()
	isBody := methods.HasBody(r.Method)
	q := interpolateTimeQuery(trq.Statement, trq.TimestampDefinition, extent)
	if isBody {
		request.SetBody(r, []byte(q))
	} else {
		qi.Set(upQuery, q)
		r.URL.RawQuery = qi.Encode()
	}
}

func interpolateTimeQuery(template string, tfd timeseries.FieldDefinition,
	extent *timeseries.Extent) string {

	var start, end string

	// tfd.DataType holds the database internal format for the timestamp used
	// when setting extents
	switch tfd.DataType {
	case timeseries.DateTimeUnixMilli: // epoch millisecs
		start = strconv.FormatInt(extent.Start.UnixNano()/1000000, 10)
		end = strconv.FormatInt(extent.End.UnixNano()/1000000, 10)
	case timeseries.DateTimeUnixNano: // epoch nanosecs
		start = strconv.FormatInt(extent.Start.UnixNano(), 10)
		end = strconv.FormatInt(extent.End.UnixNano(), 10)
	case timeseries.DateTimeSQL: // '2025-05-01 11:39:18'
		start = "'" + extent.Start.Format(sql.SQLDateTimeFormat) + "'"
		end = "'" + extent.End.Format(sql.SQLDateTimeFormat) + "'"
	default: // epoch secs
		start = strconv.FormatInt(extent.Start.Unix(), 10)
		end = strconv.FormatInt(extent.End.Unix(), 10)
	}

	trange := fmt.Sprintf("%s BETWEEN %s AND %s", tfd.Name, start, end)
	return strings.NewReplacer(
		tkRange, trange,
		tkTS1, start,
		tkTS2, end,
		tkFormat, "TSVWithNamesAndTypes",
	).Replace(template)
}
