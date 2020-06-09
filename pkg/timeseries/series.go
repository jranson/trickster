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

import (
	"encoding/binary"

	"github.com/tricksterproxy/trickster/pkg/util/fnv"
)

// Series represents a single timeseries in a Result
type Series struct {
	// Header is the Series Header describing the Series
	Header *SeriesHeader
	// Points is the list of Points in the Series
	Points Points
}

// SeriesHeader is the header section of a series, and describes its
// shape, size, and attributes
type SeriesHeader struct {
	// Name is the name of the Series
	Name string `msg:"name"`
	// Tags is the map of tags associated with the Series
	Tags Tags `msg:"tags"`
	// // FieldsLookup is map to lookup the definition of a named field
	// FieldsLookup map[string]*FieldDefinition `msg:"-"`
	// FieldsList is the ordered list of fields in the Series
	FieldsList []*FieldDefinition `msg:"fields"`
	// TimestampIndex is the index of the TimeStamp field in the output when
	// it's time to serialize the DataSet for the wire
	TimestampIndex int `msg:"ti"`
	// QueryStatement is the original query to which this DataSet is associated
	QueryStatement string `msg:"query"`
	// Hash is the FNV64a Hash for the SeriesHeader
	Hash Hash `msg:"hash"`
	// Size is the memory utilization of the Header in bytes
	Size int `msg:"size"`
}

// Hash is a numeric value representing a calculated hash
type Hash uint64

// Hashes is a slice of type Hash
type Hashes []Hash

// SeriesLookup is a map of Series searchable by Series Header Hash
type SeriesLookup map[Hash]*Series

// CalculateHash sums the FNV64a hash for the Header and stores it to the Hash member
func (sh *SeriesHeader) CalculateHash() {
	hash := fnv.NewInlineFNV64a()
	hash.Write([]byte(sh.Name))
	hash.Write([]byte(sh.QueryStatement))
	for _, k := range sh.Tags.Keys() {
		hash.Write([]byte(k))
		hash.Write([]byte(sh.Tags[k]))
	}
	for _, fd := range sh.FieldsList {
		hash.Write([]byte(fd.Name))
		hash.Write([]byte{byte(fd.DataType)})
	}
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(sh.TimestampIndex))
	hash.Write(b)
	sh.Hash = Hash(hash.Sum64())
}

// Clone returns a perfect, new copy of the SeriesHeader
func (sh *SeriesHeader) Clone() *SeriesHeader {
	clone := &SeriesHeader{
		Name:       sh.Name,
		Tags:       sh.Tags.Clone(),
		FieldsList: make([]*FieldDefinition, len(sh.FieldsList)),
		//FieldsLookup:   make(map[string]*FieldDefinition),
		TimestampIndex: sh.TimestampIndex,
		QueryStatement: sh.QueryStatement,
		Hash:           sh.Hash,
		Size:           sh.Size,
	}
	for i, fd := range sh.FieldsList {
		clone.FieldsList[i] = fd.Clone()
		//clone.FieldsLookup[fd.Name] = clone.FieldsList[i]
	}
	return clone
}

// Size returns the memory utilization of the Series in bytes
func (s *Series) Size() int {
	c := 8
	if s.Header != nil {
		c += s.Header.Size
	}
	if s.Points != nil {
		c += s.Points.Size()
	}
	return c
}

// Clone returns a perfect, new copy of the Series
func (s *Series) Clone() *Series {
	clone := &Series{}
	if s.Header != nil {
		clone.Header = s.Header.Clone()
	}
	if s.Points != nil {
		clone.Points = s.Points.Clone(clone.Header)
	}
	return clone
}
