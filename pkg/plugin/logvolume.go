package plugin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// queryLogVolume builds the full-range log-volume series for Explore. It wraps
// the user's filter with a timechart aggregation so the volume reflects the
// active query (the dedicated histogram endpoint counts the whole index and
// cannot filter). Results are grouped by the configured level field so Explore
// renders stacked per-level bars.
func (d *Datasource) queryLogVolume(ctx context.Context, qm queryModel, from, to string, interval time.Duration) backend.DataResponse {
	q := strings.TrimSpace(qm.QueryText) + " | timechart count"
	if d.settings.LevelField != "" {
		q += " by " + d.settings.LevelField
	}
	q += " span=" + durationToSpan(interval)

	body := queryRequest{Q: q, From: from, To: to}

	var env queryEnvelope
	if err := d.client.postJSON(ctx, "/api/v1/query", body, &env); err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, err.Error())
	}
	if env.Error != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, env.Error.Error())
	}
	if env.Data.Type != "timechart" {
		// Query already aggregates or returns no time axis; no volume to show.
		return backend.DataResponse{}
	}

	frame, err := columnsRowsToFrame(env.Data.Columns, env.Data.Rows, true)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, err.Error())
	}
	frame.Name = "logvolume"
	return backend.DataResponse{Frames: []*data.Frame{frame}}
}

// durationToSpan formats a Grafana interval as a LynxDB span string (e.g. 5m).
func durationToSpan(d time.Duration) string {
	if d <= 0 {
		return "1m"
	}
	switch {
	case d >= time.Hour && d%time.Hour == 0:
		return strconv.FormatInt(int64(d/time.Hour), 10) + "h"
	case d >= time.Minute && d%time.Minute == 0:
		return strconv.FormatInt(int64(d/time.Minute), 10) + "m"
	default:
		secs := int64(d / time.Second)
		if secs < 1 {
			secs = 1
		}
		return strconv.FormatInt(secs, 10) + "s"
	}
}
