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

package flux

import (
	"testing"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/timeseries"
)

const fqAbsoluteTimeMS string = `from("test-bucket")
	|> range(start: 2023-01-01T00:00:00.000Z, stop: 2023-01-08T00:00:00.000Z)
	|> window(every: 5m)
	|> mean()
`

const tokenized = `from("test-bucket")
	|> range(<TIMERANGE_TOKEN>)
	|> window(every: 5m)
	|> mean()
`

func TestParseQuery(t *testing.T) {
	s, e, d, err := ParseQuery(fqAbsoluteTimeMS)
	if s != tokenized {
		t.Error("parsing failure")
	}
	if d != time.Minute*5 {
		t.Error("invalid duration", d)
	}
	e2 := timeseries.Extent{Start: time.Unix(1672531200, 0),
		End: time.Unix(1673136000, 0)}
	if !e.Start.Equal(e2.Start) {
		t.Error("invalid extent start")
	}
	if !e.End.Equal(e2.End) {
		t.Error("invalid extent end")
	}
	if err != nil {
		t.Error(err)
	}
}
