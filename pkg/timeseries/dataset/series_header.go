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

//go:generate go tool msgp

package dataset

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/trickstercache/trickster/v2/pkg/checksum/fnv"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"
)

// SeriesHeader is the header section of a series, and describes its
// shape, size, and attributes
type SeriesHeader struct {
	// Name is the name of the Series
	Name string `msg:"name"`
	// Tags is the map of tags associated with the Series. Each key will map to
	// a fd.Name in TagFieldsList, with values representing the specific tag
	// values for this Series.
	Tags Tags `msg:"tags"`
	// TimestampField is the Field Definitions for the timestamp field.
	// Optional and used by some providers. TODO: use by more/all TSDB providers
	TimestampField timeseries.FieldDefinition `msg:"timestampField"`
	// TagFieldsList is the ordered list of tag-based Field Definitions in the
	// Series. Optional and used by some providers. TODO: use by more/all TSDB providers
	TagFieldsList timeseries.FieldDefinitions `msg:"tagFields"`
	// FieldsList is the ordered list of value-based Field Definitions in the Series.
	ValueFieldsList timeseries.FieldDefinitions `msg:"fields"`
	// TimestampIndex is the index of the TimeStamp field in the output when
	// it's time to serialize the DataSet for the wire
	TimestampIndex uint64 `msg:"ti"` // TODO: DO WE NEED THIS? We have TFD so it seems duplicative.
	// QueryStatement is the original query to which this DataSet is associated
	QueryStatement string `msg:"query"`
	// Size is the memory utilization of the Header in bytes
	Size int `msg:"size"`

	hash Hash
}

// CalculateHash sums the FNV64a hash for the Header and stores it to the Hash member
func (sh *SeriesHeader) CalculateHash() Hash {
	if sh.hash > 0 {
		return sh.hash
	}
	hash := fnv.NewInlineFNV64a()
	hash.Write([]byte(sh.Name))
	hash.Write([]byte(sh.QueryStatement))
	for _, k := range sh.Tags.Keys() {
		hash.Write([]byte(k))
		hash.Write([]byte(sh.Tags[k]))
	}
	for _, fd := range sh.ValueFieldsList {
		hash.Write([]byte(fd.Name))
		hash.Write([]byte{byte(fd.DataType)})
	}
	for _, fd := range sh.TagFieldsList {
		hash.Write([]byte(fd.Name))
		hash.Write([]byte{byte(fd.DataType)})
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, sh.TimestampIndex)
	hash.Write(b)
	sh.hash = Hash(hash.Sum64())
	return sh.hash
}

// Clone returns a perfect, new copy of the SeriesHeader
func (sh *SeriesHeader) Clone() SeriesHeader {
	clone := SeriesHeader{
		Name:            sh.Name,
		Tags:            sh.Tags.Clone(),
		ValueFieldsList: make([]timeseries.FieldDefinition, len(sh.ValueFieldsList)),
		TagFieldsList:   make([]timeseries.FieldDefinition, len(sh.TagFieldsList)),
		TimestampIndex:  sh.TimestampIndex,
		QueryStatement:  sh.QueryStatement,
		Size:            sh.Size,
		hash:            sh.hash,
	}
	copy(clone.ValueFieldsList, sh.ValueFieldsList)
	copy(clone.TagFieldsList, sh.TagFieldsList)
	return clone
}

// CalculateSize sets and returns the header size
func (sh *SeriesHeader) CalculateSize() int {
	c := len(sh.Name) + sh.Tags.Size() + 8 + len(sh.QueryStatement) + 28
	for i := range sh.ValueFieldsList {
		c += len(sh.ValueFieldsList[i].Name) + 17
	}
	for i := range sh.TagFieldsList {
		c += len(sh.TagFieldsList[i].Name) + 17
	}
	sh.Size = c
	return c
}

func (sh *SeriesHeader) String() string {
	sb := &strings.Builder{}
	sb.WriteByte('{')
	if sh.Name != "" {
		fmt.Fprintf(sb, `"name":"%s",`, sh.Name)
	}
	if sh.QueryStatement != "" {
		fmt.Fprintf(sb, `"query":"%s",`, sh.QueryStatement)
	}
	if len(sh.Tags) > 0 {
		fmt.Fprintf(sb, `"tags":"%s",`, sh.Tags.String())
	}
	if len(sh.ValueFieldsList) > 0 {
		sb.WriteString(`"fields":[`)
		l := len(sh.ValueFieldsList)
		for i, fd := range sh.ValueFieldsList {
			fmt.Fprintf(sb, `"%s"`, fd.Name)
			if i < l-1 {
				sb.WriteByte(',')
			}
		}
		sb.WriteString("],")
	}
	if len(sh.TagFieldsList) > 0 {
		sb.WriteString(`"tagsList":[`)
		l := len(sh.TagFieldsList)
		for i, fd := range sh.TagFieldsList {
			fmt.Fprintf(sb, `"%s"`, fd.Name)
			if i < l-1 {
				sb.WriteByte(',')
			}
		}
		sb.WriteString("],")
	}
	sb.WriteString(`"timestampIndex":` + strconv.FormatUint(sh.TimestampIndex, 10))
	sb.WriteByte('}')
	return sb.String()
}
