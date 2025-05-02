package influxql

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/proxy/errors"
	"github.com/trickstercache/trickster/v2/pkg/proxy/headers"
	"github.com/trickstercache/trickster/v2/pkg/proxy/methods"
	"github.com/trickstercache/trickster/v2/pkg/proxy/params"
	"github.com/trickstercache/trickster/v2/pkg/proxy/urls"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"

	"github.com/influxdata/influxql"
)

var epochToFlag = map[string]byte{
	"ns": 1,
	"u":  2, "µ": 2,
	"ms": 3,
	"s":  4,
	"m":  5,
	"h":  6,
}

// Common URL Parameter Names
const (
	ParamQuery   = "q"
	ParamDB      = "db"
	ParamEpoch   = "epoch"
	ParamPretty  = "pretty"
	ParamChunked = "chunked"
)

func ParseTimeRangeQuery(r *http.Request, b []byte,
	values url.Values, trq *timeseries.TimeRangeQuery,
	rlo *timeseries.RequestOptions) (*timeseries.TimeRangeQuery,
	*timeseries.RequestOptions, bool, error) {
	if trq.Statement == "" {
		return nil, nil, false, errors.MissingURLParam(ParamQuery)
	}

	var valuer = &influxql.NowValuer{Now: time.Now()}

	if b, ok := epochToFlag[values.Get(ParamEpoch)]; ok {
		rlo.TimeFormat = b
	}

	if values.Get(ParamPretty) == "true" {
		rlo.OutputFormat = 1
	} else if r != nil && r.Header != nil &&
		r.Header.Get(headers.NameAccept) == headers.ValueApplicationCSV {
		rlo.OutputFormat = 2
	}

	var cacheError error

	p := influxql.NewParser(strings.NewReader(trq.Statement))
	q, err := p.ParseQuery()
	if err != nil {
		return nil, nil, false, err
	}

	trq.Step = -1
	var hasTimeQueryParts bool
	statements := make([]string, 0, len(q.Statements))
	var canObjectCache bool
	for _, v := range q.Statements {
		sel, ok := v.(*influxql.SelectStatement)
		if !ok || sel.Condition == nil {
			cacheError = errors.ErrNotTimeRangeQuery
		} else {
			canObjectCache = true
		}
		step, err := sel.GroupByInterval()
		if err != nil {
			cacheError = err
		} else {
			if trq.Step == -1 && step > 0 {
				trq.Step = step
			} else if trq.Step != step {
				// this condition means multiple queries were present, and had
				// different step widths
				cacheError = errors.ErrStepParse
			}
		}
		_, tr, err := influxql.ConditionExpr(sel.Condition, valuer)
		if err != nil {
			cacheError = err
		}

		// this section determines the time range of the query
		ex := timeseries.Extent{Start: tr.Min, End: tr.Max}
		if ex.Start.IsZero() {
			ex.Start = time.Unix(0, 0)
		}
		if ex.End.IsZero() {
			ex.End = time.Now()
		}
		if trq.Extent.Start.IsZero() {
			trq.Extent = ex
		} else if trq.Extent != ex {
			// this condition means multiple queries were present, and had
			// different time ranges
			cacheError = errors.ErrNotTimeRangeQuery
		}

		// this sets a zero time range for normalizing the query for cache key hashing
		sel.SetTimeRange(time.Time{}, time.Time{})
		statements = append(statements, sel.String())

		hasTimeQueryParts = true
	}

	if !hasTimeQueryParts {
		cacheError = errors.ErrNotTimeRangeQuery
	}

	// this field is used as part of the data that calculates the cache key
	trq.Statement = strings.Join(statements, " ; ")
	trq.ParsedQuery = q
	trq.TemplateURL = urls.Clone(r.URL)
	trq.CacheKeyElements = map[string]string{
		ParamQuery: trq.Statement,
	}
	qt := url.Values(http.Header(values).Clone())
	qt.Set(ParamQuery, trq.Statement)

	// Swap in the Tokenzed Query in the Url Params
	trq.TemplateURL.RawQuery = qt.Encode()

	if cacheError != nil {
		return trq, rlo, true, cacheError
	}

	return trq, rlo, canObjectCache, nil
}

func SetExtent(r *http.Request, trq *timeseries.TimeRangeQuery,
	extent *timeseries.Extent, q *influxql.Query) {
	for _, s := range q.Statements {
		if sel, ok := s.(*influxql.SelectStatement); ok {
			// since setting timerange results in a clause of '>= start AND < end', we add the
			// size of 1 step onto the end time so as to ensure it is included in the results
			sel.SetTimeRange(extent.Start, extent.End.Add(trq.Step))
		}
	}
	uq := q.String()
	v, _, isBody := params.GetRequestValues(r)
	v.Set(ParamQuery, uq)
	if isBody {
		r.Body = io.NopCloser(strings.NewReader(uq))
		r.ContentLength = int64(len(uq))
	}
	v.Set(ParamEpoch, "ns") // request nanosecond epoch timestamp format from server
	v.Del(ParamChunked)     // we do not support chunked output or handling chunked server responses
	v.Del(ParamPretty)
	if !methods.HasBody(r.Method) {
		r.URL.RawQuery = v.Encode()
	}
	// Need to set template URL for cache key derivation
	trq.TemplateURL = urls.Clone(r.URL)
	qt := url.Values(http.Header(v).Clone())
	trq.TemplateURL.RawQuery = qt.Encode()
}

/*
	var uq string
	if q, ok := trq.ParsedQuery.(*influxql.Query); ok {
		for _, s := range q.Statements {
			if sel, ok := s.(*influxql.SelectStatement); ok {
				// since setting timerange results in a clause of '>= start AND < end', we add the
				// size of 1 step onto the end time so as to ensure it is included in the results
				sel.SetTimeRange(extent.Start, extent.End.Add(trq.Step))
			}
		}
		uq = q.String()
		// } else if trq.ProviderData1 == flux.LanuageFlux {
		// 	q.SetExtent(*extent)
		// 	uq = q.String()
		// } else {
		// 	return
	}

	v.Set(ti.ParamQuery, uq)
	if isBody {
		r.Body = io.NopCloser(strings.NewReader(uq))
		r.ContentLength = int64(len(uq))
	}
	v.Set(ti.ParamEpoch, "ns") // request nanosecond epoch timestamp format from server
	v.Del(ti.ParamChunked)     // we do not support chunked output or handling chunked server responses
	v.Del(ti.ParamPretty)
	if !methods.HasBody(r.Method) {
		r.URL.RawQuery = v.Encode()
	}
	// Need to set template URL for cache key derivation
	trq.TemplateURL = urls.Clone(r.URL)
	qt := url.Values(http.Header(v).Clone())
	trq.TemplateURL.RawQuery = qt.Encode()
*/
