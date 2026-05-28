package plugin

import "encoding/json"

// queryRequest is the body sent to POST /api/v1/query and /api/v1/query/stream.
type queryRequest struct {
	Q         string            `json:"q"`
	From      string            `json:"from,omitempty"`
	To        string            `json:"to,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// apiError is the error object returned under the response envelope.
type apiError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *apiError) Error() string {
	if e == nil {
		return ""
	}
	if e.Suggestion != "" {
		return e.Message + " (" + e.Suggestion + ")"
	}
	return e.Message
}

// queryEnvelope is the {data, meta, error} wrapper for POST /api/v1/query.
type queryEnvelope struct {
	Data  queryData `json:"data"`
	Meta  respMeta  `json:"meta"`
	Error *apiError `json:"error"`
}

// respMeta carries the advisory metadata used by the plugin.
type respMeta struct {
	Lints []queryLint `json:"lints"`
}

// queryLint is an advisory query warning returned by LynxDB.
type queryLint struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Reason   string `json:"reason"`
	Severity string `json:"severity"`
	Position int    `json:"position"`
}

// queryData is the polymorphic data payload of a query response.
// The Type field selects which other fields are populated.
type queryData struct {
	Type string `json:"type"`

	// type == "events"
	Events  []map[string]interface{} `json:"events"`
	Total   int                      `json:"total"`
	HasMore bool                     `json:"has_more"`

	// type == "timechart" | "aggregate"
	Columns   []string        `json:"columns"`
	Rows      [][]interface{} `json:"rows"`
	Interval  string          `json:"interval"`
	TotalRows int             `json:"total_rows"`

	// type == "job"
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// healthEnvelope wraps GET /health.
type healthEnvelope struct {
	Data healthData `json:"data"`
}

type healthData struct {
	Status   string `json:"status"`
	Degraded bool   `json:"degraded"`
	Version  string `json:"version"`
}

// rawEnvelope unwraps the {data, error} wrapper for resource passthrough where
// the inner data shape is forwarded to the frontend verbatim.
type rawEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}
