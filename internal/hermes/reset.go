package hermes

import (
	"context"
	"fmt"
)

// ResetMemory clears Hermes workspace memory for sk via POST
// /admin/reset-memory on the bridge (EP-16 TK-16.3.1, admin-only). Mirrors
// Configure's request shape: Bearer auth is set by newReq, sk goes in the
// X-Hermes-Session-Key header rather than the body.
func (c *httpClient) ResetMemory(ctx context.Context, sk SessionKey) error {
	req, err := c.newReq(ctx, "POST", "/admin/reset-memory", nil, sk, "")
	if err != nil {
		return fmt.Errorf("hermes reset memory: build request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("hermes reset memory: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hermes reset memory: %w", readErr(resp))
	}

	return nil
}
