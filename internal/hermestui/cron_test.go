package hermestui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"salespilot/internal/config"
)

func TestNewCronClient_NilWithoutToken(t *testing.T) {
	cfg := &config.Config{HermesTuiBaseURL: "http://example.com", HermesTuiSessionToken: ""}
	if c := NewCronClient(cfg); c != nil {
		t.Fatalf("NewCronClient() = %v, want nil when HermesTuiSessionToken is unset", c)
	}
}

func TestNewCronClient_NonNilWithToken(t *testing.T) {
	cfg := &config.Config{HermesTuiBaseURL: "http://example.com", HermesTuiSessionToken: "tok"}
	if c := NewCronClient(cfg); c == nil {
		t.Fatal("NewCronClient() = nil, want non-nil when HermesTuiSessionToken is set")
	}
}

func newTestCronClient(baseURL string) *CronClient {
	return &CronClient{baseURL: baseURL, token: "test-token", hc: http.DefaultClient}
}

func TestCronClient_UpsertJob_CreatesWhenAbsent(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/cron/jobs":
			_ = json.NewEncoder(w).Encode([]CronJob{}) // no existing jobs
		case r.Method == http.MethodPost && r.URL.Path == "/api/cron/jobs":
			gotAuth = r.Header.Get("Authorization")
			gotMethod = r.Method
			gotPath = r.URL.Path
			var body cronJobCreateReq
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Name != "salespilot-discovery-crawl" {
				t.Errorf("create body.Name = %q, want salespilot-discovery-crawl", body.Name)
			}
			if body.Schedule != "0 8 * * *" {
				t.Errorf("create body.Schedule = %q, want '0 8 * * *'", body.Schedule)
			}
			_ = json.NewEncoder(w).Encode(CronJob{ID: "job-1", Name: body.Name, Schedule: map[string]string{"kind": "cron", "expr": body.Schedule, "display": body.Schedule}, Enabled: true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := newTestCronClient(srv.URL)
	err := c.UpsertJob(context.Background(), "salespilot-discovery-crawl", "0 8 * * *", "do the thing")
	if err != nil {
		t.Fatalf("UpsertJob: unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want 'Bearer test-token'", gotAuth)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/cron/jobs" {
		t.Errorf("create request = %s %s, want POST /api/cron/jobs", gotMethod, gotPath)
	}
}

func TestCronClient_UpsertJob_UpdatesAndResumesWhenPresent(t *testing.T) {
	var putCalled, resumeCalled bool
	var putBody cronJobUpdateReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/cron/jobs":
			// Schedule is a nested object on read ({"kind":...,"expr":...,
			// "display":...}), not the plain cron string the write side
			// takes — mirrors the real hermes-tui API (see CronJob's doc
			// comment); a plain-string mock here previously masked a real
			// decode bug that silently broke every upsert/pause call.
			_ = json.NewEncoder(w).Encode([]CronJob{
				{ID: "job-existing", Name: "salespilot-discovery-crawl", Schedule: map[string]string{"kind": "cron", "expr": "0 8 * * 1", "display": "0 8 * * 1"}, Enabled: false},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/cron/jobs/job-existing":
			putCalled = true
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			_ = json.NewEncoder(w).Encode(CronJob{ID: "job-existing", Name: "salespilot-discovery-crawl", Schedule: map[string]string{"kind": "cron", "expr": "0 8 * * *", "display": "0 8 * * *"}, Enabled: true})
		case r.Method == http.MethodPost && r.URL.Path == "/api/cron/jobs/job-existing/resume":
			resumeCalled = true
			_ = json.NewEncoder(w).Encode(CronJob{ID: "job-existing", Enabled: true})
		case r.Method == http.MethodPost:
			t.Fatalf("unexpected POST %s (create should not run when the job exists)", r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := newTestCronClient(srv.URL)
	err := c.UpsertJob(context.Background(), "salespilot-discovery-crawl", "0 8 * * *", "new prompt")
	if err != nil {
		t.Fatalf("UpsertJob: unexpected error: %v", err)
	}
	if !putCalled {
		t.Fatal("PUT (update) was never called")
	}
	if putBody.Updates["schedule"] != "0 8 * * *" {
		t.Errorf("update schedule = %v, want '0 8 * * *'", putBody.Updates["schedule"])
	}
	// enabled must NOT be part of the PUT body — update_job would write the
	// flag but leave state:"paused"/stale next_run_at, silently keeping the
	// job paused; re-enabling goes through /resume instead.
	if _, has := putBody.Updates["enabled"]; has {
		t.Errorf("update body must not contain 'enabled' (got %v) — resume endpoint owns re-enabling", putBody.Updates["enabled"])
	}
	if !resumeCalled {
		t.Fatal("POST /resume was never called — job would stay paused")
	}
}

func TestCronClient_PauseJobByName_PausesWhenEnabled(t *testing.T) {
	var pauseCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/cron/jobs":
			_ = json.NewEncoder(w).Encode([]CronJob{
				{ID: "job-1", Name: "salespilot-discovery-crawl", Enabled: true},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/cron/jobs/job-1/pause":
			pauseCalled = true
			_ = json.NewEncoder(w).Encode(CronJob{ID: "job-1", Enabled: false})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := newTestCronClient(srv.URL)
	if err := c.PauseJobByName(context.Background(), "salespilot-discovery-crawl"); err != nil {
		t.Fatalf("PauseJobByName: unexpected error: %v", err)
	}
	if !pauseCalled {
		t.Fatal("pause endpoint was never called")
	}
}

func TestCronClient_PauseJobByName_NoopWhenAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/cron/jobs" {
			_ = json.NewEncoder(w).Encode([]CronJob{}) // no jobs at all
			return
		}
		t.Fatalf("unexpected request: %s %s (should have been a no-op)", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestCronClient(srv.URL)
	if err := c.PauseJobByName(context.Background(), "salespilot-discovery-crawl"); err != nil {
		t.Fatalf("PauseJobByName: unexpected error: %v", err)
	}
}

func TestCronClient_PauseJobByName_NoopWhenAlreadyPaused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/cron/jobs" {
			_ = json.NewEncoder(w).Encode([]CronJob{
				{ID: "job-1", Name: "salespilot-discovery-crawl", Enabled: false},
			})
			return
		}
		t.Fatalf("unexpected request: %s %s (already paused, should have been a no-op)", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := newTestCronClient(srv.URL)
	if err := c.PauseJobByName(context.Background(), "salespilot-discovery-crawl"); err != nil {
		t.Fatalf("PauseJobByName: unexpected error: %v", err)
	}
}

func TestCronClient_UpsertJob_ListErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestCronClient(srv.URL)
	err := c.UpsertJob(context.Background(), "salespilot-discovery-crawl", "0 8 * * *", "prompt")
	if err == nil {
		t.Fatal("UpsertJob should propagate a list-jobs failure, got nil")
	}
}
