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

// Package graphite provides the Graphite Origin Type
package graphite

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/tricksterproxy/trickster/pkg/cache"
	"github.com/tricksterproxy/trickster/pkg/proxy"
	"github.com/tricksterproxy/trickster/pkg/proxy/errors"
	"github.com/tricksterproxy/trickster/pkg/proxy/origins"
	oo "github.com/tricksterproxy/trickster/pkg/proxy/origins/options"
	"github.com/tricksterproxy/trickster/pkg/proxy/params"
	"github.com/tricksterproxy/trickster/pkg/proxy/urls"
	"github.com/tricksterproxy/trickster/pkg/timeseries"
	"github.com/tricksterproxy/trickster/pkg/util/timeconv"
)

var _ origins.Client = (*Client)(nil)

// Graphite API
const (
	mnRender = "render"
)

// Common URL Parameter Names
const (
	upTarget = "target"
	upFrom   = "from"
	upUntil  = "until"
	upFormat = "format"
)

var supportedFormats = map[string]bool{
	"raw":    true,
	"csv":    true,
	"json":   true,
	"pickle": true,
}

// Client Implements Proxy Client Interface
type Client struct {
	name               string
	config             *oo.Options
	cache              cache.Cache
	webClient          *http.Client
	handlers           map[string]http.Handler
	handlersRegistered bool
	baseUpstreamURL    *url.URL
	healthURL          *url.URL
	healthHeaders      http.Header
	healthMethod       string
	router             http.Handler
	modeler            *timeseries.Modeler
}

// NewClient returns a new Client Instance
func NewClient(name string, oc *oo.Options, router http.Handler,
	cache cache.Cache, modeler *timeseries.Modeler) (origins.Client, error) {
	c, err := proxy.NewHTTPClient(oc)
	bur := urls.FromParts(oc.Scheme, oc.Host, oc.PathPrefix, "", "")
	return &Client{name: name, config: oc, router: router, cache: cache,
		webClient: c, baseUpstreamURL: bur, modeler: modeler}, err
}

// SetCache sets the Cache object the client will use for caching origin content
func (c *Client) SetCache(cc cache.Cache) {
	c.cache = cc
}

// Configuration returns the upstream Configuration for this Client
func (c *Client) Configuration() *oo.Options {
	return c.config
}

// HTTPClient returns the HTTP Client for this origin
func (c *Client) HTTPClient() *http.Client {
	return c.webClient
}

// Name returns the name of the upstream Configuration proxied by the Client
func (c *Client) Name() string {
	return c.name
}

// Cache returns and handle to the Cache instance used by the Client
func (c *Client) Cache() cache.Cache {
	return c.cache
}

// Router returns the http.Handler that handles request routing for this Client
func (c *Client) Router() http.Handler {
	return c.router
}

func parseTime(input string, now time.Time) (time.Time, error) {
	if input == "" {
		return time.Time{}, timeseries.ErrInvalidTimeFormat
	}
	if input == "now" {
		return now, nil
	}
	if input[0] == '-' {
		d, err := timeconv.ParseDuration(input[1:])
		if err != nil {
			return time.Time{}, err
		}
		return now.Add(-d), nil
	}
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	t := time.Unix(i, 0) // assume it's epoch seconds
	return t, nil
}

// ParseTimeRangeQuery parses the key parts of a TimeRangeQuery from the inbound HTTP Request
func (c *Client) ParseTimeRangeQuery(r *http.Request) (*timeseries.TimeRangeQuery, error) {

	trq := &timeseries.TimeRangeQuery{Extent: timeseries.Extent{}, Step: time.Second * 300}

	qp, _, _ := params.GetRequestValues(r)

	of := qp.Get(upFormat)
	if of == "" {
		return nil, errors.MissingURLParam(upFormat)
	}
	if ok := supportedFormats[of]; !ok {
		return nil, errors.ErrUnsupportedEncoding
	}

	trq.Statement = qp.Get(upTarget)
	if trq.Statement == "" {
		return nil, errors.MissingURLParam(upTarget)
	}

	n := time.Now()
	var t time.Time
	var err error

	if p := qp.Get(upFrom); p != "" {
		t, err = parseTime(p, n)
		if err != nil {
			t = n.Add(-24 * time.Hour) // graphite default from is -24h
		}
	} else {
		t = n.Add(-24 * time.Hour)
	}
	trq.Extent.Start = t

	if p := qp.Get(upUntil); p != "" {
		t, err = parseTime(p, n)
		if err != nil {
			t = n // graphite default until is now()
		}
	} else {
		t = n
	}
	trq.Extent.End = t

	return trq, nil
}
