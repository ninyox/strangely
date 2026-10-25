package routes_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUnknownCookieIsRejectedAndCleared(t *testing.T) {
	env := newTestEnv(t)

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/matchmaking/status", nil)
	if err != nil {
		t.Fatalf("could not build request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "strg_session", Value: strings.Repeat("f", 64)})

	plain := &http.Client{Timeout: 10 * time.Second}
	resp, err := plain.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	cleared := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "strg_session" && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("a stale session cookie should be cleared")
	}
}

func TestHealthReadyAndConfigEndpoints(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	code, body := c.get("/healthz")
	if code != http.StatusOK || !body.Success {
		t.Fatalf("/healthz = %d %+v, want 200", code, body)
	}

	code, body = c.get("/readyz")
	if code != http.StatusOK {
		t.Fatalf("/readyz = %d, want 200", code)
	}
	ready := decode[struct {
		Status string `json:"status"`
	}](t, body.Data)
	if ready.Status != "ready" {
		t.Errorf("/readyz status = %q, want ready", ready.Status)
	}

	code, body = c.get("/api/config")
	if code != http.StatusOK {
		t.Fatalf("/api/config = %d, want 200", code)
	}
	cfg := decode[struct {
		IceServers []struct {
			URLs []string `json:"urls"`
		} `json:"iceServers"`
		SkipLockSeconds   float64  `json:"skipLockSeconds"`
		MaxSignalingBytes int      `json:"maxSignalingBytes"`
		ReportReasons     []string `json:"reportReasons"`
	}](t, body.Data)

	if len(cfg.IceServers) == 0 || len(cfg.IceServers[0].URLs) == 0 {
		t.Error("the public config should carry at least one ICE server")
	} else if cfg.IceServers[0].URLs[0] != "stun:stun.example.test:3478" {
		t.Errorf("ICE url = %q, want the configured STUN url", cfg.IceServers[0].URLs[0])
	}
	if cfg.SkipLockSeconds != testSkipLock.Seconds() {
		t.Errorf("skipLockSeconds = %v, want %v", cfg.SkipLockSeconds, testSkipLock.Seconds())
	}
	if cfg.MaxSignalingBytes <= 0 {
		t.Error("maxSignalingBytes should be published to the client")
	}
	if len(cfg.ReportReasons) == 0 {
		t.Error("report reasons should be published so the UI cannot invent them")
	}

	// An unknown endpoint is a JSON 404, not gin's HTML default.
	code, body = c.get("/api/nope")
	if code != http.StatusNotFound || body.Code != "not_found" {
		t.Errorf("/api/nope = %d %q, want 404 not_found", code, body.Code)
	}
}

func TestReadyzReportsDraining(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	env.api.MarkNotReady()

	code, body := c.get("/readyz")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("/readyz while draining = %d, want 503", code)
	}
	if body.Code != "not_ready" {
		t.Errorf("code = %q, want not_ready", body.Code)
	}
}
