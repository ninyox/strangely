package routes_test

import (
	"net/http"
	"testing"
	"time"
)

func TestJoinRequiresAUsername(t *testing.T) {
	env := newTestEnv(t)
	c := newClient(t, env)

	// A session without a name cannot start matching.
	c.post("/api/session", nil)

	code, body := c.post("/api/matchmaking/join", nil)
	if code != http.StatusBadRequest || body.Code != "username_required" {
		t.Fatalf("join without a name = %d %q, want 400 username_required", code, body.Code)
	}

	// Sending the name with the join is enough.
	code, body = c.join("Ada")
	if code != http.StatusOK {
		t.Fatalf("join = %d %+v, want 200", code, body)
	}
	state := decode[stateData](t, body.Data)
	if state.State != "queued" || state.QueuePosition != 1 {
		t.Errorf("join state = %+v, want queued at position 1", state)
	}

	// Joining again while waiting is a conflict the client can recover from.
	code, body = c.join("Ada")
	if code != http.StatusConflict || body.Code != "already_queued" {
		t.Fatalf("second join = %d %q, want 409 already_queued", code, body.Code)
	}

	// Cancelling puts the visitor back to idle.
	code, body = c.post("/api/matchmaking/cancel", nil)
	if code != http.StatusOK {
		t.Fatalf("cancel = %d %+v, want 200", code, body)
	}
	if state := decode[stateData](t, body.Data); state.State != "idle" {
		t.Errorf("cancel state = %q, want idle", state.State)
	}
}

func TestTwoClientsMatchAndTheSkipLockIsEnforced(t *testing.T) {
	env := newTestEnv(t)

	ada := newClient(t, env)
	grace := newClient(t, env)

	ada.createSession("Ada")
	grace.createSession("Grace")

	if code, body := ada.join("Ada"); code != http.StatusOK {
		t.Fatalf("ada join = %d %+v", code, body)
	}

	code, body := grace.join("Grace")
	if code != http.StatusOK {
		t.Fatalf("grace join = %d %+v", code, body)
	}
	graceState := decode[stateData](t, body.Data)
	if graceState.MatchID == "" {
		t.Fatal("the second join should produce a match")
	}
	if graceState.PartnerUsername != "Ada" {
		t.Errorf("partner = %q, want Ada", graceState.PartnerUsername)
	}
	if graceState.Initiator {
		t.Error("the second visitor should not be the initiator")
	}

	// Ada sees the same match, as the initiating side.
	adaState := ada.status()
	if adaState.MatchID != graceState.MatchID {
		t.Fatalf("match ids differ: %q vs %q", adaState.MatchID, graceState.MatchID)
	}
	if !adaState.Initiator || adaState.PartnerUsername != "Grace" {
		t.Errorf("ada state = %+v, want the initiator side matched with Grace", adaState)
	}
	if adaState.RemainingMs <= 0 || adaState.RemainingMs > testSkipLock.Milliseconds() {
		t.Errorf("remaining = %dms, want between 0 and %d", adaState.RemainingMs, testSkipLock.Milliseconds())
	}

	// Skip inside the window is refused, with the server's remaining time.
	code, body = ada.post("/api/matches/"+graceState.MatchID+"/skip", nil)
	if code != http.StatusConflict || body.Code != "skip_locked" {
		t.Fatalf("early skip = %d %q, want 409 skip_locked", code, body.Code)
	}
	locked := decode[skipLockedData](t, body.Data)
	if locked.RemainingMs <= 0 || locked.RemainingMs > testSkipLock.Milliseconds() {
		t.Errorf("remainingMs = %d, want between 0 and %d", locked.RemainingMs, testSkipLock.Milliseconds())
	}
	if locked.SkipLockMs != testSkipLock.Milliseconds() {
		t.Errorf("skipLockMs = %d, want %d", locked.SkipLockMs, testSkipLock.Milliseconds())
	}

	// The refused skip changed nothing.
	if after := ada.status(); after.MatchID != graceState.MatchID {
		t.Fatal("a refused skip must not end the call")
	}
}

