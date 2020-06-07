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

// Field Data Types
const (
	Int64 FieldDataType = iota
	Float64
	String
	Bool
	Byte
	Int16
)

// FieldDataType is a byte representing the data type of a Field
// when stored in a Point's Values list
type FieldDataType byte

// FieldDefinition describes a field by name and type
type FieldDefinition struct {
	Name     string
	DataType FieldDataType
}

// Clone returns a perfect, new copy of the FieldDefinition
func (fd *FieldDefinition) Clone() *FieldDefinition {
	return &FieldDefinition{
		Name:     fd.Name,
		DataType: fd.DataType,
	}
}
