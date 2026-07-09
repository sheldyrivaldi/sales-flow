package mcp

import (
	"context"
	"encoding/json"
	"log"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"salespilot/internal/domain"
)

// writeToolWhitelist is the only set of tool names registerWriteTools may
// register (P-9: write tool "hanya bila di whitelist tools.include"). Must
// stay in sync with deploy/hermes/config.yaml.example tools.include.
var writeToolWhitelist = map[string]bool{
	"update_prospect_stage": true,
	"save_playbook_draft":   true,
}

// addWriteTool registers tool only if its name is in writeToolWhitelist — a
// defense-in-depth guard so a future write tool can't be wired in without
// deliberately adding it to the whitelist first.
func addWriteTool(s *mcpserver.MCPServer, tool mcpgo.Tool, handler mcpserver.ToolHandlerFunc) {
	if !writeToolWhitelist[tool.Name] {
		panic("mcp: write tool " + tool.Name + " tidak ada di whitelist")
	}
	s.AddTool(tool, handler)
}

// registerWriteTools registers the 2 gated write tools from
// deploy/hermes/config.yaml.example. Both are drafts/proposals, not final
// actions — human-in-the-loop review happens outside the tool call (P-9),
// and every successful write is recorded to audit_log.
func registerWriteTools(s *mcpserver.MCPServer, deps Deps) {
	addWriteTool(s, mcpgo.NewTool("update_prospect_stage",
		mcpgo.WithDescription("Ubah stage prospek (usulan aksi — perlu konteks human-in-the-loop, bukan aksi final tersembunyi). WON/LOST otomatis tercatat sebagai outcome_event."),
		mcpgo.WithString("prospect_id", mcpgo.Required(), mcpgo.Description("ID prospek (UUID)")),
		mcpgo.WithString("to_stage", mcpgo.Required(), mcpgo.Description("Stage tujuan: NEW, QUALIFIED, ENGAGED, PROPOSAL, WON, LOST")),
		mcpgo.WithString("notes", mcpgo.Description("Catatan opsional (disimpan pada outcome bila WON/LOST)")),
	), updateProspectStageHandler(deps))

	addWriteTool(s, mcpgo.NewTool("save_playbook_draft",
		mcpgo.WithDescription("Simpan draft playbook untuk tender/prospek. Ini draft, bukan aksi final — perlu ditinjau manusia sebelum dipakai."),
		mcpgo.WithString("target_type", mcpgo.Required(), mcpgo.Description("tender atau prospect")),
		mcpgo.WithString("target_id", mcpgo.Required(), mcpgo.Description("ID tender/prospek (UUID)")),
		mcpgo.WithString("title", mcpgo.Description("Judul draft (opsional)")),
		mcpgo.WithObject("content", mcpgo.Required(), mcpgo.Description("Konten draft playbook (objek JSON bebas — mis. sections)")),
	), savePlaybookDraftHandler(deps))
}

func updateProspectStageHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		prospectID, err := req.RequireString("prospect_id")
		if err != nil {
			return mcpgo.NewToolResultError("prospect_id wajib diisi"), nil
		}
		toStageRaw, err := req.RequireString("to_stage")
		if err != nil {
			return mcpgo.NewToolResultError("to_stage wajib diisi"), nil
		}
		toStage := domain.ProspectStage(toStageRaw)
		if !toStage.Valid() {
			return mcpgo.NewToolResultError("to_stage tidak valid"), nil
		}
		notes := req.GetString("notes", "")

		p, err := deps.Prospect.UpdateStage(ctx, prospectID, toStage, notes)
		if err != nil {
			return toolResultFromErr(err, "gagal mengubah stage prospek")
		}

		writeAudit(ctx, deps, "update_prospect_stage", "prospect", prospectID, map[string]any{
			"to_stage": string(toStage),
			"notes":    notes,
		})

		return mcpgo.NewToolResultJSON(p)
	}
}

func savePlaybookDraftHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		targetTypeRaw, err := req.RequireString("target_type")
		if err != nil {
			return mcpgo.NewToolResultError("target_type wajib diisi"), nil
		}
		targetType := domain.PlaybookTargetType(targetTypeRaw)
		if !targetType.Valid() {
			return mcpgo.NewToolResultError("target_type harus tender atau prospect"), nil
		}
		targetID, err := req.RequireString("target_id")
		if err != nil {
			return mcpgo.NewToolResultError("target_id wajib diisi"), nil
		}

		// Verify the target actually exists before persisting a draft against
		// it — target_id has no FK (tender/prospect are separate tables), so
		// without this check a typo'd or mismatched-type id would silently
		// create an orphan draft that EP-14 can never surface.
		switch targetType {
		case domain.PlaybookTargetTender:
			if _, err := deps.Tender.Get(ctx, targetID); err != nil {
				return toolResultFromErr(err, "tender tujuan tidak ditemukan")
			}
		case domain.PlaybookTargetProspect:
			if _, err := deps.Prospect.Get(ctx, targetID); err != nil {
				return toolResultFromErr(err, "prospek tujuan tidak ditemukan")
			}
		}

		contentArg, ok := req.GetArguments()["content"]
		if !ok || contentArg == nil {
			return mcpgo.NewToolResultError("content wajib diisi"), nil
		}
		contentJSON, err := json.Marshal(contentArg)
		if err != nil {
			return mcpgo.NewToolResultError("content tidak valid"), nil
		}

		draft := &domain.PlaybookDraft{
			TargetType: string(targetType),
			TargetID:   targetID,
			Content:    contentJSON,
			Source:     "mcp",
		}
		if title := req.GetString("title", ""); title != "" {
			draft.Title = &title
		}

		if err := deps.Playbook.Create(ctx, draft); err != nil {
			return toolResultFromErr(err, "gagal menyimpan draft playbook")
		}

		writeAudit(ctx, deps, "save_playbook_draft", string(targetType), targetID, map[string]any{
			"draft_id": draft.ID,
			"title":    draft.Title,
		})

		return mcpgo.NewToolResultJSON(draft)
	}
}

// writeAudit best-effort records an audit_log row for a write tool call.
// The underlying mutation has already been committed by the time this runs
// (same non-transactional sequencing already accepted elsewhere in this
// codebase for OutcomeEvent — see recordOutcome in internal/service/outcome.go),
// so a failure here must not undo it. It retries once immediately to absorb
// a transient blip (pool exhaustion, brief connection drop) — the most
// common real-world cause of a single write failing — and logs loudly with
// a greppable prefix if it still fails, since silent, unretried audit loss
// is the failure mode this whole tool exists to avoid. Full atomicity would
// require a shared transaction across the service and audit layers, which
// is out of scope here (EP-17).
func writeAudit(ctx context.Context, deps Deps, action, targetType, targetID string, payload map[string]any) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("mcp: AUDIT FAILURE: payload marshal gagal untuk %s (target=%s/%s): %v", action, targetType, targetID, err)
		return
	}
	e := &domain.AuditEvent{
		Actor:      "mcp",
		Action:     action,
		TargetType: &targetType,
		TargetID:   &targetID,
		Payload:    payloadJSON,
	}
	if err := deps.Audit.Create(ctx, e); err != nil {
		if err := deps.Audit.Create(ctx, e); err != nil {
			log.Printf("mcp: AUDIT FAILURE: gagal menulis audit_log untuk %s (target=%s/%s) setelah retry: %v", action, targetType, targetID, err)
		}
	}
}
