package hermes

import (
	"context"
	"encoding/json"
	"fmt"
)

// Health memeriksa ketersediaan bridge Hermes dalam dua langkah:
//  1. GET /health — liveness (bridge hidup & menerima request)
//  2. GET /v1/capabilities — baca versi & model yang tersedia
//
// Bila salah satu gagal, error dikembalikan; tidak ada panic.
func (c *httpClient) Health(ctx context.Context) (Capabilities, error) {
	// Langkah 1 — liveness.
	liveReq, err := c.newReq(ctx, "GET", "/health", nil, "", "")
	if err != nil {
		return Capabilities{}, fmt.Errorf("hermes health: build /health request: %w", err)
	}

	liveResp, err := c.hc.Do(liveReq)
	if err != nil {
		return Capabilities{}, fmt.Errorf("hermes health: do /health request: %w", err)
	}
	defer func() { _ = liveResp.Body.Close() }()

	if liveResp.StatusCode < 200 || liveResp.StatusCode >= 300 {
		return Capabilities{}, readErr(liveResp)
	}

	var live struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(liveResp.Body).Decode(&live); err != nil {
		return Capabilities{}, fmt.Errorf("hermes health: decode /health response: %w", err)
	}
	if live.Status != "ok" {
		return Capabilities{}, fmt.Errorf("hermes health: status=%q", live.Status)
	}

	// Langkah 2 — kapabilitas.
	capReq, err := c.newReq(ctx, "GET", "/v1/capabilities", nil, "", "")
	if err != nil {
		return Capabilities{}, fmt.Errorf("hermes health: build /v1/capabilities request: %w", err)
	}

	capResp, err := c.hc.Do(capReq)
	if err != nil {
		return Capabilities{}, fmt.Errorf("hermes health: do /v1/capabilities request: %w", err)
	}
	defer func() { _ = capResp.Body.Close() }()

	if capResp.StatusCode < 200 || capResp.StatusCode >= 300 {
		return Capabilities{}, readErr(capResp)
	}

	var caps Capabilities
	if err := json.NewDecoder(capResp.Body).Decode(&caps); err != nil {
		return Capabilities{}, fmt.Errorf("hermes health: decode /v1/capabilities response: %w", err)
	}

	return caps, nil
}
