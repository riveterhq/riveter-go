package riveter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func runFixtureJSON(overrides map[string]any) []byte {
	fixture := map[string]any{
		"id":           "run_abc",
		"type":         "enrichment",
		"status":       "processing",
		"progress":     map[string]any{"percent_complete": 40},
		"credits_used": 1.5,
		"app_url":      "https://app.riveterhq.com/runs/run_abc",
		"result_url":   "https://api.riveterhq.com/v1/runs/run_abc/result",
		"started_at":   "2026-01-15T12:00:00Z",
		"finished_at":  nil,
		"error":        nil,
	}
	for key, value := range overrides {
		fixture[key] = value
	}
	payload, err := json.Marshal(fixture)
	if err != nil {
		panic(err)
	}
	return payload
}

func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	options := append([]Option{WithAPIKey("rk_test"), WithBaseURL(server.URL + "/v1")}, opts...)
	client, err := NewClient(options...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestMissingAPIKey(t *testing.T) {
	t.Setenv("RIVETER_API_KEY", "")
	if _, err := NewClient(); err == nil {
		t.Fatal("expected an error without an API key")
	}
}

func TestEnrichSendsAuthAndBody(t *testing.T) {
	var captured *http.Request
	var capturedBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(runFixtureJSON(nil))
	})

	run, err := client.Enrich(context.Background(), EnrichParams{
		EnrichmentID: "enr_1",
		Input:        map[string][]string{"Company": {"Acme"}},
	})
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	if captured.Method != http.MethodPost || captured.URL.Path != "/v1/enrich" {
		t.Fatalf("unexpected request: %s %s", captured.Method, captured.URL.Path)
	}
	if got := captured.Header.Get("Authorization"); got != "Bearer rk_test" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := captured.Header.Get("User-Agent"); !strings.HasPrefix(got, "riveter-sdk-go/") {
		t.Fatalf("User-Agent = %q", got)
	}
	if capturedBody["enrichment_id"] != "enr_1" {
		t.Fatalf("body = %v", capturedBody)
	}
	if _, hasOutput := capturedBody["output"]; hasOutput {
		t.Fatalf("empty optional fields must be omitted, body = %v", capturedBody)
	}
	if run.ID != "run_abc" || run.Progress == nil || *run.Progress.PercentComplete != 40 {
		t.Fatalf("run = %+v", run)
	}
}

func TestCleanEnvelopeError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"type":"not_found","message":"No run found"}}`)
	})

	_, err := client.Runs.Get(context.Background(), "run_missing")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Type != "not_found" || apiErr.Message != "No run found" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
}

func TestLegacy401Error(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"request_status":"error","message":"Invalid or missing API key","error_type":"unauthorized"}`)
	})

	_, err := client.Account(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 || apiErr.Type != "unauthorized" {
		t.Fatalf("apiErr = %+v", apiErr)
	}
}

func TestRateLimitRetryOnPost(t *testing.T) {
	attempts := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix(), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "Retry later")
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(runFixtureJSON(nil))
	})

	run, err := client.Enrich(context.Background(), EnrichParams{EnrichmentID: "enr_1"})
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	if run.ID != "run_abc" || attempts != 2 {
		t.Fatalf("run = %+v, attempts = %d", run, attempts)
	}
}

func TestRateLimitGivesUpAfterMaxRetries(t *testing.T) {
	attempts := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix(), 10))
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "Retry later")
	}, WithMaxRetries(1))

	_, err := client.Runs.Get(context.Background(), "run_abc")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != 429 || apiErr.Type != "rate_limited" || attempts != 2 {
		t.Fatalf("apiErr = %+v, attempts = %d", apiErr, attempts)
	}
}

func TestGet5xxRetry(t *testing.T) {
	attempts := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprint(w, "Bad gateway")
			return
		}
		w.Write(runFixtureJSON(nil))
	})

	run, err := client.Runs.Get(context.Background(), "run_abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if run.ID != "run_abc" || attempts != 2 {
		t.Fatalf("run = %+v, attempts = %d", run, attempts)
	}
}

