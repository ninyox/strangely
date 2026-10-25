package routes_test

import (
	"net/http"
	"testing"
)

func TestCallsCannotBeTouchedByAnotherSession(t *testing.T) {
	env := newTestEnv(t)

	ada := newClient(t, env)
	grace := newClient(t, env)
	outsider := newClient(t, env)

	ada.createSession("Ada")
	grace.createSession("Grace")
	outsider.createSession("Outsider")

	ada.join("Ada")
	_, body := grace.join("Grace")
	matchID := decode[stateData](t, body.Data).MatchID

	for _, action := range []string{"skip", "stop", "report"} {
		code, body := outsider.post("/api/matches/"+matchID+"/"+action, map[string]string{"reason": "other"})
		if code != http.StatusNotFound {
			t.Errorf("outsider %s = %d %+v, want 404", action, code, body)
		}
		if body.Code != "match_not_found" {
			t.Errorf("outsider %s code = %q, want match_not_found", action, body.Code)
		}
	}

	// A session in a call cannot cancel the queue instead.
	code, body := ada.post("/api/matchmaking/cancel", nil)
	if code != http.StatusConflict || body.Code != "already_in_match" {
		t.Errorf("cancel while in a call = %d %q, want 409 already_in_match", code, body.Code)
	}

	// Joining while in a call reports the existing match instead of opening a
	// second one.
	code, body = ada.join("Ada")
	if code != http.StatusConflict || body.Code != "already_in_match" {
		t.Fatalf("join while in a call = %d %q, want 409 already_in_match", code, body.Code)
	}
	if state := decode[stateData](t, body.Data); state.MatchID != matchID {
		t.Errorf("join payload match = %q, want the existing %q", state.MatchID, matchID)
	}

	// The call is still running: nothing the outsider sent changed it.
	if state := ada.status(); state.MatchID != matchID {
		t.Error("the call should be untouched by the outsider's requests")
	}
}

func TestReportValidation(t *testing.T) {
	env := newTestEnv(t)

	ada := newClient(t, env)
	grace := newClient(t, env)

	ada.createSession("Ada")
	grace.createSession("Grace")
	ada.join("Ada")
	_, body := grace.join("Grace")
	matchID := decode[stateData](t, body.Data).MatchID

	code, body := ada.post("/api/matches/"+matchID+"/report", map[string]string{"reason": "made-up-reason"})
	if code != http.StatusBadRequest || body.Code != "invalid_reason" {
		t.Fatalf("bad reason = %d %q, want 400 invalid_reason", code, body.Code)
	}
	allowed := decode[struct {
		Allowed []string `json:"allowed"`
	}](t, body.Data)
	if len(allowed.Allowed) == 0 {
		t.Error("the rejection should list the accepted reasons")
	}

	// An unknown match cannot be reported.
	code, _ = ada.post("/api/matches/does-not-exist/report", map[string]string{"reason": "other"})
	if code != http.StatusNotFound {
		t.Errorf("unknown match report = %d, want 404", code)
	}
}

func TestReportRateLimit(t *testing.T) {
	env := newTestEnv(t)

	ada := newClient(t, env)
	grace := newClient(t, env)

	ada.createSession("Ada")
	grace.createSession("Grace")
	ada.join("Ada")
	_, body := grace.join("Grace")
	matchID := decode[stateData](t, body.Data).MatchID

	// The store allows five reports per ten minutes per session.
	for i := 0; i < 5; i++ {
		if code, body := ada.post("/api/matches/"+matchID+"/report", map[string]string{"reason": "other"}); code != http.StatusOK {
			t.Fatalf("report %d = %d %+v, want 200", i+1, code, body)
		}
	}

	code, body := ada.post("/api/matches/"+matchID+"/report", map[string]string{"reason": "other"})
	if code != http.StatusTooManyRequests || body.Code != "report_rate_limited" {
		t.Fatalf("sixth report = %d %q, want 429 report_rate_limited", code, body.Code)
	}
}

func TestSessionCreationIsRateLimited(t *testing.T) {
	env := newTestEnvWith(t, map[string]string{"RATE_LIMIT_SESSION_PER_MINUTE": "2"})
	c := newClient(t, env)

	if code, _ := c.post("/api/session", nil); code != http.StatusOK {
		t.Fatalf("first session = %d, want 200", code)
	}
	if code, _ := c.post("/api/session", nil); code != http.StatusOK {
		t.Fatalf("second session = %d, want 200", code)
	}

	code, body := c.post("/api/session", nil)
	if code != http.StatusTooManyRequests || body.Code != "rate_limited" {
		t.Fatalf("third session = %d %q, want 429 rate_limited", code, body.Code)
	}
}
