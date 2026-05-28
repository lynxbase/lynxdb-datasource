package plugin

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// eventsToFrame builds a Grafana log-lines frame from LynxDB events.
// The configured time/message/level fields drive the required log columns; all
// remaining event fields are carried as JSON labels.
func (d *Datasource) eventsToFrame(events []map[string]interface{}) *data.Frame {
	n := len(events)
	times := make([]time.Time, 0, n)
	bodies := make([]string, 0, n)
	levels := make([]string, 0, n)
	ids := make([]string, 0, n)
	labels := make([]json.RawMessage, 0, n)

	timeField := d.settings.TimeField
	msgField := d.settings.MessageField
	lvlField := d.settings.LevelField

	for i, ev := range events {
		t, _ := toTime(firstNonNil(ev[timeField], ev["_time"], ev["_timestamp"], ev["time"]))
		times = append(times, t)

		body := toStringValue(firstNonNil(ev[msgField], ev["_raw"], ev["message"]))
		if body == "" {
			if b, err := json.Marshal(ev); err == nil {
				body = string(b)
			}
		}
		bodies = append(bodies, body)

		levels = append(levels, toStringValue(firstNonNil(ev[lvlField], ev["level"], ev["severity"])))

		id := toStringValue(firstNonNil(ev["id"], ev["_id"]))
		if id == "" {
			id = strconv.FormatInt(t.UnixNano(), 10) + "-" + strconv.Itoa(i)
		}
		ids = append(ids, id)

		lbl := make(map[string]interface{}, len(ev))
		for k, v := range ev {
			if k == timeField || k == msgField || k == lvlField {
				continue
			}
			lbl[k] = v
		}
		raw, err := json.Marshal(lbl)
		if err != nil {
			raw = []byte("{}")
		}
		labels = append(labels, json.RawMessage(raw))
	}

	frame := data.NewFrame("logs",
		data.NewField("timestamp", nil, times),
		data.NewField("body", nil, bodies),
		data.NewField("severity", nil, levels),
		data.NewField("id", nil, ids),
		data.NewField("labels", nil, labels),
	)
	frame.Meta = &data.FrameMeta{
		Type:                   data.FrameTypeLogLines,
		PreferredVisualization: data.VisTypeLogs,
		Custom:                 map[string]interface{}{"limit": d.settings.MaxLines},
	}
	return frame
}

// columnsRowsToFrame converts a columns/rows result into a frame. When
// timeFirst is true the first column becomes a time axis and the remaining
// columns become numeric series (time-series wide). Otherwise each column type
// is inferred and the result is a table.
func columnsRowsToFrame(columns []string, rows [][]interface{}, timeFirst bool) (*data.Frame, error) {
	if len(columns) == 0 {
		return data.NewFrame("response"), nil
	}
	ncol := len(columns)

	if timeFirst && ncol >= 1 {
		times := make([]time.Time, len(rows))
		series := make([][]*float64, ncol-1)
		for j := range series {
			series[j] = make([]*float64, len(rows))
		}
		for i, row := range rows {
			if len(row) > 0 {
				t, _ := toTime(row[0])
				times[i] = t
			}
			for j := 1; j < ncol && j < len(row); j++ {
				series[j-1][i] = toFloatPtr(row[j])
			}
		}
		fields := make([]*data.Field, 0, ncol)
		fields = append(fields, data.NewField(columns[0], nil, times))
		for j := 1; j < ncol; j++ {
			fields = append(fields, data.NewField(columns[j], nil, series[j-1]))
		}
		frame := data.NewFrame("response", fields...)
		frame.Meta = &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide}
		return frame, nil
	}

	fields := make([]*data.Field, ncol)
	for j := 0; j < ncol; j++ {
		col := make([]interface{}, len(rows))
		for i, row := range rows {
			if j < len(row) {
				col[i] = row[j]
			}
		}
		fields[j] = inferField(columns[j], col)
	}
	return data.NewFrame("response", fields...), nil
}

// inferField builds a typed, nullable field by classifying its values.
func inferField(name string, vals []interface{}) *data.Field {
	switch classifyColumn(vals) {
	case colTime:
		out := make([]*time.Time, len(vals))
		for i, v := range vals {
			if t, ok := toTimeStrict(v); ok {
				tt := t
				out[i] = &tt
			}
		}
		return data.NewField(name, nil, out)
	case colNumber:
		out := make([]*float64, len(vals))
		for i, v := range vals {
			out[i] = toFloatPtr(v)
		}
		return data.NewField(name, nil, out)
	default:
		out := make([]*string, len(vals))
		for i, v := range vals {
			if v != nil {
				s := toStringValue(v)
				out[i] = &s
			}
		}
		return data.NewField(name, nil, out)
	}
}

type colKind int

const (
	colString colKind = iota
	colNumber
	colTime
)

func classifyColumn(vals []interface{}) colKind {
	sawValue := false
	allNumber := true
	allTime := true
	for _, v := range vals {
		if v == nil {
			continue
		}
		sawValue = true
		if _, ok := toFloat(v); !ok {
			allNumber = false
		}
		if _, ok := toTimeStrict(v); !ok {
			allTime = false
		}
	}
	switch {
	case !sawValue:
		return colString
	case allNumber:
		return colNumber
	case allTime:
		return colTime
	default:
		return colString
	}
}

func firstNonNil(vals ...interface{}) interface{} {
	for _, v := range vals {
		if v != nil {
			if s, ok := v.(string); ok && s == "" {
				continue
			}
			return v
		}
	}
	return nil
}

// toFloat accepts only real numeric JSON types so numeric-looking strings stay
// strings during column classification.
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func toFloatPtr(v interface{}) *float64 {
	if f, ok := toFloat(v); ok {
		return &f
	}
	return nil
}

// toTime parses time from RFC3339 strings, time.Time, or epoch numbers.
func toTime(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case string:
		return parseTimeString(t)
	case float64:
		return epochToTime(t), true
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return epochToTime(f), true
		}
	}
	return time.Time{}, false
}

// toTimeStrict parses time only from strings or time.Time, never from numbers.
func toTimeStrict(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case string:
		return parseTimeString(t)
	}
	return time.Time{}, false
}

func parseTimeString(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if pt, err := time.Parse(layout, s); err == nil {
			return pt, true
		}
	}
	return time.Time{}, false
}

func epochToTime(f float64) time.Time {
	n := int64(f)
	switch {
	case f >= 1e18:
		return time.Unix(0, n)
	case f >= 1e15:
		return time.Unix(0, n*int64(time.Microsecond))
	case f >= 1e12:
		return time.Unix(0, n*int64(time.Millisecond))
	default:
		sec := int64(f)
		nsec := int64((f - float64(sec)) * float64(time.Second))
		return time.Unix(sec, nsec)
	}
}

func toStringValue(v interface{}) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return s
	case bool:
		return strconv.FormatBool(s)
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case json.Number:
		return s.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
