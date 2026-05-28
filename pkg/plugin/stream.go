package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Live-tail channels are addressed as `tail/<refId>/<hash>`.
const tailChannelPrefix = "tail/"

// frameSender is the subset of *backend.StreamSender used by streamSSE,
// extracted so the SSE parser can be tested without a live stream packet sender.
type frameSender interface {
	SendFrame(frame *data.Frame, include data.FrameInclude) error
}

// SubscribeStream allows subscriptions to live-tail channels.
func (d *Datasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	status := backend.SubscribeStreamStatusOK
	if !strings.HasPrefix(req.Path, tailChannelPrefix) {
		status = backend.SubscribeStreamStatusNotFound
	}
	return &backend.SubscribeStreamResponse{Status: status}, nil
}

// PublishStream denies client publishing; the stream is server-driven.
func (d *Datasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{Status: backend.PublishStreamStatusPermissionDenied}, nil
}

// RunStream opens the LynxDB SSE tail for the channel's query and forwards each
// event as a single-row log frame until the client disconnects.
func (d *Datasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	var qm queryModel
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &qm); err != nil {
			return fmt.Errorf("decode stream query: %w", err)
		}
	}
	if qm.QueryText == "" {
		return nil
	}
	if d.client.baseURL == "" {
		return errMissingURL
	}

	resp, err := d.client.tailStream(ctx, qm.QueryText, "-1m", 100)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return d.streamSSE(ctx, resp.Body, sender)
}

// streamSSE parses LynxDB tail server-sent events and forwards `result` events
// as log frames. Other events (catchup_done, heartbeat, warning) are ignored.
func (d *Datasource) streamSSE(ctx context.Context, body io.Reader, sender frameSender) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var event string
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		switch {
		case line == "":
			event = ""
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			switch event {
			case "error":
				return fmt.Errorf("lynxdb tail: %s", payload)
			case "close":
				return nil
			case "result":
				var ev map[string]interface{}
				if json.Unmarshal([]byte(payload), &ev) != nil {
					continue
				}
				frame := d.eventsToFrame([]map[string]interface{}{ev})
				if err := sender.SendFrame(frame, data.IncludeAll); err != nil {
					return err
				}
			}
		}
	}

	return scanner.Err()
}
