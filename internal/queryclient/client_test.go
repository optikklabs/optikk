package queryclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/optikklabs/optikk/internal/clierr"
)

func kindOf(t *testing.T, err error) clierr.Kind {
	t.Helper()
	var e *clierr.Error
	if !errors.As(err, &e) {
		t.Fatalf("error %v is not a *clierr.Error", err)
	}
	return e.Kind
}

func TestDoClassifiesAuthFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED","message":"token expired"}}`))
	}))
	defer ts.Close()

	err := New(ts.URL, "tok", 0).do(context.Background(), "GET", "/v1/x", nil, nil)
	if kindOf(t, err) != clierr.Auth {
		t.Errorf("401 should classify as Auth, got %v", err)
	}
}

func TestDoClassifiesAPIFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":{"code":"QUERY_TIMEOUT","message":"too slow"}}`))
	}))
	defer ts.Close()

	err := New(ts.URL, "tok", 0).do(context.Background(), "GET", "/v1/x", nil, nil)
	var e *clierr.Error
	if !errors.As(err, &e) || e.Kind != clierr.API || e.APICode != "QUERY_TIMEOUT" {
		t.Errorf("500 should classify as API with the server code, got %v", err)
	}
}

func TestDoClassifiesNetworkFailure(t *testing.T) {
	// A closed server guarantees connection refused.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	err := New(ts.URL, "tok", 0).do(context.Background(), "GET", "/v1/x", nil, nil)
	if kindOf(t, err) != clierr.Network {
		t.Errorf("dial failure should classify as Network, got %v", err)
	}
}

func TestErrorGroupPaths(t *testing.T) {
	var gotPath, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.Write([]byte(`{"success":true,"data":{"results":[{"group_id":"g1","sample_trace_id":"t1"}],"pageInfo":{"hasMore":false}}}`))
	}))
	defer ts.Close()

	resp, err := New(ts.URL, "tok", 0).ListErrorGroups(context.Background(), 1000, 2000, "checkout svc", 50, "")
	if err != nil {
		t.Fatalf("ListErrorGroups: %v", err)
	}
	if gotPath != "/api/v1/errors/groups" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "startTime=1000&endTime=2000&limit=50&serviceName=checkout+svc" {
		t.Errorf("query = %q", gotQuery)
	}
	if len(resp.Results) != 1 || resp.Results[0].SampleTraceID != "t1" {
		t.Errorf("decode = %+v, want one group with sample_trace_id t1", resp)
	}
}

func TestIngestionSummaryDecodesTotals(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ingestion/summary" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"success":true,"data":{"totals":{"logs":5,"spans":7,"metricDatapoints":11,"records":23}}}`))
	}))
	defer ts.Close()

	totals, err := New(ts.URL, "tok", 0).IngestionSummary(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("IngestionSummary: %v", err)
	}
	if totals.Records != 23 || totals.Spans != 7 || totals.Logs != 5 || totals.MetricDatapoints != 11 {
		t.Errorf("totals = %+v", totals)
	}
}

func TestLogsByTracePath(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"success":true,"data":[{"id":"1","body":"hello"}]}`))
	}))
	defer ts.Close()

	logs, err := New(ts.URL, "tok", 0).LogsByTrace(context.Background(), "abc123", 100)
	if err != nil {
		t.Fatalf("LogsByTrace: %v", err)
	}
	if gotPath != "/api/v1/logs/trace/abc123" {
		t.Errorf("path = %q", gotPath)
	}
	if len(logs) != 1 || logs[0].Body != "hello" {
		t.Errorf("decode = %+v", logs)
	}
}
