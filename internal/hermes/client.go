package hermes

import (
	"context"
	"encoding/json"
)

// SessionKey adalah kunci sesi workspace yang dikirim via header X-Hermes-Session-Key.
type SessionKey string

// ToolCall merepresentasikan satu tool call dari Hermes (flat, tidak nested).
type ToolCall struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"` // selalu "function"
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Message adalah satu pesan dalam percakapan.
type Message struct {
	Role      string     `json:"role"` // system | user | assistant | tool
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatRequest adalah permintaan chat ke Hermes.
// SessionKey dan SessionID dikirim sebagai header, bukan body JSON.
type ChatRequest struct {
	Messages   []Message  `json:"messages"`
	Stream     bool       `json:"stream"`
	SessionKey SessionKey `json:"-"`
	SessionID  string     `json:"-"`
	// DocumentBase64/DocumentFilename attach one document (PDF/image) to the
	// LAST user message — the bridge renders PDFs to page images and sends
	// them as native vision input (mirrors GenerateJSONFromDocument's wire).
	DocumentBase64   string `json:"-"`
	DocumentFilename string `json:"-"`
}

// ChatResponse adalah respons non-stream dari Hermes.
type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Chunk adalah satu potongan data dari stream SSE Hermes.
type Chunk struct {
	Delta    string    // potongan teks inkremental
	ToolCall *ToolCall // delta tool-call bila ada (nil bila tidak)
	Done     bool      // true pada akhir stream ([DONE])
	Err      error     // != nil bila terjadi error di tengah stream
}

// Capabilities merepresentasikan kapabilitas yang dilaporkan Hermes.
type Capabilities struct {
	Version  string   `json:"version"`
	Models   []string `json:"models"`
	Features []string `json:"features"`
}

// ProviderConfig berisi konfigurasi AI provider yang dikirim ke bridge via /admin/config.
type ProviderConfig struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	BaseURL  *string `json:"base_url"` // null bila memakai default provider
	APIKey   string  `json:"api_key"`
	// ToolSets menimpa default enabled_toolsets bridge (env ENABLED_TOOLSETS)
	// untuk mode chat. omitempty: slice kosong/nil sengaja tidak dikirim agar
	// bridge tetap memakai default-nya, bukan "toolset kosong".
	ToolSets []string `json:"enabled_toolsets,omitempty"`
}

// Client adalah kontrak abstrak ke Hermes — satu-satunya antarmuka yang boleh dipakai
// layer service/AI. Semua implementasi detail HTTP tersembunyi di balik interface ini.
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan Chunk, error)
	GenerateJSON(ctx context.Context, prompt string, schema any, sk SessionKey) (json.RawMessage, error)
	Health(ctx context.Context) (Capabilities, error)
	Configure(ctx context.Context, cfg ProviderConfig) error
	// ResetMemory clears Hermes workspace memory for sk (EP-16 TK-16.3.1,
	// admin-only). Additive to the interface — mirrors Configure's shape.
	ResetMemory(ctx context.Context, sk SessionKey) error
}

// DocumentExtractor is an optional capability — implemented by the real
// httpClient, sending the actual file bytes to Hermes so the model reads the
// document natively (vision) instead of a lossy Go-side text extraction
// (EP-13 Company Profile PDF ingest: the RFI source document is
// table-heavy, and plain-text extraction mangles table structure).
//
// Deliberately kept OUT of the Client interface above: dozens of test files
// across the codebase define their own minimal stubHermesClient implementing
// just Client's 6 methods, and adding a 7th method there would break every
// one of them for a capability only PDF ingest needs. Callers that need this
// type-assert: `de, ok := hc.(hermes.DocumentExtractor)`.
type DocumentExtractor interface {
	GenerateJSONFromDocument(ctx context.Context, prompt, filename string, fileBytes []byte, schema any, sk SessionKey) (json.RawMessage, error)
}

// MultiDocumentExtractor is like DocumentExtractor but attaches MANY documents
// to a single synchronous JSON generation (bridge renders every page of every
// doc to vision input). Kept optional/out of Client for the same reason as
// DocumentExtractor: callers type-assert `de, ok := hc.(hermes.MultiDocumentExtractor)`.
type MultiDocumentExtractor interface {
	GenerateJSONFromDocuments(ctx context.Context, prompt string, docs []AgentDocument, schema any, sk SessionKey) (json.RawMessage, error)
}
