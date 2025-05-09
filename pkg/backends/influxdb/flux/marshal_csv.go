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
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/trickstercache/trickster/v2/pkg/observability/logging"
	"github.com/trickstercache/trickster/v2/pkg/observability/logging/logger"
	"github.com/trickstercache/trickster/v2/pkg/proxy/headers"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"
	"github.com/trickstercache/trickster/v2/pkg/timeseries/dataset"
	"github.com/trickstercache/trickster/v2/pkg/timeseries/epoch"
	"github.com/trickstercache/trickster/v2/pkg/util/sets"
)

// type marshaler func(*dataset.DataSet, *JSONRequestBody, int, io.Writer) error

const tableColumnName = "table"
const startColumnName = "_start"
const stopColumnName = "_stop"

func marshalTimeseriesCSVWriter(ds *dataset.DataSet, frb *JSONRequestBody,
	status int, w io.Writer) error {

	if hw, ok := w.(http.ResponseWriter); ok {
		hw.Header().Set(headers.NameContentType, headers.ValueApplicationCSV)
		hw.WriteHeader(status)
	}

	fds, tagFields, _, _ := ds.FieldDefinitions()

	// TODO: Ensure TRQ is always the user request (eg., the one with THEIR extents)

	var wantDatatypeAnnotation, wantGroupAnnotation, wantDefaultsAnnotation bool
	for _, s := range frb.Dialect.Annotations {
		switch s {
		case AnnotationDatatype:
			wantDatatypeAnnotation = true
		case AnnotationGroup:
			wantGroupAnnotation = true
		case AnnotationDefault:
			wantDefaultsAnnotation = true
		}
	}
	if wantDatatypeAnnotation {
		if err := printCsvDatatypeAnnotationRow(w, fds); err != nil {
			return err
		}
	}
	if wantGroupAnnotation {
		if err := printCsvGroupAnnotationRow(w, fds, tagFields); err != nil {
			return err
		}
	}
	if wantDefaultsAnnotation {
		if err := printCsvDefaultAnnotationRow(w, fds); err != nil {
			return err
		}
	}
	if frb.Dialect.Header == nil || *frb.Dialect.Header {
		if err := printCsvHeaderRow(w, fds); err != nil {
			return err
		}
	}
	if err := printCsvDataRows(w, ds, fds, ds.TimeRangeQuery.Extent); err != nil {
		return err
	}
	return nil
}

func assignTableOrders(ds *dataset.DataSet, e timeseries.Extent) {
	var k int
	for _, r := range ds.Results {
		for _, s := range r.SeriesList {
			for i := range s.Header.UntrackedFieldsList {
				switch s.Header.UntrackedFieldsList[i].Name {
				case tableColumnName:
					s.Header.UntrackedFieldsList[i].DefaultValue = strconv.Itoa(k)
					k++
				case startColumnName:
					s.Header.UntrackedFieldsList[i].DefaultValue =
						epoch.FormatTime(e.Start, s.Header.UntrackedFieldsList[i].DataType, false)
				case stopColumnName:
					s.Header.UntrackedFieldsList[i].DefaultValue =
						epoch.FormatTime(e.End, s.Header.UntrackedFieldsList[i].DataType, false)
				}
			}

		}
	}
}

func printCsvDatatypeAnnotationRow(w io.Writer,
	fds timeseries.FieldDefinitions) error {
	cells := make([]string, len(fds)+2)
	cells[0] = "#datatype"
	cells[1] = "string"
	i := 2
	for _, fd := range fds {
		cells[i] = fieldTypeToFluxType(fd.DataType)
		i++
	}
	_, err := w.Write([]byte(strings.Join(cells, ",") + "\n"))
	return err
}

func printCsvGroupAnnotationRow(w io.Writer,
	fds, tagFields timeseries.FieldDefinitions) error {
	cells := make([]string, len(fds)+2)
	cells[0] = "#group"
	cells[1] = "false"
	i := 2
	tags := sets.NewStringSet()
	tags.AddAll([]string{"_start", "_stop"})
	for _, fd := range tagFields {
		tags.Add(fd.Name)
	}
	for _, fd := range fds {
		if tags.Contains(fd.Name) {
			cells[i] = "true"
		} else {
			cells[i] = "false"
		}
		i++
	}
	_, err := w.Write([]byte(strings.Join(cells, ",") + "\n"))
	return err
}

func printCsvDefaultAnnotationRow(w io.Writer,
	fds timeseries.FieldDefinitions) error {
	cells := make([]string, len(fds)+2)
	cells[0] = "#default"
	cells[1] = "_result"
	i := 2
	for _, fd := range fds {
		cells[i] = fd.DefaultValue
		i++
	}
	_, err := w.Write([]byte(strings.Join(cells, ",") + "\n"))
	return err
}

func printCsvHeaderRow(w io.Writer, fds timeseries.FieldDefinitions) error {
	cells := make([]string, len(fds)+2)
	cells[1] = "result"
	i := 2
	for _, fd := range fds {
		cells[i] = fd.Name
		i++
	}
	_, err := w.Write([]byte(strings.Join(cells, ",") + "\n"))
	return err
}

func printCsvDataRows(w io.Writer, ds *dataset.DataSet,
	fds timeseries.FieldDefinitions, e timeseries.Extent) error {
	// we have to make the rows first, and then sort them by time, then table #
	// flux csv results are not sorted by series / tags
	rows := make([][]string, ds.PointCount())
	var k int
	assignTableOrders(ds, e)
	for _, r := range ds.Results {
		for _, s := range r.SeriesList {
			for _, p := range s.Points {
				if row, ok := processCsvDataRow(fds, s, p); ok {
					rows[k] = row
					k++
				}
			}
		}
	}
	rows = rows[:k]
	// TODO: SORT BY TIME THEN TABLE
	cw := csv.NewWriter(w)
	var err error
	for _, row := range rows {
		err = cw.Write(row)
		if err != nil {
			logger.Error("failed to write csv row", logging.Pairs{"error": err})
		}
	}
	cw.Flush()
	return nil
}

func processCsvDataRow(fds timeseries.FieldDefinitions, s *dataset.Series,
	p dataset.Point) ([]string, bool) {
	row := make([]string, len(fds)+2)
	fdl := fds.ToLookup()
	// TODO: how to fill in result if not default? in row[1]
	for k, v := range s.Header.Tags {
		if fd, ok := fdl[k]; ok && fd.OutputPosition > 2 {
			row[fd.OutputPosition] = v
		}
	}
	for _, fd := range s.Header.UntrackedFieldsList {
		switch fd.Name {
		case tableColumnName:
			row[fd.OutputPosition] = getTableID(s)
		case startColumnName, stopColumnName:
			row[fd.OutputPosition] = fd.DefaultValue
		}
	}
	if strings.HasSuffix(s.Header.TimestampField.Name, "time") {
		row[s.Header.TimestampField.OutputPosition] = p.Epoch.Format(s.Header.TimestampField.DataType, false)
	}
	for i, fd := range s.Header.ValueFieldsList {
		// TODO: print it based on the underlying type.
		row[fd.OutputPosition] = fmt.Sprintf("%v", p.Values[i])
	}
	return row, true
}

func getTableID(s *dataset.Series) string {
	if s == nil || len(s.Header.UntrackedFieldsList) == 0 {
		return "0"
	}
	for _, fd := range s.Header.UntrackedFieldsList {
		if fd.Name == tableColumnName {
			if fd.DefaultValue != "" {
				return fd.DefaultValue
			}
			return "0"
		}
	}
	return "0"
}
