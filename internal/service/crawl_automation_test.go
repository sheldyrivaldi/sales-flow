package service

import (
	"context"
	"strings"
	"testing"
)

func TestCronScheduleForFrequency(t *testing.T) {
	cases := map[string]string{
		"harian":   "0 8 * * *",
		"2-3x":     "0 8 * * 1,3,5",
		"mingguan": "0 8 * * 1",
		"unknown":  "0 8 * * *",
		"":         "0 8 * * *",
	}
	for freq, want := range cases {
		if got := cronScheduleForFrequency(freq); got != want {
			t.Errorf("cronScheduleForFrequency(%q) = %q, want %q", freq, got, want)
		}
	}
}

func TestTriggerPrompt_ContainsCallbackURL(t *testing.T) {
	url := "http://api:8080/internal/discovery/trigger?secret=abc123"
	prompt := triggerPrompt(url)
	if !strings.Contains(prompt, url) {
		t.Errorf("triggerPrompt does not contain callback URL:\n%s", prompt)
	}
}

// stubCronUpserter is a test double for CronJobUpserter.
type stubCronUpserter struct {
	upsertCalls []struct{ jobName, schedule, prompt string }
	pauseCalls  []string
	upsertErr   error
	pauseErr    error
}

func (s *stubCronUpserter) UpsertJob(_ context.Context, jobName, schedule, prompt string) error {
	s.upsertCalls = append(s.upsertCalls, struct{ jobName, schedule, prompt string }{jobName, schedule, prompt})
	return s.upsertErr
}

func (s *stubCronUpserter) PauseJobByName(_ context.Context, jobName string) error {
	s.pauseCalls = append(s.pauseCalls, jobName)
	return s.pauseErr
}

func TestSyncCrawlAutomation_NilCrawl_Noop(t *testing.T) {
	svc := &ProfileService{crawl: nil}
	// Must not panic.
	svc.syncCrawlAutomation(context.Background(), "PT Contoh", true, "harian")
}

func TestSyncCrawlAutomation_NilClient_Noop(t *testing.T) {
	svc := &ProfileService{crawl: &CrawlAutomation{Client: nil}}
	// Must not panic.
	svc.syncCrawlAutomation(context.Background(), "PT Contoh", true, "harian")
}

func TestSyncCrawlAutomation_Enabled_Upserts(t *testing.T) {
	stub := &stubCronUpserter{}
	svc := &ProfileService{crawl: &CrawlAutomation{
		Client:             stub,
		TriggerSecret:      "sekret",
		InternalAPIBaseURL: "http://api:8080",
	}}

	svc.syncCrawlAutomation(context.Background(), "PT Contoh", true, "mingguan")

	if len(stub.upsertCalls) != 1 {
		t.Fatalf("upsertCalls = %d, want 1", len(stub.upsertCalls))
	}
	call := stub.upsertCalls[0]
	if call.jobName != discoveryCronJobName {
		t.Errorf("jobName = %q, want %q", call.jobName, discoveryCronJobName)
	}
	if call.schedule != "0 8 * * 1" {
		t.Errorf("schedule = %q, want '0 8 * * 1' (mingguan)", call.schedule)
	}
	if !strings.Contains(call.prompt, "http://api:8080/internal/discovery/trigger?secret=sekret") {
		t.Errorf("prompt missing callback URL: %s", call.prompt)
	}
	if len(stub.pauseCalls) != 0 {
		t.Errorf("pauseCalls = %d, want 0 when crawl is enabled", len(stub.pauseCalls))
	}
}

func TestSyncCrawlAutomation_Enabled_NoSecret_SkipsUpsert(t *testing.T) {
	stub := &stubCronUpserter{}
	svc := &ProfileService{crawl: &CrawlAutomation{
		Client:        stub,
		TriggerSecret: "", // unset
	}}

	svc.syncCrawlAutomation(context.Background(), "PT Contoh", true, "harian")

	if len(stub.upsertCalls) != 0 {
		t.Errorf("upsertCalls = %d, want 0 when TriggerSecret is unset", len(stub.upsertCalls))
	}
}

func TestSyncCrawlAutomation_Disabled_Pauses(t *testing.T) {
	stub := &stubCronUpserter{}
	svc := &ProfileService{crawl: &CrawlAutomation{Client: stub, TriggerSecret: "sekret"}}

	svc.syncCrawlAutomation(context.Background(), "PT Contoh", false, "harian")

	if len(stub.pauseCalls) != 1 || stub.pauseCalls[0] != discoveryCronJobName {
		t.Errorf("pauseCalls = %v, want [%q]", stub.pauseCalls, discoveryCronJobName)
	}
	if len(stub.upsertCalls) != 0 {
		t.Errorf("upsertCalls = %d, want 0 when crawl is disabled", len(stub.upsertCalls))
	}
}

func TestSyncCrawlAutomation_UpsertError_DoesNotPanic(t *testing.T) {
	stub := &stubCronUpserter{upsertErr: context.DeadlineExceeded}
	svc := &ProfileService{crawl: &CrawlAutomation{Client: stub, TriggerSecret: "sekret"}}
	svc.syncCrawlAutomation(context.Background(), "PT Contoh", true, "harian")
}

func TestSyncCrawlAutomation_PauseError_DoesNotPanic(t *testing.T) {
	stub := &stubCronUpserter{pauseErr: context.DeadlineExceeded}
	svc := &ProfileService{crawl: &CrawlAutomation{Client: stub}}
	svc.syncCrawlAutomation(context.Background(), "PT Contoh", false, "harian")
}
