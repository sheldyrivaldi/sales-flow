package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"gorm.io/gorm"

	"salespilot/internal/domain"
	"salespilot/internal/mcp"
	"salespilot/internal/service"
)

// --- In-memory fakes (no DB needed — this test must stay deterministic and
// runnable in CI without Postgres; see task.plan.md TK-09.4.1). ---

type fakeTenderRepo struct{ items map[string]domain.Tender }

func (r *fakeTenderRepo) Create(_ context.Context, t *domain.Tender) error {
	if t.ID == "" {
		t.ID = fmt.Sprintf("tender-%d", len(r.items)+1)
	}
	r.items[t.ID] = *t
	return nil
}
func (r *fakeTenderRepo) GetByID(_ context.Context, id string) (*domain.Tender, error) {
	t, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &t, nil
}
func (r *fakeTenderRepo) List(_ context.Context, f domain.TenderFilter, _, _ int) ([]domain.Tender, int64, error) {
	var out []domain.Tender
	for _, t := range r.items {
		if f.Status != nil && t.Status != *f.Status {
			continue
		}
		if f.Search != "" {
			q := strings.ToLower(f.Search)
			titleMatch := strings.Contains(strings.ToLower(t.Title), q)
			buyerMatch := t.BuyerName != nil && strings.Contains(strings.ToLower(*t.BuyerName), q)
			if !titleMatch && !buyerMatch {
				continue
			}
		}
		out = append(out, t)
	}
	return out, int64(len(out)), nil
}
func (r *fakeTenderRepo) Update(_ context.Context, t *domain.Tender) error {
	r.items[t.ID] = *t
	return nil
}
func (r *fakeTenderRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}

type fakeEventRepo struct{ items map[string]domain.Event }

func (r *fakeEventRepo) Create(_ context.Context, e *domain.Event) error {
	if e.ID == "" {
		e.ID = fmt.Sprintf("event-%d", len(r.items)+1)
	}
	r.items[e.ID] = *e
	return nil
}
func (r *fakeEventRepo) GetByID(_ context.Context, id string) (*domain.Event, error) {
	e, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &e, nil
}
func (r *fakeEventRepo) List(_ context.Context, _ domain.EventFilter, _, _ int) ([]domain.Event, int64, error) {
	var out []domain.Event
	for _, e := range r.items {
		out = append(out, e)
	}
	return out, int64(len(out)), nil
}
func (r *fakeEventRepo) Update(_ context.Context, e *domain.Event) error {
	r.items[e.ID] = *e
	return nil
}
func (r *fakeEventRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}

type fakeProspectRepo struct{ items map[string]domain.Prospect }

func (r *fakeProspectRepo) Create(_ context.Context, p *domain.Prospect) error {
	if p.ID == "" {
		p.ID = fmt.Sprintf("prospect-%d", len(r.items)+1)
	}
	r.items[p.ID] = *p
	return nil
}
func (r *fakeProspectRepo) GetByID(_ context.Context, id string) (*domain.Prospect, error) {
	p, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}
func (r *fakeProspectRepo) GetBySource(_ context.Context, _ domain.ProspectSource, _ string) (*domain.Prospect, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeProspectRepo) List(_ context.Context, f domain.ProspectFilter, _, _ int) ([]domain.Prospect, int64, error) {
	var out []domain.Prospect
	for _, p := range r.items {
		if f.Stage != nil && p.Stage != *f.Stage {
			continue
		}
		out = append(out, p)
	}
	return out, int64(len(out)), nil
}
func (r *fakeProspectRepo) Update(_ context.Context, p *domain.Prospect) error {
	r.items[p.ID] = *p
	return nil
}
func (r *fakeProspectRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}
func (r *fakeProspectRepo) SummaryByStage(_ context.Context) ([]domain.ProspectStageSummary, error) {
	agg := map[domain.ProspectStage]*domain.ProspectStageSummary{}
	for _, p := range r.items {
		s, ok := agg[p.Stage]
		if !ok {
			s = &domain.ProspectStageSummary{Stage: p.Stage}
			agg[p.Stage] = s
		}
		s.Count++
		if p.EstValue != nil {
			s.TotalValue += *p.EstValue
		}
	}
	var out []domain.ProspectStageSummary
	for _, s := range agg {
		out = append(out, *s)
	}
	return out, nil
}

