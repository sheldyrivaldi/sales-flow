package hermestui

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// TUIBasePath is where Go mounts the hermes-tui proxy. Unlike the old ttyd
// setup (which had a matching -b/--base-path flag so paths could be
// forwarded unchanged), the Hermes dashboard always serves its own routes
// rooted at "/" and has no base-path flag — it instead honours the
// X-Forwarded-Prefix request header for sub-path deployments (see
// hermes_cli/web_server.py's mount_spa/_normalise_prefix in the official
// image). So the proxy must both strip this prefix from the forwarded
// request path AND set X-Forwarded-Prefix to this value (see ProxyHTTP/
// ProxyWS in hermes_tui_handler.go) — ttyd's "forward unchanged" trick no
// longer applies.
const TUIBasePath = "/api/admin/hermes/tui"

// SessionCookieName is the HttpOnly cookie carrying the TUI session JWT
// (auth.TokenTUISession). A cookie, not a header, because browsers attach
// cookies automatically to same-origin requests — including the WS
// handshake ttyd's frontend initiates itself, which cannot be given a
// custom Authorization header from JS in the way apiFetch normally does.
const SessionCookieName = "hermes_tui_session"

// WSURL converts an http(s) base URL (e.g. "http://hermes-tui:7681") into a
// ws(s) URL with path appended unchanged.
func WSURL(baseURL, path string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("hermestui.WSURL: parse base URL: %w", err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		return "", fmt.Errorf("hermestui.WSURL: unsupported scheme %q", u.Scheme)
	}
	u.Path = path
	u.RawQuery = ""
	return u.String(), nil
}

// Pump bidirectionally relays WS frames between client and bridge, preserving
// message type (text vs binary) on both legs without ever parsing them —
// the upstream's wire protocol (whichever of the dashboard's several WS
// endpoints this is) stays opaque to Go (see plan §Cross-cutting design:
// framing convention). touch (may be nil) is called on every successfully
// relayed frame, in either direction, for idle-timeout tracking (wires into
// Registry.Touch). Pump blocks until both directions have stopped — either
// side closing/erroring on its own, or ctx being cancelled by an explicit
// end or sweeper reaping the session — then returns.
func Pump(ctx context.Context, client, bridge *websocket.Conn, touch func()) {
	var closeOnce sync.Once
	closeBoth := func() {
		closeOnce.Do(func() {
			_ = client.Close()
			_ = bridge.Close()
		})
	}

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			closeBoth()
		case <-stop:
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	relay := func(dst, src *websocket.Conn) {
		defer wg.Done()
		defer closeBoth()
		for {
			mt, data, err := src.ReadMessage()
			if err != nil {
				return
			}
			if touch != nil {
				touch()
			}
			if err := dst.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}
	go relay(bridge, client)
	go relay(client, bridge)
	wg.Wait()
}
