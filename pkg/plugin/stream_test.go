package plugin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type captureFrameSender struct {
	frames []*data.Frame
}

func (c *captureFrameSender) SendFrame(f *data.Frame, _ data.FrameInclude) error {
	c.frames = append(c.frames, f)
	return nil
}

func TestStreamSSEParsing(t *testing.T) {
	d := testDatasource()
	sse := strings.Join([]string{
		"event: result",
		`data: {"_time":"2026-05-28T10:00:00Z","_raw":"first","level":"ERROR"}`,
		"",
		"event: catchup_done",
		`data: {"count":1}`,
		"",
		"event: heartbeat",
		`data: {"ts":"2026-05-28T10:00:05Z"}`,
		"",
		"event: result",
		`data: {"_time":"2026-05-28T10:00:01Z","_raw":"second"}`,
		"",
		"event: close",
		`data: {"reason":"done"}`,
		"",
	}, "\n")

	cs := &captureFrameSender{}
	if err := d.streamSSE(context.Background(), strings.NewReader(sse), cs); err != nil {
		t.Fatal(err)
	}

	if len(cs.frames) != 2 {
		t.Fatalf("expected 2 result frames, got %d", len(cs.frames))
	}
	if cs.frames[0].Meta == nil || cs.frames[0].Meta.Type != data.FrameTypeLogLines {
		t.Fatalf("expected log-lines frame, got %+v", cs.frames[0].Meta)
	}
	if got := cs.frames[0].Fields[1].At(0).(string); got != "first" {
		t.Errorf("body = %q, want first", got)
	}
}

func TestStreamSSEError(t *testing.T) {
	d := testDatasource()
	sse := "event: error\ndata: {\"error\":\"boom\"}\n\n"
	cs := &captureFrameSender{}
	err := d.streamSSE(context.Background(), strings.NewReader(sse), cs)
	if err == nil {
		t.Fatal("expected error from error event")
	}
}

func TestIntegrationLiveTail(t *testing.T) {
	d := liveDatasource(t)
	resp, err := d.client.tailStream(context.Background(), "*", "-1m", 5)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cs := &captureFrameSender{}
	// streamSSE returns the context error when the deadline elapses; that is the
	// expected way this bounded test ends, so the error is intentionally ignored.
	_ = d.streamSSE(ctx, resp.Body, cs)

	if len(cs.frames) == 0 {
		t.Fatal("no frames streamed from live tail")
	}
	t.Logf("streamed %d frames", len(cs.frames))
}