type fakeProfileRepo struct{ current *domain.ProfileAggregate }

func (r *fakeProfileRepo) GetCurrent(_ context.Context) (*domain.ProfileAggregate, error) {
	if r.current == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.current, nil
}
func (r *fakeProfileRepo) CreateVersion(_ context.Context, agg *domain.ProfileAggregate) (*domain.ProfileAggregate, error) {
	r.current = agg
	return agg, nil
}

type fakeOutcomeRepo struct{ events []domain.OutcomeEvent }

func (r *fakeOutcomeRepo) Create(_ context.Context, e *domain.OutcomeEvent) error {
	r.events = append(r.events, *e)
	return nil
}

type fakeAuditRepo struct{ events []domain.AuditEvent }

func (r *fakeAuditRepo) Create(_ context.Context, e *domain.AuditEvent) error {
	r.events = append(r.events, *e)
	return nil
}

type fakePlaybookRepo struct{ drafts []domain.PlaybookDraft }

func (r *fakePlaybookRepo) Create(_ context.Context, d *domain.PlaybookDraft) error {
	if d.ID == "" {
		d.ID = fmt.Sprintf("draft-%d", len(r.drafts)+1)
	}
	r.drafts = append(r.drafts, *d)
	return nil
}
func (r *fakePlaybookRepo) List(_ context.Context, targetType, targetID string) ([]domain.PlaybookDraft, error) {
	var out []domain.PlaybookDraft
	for _, d := range r.drafts {
		if d.TargetType == targetType && d.TargetID == targetID {
			out = append(out, d)
		}
	}
	return out, nil
}

// testHarness wires fakes into real services (same construction as
// internal/http/router.go) and starts an httptest server exposing /mcp
// through the same Echo + BearerAuth path used in production.
type testHarness struct {
	server    *httptest.Server
	token     string
	prospects *fakeProspectRepo
	tenders   *fakeTenderRepo
	audit     *fakeAuditRepo
	playbook  *fakePlaybookRepo
}

func newHarness(t *testing.T) *testHarness {
	t.Helper()

	tenderRepo := &fakeTenderRepo{items: map[string]domain.Tender{}}
	eventRepo := &fakeEventRepo{items: map[string]domain.Event{}}
	prospectRepo := &fakeProspectRepo{items: map[string]domain.Prospect{}}
	profileRepo := &fakeProfileRepo{}
	outcomeRepo := &fakeOutcomeRepo{}
	auditRepo := &fakeAuditRepo{}
	playbookRepo := &fakePlaybookRepo{}

	deps := mcp.Deps{
		Tender:       service.NewTenderService(tenderRepo, outcomeRepo, service.NoopLearningHook()),
		Event:        service.NewEventService(eventRepo, prospectRepo),
		Prospect:     service.NewProspectService(prospectRepo, outcomeRepo, service.NoopLearningHook()),
		Profile:      service.NewProfileService(profileRepo),
		ProspectRepo: prospectRepo,
		Audit:        auditRepo,
		Playbook:     playbookRepo,
	}

	srv := mcp.NewServer(deps)
	e := echo.New()
	token := "test-mcp-token"
	e.Any("/mcp", echo.WrapHandler(mcp.Handler(srv)), mcp.BearerAuth(token))

	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)

	return &testHarness{
		server:    ts,
		token:     token,
		prospects: prospectRepo,
		tenders:   tenderRepo,
		audit:     auditRepo,
		playbook:  playbookRepo,
	}
}

func (h *testHarness) mcpClient(t *testing.T) *mcpclient.Client {
	t.Helper()
	c, err := mcpclient.NewStreamableHttpClient(h.server.URL+"/mcp", transport.WithHTTPHeaders(map[string]string{
		"Authorization": "Bearer " + h.token,
	}))
	if err != nil {
		t.Fatalf("NewStreamableHttpClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("client Start: %v", err)
	}
	_, err = c.Initialize(ctx, mcpgo.InitializeRequest{
		Params: mcpgo.InitializeParams{
			ProtocolVersion: mcpgo.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcpgo.Implementation{Name: "mcp-test", Version: "1.0.0"},
		},
	})
	if err != nil {
		t.Fatalf("client Initialize: %v", err)
	}
	return c
}

// toolResultJSON decodes the text-fallback JSON of a CallToolResult into v.
func toolResultJSON(t *testing.T, res *mcpgo.CallToolResult, v any) {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool call returned isError=true: %+v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatalf("tool call returned no content")
	}
	text, ok := mcpgo.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("tool call content[0] is not text content: %+v", res.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), v); err != nil {
		t.Fatalf("decode tool result JSON: %v (raw=%s)", err, text.Text)
	}
}

