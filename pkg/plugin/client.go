package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// lynxClient is a thin HTTP client for the LynxDB REST API. It reuses the
// Grafana-managed http.Client so datasource TLS, proxy, and timeout settings
// are honored, and adds the bearer token on every request.
type lynxClient struct {
	httpClient   *http.Client
	streamClient *http.Client
	baseURL      string
	token        string
}

func newClient(ctx context.Context, settings backend.DataSourceInstanceSettings, token string) (*lynxClient, error) {
	opts, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}

	cl, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("new http client: %w", err)
	}

	// Live tail is a long-lived SSE stream, so the request timeout is removed
	// for the streaming client while keeping the configured transport.
	streamOpts := opts
	timeouts := httpclient.DefaultTimeoutOptions
	if streamOpts.Timeouts != nil {
		timeouts = *streamOpts.Timeouts
	}
	timeouts.Timeout = 0
	streamOpts.Timeouts = &timeouts
	streamCl, err := httpclient.New(streamOpts)
	if err != nil {
		return nil, fmt.Errorf("new stream client: %w", err)
	}

	return &lynxClient{
		httpClient:   cl,
		streamClient: streamCl,
		baseURL:      strings.TrimRight(settings.URL, "/"),
		token:        token,
	}, nil
}

// tailStream opens the LynxDB live-tail SSE stream. The caller must close the
// returned response body.
func (c *lynxClient) tailStream(ctx context.Context, q, from string, count int) (*http.Response, error) {
	query := url.Values{}
	query.Set("q", q)
	if from != "" {
		query.Set("from", from)
	}
	if count > 0 {
		query.Set("count", strconv.Itoa(count))
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/api/v1/tail", query, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("lynxdb tail %s: %s", resp.Status, decodeAPIError(body))
	}
	return resp, nil
}

func (c *lynxClient) newRequest(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Request, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

// getJSON issues a GET and unmarshals the body into out.
func (c *lynxClient) getJSON(ctx context.Context, path string, query url.Values, out interface{}) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

// postJSON issues a POST with a JSON body and unmarshals the body into out.
func (c *lynxClient) postJSON(ctx context.Context, path string, payload, out interface{}) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, nil, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

// getRaw issues a GET and returns the raw body and status code without
// decoding, used to forward resource responses to the frontend.
func (c *lynxClient) getRaw(ctx context.Context, path string, query url.Values) ([]byte, int, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	return body, resp.StatusCode, nil
}

func (c *lynxClient) do(req *http.Request, out interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if msg := decodeAPIError(data); msg != "" {
			return fmt.Errorf("lynxdb %s: %s", resp.Status, msg)
		}
		return fmt.Errorf("lynxdb %s", resp.Status)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// decodeAPIError extracts a human message from a {error:{...}} envelope.
func decodeAPIError(data []byte) string {
	var env struct {
		Error *apiError `json:"error"`
	}
	if json.Unmarshal(data, &env) == nil && env.Error != nil {
		return env.Error.Error()
	}
	return strings.TrimSpace(string(data))
}
