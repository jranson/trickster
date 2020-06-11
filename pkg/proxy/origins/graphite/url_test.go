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
	"testing"
	"time"

	"github.com/tricksterproxy/trickster/pkg/config"
	"github.com/tricksterproxy/trickster/pkg/proxy/urls"
	"github.com/tricksterproxy/trickster/pkg/timeseries"
)

func TestSetExtent(t *testing.T) {

	expected := "format=raw&from=300&target=test&until=600"

	conf, _, err := config.Load("trickster", "test",
		[]string{"-origin-url", "none:9090", "-origin-type", "graphite", "-log-level", "debug"})
	if err != nil {
		t.Fatalf("Could not load configuration: %s", err.Error())
	}

	oc := conf.Origins["default"]
	client := Client{config: oc}

	u := &url.URL{RawQuery: "target=test"}

	r, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	e := &timeseries.Extent{Start: time.Unix(300, 0), End: time.Unix(600, 0)}
	client.SetExtent(r, nil, e)

	if expected != r.URL.RawQuery {
		t.Errorf("\nexpected [%s]\ngot [%s]", expected, r.URL.RawQuery)
	}

	u2 := urls.Clone(u)
	u2.RawQuery = ""

	b := bytes.NewBufferString(expected)
	r, _ = http.NewRequest(http.MethodPost, u2.String(), b)

	client.SetExtent(r, nil, e)
	if r.ContentLength != 29 {
		t.Errorf("expected 29 got %d", r.ContentLength)
	}

}

func TestFastForwardRequest(t *testing.T) {
	client := Client{}
	x, y := client.FastForwardRequest(nil)
	if x != nil || y != nil {
		t.Error("stub function should return nils")
	}
}