// wantAuthorized is not a full JSON-RPC MCP request, but any well-formed
// POST reaching the handler proves BearerAuth passed the request through
// (auth is asserted separately without a full handshake).
func TestMCP_Unauthenticated_Rejected(t *testing.T) {
	h := newHarness(t)

	resp, err := http.Post(h.server.URL+"/mcp", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (no Authorization header)", resp.StatusCode)
	}
}

func TestMCP_WrongToken_Rejected(t *testing.T) {
	h := newHarness(t)

	req, _ := http.NewRequest(http.MethodPost, h.server.URL+"/mcp", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (wrong token)", resp.StatusCode)
	}
}

// wantTools is the stable contract from deploy/hermes/config.yaml.example
// (mcp_servers.sales.tools.include) — must stay exact and additive-only.
var wantTools = []string{
	"list_tenders", "get_tender", "search_tenders",
	"list_events", "get_event",
	"list_prospects", "get_prospect",
	"get_pipeline_summary", "get_revenue_summary", "get_company_profile",
	"update_prospect_stage", "save_playbook_draft",
}

func TestMCP_ToolsList_MatchesContract(t *testing.T) {
	h := newHarness(t)
	c := h.mcpClient(t)

	res, err := c.ListTools(context.Background(), mcpgo.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	for _, name := range wantTools {
		if !got[name] {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
	if len(res.Tools) != len(wantTools) {
		t.Errorf("tools/list returned %d tools, want %d (%v)", len(res.Tools), len(wantTools), res.Tools)
	}
}

func TestMCP_ListTenders_ReturnsSeededData(t *testing.T) {
	h := newHarness(t)
	h.tenders.items["t1"] = domain.Tender{
		ID: "t1", Title: "Pengadaan Server", Status: domain.TenderStatusIdentified,
		Currency: "IDR", Origin: domain.OriginManual,
	}
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "list_tenders"},
	})
	if err != nil {
		t.Fatalf("CallTool list_tenders: %v", err)
	}

	var out struct {
		Items []domain.Tender `json:"items"`
		Total int64           `json:"total"`
	}
	toolResultJSON(t, res, &out)

	if out.Total != 1 || len(out.Items) != 1 || out.Items[0].ID != "t1" {
		t.Errorf("list_tenders result = %+v, want 1 item t1", out)
	}
}

func TestMCP_ListTenders_EmptyResult_ItemsIsEmptyArrayNotNull(t *testing.T) {
	h := newHarness(t)
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "list_tenders"},
	})
	if err != nil {
		t.Fatalf("CallTool list_tenders: %v", err)
	}
	if res.IsError {
		t.Fatalf("list_tenders on empty data returned error: %+v", res.Content)
	}
	text, ok := mcpgo.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("content[0] is not text content: %+v", res.Content[0])
	}
	// Asserted on the raw JSON text, not the decoded Go struct: json.Unmarshal
	// maps both "items":[] and "items":null to a nil []domain.Tender, so
	// decoding would hide exactly the regression this test guards against.
	if !strings.Contains(text.Text, `"items":[]`) {
		t.Errorf("list_tenders on empty data: items is not an empty array: raw=%s", text.Text)
	}
}

func TestMCP_GetProspect_NotFound_ReturnsToolError(t *testing.T) {
	h := newHarness(t)
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "get_prospect", Arguments: map[string]any{"id": "missing"}},
	})
	if err != nil {
		t.Fatalf("CallTool get_prospect: %v", err)
	}
	if !res.IsError {
		t.Errorf("get_prospect with missing id: IsError = false, want true")
	}
}

