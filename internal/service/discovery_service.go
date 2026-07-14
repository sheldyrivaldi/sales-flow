package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
)

// pgUniqueViolation is the Postgres error code for a unique constraint
// violation (23505) — used to detect a dedup_key race (two concurrent
// creates for the same candidate) without a dedicated repository method.
const pgUniqueViolation = "23505"

// discoveryRunTimeout bounds how long a single background discovery run
// (crawl + extract + score every candidate) may run before its detached
// context is canceled — long enough for a real multi-source crawl, short
// enough that a stuck run doesn't leak forever.
const discoveryRunTimeout = 10 * time.Minute

// candidateCollector is the slice of *ai.DiscoveryOrchestrator that
// DiscoveryService needs. A narrow interface (rather than depending on the
// concrete orchestrator type) keeps the persist+score pipeline below
// unit-testable without exercising the real crawler/compliance chain again
// (already covered by internal/ai's own tests) — tests here inject a fake.
type candidateCollector interface {
	CollectCandidates(ctx context.Context) ([]ai.CandidateTender, []domain.Source, error)
}

// DiscoveryService orchestrates one discovery run at the service layer:
// collect candidates (internal/ai), dedup, persist each as a
// discovery-origin tender, and score it (EP-10). It lives here rather than
// in internal/ai because it depends on TenderService/ScoreService, which
// themselves import internal/ai — putting this logic in internal/ai would
// create an import cycle.
type DiscoveryService struct {
	collector candidateCollector
	tenders   *TenderService
	scores    *ScoreService
	runs      domain.DiscoveryRunRepository
}

func NewDiscoveryService(collector candidateCollector, tenders *TenderService, scores *ScoreService, runs domain.DiscoveryRunRepository) *DiscoveryService {
	return &DiscoveryService{collector: collector, tenders: tenders, scores: scores, runs: runs}
}

// StartRun begins a new discovery_run row, or returns the existing run
// unchanged if correlationKey already matches one (EP-12 ST-12.3.2: a
// retried/scheduled trigger for the same logical request must not start a
// second concurrent run). started=false means an existing run was returned
// rather than a new one created — callers should not re-execute the pipeline
// for it. A nil/empty correlationKey always starts a fresh run (manual runs
// have no natural idempotency key).
func (s *DiscoveryService) StartRun(ctx context.Context, correlationKey *string) (run *domain.DiscoveryRun, started bool, err error) {
	hasKey := correlationKey != nil && *correlationKey != ""

	if hasKey {
		existing, err := s.runs.GetByCorrelationKey(ctx, *correlationKey)
		if err != nil {
			return nil, false, fmt.Errorf("discovery.StartRun: lookup: %w", err)
		}
		if existing != nil {
			return existing, false, nil
		}
	}

	newRun := &domain.DiscoveryRun{
		StartedAt: time.Now(),
		// SourceIDs must be a non-nil empty slice, not nil: the serializer
		// marshals a nil slice to SQL NULL, which violates the column's
		// NOT NULL constraint (the DB-side DEFAULT '[]' only applies when the
		// column is omitted from the INSERT entirely, not when NULL is sent
		// explicitly).
		SourceIDs:      []string{},
		Status:         domain.DiscoveryStatusPending,
		CorrelationKey: correlationKey,
	}
	if err := s.runs.Create(ctx, newRun); err != nil {
		// Race: another request created a run with the same key between our
		// lookup and this create. Fetch and return that one instead of
		// surfacing a spurious error — same treatment as the dedup_key race
		// in persistCandidate.
		if hasKey && isUniqueViolation(err) {
			if existing, gerr := s.runs.GetByCorrelationKey(ctx, *correlationKey); gerr == nil && existing != nil {
				return existing, false, nil
			}
		}
		return nil, false, fmt.Errorf("discovery.StartRun: create: %w", err)
	}
	return newRun, true, nil
}

