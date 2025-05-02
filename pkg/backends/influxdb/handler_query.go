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

package influxdb

import (
	"net/http"
	"strings"

	"github.com/trickstercache/trickster/v2/pkg/backends/influxdb/flux"
	"github.com/trickstercache/trickster/v2/pkg/backends/influxdb/influxql"
	"github.com/trickstercache/trickster/v2/pkg/proxy/engines"
	"github.com/trickstercache/trickster/v2/pkg/proxy/errors"
	"github.com/trickstercache/trickster/v2/pkg/proxy/params"
	"github.com/trickstercache/trickster/v2/pkg/proxy/urls"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"
)

// QueryHandler handles timeseries requests for InfluxDB and processes them through the delta proxy cache
func (c *Client) QueryHandler(w http.ResponseWriter, r *http.Request) {
	qp, qb, fromBody := params.GetRequestValues(r)
	q := strings.Trim(strings.ToLower(qp.Get(influxql.ParamQuery)), " \t\n")
	if q == "" {
		if len(qb) == 0 || !fromBody {
			c.ProxyHandler(w, r)
			return
		}
		q = string(qb)
	}
	// if it's not a select statement or flux query, just proxy it instead
	if !strings.Contains(q, "select ") && !strings.Contains(q, "from(") {
		c.ProxyHandler(w, r)
		return
	}
	r.URL = urls.BuildUpstreamURL(r, c.BaseUpstreamURL())
	engines.DeltaProxyCacheRequest(w, r, c.Modeler())
}

// ParseTimeRangeQuery parses the key parts of a TimeRangeQuery from the inbound HTTP Request
func (c *Client) ParseTimeRangeQuery(r *http.Request) (*timeseries.TimeRangeQuery,
	*timeseries.RequestOptions, bool, error) {
	trq := &timeseries.TimeRangeQuery{}
	rlo := &timeseries.RequestOptions{}
	values, b, isBody := params.GetRequestValues(r)
	if isBody {
		trq.OriginalBody = b
	}
	statement := values.Get(influxql.ParamQuery)
	if statement == "" && isBody && len(b) > 0 {
		s := string(b)
		if strings.Contains(s, "aggregateWindow(") && strings.Contains(s, "bucket:") {
			return trq, rlo, true, flux.ParseTimeRangeQuery(r, b, trq, rlo)
		}
		return nil, nil, false, errors.MissingURLParam(influxql.ParamQuery)
	}
	trq.Statement = statement
	t, err := influxql.ParseTimeRangeQuery(r, b, values, trq, rlo)
	return trq, rlo, t, err
}
