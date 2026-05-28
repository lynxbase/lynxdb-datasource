package plugin

import (
	"context"
	"errors"
	"net/url"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/lynxbase/lynxdb-datasource/pkg/models"
)

// Interface assertions: the datasource handles queries, health checks, and
// resource calls (field/value autocomplete and query validation).
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
	_ backend.StreamHandler         = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// Datasource is a LynxDB datasource instance. One instance exists per
// configured datasource and is recreated when its settings change.
type Datasource struct {
	settings *models.PluginSettings
	client   *lynxClient
}

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	settings, err := models.LoadPluginSettings(s)
	if err != nil {
		return nil, err
	}

	client, err := newClient(ctx, s, settings.Secrets.Token)
	if err != nil {
		return nil, err
	}

	return &Datasource{settings: settings, client: client}, nil
}

// Dispose cleans up datasource instance resources.
func (d *Datasource) Dispose() {}

// QueryData runs each query in the request and collects the responses by RefID.
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		response.Responses[q.RefID] = d.query(ctx, q)
	}

	return response, nil
}

// CheckHealth verifies the datasource can reach LynxDB and reports its version.
func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if d.client.baseURL == "" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "LynxDB URL is not set",
		}, nil
	}

	var env healthEnvelope
	if err := d.client.getJSON(ctx, "/health", url.Values{}, &env); err != nil {
		log.DefaultLogger.Error("health check failed", "err", err)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Could not reach LynxDB: " + err.Error(),
		}, nil
	}

	if env.Data.Status != "" && env.Data.Status != "healthy" && env.Data.Status != "ok" {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "LynxDB reported status: " + env.Data.Status,
		}, nil
	}

	msg := "Connected to LynxDB"
	if env.Data.Version != "" {
		msg += " " + env.Data.Version
	}
	if env.Data.Degraded {
		msg += " (degraded: running on in-memory store)"
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: msg,
	}, nil
}

// errMissingURL is returned when a request is attempted without a base URL.
var errMissingURL = errors.New("LynxDB URL is not set")
