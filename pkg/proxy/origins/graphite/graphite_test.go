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

package graphite

import (
	"bytes"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	cr "github.com/tricksterproxy/trickster/pkg/cache/registration"
	"github.com/tricksterproxy/trickster/pkg/config"
	pe "github.com/tricksterproxy/trickster/pkg/proxy/errors"
	"github.com/tricksterproxy/trickster/pkg/proxy/origins"
	"github.com/tricksterproxy/trickster/pkg/proxy/origins/graphite/model"
	oo "github.com/tricksterproxy/trickster/pkg/proxy/origins/options"
	"github.com/tricksterproxy/trickster/pkg/timeseries"
	"github.com/tricksterproxy/trickster/pkg/timeseries/dataset"
	tl "github.com/tricksterproxy/trickster/pkg/util/log"
)

var testModeler = timeseries.NewModeler(model.UnmarshalTimeseries,
	model.MarshalTimeseries, model.MarshalTimeseriesWriter,
	dataset.UnmarshalDataSet, dataset.MarshalDataSet)

func TestGraphiteClientInterfacing(t *testing.T) {

	// this test ensures the client will properly conform to the
	// Client and TimeseriesClient interfaces

	c := &Client{name: "test"}
	var oc origins.Client = c
	var tc origins.TimeseriesClient = c

	if oc.Name() != "test" {
		t.Errorf("expected %s got %s", "test", oc.Name())
	}

	if tc.Name() != "test" {
		t.Errorf("expected %s got %s", "test", tc.Name())
	}
}

func TestNewClient(t *testing.T) {

	conf, _, err := config.Load("trickster", "test", []string{"-origin-url", "http://1", "-origin-type", "test"})
	if err != nil {
		t.Fatalf("Could not load configuration: %s", err.Error())
	}

	caches := cr.LoadCachesFromConfig(conf, tl.ConsoleLogger("error"))
	defer cr.CloseCaches(caches)
	cache, ok := caches["default"]
	if !ok {
		t.Errorf("Could not find default configuration")
	}

	oc := &oo.Options{OriginType: "TEST_CLIENT"}
	c, err := NewClient("default", oc, nil, cache, testModeler)
	if err != nil {
		t.Error(err)
	}

	if c.Name() != "default" {
		t.Errorf("expected %s got %s", "default", c.Name())
	}

	if c.Cache().Configuration().CacheType != "memory" {
		t.Errorf("expected %s got %s", "memory", c.Cache().Configuration().CacheType)
	}

	if c.Configuration().OriginType != "TEST_CLIENT" {
		t.Errorf("expected %s got %s", "TEST_CLIENT", c.Configuration().OriginType)
	}
}

func TestParseTime(t *testing.T) {

	n := time.Now()

	e1, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", "2018-04-07 05:08:53 +0000 UTC")

	fixtures := []struct {
		input    string
		expected time.Time
	}{
		{"-1h", n.Add(-1 * time.Hour)},
		{"1523077733", e1},
		// {"1523077733.2", "2018-04-07 05:08:53.2 +0000 UTC"},
	}

	for _, f := range fixtures {
		out, err := parseTime(f.input, n)
		if err != nil {
			t.Error(err)
		}

		if !out.Equal(f.expected) {
			t.Errorf("Expected %d, got %d for input %s", f.expected.Unix(), out.Unix(), f.input)
		}
	}

	_, err := parseTime("", n)
	if err != timeseries.ErrInvalidTimeFormat {
		t.Error("expected invalid timeseries format error")
	}

	var ts time.Time
	ts, err = parseTime("now", n)
	if err != nil {
		t.Error(err)
	}
	if ts != n {
		t.Error("time parse mismatch")
	}

	const expected = "unable to parse duration: ."
	_, err = parseTime("-.", n)
	if err.Error() != expected {
		t.Error("expected invalid time format error")
	}

}

