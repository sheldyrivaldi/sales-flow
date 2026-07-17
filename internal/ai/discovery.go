package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// CandidateTender is one tender-shaped result extracted by a Crawler, before
// it becomes a persisted domain.Tender (EP-12 orchestrator pipeline).
type CandidateTender struct {
	Title                   string
	BuyerName               string
	BuyerCountry            string
	BuyerIndustry           string
	ValueEstimate           *float64
	SubmissionDeadline      *time.Time
	SourceName              string
	SourceURL               string
	ServiceCategory         string
	ScopeSummary            string
	EligibilityRequirements string
	TechnicalRequirements   string
}

// candidateWire is the JSON-facing shape GenerateJSON unmarshals into — it
// needs explicit snake_case tags (CandidateTender has none, since it's an
// internal Go-side type used well beyond this one extraction step) and a
// string SubmissionDeadline: Go's time.Time JSON unmarshaling requires full
// RFC3339, but the extraction prompt asks for a bare "YYYY-MM-DD" (mirrors
// how dto.TenderCreateRequest takes date fields as strings and parses them
// manually rather than relying on json.Unmarshal into time.Time directly).
type candidateWire struct {
	Title                   string   `json:"title"`
	BuyerName               string   `json:"buyer_name"`
	BuyerCountry            string   `json:"buyer_country"`
	BuyerIndustry           string   `json:"buyer_industry"`
	ValueEstimate           *float64 `json:"value_estimate"`
	SubmissionDeadline      string   `json:"submission_deadline"`
	SourceName              string   `json:"source_name"`
	SourceURL               string   `json:"source_url"`
	ServiceCategory         string   `json:"service_category"`
	ScopeSummary            string   `json:"scope_summary"`
	EligibilityRequirements string   `json:"eligibility_requirements"`
	TechnicalRequirements   string   `json:"technical_requirements"`
}

// toCandidateTender converts a wire candidate to the internal shape,
// best-effort-parsing SubmissionDeadline: an empty or unparseable date
// yields nil (deadline unknown), not an error — one candidate's malformed
// date must not fail the whole extraction batch.
func (w candidateWire) toCandidateTender() CandidateTender {
	var deadline *time.Time
	if w.SubmissionDeadline != "" {
		if parsed, err := time.Parse("2006-01-02", w.SubmissionDeadline); err == nil {
			deadline = &parsed
		}
	}
	return CandidateTender{
		Title:                   w.Title,
		BuyerName:               w.BuyerName,
		BuyerCountry:            w.BuyerCountry,
		BuyerIndustry:           w.BuyerIndustry,
		ValueEstimate:           w.ValueEstimate,
		SubmissionDeadline:      deadline,
		SourceName:              w.SourceName,
		SourceURL:               w.SourceURL,
		ServiceCategory:         w.ServiceCategory,
		ScopeSummary:            w.ScopeSummary,
		EligibilityRequirements: w.EligibilityRequirements,
		TechnicalRequirements:   w.TechnicalRequirements,
	}
}

// ComputeDedupKey derives a stable dedup key from a candidate's identifying
// fields (EP-12 AC: "dedup_key = hash(buyer+title+deadline)"). Inputs are
// normalized (trimmed, lowercased, deadline formatted to a fixed layout) so
// the same real-world tender produces the same key regardless of casing/
// whitespace differences between sources. Empty buyer+title (nothing to key
// on) returns "" — callers must treat that as "cannot dedup this candidate".
func ComputeDedupKey(buyer, title string, deadline *time.Time) string {
	buyer = strings.ToLower(strings.TrimSpace(buyer))
	title = strings.ToLower(strings.TrimSpace(title))
	if buyer == "" && title == "" {
		return ""
	}

	deadlineStr := ""
	if deadline != nil {
		deadlineStr = deadline.UTC().Format("2006-01-02")
	}

	sum := sha256.Sum256([]byte(buyer + "|" + title + "|" + deadlineStr))
	return hex.EncodeToString(sum[:])
}

