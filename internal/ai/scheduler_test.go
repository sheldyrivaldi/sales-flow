package ai

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"salespilot/internal/domain"
)

type fakeSchedulerProfileFetcher struct {
	agg *domain.ProfileAggregate
	err error
}

func (f *fakeSchedulerProfileFetcher) GetCurrent(_ context.Context) (*domain.ProfileAggregate, error) {
	return f.agg, f.err
}

type fakeRunTrigger struct {
	mu      sync.Mutex
	calls   int
	lastKey *string
}

func (f *fakeRunTrigger) RunAsync(_ context.Context, key *string) (*domain.DiscoveryRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastKey = key
	return &domain.DiscoveryRun{ID: "run"}, nil
}

func (f *fakeRunTrigger) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeRunTrigger) LastKey() *string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastKey
}

func TestScheduler_TriggersWhenEnabled(t *testing.T) {
	profiles := &fakeSchedulerProfileFetcher{agg: &domain.ProfileAggregate{
		Profile: domain.CompanyProfile{CrawlEnabled: true, CrawlFrequency: "harian"},
	}}
	runner := &fakeRunTrigger{}
	sched := NewScheduler(profiles, runner, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	sched.Start(ctx)

	if runner.Calls() == 0 {
		t.Fatal("expected RunAsync to be called at least once when crawl_enabled=true")
	}
	key := runner.LastKey()
	if key == nil || *key == "" {
		t.Error("expected a non-empty correlation key to be passed to RunAsync")
	}
}

func TestScheduler_SkipsWhenDisabled(t *testing.T) {
	profiles := &fakeSchedulerProfileFetcher{agg: &domain.ProfileAggregate{
		Profile: domain.CompanyProfile{CrawlEnabled: false, CrawlFrequency: "harian"},
	}}
	runner := &fakeRunTrigger{}
	sched := NewScheduler(profiles, runner, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	sched.Start(ctx)

	if runner.Calls() != 0 {
		t.Errorf("RunAsync called %d times, want 0 when crawl_enabled=false", runner.Calls())
	}
}

func TestScheduler_ProfileError_SkipsTickWithoutCrashing(t *testing.T) {
	profiles := &fakeSchedulerProfileFetcher{err: errors.New("db down")}
	runner := &fakeRunTrigger{}
	sched := NewScheduler(profiles, runner, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	sched.Start(ctx) // must not panic despite the profile lookup always failing

	if runner.Calls() != 0 {
		t.Errorf("RunAsync called %d times, want 0 when profile lookup fails", runner.Calls())
	}
}

func TestScheduler_StopsPromptlyOnCtxCancel(t *testing.T) {
	profiles := &fakeSchedulerProfileFetcher{agg: &domain.ProfileAggregate{
		Profile: domain.CompanyProfile{CrawlEnabled: false},
	}}
	sched := NewScheduler(profiles, &fakeRunTrigger{}, time.Hour) // long tick — shouldn't matter

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sched.Start(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Start returned — good.
	case <-time.After(2 * time.Second):
		t.Fatal("Scheduler.Start did not return promptly after ctx cancellation")
	}
}
