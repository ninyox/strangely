package routes_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestSessionCreationSetsAnOpaqueHttpOnlyCookie(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	first := c.createSession("Ada")

	cookie := c.lastSessionCookie()
	if cookie == nil {
		t.Fatal("the session cookie was not set")
	}
	if !cookie.HttpOnly {
		t.Error("the session cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.Secure {
		t.Error("the cookie must not be Secure in development over plain HTTP")
	}
	if len(cookie.Value) != 64 {
		t.Errorf("cookie value length = %d, want 64 hex characters", len(cookie.Value))
	}
	if cookie.Value == first.SessionID {
		t.Error("the cookie must not contain the session id")
	}

	// The same cookie resumes the same visit instead of minting a new one.
	second := c.createSession("Ada")
	if second.SessionID != first.SessionID {
		t.Errorf("resumed session id = %q, want %q", second.SessionID, first.SessionID)
	}
	if first.SkipLockMs != testSkipLock.Milliseconds() {
		t.Errorf("skipLockMs = %d, want %d", first.SkipLockMs, testSkipLock.Milliseconds())
	}
}

func TestInvalidUsernamesAreRejected(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	cases := []struct {
		name     string
		username string
	}{
		{"too short", "ab"},
		{"markup", "<script>alert(1)</script>"},
		{"control characters", "ada\u0007lovelace"},
		{"too long", "thisnameisfarmorethantwentycharacterslong"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, body := c.post("/api/session", map[string]string{"username": tc.username})
			if code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %+v)", code, body)
			}
			if body.Code != "invalid_username" {
				t.Errorf("code = %q, want invalid_username", body.Code)
			}
		})
	}
}

func TestUsernameIsNormalizedNotRejected(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	data := c.createSession("   Ada   Lovelace  ")
	if data.Username != "Ada Lovelace" {
		t.Errorf("username = %q, want the trimmed and collapsed form", data.Username)
	}
}

func TestMalformedAndOversizedBodiesAreRejected(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	code, body := c.post("/api/session", rawBody("not json at all"))
	if code != http.StatusBadRequest || body.Code != "invalid_body" {
		t.Errorf("malformed body: status = %d code = %q, want 400 invalid_body", code, body.Code)
	}

	huge := rawBody(`{"username":"` + strings.Repeat("a", 16*1024) + `"}`)
	code, body = c.post("/api/session", huge)
	if code != http.StatusBadRequest {
		t.Errorf("oversized body: status = %d, want 400 (body %+v)", code, body)
	}
}

func TestProtectedRoutesRequireASession(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/matchmaking/status"},
		{http.MethodPost, "/api/matchmaking/join"},
		{http.MethodPost, "/api/matchmaking/cancel"},
		{http.MethodPost, "/api/matches/whatever/skip"},
		{http.MethodPost, "/api/matches/whatever/stop"},
		{http.MethodPost, "/api/matches/whatever/report"},
		{http.MethodGet, "/api/session"},
	} {
		code, body := c.do(route.method, route.path, nil, nil)
		if code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401", route.method, route.path, code)
		}
		if body.Code != "no_session" {
			t.Errorf("%s %s: code = %q, want no_session", route.method, route.path, body.Code)
		}
	}
}