// DiscoverInput is everything a Crawler needs to search for tenders: the
// (already compliance-filtered — see filterCrawlableSources) sources to
// search, the discovery keywords drawn from the Company Profile's keyword
// sets, and the buyer/opportunity target criteria to guide relevance.
type DiscoverInput struct {
	Sources  []domain.Source
	Keywords []string
	Target   *domain.TargetCriteria
	// Products, Vision, Mission come straight from the current Company
	// Profile (EP-18-ish profile revamp) and are injected into the discovery
	// prompt so the agent judges relevance against what the company actually
	// sells and aims for, not just keywords/target criteria.
	Products []string
	Vision   *string
	Mission  *string
}

// Crawler is the seam between the discovery orchestrator and however
// candidates actually get found. The live implementation (hermesCrawler)
// drives Hermes's web toolset; tests substitute a fake so the orchestrator
// pipeline (compliance filter -> crawl -> extract -> dedup -> score -> save)
// is fully unit-testable without a real Hermes bridge.
type Crawler interface {
	Discover(ctx context.Context, in DiscoverInput) ([]CandidateTender, error)
}

// defaultCrawlMinInterval is the minimum pause between two consecutive
// per-source crawl requests to Hermes — a courtesy rate limit (PRD §9:
// "frekuensi wajar + backoff") so discovery never hammers a source or the
// LLM API back-to-back.
const defaultCrawlMinInterval = 3 * time.Second

// defaultCrawlMaxRetries is how many extra attempts a single source gets
// after a transient failure (network blip, malformed LLM output) before
// that source is given up on — the source's error then aborts the whole
// Discover call (a systemic Hermes outage should surface, not be silently
// swallowed source-by-source).
const defaultCrawlMaxRetries = 2

// hermesCrawler is the live Crawler: for each source, it asks the Hermes
// agent (via Chat, so it can use whatever web/browse toolset the bridge
// exposes) to search it, then asks GenerateJSON to extract the results into
// CandidateTender shape. Sources are visited one at a time — never in a
// single combined request — so a per-source rate limit/backoff (ST-12.4.2)
// and the §9 compliance boundary (one source's failure/skip never touches
// another's request) both have somewhere real to attach.
//
// Known limitation (mirrors the EP-09 MCP gap): whether hermes-bridge
// actually wires a web/browse toolset into the agent is not verified from
// the Go side — see epic.plan.md EP-09 notes. This type isolates every
// Hermes interaction so that gap cannot leak into the rest of the discovery
// pipeline, which is tested against the Crawler interface instead.
type hermesCrawler struct {
	hc hermes.Client
	sk hermes.SessionKey
	// minInterval is the pause between sources and the base delay for
	// backoff between retries of the same source. Field (not a constant) so
	// tests can inject a millisecond-scale value and stay fast/deterministic.
	minInterval time.Duration
	// maxRetries is how many retries (beyond the first attempt) a single
	// source gets on a transient error.
	maxRetries int
}

func NewHermesCrawler(hc hermes.Client, sk hermes.SessionKey) Crawler {
	return &hermesCrawler{hc: hc, sk: sk, minInterval: defaultCrawlMinInterval, maxRetries: defaultCrawlMaxRetries}
}

type discoverExtractResult struct {
	Candidates []candidateWire `json:"candidates"`
}

