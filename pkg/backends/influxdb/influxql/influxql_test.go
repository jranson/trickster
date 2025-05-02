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

package influxql

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/proxy/headers"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"
)

const testFluxQuery = `SELECT mean("value") FROM "monthly"."rollup.1min" WHERE ("application" = 'web') AND time >= now() - 6h ` +
	`GROUP BY time(15s), "cluster" fill(null)`

var testVals = url.Values(map[string][]string{"q": {testFluxQuery},
	"epoch": {"ms"}})
var testRawQuery = testVals.Encode()

func TestParseTimeRangeQuery(t *testing.T) {

	// test GET
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme:   "https",
			Host:     "blah.com",
			Path:     "/",
			RawQuery: testRawQuery,
		}}
	trq := &timeseries.TimeRangeQuery{Statement: testFluxQuery}
	rlo := &timeseries.RequestOptions{}

	_, err := ParseTimeRangeQuery(req, nil, testVals, trq, rlo)
	if err != nil {
		t.Error(err)
	} else {
		if trq.Step.Seconds() != 15 {
			t.Errorf("expected %d got %d", 15, int(trq.Step.Seconds()))
		}
		if int(trq.Extent.End.Sub(trq.Extent.Start).Hours()) != 6 {
			t.Errorf("expected %d got %d", 6, int(trq.Extent.End.Sub(trq.Extent.Start).Hours()))
		}
	}

	trq = &timeseries.TimeRangeQuery{Statement: testFluxQuery}
	rlo = &timeseries.RequestOptions{}

	req, _ = http.NewRequest(http.MethodPost, "http://blah.com/", io.NopCloser(strings.NewReader(testRawQuery)))
	req.Header.Set(headers.NameContentLength, strconv.Itoa(len(testRawQuery)))
	req.Header.Set(headers.NameContentType, headers.ValueApplicationJSON)

	_, err = ParseTimeRangeQuery(req, []byte(testRawQuery), testVals, trq, rlo)
	if err != nil {
		t.Error(err)
	} else {
		if trq.Step.Seconds() != 15 {
			t.Errorf("expected %d got %d", 15, int(trq.Step.Seconds()))
		}
		if int(trq.Extent.End.Sub(trq.Extent.Start).Hours()) != 6 {
			t.Errorf("expected %d got %d", 6, int(trq.Extent.End.Sub(trq.Extent.Start).Hours()))
		}
	}

}
