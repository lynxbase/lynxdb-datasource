package plugin

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Integration tests run only when LYNXDB_TEST_URL points at a live LynxDB
// (for example `lynxdb demo`). They exercise the real client and frame
// conversion against the API.

func liveDatasource(t *testing.T) *Datasource {
	t.Helper()
	url := os.Getenv("LYNXDB_TEST_URL")
	if url == "" {
		t.Skip("set LYNXDB_TEST_URL to run integration tests")
	}
	ds, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{URL: url})
	if err != nil {
		t.Fatalf("new datasource: %v", err)
	}
	return ds.(*Datasource)
}

func recentRange() backend.TimeRange {
	now := time.Now()
	return backend.TimeRange{From: now.Add(-30 * time.Minute), To: now}
}

func TestIntegrationHealth(t *testing.T) {
	d := liveDatasource(t)
	res, err := d.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != backend.HealthStatusOk {
		t.Fatalf("health not ok: %s", res.Message)
	}
	t.Logf("health: %s", res.Message)
}

func TestIntegrationQueryEvents(t *testing.T) {
	d := liveDatasource(t)
	resp := d.query(context.Background(), backend.DataQuery{
		RefID:     "A",
		JSON:      []byte(`{"queryText":"*","queryType":"logs","maxLines":5}`),
		TimeRange: recentRange(),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	if len(resp.Frames) == 0 {
		t.Fatal("no frames returned")
	}
	f := resp.Frames[0]
	if f.Meta == nil || f.Meta.Type != data.FrameTypeLogLines {
		t.Fatalf("expected log-lines frame, got %+v", f.Meta)
	}
	if len(f.Fields) != 5 {
		t.Fatalf("expected 5 log fields, got %d", len(f.Fields))
	}
	t.Logf("events frame rows: %d", f.Fields[0].Len())
}

func TestIntegrationTimechartGrouped(t *testing.T) {
	d := liveDatasource(t)
	resp := d.query(context.Background(), backend.DataQuery{
		RefID:     "A",
		JSON:      []byte(`{"queryText":"* | timechart count() span=1m by level","queryType":"metrics"}`),
		TimeRange: recentRange(),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	if len(resp.Frames) == 0 {
		t.Fatal("no frames returned")
	}
	f := resp.Frames[0]
	if f.Meta == nil || f.Meta.Type != data.FrameTypeTimeSeriesWide {
		t.Fatalf("expected wide time series, got %+v", f.Meta)
	}
	if len(f.Fields) < 2 {
		t.Fatalf("expected time + at least one series, got %d fields", len(f.Fields))
	}
	t.Logf("timechart fields: %d, rows: %d", len(f.Fields), f.Fields[0].Len())
}

func TestIntegrationLogVolume(t *testing.T) {
	d := liveDatasource(t)
	r := recentRange()
	resp := d.queryLogVolume(
		context.Background(),
		queryModel{QueryText: "*", QueryType: queryTypeLogVolume},
		r.From.UTC().Format(time.RFC3339Nano),
		r.To.UTC().Format(time.RFC3339Nano),
		time.Minute,
	)
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	if len(resp.Frames) == 0 {
		t.Fatal("no log-volume frames returned")
	}
	t.Logf("log volume fields: %d", len(resp.Frames[0].Fields))
}

type captureSender struct {
	resp *backend.CallResourceResponse
}

func (c *captureSender) Send(r *backend.CallResourceResponse) error {
	c.resp = r
	return nil
}

func TestIntegrationResources(t *testing.T) {
	d := liveDatasource(t)
	for _, path := range []string{"fields", "sources"} {
		cs := &captureSender{}
		err := d.CallResource(context.Background(), &backend.CallResourceRequest{
			Path:   path,
			Method: "GET",
			URL:    path,
		}, cs)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if cs.resp == nil || cs.resp.Status != 200 {
			t.Fatalf("%s: unexpected response %+v", path, cs.resp)
		}
	}

	cs := &captureSender{}
	if err := d.CallResource(context.Background(), &backend.CallResourceRequest{
		Path:   "explain",
		Method: "GET",
		URL:    "explain?q=" + "level%3Derror",
	}, cs); err != nil {
		t.Fatal(err)
	}
	if cs.resp.Status != 200 {
		t.Fatalf("explain status %d", cs.resp.Status)
	}
}
