package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// CallResource serves autocomplete and validation requests made by the query
// editor. It forwards to the matching LynxDB endpoint and returns the inner
// data payload to the frontend.
//
// Routes:
//
//	GET  fields[?prefix=]                     -> /api/v1/fields
//	GET  field-values?field=NAME[&limit=N]    -> /api/v1/fields/{NAME}/values
//	GET  sources[?pattern=]                    -> /api/v1/sources
//	GET  explain?q=QUERY (or POST {"q":...})   -> /api/v1/query/explain
func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if d.client.baseURL == "" {
		return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": errMissingURL.Error()})
	}

	q := parseResourceQuery(req.URL)

	switch req.Path {
	case "fields":
		fwd := url.Values{}
		if p := q.Get("prefix"); p != "" {
			fwd.Set("prefix", p)
		}
		return d.proxyResource(ctx, sender, "/api/v1/fields", fwd)

	case "field-values":
		field := q.Get("field")
		if field == "" {
			return sendJSON(sender, http.StatusBadRequest, map[string]string{"error": "field is required"})
		}
		fwd := url.Values{}
		if l := q.Get("limit"); l != "" {
			fwd.Set("limit", l)
		}
		return d.proxyResource(ctx, sender, "/api/v1/fields/"+url.PathEscape(field)+"/values", fwd)

	case "sources":
		fwd := url.Values{}
		if p := q.Get("pattern"); p != "" {
			fwd.Set("pattern", p)
		}
		return d.proxyResource(ctx, sender, "/api/v1/sources", fwd)

	case "explain":
		text := q.Get("q")
		if text == "" && len(req.Body) > 0 {
			var b struct {
				Q string `json:"q"`
			}
			if err := json.Unmarshal(req.Body, &b); err == nil {
				text = b.Q
			}
		}
		fwd := url.Values{}
		fwd.Set("q", text)
		return d.proxyResource(ctx, sender, "/api/v1/query/explain", fwd)

	default:
		return sendJSON(sender, http.StatusNotFound, map[string]string{"error": "unknown resource: " + req.Path})
	}
}

// proxyResource forwards a GET to LynxDB and returns the inner data payload
// (unwrapping the {data, error} envelope when present).
func (d *Datasource) proxyResource(ctx context.Context, sender backend.CallResourceResponseSender, path string, query url.Values) error {
	body, status, err := d.client.getRaw(ctx, path, query)
	if err != nil {
		return sendJSON(sender, http.StatusBadGateway, map[string]string{"error": err.Error()})
	}

	if status < http.StatusBadRequest {
		var env rawEnvelope
		if json.Unmarshal(body, &env) == nil && len(env.Data) > 0 {
			body = env.Data
		}
	}

	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    body,
	})
}

func parseResourceQuery(rawURL string) url.Values {
	u, err := url.Parse(rawURL)
	if err != nil {
		return url.Values{}
	}
	return u.Query()
}

func sendJSON(sender backend.CallResourceResponseSender, status int, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		b = []byte(`{"error":"failed to encode response"}`)
	}
	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    b,
	})
}
