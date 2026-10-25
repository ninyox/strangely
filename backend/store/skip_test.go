package store

import (
	"testing"
	"time"

	"strgback/models"
)

func TestSkipIsRejectedInsideTheFiveSecondWindow(t *testing.T) {
	st, sink, clock := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	clock.Advance(1500 * time.Millisecond)

	res, err := st.Skip(first, matchID)
	if err != nil {
		t.Fatalf("Skip returned an error: %v", err)
	}
	if !res.TooEarly {
		t.Fatal("a skip inside the five second window must be refused")
	}
	if res.Ended {
		t.Fatal("a refused skip must not end the match")
	}
	if res.RemainingMs < 3400 || res.RemainingMs > 3600 {
		t.Errorf("remaining = %dms, want about 3500ms", res.RemainingMs)
	}

	// The call is untouched and the partner hears nothing.
	view, err := st.Status(first)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if view.MatchID != matchID || view.State == models.StateEnded {
		t.Fatalf("match should still be running: %+v", view)
	}
	if view.RemainingMs < 3400 || view.RemainingMs > 3600 {
		t.Errorf("status remaining = %dms, want about 3500ms", view.RemainingMs)
	}

	for _, ev := range sink.eventsFor(second) {
		if ev.Type == EventPartnerLeft {
			t.Fatal("the partner must not be told about a refused skip")
		}
	}
}

func TestSkipEndsTheMatchAfterTheWindow(t *testing.T) {
	st, sink, clock := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	clock.Advance(5 * time.Second)

	res, err := st.Skip(first, matchID)
	if err != nil {
		t.Fatalf("Skip failed: %v", err)
	}
	if res.TooEarly {
		t.Fatal("a skip after five seconds must be allowed")
	}
	if !res.Ended || res.AlreadyEnded {
		t.Fatalf("skip result = %+v, want a fresh end", res)
	}

	// Both sides are out of the call, and the reason is recorded.
	for _, id := range []string{first, second} {
		view, err := st.Status(id)
		if err != nil {
			t.Fatalf("status failed: %v", err)
		}
		if view.MatchID != "" {
			t.Errorf("session %s is still attached to match %s", id, view.MatchID)
		}
		if view.State != models.StateEnded {
			t.Errorf("session %s state = %q, want ended", id, view.State)
		}
		if view.LastEndedReason != models.ReasonSkipped {
			t.Errorf("session %s reason = %q, want %q", id, view.LastEndedReason, models.ReasonSkipped)
		}
	}

	// The partner is told promptly, with the reason.
	last, ok := sink.lastFor(second)
	if !ok || last.Type != EventPartnerLeft {
		t.Fatalf("partner event = %+v, want partner_left", last)
	}
	payload, ok := last.Data.(PartnerLeftPayload)
	if !ok {
		t.Fatalf("partner_left payload has unexpected type %T", last.Data)
	}
	if payload.Reason != models.ReasonSkipped || payload.MatchID != matchID {
		t.Errorf("partner_left payload = %+v, want a skip of %s", payload, matchID)
	}

	// Both can start again and are matched with each other.
	rejoin, err := st.Join(second)
	if err != nil {
		t.Fatalf("re-join failed: %v", err)
	}
	if rejoin.State != models.StateQueued {
		t.Fatalf("re-join state = %q, want queued", rejoin.State)
	}

	rematch, err := st.Join(first)
	if err != nil || rematch.MatchID == "" {
		t.Fatalf("second match failed: %+v err=%v", rematch, err)
	}
	if rematch.PartnerUsername != "Grace" {
		t.Errorf("re-match partner = %q, want Grace", rematch.PartnerUsername)
	}
}

func TestSkipIsIdempotent(t *testing.T) {
	st, _, clock := newTestStore(t)

	first, second, matchID := matchTwo(t, st)
	clock.Advance(6 * time.Second)

	if _, err := st.Skip(first, matchID); err != nil {
		t.Fatalf("first Skip failed: %v", err)
	}

	res, err := st.Skip(first, matchID)
	if err != nil {
		t.Fatalf("second Skip returned an error: %v", err)
	}
	if !res.AlreadyEnded || !res.Ended {
		t.Fatalf("second Skip = %+v, want an idempotent already-ended result", res)
	}

	// The partner skipping the already finished call is fine too.
	res, err = st.Skip(second, matchID)
	if err != nil {
		t.Fatalf("partner Skip returned an error: %v", err)
	}
	if !res.AlreadyEnded {
		t.Fatalf("partner Skip = %+v, want already ended", res)
	}
}

func TestStopWorksInsideTheFiveSecondWindow(t *testing.T) {
	st, sink, clock := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	clock.Advance(time.Second)

	res, err := st.Stop(first, matchID)
	if err != nil {
		t.Fatalf("Stop inside the Skip window failed: %v", err)
	}
	if !res.Ended || res.TooEarly {
		t.Fatalf("Stop = %+v, want an immediate end", res)
	}

	last, ok := sink.lastFor(second)
	if !ok || last.Type != EventPartnerLeft {
		t.Fatalf("partner event = %+v, want partner_left", last)
	}
	payload := last.Data.(PartnerLeftPayload)
	if payload.Reason != models.ReasonStopped {
		t.Errorf("partner reason = %q, want %q", payload.Reason, models.ReasonStopped)
	}

	view, _ := st.Status(first)
	if view.State != models.StateEnded || view.MatchID != "" {
		t.Errorf("state after stop = %+v, want ended", view)
	}
}

func TestStopIsIdempotentAndCannotBeForcedOnAnotherMatch(t *testing.T) {
	st, _, _ := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	if _, err := st.Stop(first, matchID); err != nil {
		t.Fatalf("first Stop failed: %v", err)
	}

	res, err := st.Stop(first, matchID)
	if err != nil {
		t.Fatalf("second Stop returned an error: %v", err)
	}
	if !res.AlreadyEnded {
		t.Fatalf("second Stop = %+v, want already ended", res)
	}

	// An unrelated session, and an unknown match id, are both refused.
	outsider := newSession(t, st, "Outsider")
	if _, err := st.Stop(outsider, matchID); err != ErrNotInMatch {
		t.Errorf("outsider stop error = %v, want ErrNotInMatch", err)
	}
	if _, err := st.Stop(second, "not-a-match"); err != ErrNotInMatch {
		t.Errorf("unknown match stop error = %v, want ErrNotInMatch", err)
	}
	if _, err := st.Skip(second, "not-a-match"); err != ErrNotInMatch {
		t.Errorf("unknown match skip error = %v, want ErrNotInMatch", err)
	}
}
