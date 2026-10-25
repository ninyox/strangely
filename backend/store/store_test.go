package store

import (
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"strgback/models"
	"strgback/utils"
)

// fakeClock lets the tests move time without sleeping.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// recordingSink captures the events the store would push over a socket.
type recordingSink struct {
	mu     sync.Mutex
	events map[string][]Event
}

func (r *recordingSink) Send(sessionID string, ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[sessionID] = append(r.events[sessionID], ev)
}

func (r *recordingSink) eventsFor(sessionID string) []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Event(nil), r.events[sessionID]...)
}

func (r *recordingSink) lastFor(sessionID string) (Event, bool) {
	events := r.eventsFor(sessionID)
	if len(events) == 0 {
		return Event{}, false
	}
	return events[len(events)-1], true
}

func newTestStore(t *testing.T) (*Store, *recordingSink, *fakeClock) {
	t.Helper()

	clock := &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	sink := &recordingSink{events: make(map[string][]Event)}

	st := New(Options{
		SessionTTL:           2 * time.Hour,
		QueueTTL:             2 * time.Minute,
		MatchIdleTTL:         20 * time.Second,
		MaxMatchDuration:     60 * time.Minute,
		SkipLock:             5 * time.Second,
		ReportsPerTenMinutes: 5,
		Now:                  clock.Now,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	st.SetSink(sink)
	t.Cleanup(st.StopJanitor)

	return st, sink, clock
}

// newSession creates a session with a display name and returns its id.
func newSession(t *testing.T, st *Store, name string) string {
	t.Helper()

	view, _, err := st.CreateSession(name)
	if err != nil {
		t.Fatalf("CreateSession(%q) failed: %v", name, err)
	}
	return view.ID
}

// matchTwo queues two sessions and returns their ids and the match id.
func matchTwo(t *testing.T, st *Store) (string, string, string) {
	t.Helper()

	first := newSession(t, st, "Ada")
	second := newSession(t, st, "Grace")

	res, err := st.Join(first)
	if err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	if res.State != models.StateQueued {
		t.Fatalf("first join state = %q, want %q", res.State, models.StateQueued)
	}

	res, err = st.Join(second)
	if err != nil {
		t.Fatalf("second join failed: %v", err)
	}
	if res.MatchID == "" {
		t.Fatal("second join did not produce a match")
	}

	return first, second, res.MatchID
}

func TestSessionTokensAreOpaqueAndStoredHashed(t *testing.T) {
	st, _, _ := newTestStore(t)

	view, token, err := st.CreateSession("Ada")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if len(token) != 64 {
		t.Errorf("token length = %d, want 64 hex characters", len(token))
	}
	if view.ID == token {
		t.Error("session id must not be the token itself")
	}
	if view.ID != utils.ShortID(utils.HashToken(token)) {
		t.Error("session id should be derived from the token hash")
	}

	// The raw token must not be recoverable from the store.
	st.mu.Lock()
	for hash, sess := range st.byHash {
		if hash == token || sess.TokenHash == token {
			st.mu.Unlock()
			t.Fatal("store kept the raw session token")
		}
	}
	st.mu.Unlock()

	second, secondToken, err := st.CreateSession("Grace")
	if err != nil {
		t.Fatalf("second CreateSession failed: %v", err)
	}
	if secondToken == token || second.ID == view.ID {
		t.Error("two sessions must not share an id or token")
	}
}

func TestAuthenticateSlidesExpiryAndRejectsExpired(t *testing.T) {
	st, _, clock := newTestStore(t)

	view, token, err := st.CreateSession("Ada")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	clock.Advance(time.Hour)
	got, ok := st.Authenticate(token)
	if !ok || got.ID != view.ID {
		t.Fatal("a session inside its TTL must authenticate")
	}

	// Authenticating pushed the expiry out by another full TTL.
	clock.Advance(90 * time.Minute)
	if _, ok := st.Authenticate(token); !ok {
		t.Fatal("expiry should slide forward on activity")
	}

	// Idle past the TTL: gone.
	clock.Advance(3 * time.Hour)
	if _, ok := st.Authenticate(token); ok {
		t.Fatal("a session past its TTL must not authenticate")
	}
	if st.SessionExists(view.ID) {
		t.Error("an expired session should be removed")
	}
}