func TestMCP_GetPipelineSummary_AggregatesByStage(t *testing.T) {
	h := newHarness(t)
	v1, v2 := 1_000_000.0, 2_000_000.0
	h.prospects.items["p1"] = domain.Prospect{ID: "p1", Name: "A", Stage: domain.ProspectStageNew, EstValue: &v1}
	h.prospects.items["p2"] = domain.Prospect{ID: "p2", Name: "B", Stage: domain.ProspectStageWon, EstValue: &v2}
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "get_pipeline_summary"},
	})
	if err != nil {
		t.Fatalf("CallTool get_pipeline_summary: %v", err)
	}

	var out struct {
		TotalCount int64   `json:"total_count"`
		TotalValue float64 `json:"total_value"`
	}
	toolResultJSON(t, res, &out)

	if out.TotalCount != 2 || out.TotalValue != 3_000_000.0 {
		t.Errorf("get_pipeline_summary = %+v, want count=2 value=3000000", out)
	}
}

func TestMCP_GetCompanyProfile_ReturnsDefaultTemplate(t *testing.T) {
	h := newHarness(t)
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "get_company_profile"},
	})
	if err != nil {
		t.Fatalf("CallTool get_company_profile: %v", err)
	}

	// Shape must match dto.ProfileResponse (the same DTO GET /api/profile
	// returns) — flat, snake_case, "version" at top level — not the
	// untagged domain.ProfileAggregate's Go field names.
	var out struct {
		Version int `json:"version"`
	}
	toolResultJSON(t, res, &out)
	if out.Version != 0 {
		t.Errorf("get_company_profile on empty workspace: version = %d, want 0 (default template)", out.Version)
	}
}

func TestMCP_UpdateProspectStage_UpdatesAndWritesAudit(t *testing.T) {
	h := newHarness(t)
	h.prospects.items["p1"] = domain.Prospect{ID: "p1", Name: "A", Stage: domain.ProspectStageNew}
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "update_prospect_stage", Arguments: map[string]any{
			"prospect_id": "p1",
			"to_stage":    "QUALIFIED",
			"notes":       "dari MCP test",
		}},
	})
	if err != nil {
		t.Fatalf("CallTool update_prospect_stage: %v", err)
	}
	if res.IsError {
		t.Fatalf("update_prospect_stage returned error: %+v", res.Content)
	}

	if got := h.prospects.items["p1"].Stage; got != domain.ProspectStageQualified {
		t.Errorf("prospect stage = %s, want QUALIFIED", got)
	}
	if len(h.audit.events) != 1 || h.audit.events[0].Action != "update_prospect_stage" {
		t.Errorf("audit events = %+v, want 1 update_prospect_stage entry", h.audit.events)
	}
}

func TestMCP_SavePlaybookDraft_PersistsAndWritesAudit(t *testing.T) {
	h := newHarness(t)
	h.prospects.items["p1"] = domain.Prospect{ID: "p1", Name: "A", Stage: domain.ProspectStageNew}
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "save_playbook_draft", Arguments: map[string]any{
			"target_type": "prospect",
			"target_id":   "p1",
			"title":       "Draft awal",
			"content":     map[string]any{"ringkasan": "test"},
		}},
	})
	if err != nil {
		t.Fatalf("CallTool save_playbook_draft: %v", err)
	}
	if res.IsError {
		t.Fatalf("save_playbook_draft returned error: %+v", res.Content)
	}

	if len(h.playbook.drafts) != 1 {
		t.Fatalf("playbook drafts = %d, want 1", len(h.playbook.drafts))
	}
	if len(h.audit.events) != 1 || h.audit.events[0].Action != "save_playbook_draft" {
		t.Errorf("audit events = %+v, want 1 save_playbook_draft entry", h.audit.events)
	}
}

func TestMCP_SavePlaybookDraft_UnknownTarget_RejectedWithoutPersisting(t *testing.T) {
	h := newHarness(t)
	c := h.mcpClient(t)

	res, err := c.CallTool(context.Background(), mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: "save_playbook_draft", Arguments: map[string]any{
			"target_type": "prospect",
			"target_id":   "does-not-exist",
			"content":     map[string]any{"ringkasan": "test"},
		}},
	})
	if err != nil {
		t.Fatalf("CallTool save_playbook_draft: %v", err)
	}
	if !res.IsError {
		t.Fatalf("save_playbook_draft with unknown target_id: IsError = false, want true")
	}
	if len(h.playbook.drafts) != 0 {
		t.Errorf("playbook drafts = %d, want 0 (draft must not persist for a nonexistent target)", len(h.playbook.drafts))
	}
	if len(h.audit.events) != 0 {
		t.Errorf("audit events = %d, want 0 (no audit for a rejected write)", len(h.audit.events))
	}
}