// ExecuteRun runs the discovery pipeline for an already-started run row,
// advancing its lifecycle pending -> running -> success/failed and recording
// found_count/summary. Callers (e.g. the async HTTP handler, TK-12.4.1, or
// the scheduler, TK-12.5.1) decide whether this runs synchronously or in a
// detached goroutine — this method itself is agnostic to that.
func (s *DiscoveryService) ExecuteRun(ctx context.Context, runID string) error {
	run, err := s.runs.GetByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("discovery.ExecuteRun: get run: %w", err)
	}

	run.Status = domain.DiscoveryStatusRunning
	if err := s.runs.Update(ctx, run); err != nil {
		return fmt.Errorf("discovery.ExecuteRun: mark running: %w", err)
	}

	saved, duplicates, errs := s.runOnce(ctx)

	now := time.Now()
	run.FinishedAt = &now
	run.FoundCount = saved

	summary := fmt.Sprintf("%d tender baru, %d duplikat dilewati", saved, duplicates)
	if len(errs) > 0 {
		summary += fmt.Sprintf(", %d error", len(errs))
	}
	run.Summary = &summary

	// Only a run that produced nothing at all (no saves, no recognized
	// duplicates) alongside errors counts as failed — a run with partial
	// success (some candidates saved despite some scoring/persist errors on
	// others) is still a success, per runOnce's own non-abort contract.
	if saved == 0 && duplicates == 0 && len(errs) > 0 {
		run.Status = domain.DiscoveryStatusFailed
	} else {
		run.Status = domain.DiscoveryStatusSuccess
	}

	if err := s.runs.Update(ctx, run); err != nil {
		return fmt.Errorf("discovery.ExecuteRun: update final: %w", err)
	}
	return nil
}

// RunAsync starts (or, per StartRun's idempotency, reuses) a discovery_run
// row and returns immediately with it — the caller (the HTTP handler, TK-
// 12.4.1) responds without waiting for the crawl to finish. If a genuinely
// new run was started, the pipeline executes in a background goroutine with
// its own timeout-bound, request-independent context, so it isn't canceled
// the instant the HTTP response is written (mirrors the detached-context
// persist pattern in internal/http/handlers/chat_handler.go). A run that was
// merely reused (started=false, an idempotent replay) is not re-executed —
// it is either already running or already finished.
func (s *DiscoveryService) RunAsync(ctx context.Context, correlationKey *string) (*domain.DiscoveryRun, error) {
	run, started, err := s.StartRun(ctx, correlationKey)
	if err != nil {
		return nil, err
	}

	if started {
		go func(runID string) {
			detachedCtx, cancel := context.WithTimeout(context.Background(), discoveryRunTimeout)
			defer cancel()
			if err := s.ExecuteRun(detachedCtx, runID); err != nil {
				log.Printf("discovery: ExecuteRun failed for run %s: %v", runID, err)
			}
		}(run.ID)
	}

	return run, nil
}

// ListRuns returns paginated discovery_run history, newest first.
func (s *DiscoveryService) ListRuns(ctx context.Context, page, pageSize int) ([]domain.DiscoveryRun, int64, error) {
	return s.runs.List(ctx, page, pageSize)
}

// runOnce collects candidates and persists+scores each one that isn't a
// duplicate of an already-known tender. A failure persisting or scoring a
// single candidate is collected into errs and does NOT abort the rest of
// the batch (AC: "satu kandidat gagal tidak menggagalkan kandidat lain"). A
// tender that fails to score is still kept — a legitimate discovery-origin
// tender for the inbox, just unscored; only a persist failure means that
// candidate produces no tender.
func (s *DiscoveryService) runOnce(ctx context.Context) (savedCount, duplicateCount int, errs []error) {
	candidates, _, err := s.collector.CollectCandidates(ctx)
	if err != nil {
		return 0, 0, []error{fmt.Errorf("discovery.runOnce: collect: %w", err)}
	}

	for _, c := range candidates {
		saved, duplicate, err := s.persistCandidate(ctx, c)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if duplicate {
			duplicateCount++
			continue
		}
		if saved != nil {
			savedCount++
			if _, err := s.scores.ScoreTarget(ctx, ai.ScoreTargetTender, saved.ID); err != nil {
				errs = append(errs, fmt.Errorf("discovery.runOnce: score %q (tender %s): %w", c.Title, saved.ID, err))
			}
		}
	}

	return savedCount, duplicateCount, errs
}

// persistCandidate dedups a single candidate against existing tenders
// (EP-12 ST-12.3: "tender sama dari beberapa sumber digabung") before
// creating it. The dedup_key lookup-then-create is inherently racy under
// concurrent runs; the DB's partial unique index on dedup_key is the
// authoritative guard, and a unique-violation on Create here is treated the
// same as a lookup hit — a duplicate, not an error.
func (s *DiscoveryService) persistCandidate(ctx context.Context, c ai.CandidateTender) (tender *domain.Tender, duplicate bool, err error) {
	key := ai.ComputeDedupKey(c.BuyerName, c.Title, c.SubmissionDeadline)
	if key != "" {
		existing, err := s.tenders.FindByDedupKey(ctx, key)
		if err != nil {
			return nil, false, fmt.Errorf("discovery.persistCandidate: dedup lookup %q: %w", c.Title, err)
		}
		if existing != nil {
			return nil, true, nil
		}
	}

	created, err := s.tenders.CreateDiscovery(ctx, c)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("discovery.persistCandidate: persist %q: %w", c.Title, err)
	}
	return created, false, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