func TestParseTimeFails(t *testing.T) {
	_, err := parseTime("a", time.Now())
	if err == nil {
		t.Errorf(`expected error 'cannot parse "a" to a valid timestamp'`)
	}
}

func TestConfiguration(t *testing.T) {
	oc := &oo.Options{OriginType: "TEST"}
	client := Client{config: oc}
	c := client.Configuration()
	if c.OriginType != "TEST" {
		t.Errorf("expected %s got %s", "TEST", c.OriginType)
	}
}

func TestHTTPClient(t *testing.T) {
	oc := &oo.Options{OriginType: "TEST"}

	client, err := NewClient("test", oc, nil, nil, testModeler)
	if err != nil {
		t.Error(err)
	}

	if client.HTTPClient() == nil {
		t.Errorf("missing http client")
	}
}

func TestCache(t *testing.T) {

	conf, _, err := config.Load("trickster", "test", []string{"-origin-url", "http://1", "-origin-type", "test"})
	if err != nil {
		t.Fatalf("Could not load configuration: %s", err.Error())
	}

	caches := cr.LoadCachesFromConfig(conf, tl.ConsoleLogger("error"))
	defer cr.CloseCaches(caches)
	cache, ok := caches["default"]
	if !ok {
		t.Errorf("Could not find default configuration")
	}
	client := Client{cache: cache}
	c := client.Cache()

	if c.Configuration().CacheType != "memory" {
		t.Errorf("expected %s got %s", "memory", c.Configuration().CacheType)
	}
}

func TestName(t *testing.T) {

	client := Client{name: "TEST"}
	c := client.Name()

	if c != "TEST" {
		t.Errorf("expected %s got %s", "TEST", c)
	}

}

func TestParseTimeRangeQuery(t *testing.T) {

	qp := url.Values(map[string][]string{
		"target": {"test"},
		"from":   {"-6h"},
		"format": {"json"},
	})

	u := &url.URL{
		Scheme:   "https",
		Host:     "blah.com",
		Path:     "/",
		RawQuery: qp.Encode(),
	}

	req := &http.Request{URL: u}
	client := &Client{}
	res, err := client.ParseTimeRangeQuery(req)
	if err != nil {
		t.Error(err)
	} else {
		if res.Statement != "test" {
			t.Errorf("expected %s got %s", "test", res.Statement)
		}

		if int(res.Extent.End.Sub(res.Extent.Start).Hours()) != 6 {
			t.Errorf("expected 6 got %d", int(res.Extent.End.Sub(res.Extent.Start).Hours()))
		}
	}

	b := bytes.NewBufferString(qp.Encode())
	u.RawQuery = ""
	req, _ = http.NewRequest(http.MethodPost, u.String(), b)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.ParseTimeRangeQuery(req)
	if err != nil {
		t.Error(err)
	}
}

func TestParseTimeRangeQueryMissingTarget(t *testing.T) {
	expected := pe.MissingURLParam(upTarget).Error()
	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "blah.com",
		Path:   "/",
		RawQuery: url.Values(map[string][]string{
			"target_": {`up`},
			"from":    {strconv.Itoa(int(time.Now().Add(time.Duration(-6) * time.Hour).Unix()))},
			"until":   {strconv.Itoa(int(time.Now().Unix()))},
			"format":  {"json"},
		}).Encode(),
	}}
	client := &Client{}
	_, err := client.ParseTimeRangeQuery(req)
	if err == nil {
		t.Errorf(`expected "%s", got NO ERROR`, expected)
		return
	}
	if err.Error() != expected {
		t.Errorf(`expected "%s", got "%s"`, expected, err.Error())
	}
}

func TestParseTimeRangeQueryMissingFormat(t *testing.T) {
	expected := pe.MissingURLParam(upFormat).Error()
	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "blah.com",
		Path:   "/",
		RawQuery: url.Values(map[string][]string{
			"target": {`up`},
			"from":   {"-1h"},
		}).Encode(),
	}}
	client := &Client{}
	_, err := client.ParseTimeRangeQuery(req)
	if err == nil {
		t.Errorf(`expected "%s", got NO ERROR`, expected)
		return
	}
	if err.Error() != expected {
		t.Errorf(`expected "%s", got "%s"`, expected, err.Error())
	}
}