func TestSafetyActionsWorkInsideTheSkipWindow(t *testing.T) {
	env := newTestEnv(t)

	ada := newClient(t, env)
	grace := newClient(t, env)

	ada.createSession("Ada")
	grace.createSession("Grace")
	ada.join("Ada")
	_, body := grace.join("Grace")
	matchID := decode[stateData](t, body.Data).MatchID

	// Reporting is available inside the window, even though skipping is not.
	code, body := ada.post("/api/matches/"+matchID+"/report",
		map[string]string{"reason": "harassment_or_hate", "note": "was shouting"})
	if code != http.StatusOK {
		t.Fatalf("report during the lock = %d %+v, want 200", code, body)
	}
	report := decode[struct {
		ReportID string `json:"reportId"`
	}](t, body.Data)
	if report.ReportID == "" {
		t.Error("a report should be acknowledged with an id")
	}

	// Stop is available inside the window too, and ends the call.
	code, body = ada.post("/api/matches/"+matchID+"/stop", nil)
	if code != http.StatusOK {
		t.Fatalf("stop during the lock = %d %+v, want 200", code, body)
	}
	end := decode[struct {
		State  string `json:"state"`
		Reason string `json:"reason"`
	}](t, body.Data)
	if end.State != "ended" || end.Reason != "stopped" {
		t.Errorf("stop payload = %+v, want ended/stopped", end)
	}

	// Grace sees a clean end state, not a frozen video.
	after := grace.status()
	if after.State != "ended" || after.MatchID != "" {
		t.Fatalf("grace state after stop = %+v, want ended with no match", after)
	}
	if after.LastEndedReason != "stopped" {
		t.Errorf("grace reason = %q, want stopped", after.LastEndedReason)
	}

	// Both can start a fresh match afterwards.
	if code, body := grace.join("Grace"); code != http.StatusOK {
		t.Fatalf("grace re-join = %d %+v", code, body)
	}
	if code, body := ada.join("Ada"); code != http.StatusOK {
		t.Fatalf("ada re-join = %d %+v", code, body)
	} else if state := decode[stateData](t, body.Data); state.MatchID == "" {
		t.Fatal("a fresh match should be created")
	}
}

func TestSkipAfterTheWindowEndsTheCallForBoth(t *testing.T) {
	env := newTestEnv(t)

	ada := newClient(t, env)
	grace := newClient(t, env)

	ada.createSession("Ada")
	grace.createSession("Grace")

	ada.join("Ada")
	_, body := grace.join("Grace")
	matchID := decode[stateData](t, body.Data).MatchID

	time.Sleep(testSkipLock + 60*time.Millisecond)

	code, body := ada.post("/api/matches/"+matchID+"/skip", nil)
	if code != http.StatusOK {
		t.Fatalf("skip after the window = %d %+v, want 200", code, body)
	}
	end := decode[struct {
		State        string `json:"state"`
		Reason       string `json:"reason"`
		AlreadyEnded bool   `json:"alreadyEnded"`
	}](t, body.Data)
	if end.State != "ended" || end.Reason != "skipped" || end.AlreadyEnded {
		t.Errorf("skip payload = %+v, want ended/skipped", end)
	}

	after := grace.status()
	if after.State != "ended" || after.LastEndedReason != "skipped" {
		t.Fatalf("grace state = %+v, want ended because the partner skipped", after)
	}
	if after.MatchID != "" {
		t.Error("grace should be detached from the ended match")
	}

	// Skipping again is idempotent.
	code, body = ada.post("/api/matches/"+matchID+"/skip", nil)
	if code != http.StatusOK {
		t.Fatalf("second skip = %d, want 200", code)
	}
	if again := decode[struct {
		AlreadyEnded bool `json:"alreadyEnded"`
	}](t, body.Data); !again.AlreadyEnded {
		t.Error("a second skip should report alreadyEnded")
	}

	// Stop is idempotent as well.
	code, body = grace.post("/api/matches/"+matchID+"/stop", nil)
	if code != http.StatusOK {
		t.Fatalf("stop after end = %d, want 200", code)
	}
	if again := decode[struct {
		AlreadyEnded bool `json:"alreadyEnded"`
	}](t, body.Data); !again.AlreadyEnded {
		t.Error("a stop after the end should report alreadyEnded")
	}
}
