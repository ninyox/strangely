package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/config"
	"strgback/handlers"
	"strgback/routes"
	"strgback/store"
)

// testOrigin is the origin the test environment allows. Anything else must be
// refused by CORS and by the WebSocket upgrade check.
const testOrigin = "http://localhost:5173"

// testSkipLock is short so the "after the window" cases do not slow the suite
// down, while still being long enough that an immediate skip is refused.
const testSkipLock = 300 * time.Millisecond

type testEnv struct {
	server *httptest.Server
	api    *handlers.API
	store  *store.Store
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWith(t, nil)
}

// newTestEnvWith builds the environment with per-test configuration overrides,
// which is how the rate limit tests tighten a budget.
func newTestEnvWith(t *testing.T, overrides map[string]string) *testEnv {
	t.Helper()

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard

	// Deterministic configuration, independent of the developer's shell.
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("ALLOWED_ORIGINS", testOrigin)
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("STUN_URLS", "stun:stun.example.test:3478")
	t.Setenv("RATE_LIMIT_SESSION_PER_MINUTE", "20")
	t.Setenv("RATE_LIMIT_JOIN_PER_MINUTE", "60")
	t.Setenv("RATE_LIMIT_SIGNAL_PER_SECOND", "200")
	t.Setenv("RATE_LIMIT_REPORT_PER_10MIN", "5")
	for key, value := range overrides {
		t.Setenv(key, value)
	}
	config.LoadEnv()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	st := store.New(store.Options{
		SessionTTL:           time.Hour,
		QueueTTL:             time.Minute,
		MatchIdleTTL:         time.Minute,
		MaxMatchDuration:     time.Hour,
		SkipLock:             testSkipLock,
		ReportsPerTenMinutes: 5,
	}, logger)

	api := handlers.New(st, logger)
	server := httptest.NewServer(routes.NewRouter(api))
	t.Cleanup(server.Close)

	return &testEnv{server: server, api: api, store: st}
}

// apiResponse mirrors the shared response envelope.
type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Code    string          `json:"code"`
	Data    json.RawMessage `json:"data"`
}

// client is one browser: it has its own cookie jar, so two clients are two
// separate anonymous visitors.
type client struct {
	t    *testing.T
	base string
	http *http.Client

	// lastCookies are the cookies of the most recent response, for the tests
	// that assert on how the session cookie is set or cleared.
	lastCookies []*http.Cookie
}

// rawBody is a request body written literally, so a test can send malformed
// JSON or something oversized. The client sends it without re-encoding.
type rawBody string

func newClient(t *testing.T, env *testEnv) *client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("could not build cookie jar: %v", err)
	}

	return &client{
		t:    t,
		base: env.server.URL,
		http: &http.Client{Jar: jar, Timeout: 10 * time.Second},
	}
}

func (c *client) do(method, path string, body any, headers map[string]string) (int, apiResponse) {
	c.t.Helper()

	var reader io.Reader
	switch typed := body.(type) {
	case nil:
	case rawBody:
		// Sent exactly as written, so malformed and oversized bodies can be
		// exercised on purpose.
		reader = strings.NewReader(string(typed))
	default:
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("could not encode body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("could not build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	c.lastCookies = resp.Cookies()

	raw, _ := io.ReadAll(resp.Body)

	var parsed apiResponse
	_ = json.Unmarshal(raw, &parsed)

	return resp.StatusCode, parsed
}

func (c *client) post(path string, body any) (int, apiResponse) {
	return c.do(http.MethodPost, path, body, nil)
}

func (c *client) get(path string) (int, apiResponse) {
	return c.do(http.MethodGet, path, nil, nil)
}

// createSession starts a visit and returns the payload.
func (c *client) createSession(username string) sessionData {
	c.t.Helper()

	status, body := c.post("/api/session", map[string]string{"username": username})
	if status != http.StatusOK {
		c.t.Fatalf("create session status = %d, body = %+v", status, body)
	}

	data := decode[sessionData](c.t, body.Data)
	if data.SessionID == "" {
		c.t.Fatal("session payload is missing a sessionId")
	}
	return data
}

// joinMatchmaking asks to be matched.
func (c *client) join(username string) (int, apiResponse) {
	return c.post("/api/matchmaking/join", map[string]string{"username": username})
}

// status reads the current state.
func (c *client) status() stateData {
	c.t.Helper()

	code, body := c.get("/api/matchmaking/status")
	if code != http.StatusOK {
		c.t.Fatalf("status code = %d, body = %+v", code, body)
	}
	return decode[stateData](c.t, body.Data)
}

// sessionData is the POST /api/session payload.
type sessionData struct {
	SessionID        string `json:"sessionId"`
	Username         string `json:"username"`
	SkipLockMs       int64  `json:"skipLockMs"`
	ExpiresInSeconds int64  `json:"expiresInSeconds"`
}

// stateData is the matchmaking state payload.
type stateData struct {
	State            string `json:"state"`
	MatchID          string `json:"matchId"`
	PartnerUsername  string `json:"partnerUsername"`
	Initiator        bool   `json:"initiator"`
	SkipLockMs       int64  `json:"skipLockMs"`
	RemainingMs      int64  `json:"remainingMs"`
	QueuePosition    int    `json:"queuePosition"`
	Ready            bool   `json:"ready"`
	PeerReady        bool   `json:"peerReady"`
	LastEndedReason  string `json:"lastEndedReason"`
	LastEndedMatchID string `json:"lastEndedMatchId"`
}

// skipLockedData is the body of a refused skip.
type skipLockedData struct {
	MatchID     string `json:"matchId"`
	RemainingMs int64  `json:"remainingMs"`
	SkipLockMs  int64  `json:"skipLockMs"`
}

func decode[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()

	var value T
	if len(raw) == 0 {
		t.Fatal("expected a data payload but the response had none")
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("could not decode payload %s: %v", raw, err)
	}
	return value
}

// sessionCookie finds the live session cookie, preferring the jar so a request
// that does not itself set the cookie still has it available.
func (c *client) sessionCookie() *http.Cookie {
	c.t.Helper()

	if parsed, err := url.Parse(c.base); err == nil && c.http.Jar != nil {
		for _, cookie := range c.http.Jar.Cookies(parsed) {
			if cookie.Name == "strg_session" {
				return cookie
			}
		}
	}

	return c.lastSessionCookie()
}

// lastSessionCookie looks only at the most recent response, which is where the
// cookie's attributes can be asserted on.
func (c *client) lastSessionCookie() *http.Cookie {
	c.t.Helper()

	for _, cookie := range c.lastCookies {
		if cookie.Name == "strg_session" {
			return cookie
		}
	}
	return nil
}
