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

package prometheus

import (
	"github.com/tricksterproxy/trickster/pkg/proxy/origins/prometheus/model"
	"github.com/tricksterproxy/trickster/pkg/timeseries"
)

// MarshalTimeseries converts a Timeseries into a JSON blob
// -----> change from interface
func (c *Client) MarshalTimeseries(ts timeseries.Timeseries) ([]byte, error) {
	// Marshal the Envelope back to a json object for Cache Storage
	return model.MarshalTimeseries(ts)
}

// UnmarshalTimeseries converts a JSON blob into a Timeseries
func (c *Client) UnmarshalTimeseries(data []byte) (timeseries.Timeseries, error) {
	return model.UnmarshalTimeseries(data)
}

// UnmarshalInstantaneous converts a JSON blob into an Instantaneous Data Point
func (c *Client) UnmarshalInstantaneous(data []byte) (timeseries.Timeseries, error) {
	return model.UnmarshalTimeseries(data)
}
