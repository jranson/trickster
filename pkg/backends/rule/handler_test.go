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

package rule

import (
	"github.com/trickstercache/trickster/v2/pkg/backends/providers"
	tu "github.com/trickstercache/trickster/v2/pkg/testutil"

	"testing"
)

func TestHandler(t *testing.T) {
	c, _ := newTestClient()
	_, w, r, _, _ := tu.NewTestInstance("", c.DefaultPathConfigs, 200, "{}", nil, providers.ReverseProxyCacheShort, "/health", "debug")
	c.Handler(w, r)
	if r.Header.Get("Test") == "" {
		t.Error("expected non-empty header")
	}
}