func TestParseTimeRangeBadStartTime(t *testing.T) {

	thresh1 := time.Now().Add(-24 * time.Hour).Add(2 * time.Second)
	thresh2 := time.Now().Add(-24 * time.Hour).Add(-2 * time.Second)

	const color = "red"
	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "blah.com",
		Path:   "/",
		RawQuery: url.Values(map[string][]string{
			"target": {`up`},
			"from":   {color},
			"until":  {strconv.Itoa(int(time.Now().Unix()))},
			"format": {"json"},
		}).Encode(),
	}}
	client := &Client{}
	trq, _ := client.ParseTimeRangeQuery(req)

	if trq.Extent.Start.Before(thresh2) ||
		trq.Extent.Start.After(thresh1) {
		t.Error("bad start time")
	}

}

func TestParseTimeRangeBadEndTime(t *testing.T) {

	thresh1 := time.Now().Add(2 * time.Second)
	thresh2 := time.Now().Add(-2 * time.Second)

	const color = "blue"
	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "blah.com",
		Path:   "/",
		RawQuery: url.Values(map[string][]string{
			"target": {`up`},
			"from":   {strconv.Itoa(int(time.Now().Add(time.Duration(-6) * time.Hour).Unix()))},
			"until":  {color},
			"format": {"json"},
		}).Encode(),
	}}
	client := &Client{}
	trq, _ := client.ParseTimeRangeQuery(req)
	if trq.Extent.End.Before(thresh2) ||
		trq.Extent.End.After(thresh1) {
		t.Error("bad end time")
	}
}

func TestParseTimeRangeQueryNoStart(t *testing.T) {

	thresh1 := time.Now().Add(-24 * time.Hour).Add(2 * time.Second)
	thresh2 := time.Now().Add(-24 * time.Hour).Add(-2 * time.Second)

	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "blah.com",
		Path:   "/",
		RawQuery: url.Values(map[string][]string{
			"target": {`up`},
			"until":  {strconv.Itoa(int(time.Now().Unix()))},
			"format": {"json"},
		}).Encode(),
	}}
	client := &Client{}
	trq, _ := client.ParseTimeRangeQuery(req)

	if trq.Extent.Start.Before(thresh2) ||
		trq.Extent.Start.After(thresh1) {
		t.Error("bad start time")
	}
}

func TestParseTimeRangeQueryNoEnd(t *testing.T) {

	thresh1 := time.Now().Add(2 * time.Second)
	thresh2 := time.Now().Add(-2 * time.Second)

	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "blah.com",
		Path:   "/",
		RawQuery: url.Values(map[string][]string{
			"target": {`up`},
			"from":   {strconv.Itoa(int(time.Now().Add(time.Duration(-6) * time.Hour).Unix()))},
			"format": {"json"},
		}).Encode(),
	}}
	client := &Client{}
	trq, _ := client.ParseTimeRangeQuery(req)
	if trq.Extent.End.Before(thresh2) ||
		trq.Extent.End.After(thresh1) {
		t.Error("bad end time")
	}
}

func TestSetCache(t *testing.T) {
	c, err := NewClient("test", oo.NewOptions(), nil, nil, testModeler)
	if err != nil {
		t.Error(err)
	}
	c.SetCache(nil)
	if c.Cache() != nil {
		t.Errorf("expected nil cache for client named %s", "test")
	}
}

func TestRouter(t *testing.T) {
	c, err := NewClient("test", oo.NewOptions(), nil, nil, testModeler)
	if err != nil {
		t.Error(err)
	}
	if c.Router() != nil {
		t.Error("expected nil router")
	}
}