// Discover visits in.Sources one at a time (pausing minInterval between
// each), aggregating every source's candidates. A source that keeps failing
// past maxRetries aborts the whole call — the already-collected candidates
// from prior sources are still returned alongside the error, so a caller
// that wants "best effort so far" can use them, though the current
// orchestrator treats any error here as a hard stop for the run.
func (c *hermesCrawler) Discover(ctx context.Context, in DiscoverInput) ([]CandidateTender, error) {
	if len(in.Sources) == 0 {
		return nil, nil
	}

	var all []CandidateTender
	for i, src := range in.Sources {
		if i > 0 {
			if err := c.wait(ctx, c.minInterval); err != nil {
				return all, err
			}
		}
		candidates, err := c.discoverSourceWithRetry(ctx, src, in)
		if err != nil {
			return all, fmt.Errorf("hermesCrawler.Discover: source %q: %w", src.Name, err)
		}
		all = append(all, candidates...)
	}
	return all, nil
}

// discoverSourceWithRetry attempts one source up to 1+maxRetries times,
// waiting an exponentially increasing backoff (capped at 8x minInterval)
// between attempts. Every wait is ctx-aware so a canceled run stops
// promptly instead of blocking through a long backoff.
func (c *hermesCrawler) discoverSourceWithRetry(ctx context.Context, src domain.Source, in DiscoverInput) ([]CandidateTender, error) {
	var lastErr error
	for attempt := 1; attempt <= c.maxRetries+1; attempt++ {
		if attempt > 1 {
			if err := c.wait(ctx, c.backoffDelay(attempt-1)); err != nil {
				return nil, err
			}
		}
		candidates, err := c.discoverSourceOnce(ctx, src, in)
		if err == nil {
			return candidates, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// discoverSourceOnce runs exactly one Chat+GenerateJSON round trip for a
// single source.
func (c *hermesCrawler) discoverSourceOnce(ctx context.Context, src domain.Source, in DiscoverInput) ([]CandidateTender, error) {
	prompt := buildDiscoveryPrompt(DiscoverInput{
		Sources:  []domain.Source{src},
		Keywords: in.Keywords,
		Target:   in.Target,
		Products: in.Products,
		Vision:   in.Vision,
		Mission:  in.Mission,
	})

	resp, err := c.hc.Chat(ctx, hermes.ChatRequest{
		Messages:   []hermes.Message{{Role: "user", Content: prompt}},
		SessionKey: c.sk,
	})
	if err != nil {
		return nil, fmt.Errorf("chat: %w", err)
	}

	extractPrompt := buildExtractionPrompt(resp.Content)
	var result discoverExtractResult
	if _, err := c.hc.GenerateJSON(ctx, extractPrompt, &result, c.sk); err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}

	candidates := make([]CandidateTender, len(result.Candidates))
	for i, w := range result.Candidates {
		candidates[i] = w.toCandidateTender()
	}
	return candidates, nil
}

// wait pauses for d, honoring ctx cancellation (returns ctx.Err() instead of
// blocking through a canceled run). d<=0 is a no-op — lets tests disable
// pacing entirely by zeroing minInterval.
func (c *hermesCrawler) wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// backoffDelay computes the exponential backoff for retry attempt N
// (1-indexed: the delay before the 2nd attempt, 3rd attempt, ...), capped at
// 8x minInterval so a source with many retries doesn't stall the whole run.
func (c *hermesCrawler) backoffDelay(attempt int) time.Duration {
	if c.minInterval <= 0 {
		return 0
	}
	d := c.minInterval
	for i := 1; i < attempt; i++ {
		d *= 2
	}
	if cap := c.minInterval * 8; d > cap {
		d = cap
	}
	return d
}

func buildDiscoveryPrompt(in DiscoverInput) string {
	var b strings.Builder
	b.WriteString("Kamu adalah agent yang mencari tender/pengadaan dari sumber resmi berikut. ")
	b.WriteString("Gunakan tool pencarian/browsing yang tersedia untuk mengunjungi tiap sumber dan cari tender yang relevan dengan kata kunci.\n\n")

	if len(in.Products) > 0 || (in.Vision != nil && *in.Vision != "") || (in.Mission != nil && *in.Mission != "") {
		b.WriteString("## Profil perusahaan (pakai ini untuk menilai relevansi tender, bukan hanya kata kunci)\n")
		if len(in.Products) > 0 {
			fmt.Fprintf(&b, "- Produk/layanan: %s\n", strings.Join(in.Products, ", "))
		}
		if in.Vision != nil && *in.Vision != "" {
			fmt.Fprintf(&b, "- Visi: %s\n", *in.Vision)
		}
		if in.Mission != nil && *in.Mission != "" {
			fmt.Fprintf(&b, "- Misi: %s\n", *in.Mission)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Sumber (kunjungi HANYA ini, urut prioritas)\n")
	for _, s := range in.Sources {
		fmt.Fprintf(&b, "- %s (%s)\n", s.Name, s.URL)
	}

	if len(in.Keywords) > 0 {
		fmt.Fprintf(&b, "\n## Kata kunci\n%s\n", strings.Join(in.Keywords, ", "))
	}

	if in.Target != nil {
		b.WriteString("\n## Kriteria target\n")
		if len(in.Target.Countries) > 0 {
			fmt.Fprintf(&b, "- Negara: %s\n", strings.Join(in.Target.Countries, ", "))
		}
		if len(in.Target.Industries) > 0 {
			fmt.Fprintf(&b, "- Industri: %s\n", strings.Join(in.Target.Industries, ", "))
		}
	}

	b.WriteString("\nLaporkan tiap tender yang kamu temukan: judul, buyer, negara, industri, nilai, " +
		"deadline submission, nama & URL sumber, kategori layanan, ringkasan scope, syarat eligibilitas, syarat teknis.")

	return b.String()
}

func buildExtractionPrompt(rawFindings string) string {
	return "Ekstrak daftar tender dari teks berikut menjadi JSON. " +
		"Balas HANYA JSON dengan schema persis: " +
		`{"candidates": [{"title":"...","buyer_name":"...","buyer_country":"...","buyer_industry":"...",` +
		`"value_estimate":0,"submission_deadline":"YYYY-MM-DD","source_name":"...","source_url":"...",` +
		`"service_category":"...","scope_summary":"...","eligibility_requirements":"...","technical_requirements":"..."}]}` +
		". Field yang tidak diketahui boleh dikosongkan. Tanpa penjelasan, tanpa markdown, tanpa code fence.\n\nTeks:\n" +
		rawFindings
}

// DiscoveryOrchestrator runs the EP-12 pipeline: resolve Company Profile +
// enabled sources -> crawl -> (compliance/persist/dedup added in later
// stages of this package) -> candidates.
type DiscoveryOrchestrator struct {
	crawler  Crawler
	sources  domain.SourceRepository
	profiles ProfileGetter
	audit    domain.AuditRepository
}

// ProfileGetter is the minimal slice of ProfileService the orchestrator
// needs — avoids importing internal/service (would create an import cycle,
// since service will import this package for scoring).
type ProfileGetter interface {
	GetCurrent(ctx context.Context) (*domain.ProfileAggregate, error)
}

func NewDiscoveryOrchestrator(crawler Crawler, sources domain.SourceRepository, profiles ProfileGetter, audit domain.AuditRepository) *DiscoveryOrchestrator {
	return &DiscoveryOrchestrator{crawler: crawler, sources: sources, profiles: profiles, audit: audit}
}

// filterCrawlableSources splits enabled sources into crawlable (access =
// "publik" — the only ones the compliance guard allows an automated crawl
// to touch, PRD §9) and skipped (access = "login"/"manual" — must only ever
// be marked, never crawled). crawlable is sorted by Priority, highest first.
func filterCrawlableSources(sources []domain.Source) (crawlable, skipped []domain.Source) {
	for _, s := range sources {
		if s.Access == domain.SourceAccessPublik {
			crawlable = append(crawlable, s)
		} else {
			skipped = append(skipped, s)
		}
	}
	sort.SliceStable(crawlable, func(i, j int) bool { return crawlable[i].Priority > crawlable[j].Priority })
	return crawlable, skipped
}

// auditSourceAccess writes a best-effort audit_log row per source (crawled
// or compliance-skipped). A failure here must never abort the run — it
// mirrors the same best-effort pattern as writeAudit in internal/mcp
// (EP-09): loud enough to grep, never fatal to the caller.
func (o *DiscoveryOrchestrator) auditSourceAccess(ctx context.Context, crawlable, skipped []domain.Source) {
	if o.audit == nil {
		return
	}
	for _, s := range crawlable {
		o.writeSourceAudit(ctx, "crawl_source", s, nil)
	}
	for _, s := range skipped {
		access := string(s.Access)
		o.writeSourceAudit(ctx, "skip_source_noncompliant", s, map[string]any{"access": access})
	}
}

func (o *DiscoveryOrchestrator) writeSourceAudit(ctx context.Context, action string, s domain.Source, extra map[string]any) {
	payload := map[string]any{"url": s.URL, "name": s.Name}
	for k, v := range extra {
		payload[k] = v
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("discovery: AUDIT FAILURE: marshal payload untuk %s (source=%s): %v", action, s.ID, err)
		return
	}
	targetType := "source"
	targetID := s.ID
	e := &domain.AuditEvent{
		Actor:      "discovery",
		Action:     action,
		TargetType: &targetType,
		TargetID:   &targetID,
		Payload:    payloadJSON,
	}
	if err := o.audit.Create(ctx, e); err != nil {
		log.Printf("discovery: AUDIT FAILURE: gagal menulis audit_log untuk %s (source=%s): %v", action, s.ID, err)
	}
}

// CollectCandidates resolves the current profile + enabled sources, applies
// the §9 compliance guard (filterCrawlableSources — only "publik" sources
// ever reach the crawler; "login"/"manual" are marked via audit_log and
// skipped, never bypassed), and asks the crawler to search the crawlable set.
// CrawlableSources returns the enabled+public sources a run would actually
// crawl — exposed so DiscoveryService can refuse to start a run when there
// is nothing to crawl (instead of "succeeding" in milliseconds with zero
// results and no explanation).
func (o *DiscoveryOrchestrator) CrawlableSources(ctx context.Context) ([]domain.Source, error) {
	enabled := true
	sources, _, err := o.sources.List(ctx, domain.SourceFilter{Enabled: &enabled}, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("discovery.CrawlableSources: %w", err)
	}
	crawlable, _ := filterCrawlableSources(sources)
	return crawlable, nil
}

func (o *DiscoveryOrchestrator) CollectCandidates(ctx context.Context) ([]CandidateTender, []domain.Source, error) {
	profile, err := o.profiles.GetCurrent(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("discovery.CollectCandidates: profile: %w", err)
	}

	enabled := true
	sources, _, err := o.sources.List(ctx, domain.SourceFilter{Enabled: &enabled}, 1, 1000)
	if err != nil {
		return nil, nil, fmt.Errorf("discovery.CollectCandidates: sources: %w", err)
	}

	crawlable, skipped := filterCrawlableSources(sources)
	o.auditSourceAccess(ctx, crawlable, skipped)

	if len(crawlable) == 0 {
		return nil, sources, nil
	}

	keywords := flattenKeywords(profile.Keywords)
	candidates, err := o.crawler.Discover(ctx, DiscoverInput{
		Sources:  crawlable,
		Keywords: keywords,
		Target:   profile.Target,
		Products: profile.Profile.Products,
		Vision:   profile.Profile.Vision,
		Mission:  profile.Profile.Mission,
	})
	if err != nil {
		return nil, sources, fmt.Errorf("discovery.CollectCandidates: crawl: %w", err)
	}

	return candidates, sources, nil
}

func flattenKeywords(sets []domain.KeywordSet) []string {
	var out []string
	for _, ks := range sets {
		out = append(out, ks.Keywords...)
	}
	return out
}
