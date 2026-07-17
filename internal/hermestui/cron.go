// cron.go — a plain server-to-server REST client for hermes-tui's dashboard
// cron API (/api/cron/jobs), used to upsert the Company Profile discovery
// crawl job (EP-12 crawl automation). Distinct from Pump/the ticket registry
// above, which back the interactive embedded-terminal feature — this talks
// to hermes-tui's own scheduler ("the gateway" that ticks cron jobs every
// 60s, per the vendored hermes-agent's cron/scheduler.py), not a proxy.
package hermestui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"salespilot/internal/config"
)

// CronClient authenticates to hermes-tui's dashboard REST API with the
// shared HERMES_DASHBOARD_SESSION_TOKEN (see internal/config).
type CronClient struct {
	baseURL string
	token   string
	hc      *http.Client
}

// NewCronClient returns nil when HermesTuiSessionToken is unset — callers
// must treat a nil *CronClient as "crawl automation upsert unavailable,
// degrade silently" (same non-blocking-AI-feature philosophy as
// ai.Extractor / hermes.Client elsewhere: the profile save itself must never
// fail because Hermes's scheduler couldn't be reached).
func NewCronClient(cfg *config.Config) *CronClient {
	if cfg.HermesTuiSessionToken == "" {
		return nil
	}
	return &CronClient{
		baseURL: strings.TrimRight(cfg.HermesTuiBaseURL, "/"),
		token:   cfg.HermesTuiSessionToken,
		hc:      &http.Client{},
	}
}

// CronJob mirrors the subset of hermes-tui's job record shape (cron/jobs.py
// _normalize_job_record) this client needs. Schedule is `any`, not string:
// on read, hermes-tui returns it as a nested object
// ({"kind":"cron","expr":"0 8 * * *","display":"..."}), not the plain cron
// string the create/update *request* bodies take — decoding into a string
// field errors on every response and silently breaks every call (learned by
// running this against the real service; confirmed via API logs). This
// client never reads CronJob.Schedule's value, only ID/Name/Enabled, so `any`
// is enough to make decoding succeed without needing the exact shape.
type CronJob struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Schedule any    `json:"schedule"`
	Prompt   string `json:"prompt"`
	Enabled  bool   `json:"enabled"`
}

type cronJobCreateReq struct {
	Prompt   string `json:"prompt"`
	Schedule string `json:"schedule"`
	Name     string `json:"name"`
	Deliver  string `json:"deliver,omitempty"`
}

type cronJobUpdateReq struct {
	Updates map[string]any `json:"updates"`
}

func (c *CronClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("status %d: %s", resp.StatusCode, b)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// findJobByName lists every job (hermes-tui's list endpoint always includes
// paused/disabled ones) for the default profile and returns the one whose
// name matches, or nil if none does.
func (c *CronClient) findJobByName(ctx context.Context, name string) (*CronJob, error) {
	var jobs []CronJob
	if err := c.doJSON(ctx, http.MethodGet, "/api/cron/jobs?profile=default", nil, &jobs); err != nil {
		return nil, fmt.Errorf("hermestui.CronClient.findJobByName: %w", err)
	}
	for i := range jobs {
		if jobs[i].Name == name {
			return &jobs[i], nil
		}
	}
	return nil, nil
}

// UpsertJob creates a job named jobName if none exists yet, otherwise
// updates its schedule/prompt in place and resumes it — covering both
// "never configured" and "user previously turned crawl off, then back on
// with a different frequency" in one call. jobName is the sole identity key;
// callers must use a fixed, stable name per logical job.
//
// Re-enabling MUST go through the dedicated /resume endpoint, not a PUT with
// {"enabled": true}: hermes's update_job writes the enabled flag verbatim but
// leaves state:"paused" and the stale/nil next_run_at untouched, so the job
// shows as paused in the dashboard and never actually fires. resume_job is
// the only path that flips state to "scheduled" AND recomputes next_run_at
// (found live: enable-from-UI kept the job paused until this was switched).
func (c *CronClient) UpsertJob(ctx context.Context, jobName, schedule, prompt string) error {
	existing, err := c.findJobByName(ctx, jobName)
	if err != nil {
		return err
	}

	if existing == nil {
		var created CronJob
		req := cronJobCreateReq{Prompt: prompt, Schedule: schedule, Name: jobName, Deliver: "local"}
		if err := c.doJSON(ctx, http.MethodPost, "/api/cron/jobs?profile=default", req, &created); err != nil {
			return fmt.Errorf("hermestui.CronClient.UpsertJob: create: %w", err)
		}
		return nil
	}

	updates := cronJobUpdateReq{Updates: map[string]any{
		"schedule": schedule,
		"prompt":   prompt,
	}}
	var updated CronJob
	if err := c.doJSON(ctx, http.MethodPut, "/api/cron/jobs/"+existing.ID+"?profile=default", updates, &updated); err != nil {
		return fmt.Errorf("hermestui.CronClient.UpsertJob: update: %w", err)
	}

	// Always resume, even when the job reads as enabled: resume is idempotent
	// (recomputes the next future run) and guarantees state:"scheduled".
	if err := c.doJSON(ctx, http.MethodPost, "/api/cron/jobs/"+existing.ID+"/resume?profile=default", nil, nil); err != nil {
		return fmt.Errorf("hermestui.CronClient.UpsertJob: resume: %w", err)
	}
	return nil
}

// PauseJobByName pauses the job named jobName if it exists; a no-op (not an
// error) when no such job exists yet — turning crawl off before it was ever
// turned on has nothing to pause.
func (c *CronClient) PauseJobByName(ctx context.Context, jobName string) error {
	existing, err := c.findJobByName(ctx, jobName)
	if err != nil {
		return fmt.Errorf("hermestui.CronClient.PauseJobByName: %w", err)
	}
	if existing == nil || !existing.Enabled {
		return nil
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/cron/jobs/"+existing.ID+"/pause?profile=default", nil, nil); err != nil {
		return fmt.Errorf("hermestui.CronClient.PauseJobByName: %w", err)
	}
	return nil
}
