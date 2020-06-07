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

package timeseries

// Result represents the results of a single query statement in the DataSet
type Result struct {
	// StatementID represents the ID of the statement for this result. This field may not be
	// used by all tsdb implementations
	StatementID int
	// Error represents a statement-level error
	Error error
	// SeriesList is an ordered list of the Series in this result
	SeriesList []*Series
	// SeriesLookup is map of Series in the result, searchable by SeriesHeader Hash
	SeriesLookup SeriesLookup
}

// Hashes returns the ordered list of Hashes for the SeriesList in the Result
func (r *Result) Hashes() Hashes {
	if len(r.SeriesList) == 0 {
		return nil
	}
	h := make(Hashes, len(r.SeriesList))
	for i := range r.SeriesList {
		h[i] = r.SeriesList[i].Header.Hash
	}
	return h
}

// Clone returns an exact copy of the Result
func (r *Result) Clone() *Result {
	clone := &Result{
		StatementID:  r.StatementID,
		Error:        r.Error,
		SeriesList:   make([]*Series, len(r.SeriesList)),
		SeriesLookup: make(SeriesLookup),
	}
	for i, s := range r.SeriesList {
		clone.SeriesList[i] = s.Clone()
		clone.SeriesLookup[s.Header.Hash] = clone.SeriesList[i]
	}
	return clone
}
