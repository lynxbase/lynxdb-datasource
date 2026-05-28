package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Query type discriminators sent from the frontend query editor.
const (
	queryTypeLogs      = "logs"
	queryTypeMetrics   = "metrics"
	queryTypeLogVolume = "logvolume"
)

// queryModel is the per-query JSON sent by the frontend.
type queryModel struct {
	QueryText string `json:"queryText"`
	QueryType string `json:"queryType"`
	MaxLines  int    `json:"maxLines"`
}

func (d *Datasource) query(ctx context.Context, q backend.DataQuery) backend.DataResponse {
	var qm queryModel
	if err := json.Unmarshal(q.JSON, &qm); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err))
	}

	if qm.QueryText == "" {
		return backend.DataResponse{}
	}
	if d.client.baseURL == "" {
		return backend.ErrDataResponse(backend.StatusValidationFailed, errMissingURL.Error())
	}

	from := q.TimeRange.From.UTC().Format(time.RFC3339Nano)
	to := q.TimeRange.To.UTC().Format(time.RFC3339Nano)

	if qm.QueryType == queryTypeLogVolume {
		return d.queryLogVolume(ctx, qm, from, to, q.Interval)
	}

	limit := qm.MaxLines
	if limit <= 0 {
		limit = d.settings.MaxLines
	}

	body := queryRequest{
		Q:     qm.QueryText,
		From:  from,
		To:    to,
		Limit: limit,
	}

	var env queryEnvelope
	if err := d.client.postJSON(ctx, "/api/v1/query", body, &env); err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, err.Error())
	}
	if env.Error != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, env.Error.Error())
	}

	return d.frameFromQueryData(qm, env.Data)
}

// frameFromQueryData converts a LynxDB query result into a Grafana data frame
// based on the result type.
func (d *Datasource) frameFromQueryData(qm queryModel, qd queryData) backend.DataResponse {
	switch qd.Type {
	case "events", "":
		frame := d.eventsToFrame(qd.Events)
		return backend.DataResponse{Frames: []*data.Frame{frame}}
	case "timechart":
		frame, err := columnsRowsToFrame(qd.Columns, qd.Rows, true)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusInternal, err.Error())
		}
		return backend.DataResponse{Frames: []*data.Frame{frame}}
	case "aggregate":
		frame, err := columnsRowsToFrame(qd.Columns, qd.Rows, false)
		if err != nil {
			return backend.ErrDataResponse(backend.StatusInternal, err.Error())
		}
		return backend.DataResponse{Frames: []*data.Frame{frame}}
	case "job":
		return backend.ErrDataResponse(backend.StatusInternal,
			"asynchronous queries are not supported yet; narrow the time range or lower the limit")
	default:
		return backend.ErrDataResponse(backend.StatusInternal, "unsupported result type: "+qd.Type)
	}
}
