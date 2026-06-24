package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// Configure mengirim konfigurasi AI provider ke bridge via POST /admin/config.
// Bridge menyimpan config in-memory; fallback ke env bila belum pernah di-set.
// Stateless terhadap restart bridge — Go re-push saat boot (TK-18.4.4).
func (c *httpClient) Configure(ctx context.Context, cfg ProviderConfig) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("hermes configure: marshal: %w", err)
	}

	req, err := c.newReq(ctx, "POST", "/admin/config", bytes.NewReader(body), "", "")
	if err != nil {
		return fmt.Errorf("hermes configure: build request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("hermes configure: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hermes configure: %w", readErr(resp))
	}

	return nil
}
