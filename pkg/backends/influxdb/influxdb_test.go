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
	"testing"

	"github.com/trickstercache/trickster/v2/pkg/backends"
	bo "github.com/trickstercache/trickster/v2/pkg/backends/options"
	"github.com/trickstercache/trickster/v2/pkg/backends/providers"
	cr "github.com/trickstercache/trickster/v2/pkg/cache/registry"
	"github.com/trickstercache/trickster/v2/pkg/config"
)

func TestInfluxDBClientInterfacing(t *testing.T) {

	// this test ensures the client will properly conform to the
	// Client and TimeseriesBackend interfaces

	backendClient, err := backends.NewTimeseriesBackend("test", nil, nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}

	var oc backends.Backend = backendClient
	var tc backends.TimeseriesBackend = backendClient

	if oc.Name() != "test" {
		t.Errorf("expected %s got %s", "test", oc.Name())
	}

	if tc.Name() != "test" {
		t.Errorf("expected %s got %s", "test", tc.Name())
	}
}

func TestNewClient(t *testing.T) {
	conf, err := config.Load([]string{"-provider", providers.InfluxDB, "-origin-url", "http://1"})
	if err != nil {
		t.Fatalf("Could not load configuration: %s", err.Error())
	}

	caches := cr.LoadCachesFromConfig(conf)
	defer cr.CloseCaches(caches)
	cache, ok := caches["default"]
	if !ok {
		t.Errorf("Could not find default configuration")
	}

	o := &bo.Options{Provider: "TEST_CLIENT"}
	c, err := NewClient("default", o, nil, cache, nil, nil)
	if err != nil {
		t.Error(err)
	}

	if c.Name() != "default" {
		t.Errorf("expected %s got %s", "default", c.Name())
	}

	if c.Cache().Configuration().Provider != "memory" {
		t.Errorf("expected %s got %s", "memory", c.Cache().Configuration().Provider)
	}

	if c.Configuration().Provider != "TEST_CLIENT" {
		t.Errorf("expected %s got %s", "TEST_CLIENT", c.Configuration().Provider)
	}
}