func TestPost5xxNotRetried(t *testing.T) {
	attempts := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"type":"error","message":"boom"}}`)
	})

	_, err := client.Runs.Stop(context.Background(), "run_abc")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 || attempts != 1 {
		t.Fatalf("apiErr = %+v, attempts = %d", apiErr, attempts)
	}
}

func TestConnectionError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	listener.Close()

	client, err := NewClient(
		WithAPIKey("rk_test"),
		WithBaseURL("http://"+address+"/v1"),
		WithMaxRetries(0),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Runs.Get(context.Background(), "run_abc")
	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Fatalf("expected *ConnectionError, got %T (%v)", err, err)
	}
}

func TestResultPassesWait(t *testing.T) {
	var captured *http.Request
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.Write(runFixtureJSON(map[string]any{"status": "success", "output": map[string]any{"Company": []any{}}}))
	})

	run, err := client.Runs.Result(context.Background(), "run_abc", &ResultOptions{Wait: 30})
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if captured.URL.Path != "/v1/runs/run_abc/result" || captured.URL.Query().Get("wait") != "30" {
		t.Fatalf("unexpected request: %s", captured.URL.String())
	}
	if run.Status != "success" || string(run.Output) != `{"Company":[]}` {
		t.Fatalf("run = %+v output = %s", run, run.Output)
	}
}

func TestWaitForResultPolls(t *testing.T) {
	attempts := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.URL.Query().Get("wait") != "50" {
			t.Fatalf("wait = %q", r.URL.Query().Get("wait"))
		}
		if attempts == 1 {
			w.Write(runFixtureJSON(map[string]any{"status": "processing"}))
			return
		}
		w.Write(runFixtureJSON(map[string]any{"status": "success", "output": map[string]any{"CEO": []any{map[string]any{"value": "Ada"}}}}))
	})

	run, err := client.Runs.WaitForResult(context.Background(), "run_abc", nil)
	if err != nil {
		t.Fatalf("WaitForResult: %v", err)
	}
	if !run.Finished() || attempts != 2 {
		t.Fatalf("run = %+v, attempts = %d", run, attempts)
	}
}

func TestListPagination(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "success" {
			t.Fatalf("status = %q", r.URL.Query().Get("status"))
		}
		page := r.URL.Query().Get("page")
		if page == "2" {
			fmt.Fprint(w, `{"runs":[{"id":"run_3"}],"pagination":{"page":2,"per_page":2,"total_count":3,"total_pages":2}}`)
			return
		}
		fmt.Fprint(w, `{"runs":[{"id":"run_1"},{"id":"run_2"}],"pagination":{"page":1,"per_page":2,"total_count":3,"total_pages":2}}`)
	})

	page, err := client.Runs.List(context.Background(), &ListRunsParams{Status: "success"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	var seen []string
	for {
		for _, run := range page.Runs {
			seen = append(seen, run.ID)
		}
		if !page.HasNextPage() {
			break
		}
		page, err = page.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage: %v", err)
		}
	}
	if strings.Join(seen, ",") != "run_1,run_2,run_3" {
		t.Fatalf("seen = %v", seen)
	}
	if _, err := page.NextPage(context.Background()); !errors.Is(err, ErrNoNextPage) {
		t.Fatalf("expected ErrNoNextPage, got %v", err)
	}
}

func TestMonitorRunsPath(t *testing.T) {
	var captured *http.Request
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r
		fmt.Fprint(w, `{"monitor_id":"mon_1","runs":[],"pagination":{"page":1,"per_page":25,"total_count":0,"total_pages":1}}`)
	})

	page, err := client.Monitors.Runs(context.Background(), "mon_1", nil)
	if err != nil {
		t.Fatalf("Runs: %v", err)
	}
	if captured.URL.Path != "/v1/monitors/mon_1/runs" {
		t.Fatalf("path = %q", captured.URL.Path)
	}
	if page.MonitorID != "mon_1" || page.HasNextPage() {
		t.Fatalf("page = %+v", page)
	}
}

func TestUpdateMonitorSendsEnabledFalse(t *testing.T) {
	var capturedBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"id":"mon_1","enabled":false}`)
	})

	enabled := false
	monitor, err := client.Monitors.Update(context.Background(), "mon_1", UpdateMonitorParams{Enabled: &enabled})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if value, present := capturedBody["enabled"]; !present || value != false {
		t.Fatalf("body = %v", capturedBody)
	}
	if monitor.Enabled {
		t.Fatalf("monitor = %+v", monitor)
	}
}

func TestUnwrapsListEnvelopes(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/enrichments" {
			if r.URL.Query().Get("name") != "E" || r.URL.Query().Get("page") != "1" {
				t.Errorf("enrichments query = %s", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"enrichments":[{"id":"enr_1","name":"E","input_columns":["Company"],"output_columns":["CEO"]}],"pagination":{"page":1,"per_page":25,"total_count":1,"total_pages":1}}`)
			return
		}
		fmt.Fprint(w, `{"monitors":[{"id":"mon_1"}]}`)
	})

	enrichmentsPage, err := client.Enrichments.List(context.Background(), &ListEnrichmentsParams{Name: "E"})
	if err != nil {
		t.Fatalf("Enrichments.List: %v", err)
	}
	monitors, err := client.Monitors.List(context.Background())
	if err != nil {
		t.Fatalf("Monitors.List: %v", err)
	}
	enrichments := enrichmentsPage.Enrichments
	if len(enrichments) != 1 || enrichments[0].ID != "enr_1" || enrichments[0].InputColumns[0] != "Company" {
		t.Fatalf("enrichments = %+v", enrichments)
	}
	if enrichmentsPage.HasNextPage() {
		t.Fatalf("expected a single page, got %+v", enrichmentsPage.Pagination)
	}
	if len(monitors) != 1 || monitors[0].ID != "mon_1" {
		t.Fatalf("monitors = %+v", monitors)
	}
}
