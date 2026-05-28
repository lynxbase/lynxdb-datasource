package plugin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/lynxbase/lynxdb-datasource/pkg/models"
)

func testDatasource() *Datasource {
	return &Datasource{
		settings: &models.PluginSettings{
			TimeField:    models.DefaultTimeField,
			MessageField: models.DefaultMessageField,
			LevelField:   models.DefaultLevelField,
			MaxLines:     models.DefaultMaxLines,
		},
	}
}

func TestEventsToFrame(t *testing.T) {
	d := testDatasource()
	events := []map[string]interface{}{
		{
			"_time": "2026-05-28T10:00:00Z",
			"_raw":  "boom",
			"level": "ERROR",
			"host":  "web-01",
		},
	}

	frame := d.eventsToFrame(events)

	if got := len(frame.Fields); got != 5 {
		t.Fatalf("expected 5 fields, got %d", got)
	}
	if frame.Meta == nil || frame.Meta.Type != data.FrameTypeLogLines {
		t.Fatalf("expected log-lines frame type, got %+v", frame.Meta)
	}
	if frame.Meta.PreferredVisualization != data.VisTypeLogs {
		t.Fatalf("expected logs visualization, got %q", frame.Meta.PreferredVisualization)
	}

	wantTime := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	if got := frame.Fields[0].At(0).(time.Time); !got.Equal(wantTime) {
		t.Errorf("timestamp = %v, want %v", got, wantTime)
	}
	if got := frame.Fields[1].At(0).(string); got != "boom" {
		t.Errorf("body = %q, want boom", got)
	}
	if got := frame.Fields[2].At(0).(string); got != "ERROR" {
		t.Errorf("severity = %q, want ERROR", got)
	}
	labels := string(frame.Fields[4].At(0).(json.RawMessage))
	if !contains(labels, "web-01") || contains(labels, "boom") {
		t.Errorf("labels = %q, expected host but not consumed fields", labels)
	}
}

func TestColumnsRowsToFrameTimechart(t *testing.T) {
	cols := []string{"_time", "count"}
	rows := [][]interface{}{
		{"2026-05-28T10:00:00Z", float64(5)},
		{"2026-05-28T10:01:00Z", float64(9)},
	}

	frame, err := columnsRowsToFrame(cols, rows, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(frame.Fields))
	}
	if frame.Fields[0].Type() != data.FieldTypeTime {
		t.Errorf("first field type = %v, want time", frame.Fields[0].Type())
	}
	if frame.Fields[1].Type() != data.FieldTypeNullableFloat64 {
		t.Errorf("value field type = %v, want nullable float64", frame.Fields[1].Type())
	}
	if v := frame.Fields[1].At(1).(*float64); v == nil || *v != 9 {
		t.Errorf("value[1] = %v, want 9", v)
	}
}

func TestColumnsRowsToFrameTimechartGrouped(t *testing.T) {
	cols := []string{"_time", "count", "level"}
	rows := [][]interface{}{
		{"2026-05-28T10:00:00Z", float64(5), "ERROR"},
		{"2026-05-28T10:00:00Z", float64(9), "INFO"},
		{"2026-05-28T10:01:00Z", float64(2), "ERROR"},
		{"2026-05-28T10:01:00Z", float64(7), "INFO"},
	}

	frame, err := columnsRowsToFrame(cols, rows, true)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Meta == nil || frame.Meta.Type != data.FrameTypeTimeSeriesWide {
		t.Fatalf("expected wide frame after pivot, got %+v", frame.Meta)
	}
	// One time field plus one value field per level value.
	if len(frame.Fields) != 3 {
		t.Fatalf("expected 3 fields (time + 2 levels), got %d", len(frame.Fields))
	}
	if frame.Fields[0].Type() != data.FieldTypeTime {
		t.Errorf("first field type = %v, want time", frame.Fields[0].Type())
	}
	if frame.Fields[0].Len() != 2 {
		t.Errorf("expected 2 time buckets, got %d", frame.Fields[0].Len())
	}
}

func TestColumnsRowsToFrameTable(t *testing.T) {
	cols := []string{"host", "count"}
	rows := [][]interface{}{
		{"web-01", float64(150)},
		{"web-02", float64(230)},
	}

	frame, err := columnsRowsToFrame(cols, rows, false)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Fields[0].Type() != data.FieldTypeNullableString {
		t.Errorf("host field type = %v, want nullable string", frame.Fields[0].Type())
	}
	if frame.Fields[1].Type() != data.FieldTypeNullableFloat64 {
		t.Errorf("count field type = %v, want nullable float64", frame.Fields[1].Type())
	}
}

func TestLintsToNotices(t *testing.T) {
	notices := lintsToNotices([]queryLint{
		{Code: "L035", Message: "scans everything", Severity: "warning"},
		{Code: "L013", Message: "use count()", Severity: "notice"},
		{Message: "bad", Severity: "error"},
	})
	if len(notices) != 3 {
		t.Fatalf("expected 3 notices, got %d", len(notices))
	}
	if notices[0].Severity != data.NoticeSeverityWarning {
		t.Errorf("notice[0] severity = %v, want warning", notices[0].Severity)
	}
	if notices[0].Text != "L035: scans everything" {
		t.Errorf("notice[0] text = %q", notices[0].Text)
	}
	if notices[1].Severity != data.NoticeSeverityInfo {
		t.Errorf("notice[1] severity = %v, want info", notices[1].Severity)
	}
	if notices[2].Severity != data.NoticeSeverityError {
		t.Errorf("notice[2] severity = %v, want error", notices[2].Severity)
	}
}

func TestEpochToTime(t *testing.T) {
	want := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	cases := []float64{
		float64(want.Unix()),
		float64(want.UnixMilli()),
	}
	for _, c := range cases {
		if got := epochToTime(c).UTC(); !got.Equal(want) {
			t.Errorf("epochToTime(%v) = %v, want %v", c, got, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
