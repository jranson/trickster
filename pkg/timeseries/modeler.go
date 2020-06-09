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

import "io"

// Modeler is a container object for Timeseries marshaling operations
type Modeler struct {
	Unmarshaler   UnmarshalerFunc   `msg:"-"`
	Marshaler     MarshalerFunc     `msg:"-"`
	MarshalWriter MarshalWriterFunc `msg:"-"`
}

// UnmarshalerFunc describes a function that unmarshals a Timeseries
type UnmarshalerFunc func([]byte) (Timeseries, error)

// MarshalerFunc describes a function that marshals a Timeseries
type MarshalerFunc func(Timeseries) ([]byte, error)

// MarshalWriterFunc describes a function that marshals a Timeseries to an io.Writer
type MarshalWriterFunc func(Timeseries, io.Writer) error

// NewModeler factories a modeler with the provided modeling functions
func NewModeler(u UnmarshalerFunc, m MarshalerFunc, mw MarshalWriterFunc) *Modeler {
	return &Modeler{Unmarshaler: u, Marshaler: m, MarshalWriter: mw}
}
