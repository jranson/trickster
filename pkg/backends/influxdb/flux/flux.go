package flux

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/trickstercache/trickster/v2/pkg/proxy/errors"
	"github.com/trickstercache/trickster/v2/pkg/proxy/urls"
	"github.com/trickstercache/trickster/v2/pkg/timeseries"
	"github.com/trickstercache/trickster/v2/pkg/util/timeconv"
)

const RangeFunction = "|> range("
const AggWindowFunc = "|> aggregateWindow("
const WindowFunc = "|> window("
const EveryToken = "every:"
const StartToken = "start:"
const StopToken = ",stop:"
const TimeRangeTokenPlaceholder = "<TIMERANGE_TOKEN>"

type Query struct {
	original  string
	tokenized string
	step      time.Duration
	extent    timeseries.Extent
}

type RequestBody struct {
	Query string `json:"query"`
	Type  string `json:"type"`
}

func ParseTimeRangeQuery(r *http.Request, b []byte,
	trq *timeseries.TimeRangeQuery, rlo *timeseries.RequestOptions) error {
	frb := &RequestBody{}
	err := json.Unmarshal(b, frb)
	if err != nil {
		return err
	}
	if frb.Type != "flux" || frb.Query == "" {
		return errors.MissingRequestParam("query")
	}
	trq.Statement = frb.Query
	tokenizedStmt, extent, step, err := ParseQuery(frb.Query)
	if err != nil {
		return err
	}
	q := &Query{
		original:  frb.Query,
		tokenized: tokenizedStmt,
		step:      step,
		extent:    extent,
	}
	trq.ParsedQuery = q
	trq.Step = step
	trq.Statement = tokenizedStmt
	trq.TemplateURL = urls.Clone(r.URL)
	trq.Extent = extent
	return nil
}

func SetExtent(r *http.Request, trq *timeseries.TimeRangeQuery,
	extent *timeseries.Extent, q *Query) {
}

func ParseQuery(input string) (string, timeseries.Extent, time.Duration, error) {
	var e timeseries.Extent
	var d time.Duration
	var err error
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		ri := strings.Index(line, RangeFunction)
		switch {
		case ri >= 0:
			e, err = parseRange(line)
			if err != nil {
				return "", e, d, err
			}
			lines[i] = tokenizeRangeLine(line, ri)
		case strings.Contains(line, AggWindowFunc),
			strings.Contains(line, WindowFunc):
			d, err = parseStep(line)
			if err != nil {
				return "", e, d, err
			}
		}
	}
	return strings.Join(lines, "\n"), e, d, err
}

func parseStep(input string) (time.Duration, error) {
	i := strings.Index(input, EveryToken)
	if i < 0 {
		return 0, nil // TOOD: correct error here
	}
	i += 6
	input = input[i:]
	i = strings.Index(input, ",")
	j := strings.Index(input, ")")
	if i >= 0 && i < j {
		input = strings.TrimSpace(input[:i])
	} else if j >= 0 {
		input = strings.TrimSpace(input[:j])
	} else {
		return 0, nil // TOOD: correct error here
	}
	return time.ParseDuration(input)
}

func parseRange(input string) (timeseries.Extent, error) {
	var e timeseries.Extent
	input = strings.ReplaceAll(input, " ", "")
	i := strings.Index(input, StartToken)
	if i < 0 {
		return e, nil // TODO: correct error here
	}
	i += 6
	input = input[i:]
	i = strings.Index(input, StopToken)
	if i < 0 {
		return e, nil // TODO: correct error here
	}
	input = strings.TrimSuffix(strings.ReplaceAll(input, StopToken, ","), ")")
	parts := strings.Split(input, ",")
	if len(parts) != 2 {
		return e, nil // TODO: correct error here
	}
	var err error
	e.Start, err = tryParseTimeField(parts[0])
	if err != nil {
		return e, err
	}
	e.End, err = tryParseTimeField(parts[1])
	if err != nil {
		return e, err
	}
	return e, nil
}

func tryParseTimeField(s string) (time.Time, error) {
	var t time.Time
	var erd, eat, eut error
	if t, erd = tryParseRelativeDuration(s); erd == nil {
		return t, nil
	}
	if t, eat = tryParseAbsoluteTime(s); eat == nil {
		return t, nil
	}
	if t, eut = tryParseUnixTimestamp(s); eut == nil {
		return t, nil
	}
	return time.Time{}, nil // TODO: correct error here
}

func tryParseRelativeDuration(s string) (time.Time, error) {
	d, err := timeconv.ParseDuration(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(d), nil
}

func tryParseAbsoluteTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func tryParseUnixTimestamp(s string) (time.Time, error) {
	unix, err := strconv.Atoi(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(unix), 0).UTC(), nil
}

func tokenizeRangeLine(input string, funcStart int) string {
	i := strings.Index(input[funcStart:], ")")
	if i < 0 {
		return input
	}
	return input[:funcStart+len(RangeFunction)] + TimeRangeTokenPlaceholder + input[funcStart+i:]
}
