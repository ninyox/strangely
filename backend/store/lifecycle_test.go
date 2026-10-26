package store

import (
	"testing"
	"time"

	"strgback/models"
)

func TestPartnerDisconnectEndsTheMatchOnCleanup(t *testing.T) {
	st, sink, clock := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	// Both sides have a signaling socket to start with.
	st.MarkConnected(first, true)
	st.MarkConnected(second, true)

	// The first tab goes away without saying goodbye.
	st.MarkConnected(first, false)
	clock.Advance(21 * time.Second)

	stats := st.Cleanup()
	if stats.MatchesEnded != 1 {
		t.Fatalf("cleanup ended %d matches, want 1", stats.MatchesEnded)
	}

	last, ok := sink.lastFor(second)
	if !ok || last.Type != EventPartnerLeft {
		t.Fatalf("partner event = %+v, want partner_left", last)
	}
	if payload := last.Data.(PartnerLeftPayload); payload.Reason != models.ReasonDisconnected {
		t.Errorf("reason = %q, want %q", payload.Reason, models.ReasonDisconnected)
	}

	view, _ := st.Status(second)
	if view.MatchID != "" {
		t.Error("the remaining session should be free to match again")
	}
	if view.State != models.StateEnded {
		t.Errorf("state = %q, want ended", view.State)
	}

	_ = matchID
}

func TestQueueDropsWaitingSessionsThatLostTheirSocket(t *testing.T) {
	st, _, clock := newTestStore(t)

	// A session queues with a live signaling socket, then that socket goes
	// away — the tab was closed, or the phone lost its network.
	waiting := newSession(t, st, "Waiting")
	st.MarkConnected(waiting, true)
	if _, err := st.Join(waiting); err != nil {
		t.Fatalf("join failed: %v", err)
	}
	st.MarkConnected(waiting, false)

	clock.Advance(21 * time.Second)

	stats := st.Cleanup()
	if stats.QueueDropped != 1 {
		t.Fatalf("cleanup dropped %d queue entries, want 1", stats.QueueDropped)
	}

	// A session with no socket cannot answer an offer, so the next arrival
	// must not be paired with it: that arrival would sit in front of a call
	// that can never be answered, and be told the "partner" disconnected.
	arrival := newSession(t, st, "Arrival")
	res, err := st.Join(arrival)
	if err != nil {
		t.Fatalf("arrival join failed: %v", err)
	}
	if res.MatchID != "" || res.State != models.StateQueued {
		t.Fatalf("arrival = %+v, want to be queued rather than matched with an unreachable session", res)
	}

	view, err := st.Status(waiting)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if view.State != models.StateIdle || view.MatchID != "" {
		t.Errorf("the waiting session = %+v, want it back to idle", view)
	}
}

func TestMaxMatchDurationEndsTheCall(t *testing.T) {
	st, sink, clock := newTestStore(t)

	_, second, _ := matchTwo(t, st)
	st.MarkConnected(second, true)

	clock.Advance(61 * time.Minute)

	stats := st.Cleanup()
	if stats.MatchesEnded != 1 {
		t.Fatalf("cleanup ended %d matches, want 1", stats.MatchesEnded)
	}

	last, _ := sink.lastFor(second)
	if payload := last.Data.(PartnerLeftPayload); payload.Reason != models.ReasonTooLong {
		t.Errorf("reason = %q, want %q", payload.Reason, models.ReasonTooLong)
	}
}

func TestExpiredSessionEndsTheMatchAndNotifiesThePartner(t *testing.T) {
	st, sink, clock := newTestStore(t)

	// Two sessions, matched. Ada is left to expire; Grace is kept alive by
	// authenticating her an hour later, which slides her expiry forward.
	ada, adaToken, err := st.CreateSession("Ada")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	grace, graceToken, err := st.CreateSession("Grace")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if _, err := st.Join(ada.ID); err != nil {
		t.Fatalf("Ada join failed: %v", err)
	}
	res, err := st.Join(grace.ID)
	if err != nil {
		t.Fatalf("Grace join failed: %v", err)
	}
	if res.MatchID == "" {
		t.Fatal("the two sessions should have matched")
	}

	clock.Advance(time.Hour)
	if _, ok := st.Authenticate(graceToken); !ok {
		t.Fatal("Grace should still authenticate inside her TTL")
	}

	// Ada expired at t0+2h; Grace is now good until t0+3h.
	clock.Advance(90 * time.Minute)

	stats := st.Cleanup()
	if stats.SessionsExpired != 1 {
		t.Errorf("expired %d sessions, want only Ada", stats.SessionsExpired)
	}
	if stats.MatchesEnded != 1 {
		t.Errorf("cleanup ended %d matches, want 1", stats.MatchesEnded)
	}
	if st.SessionExists(ada.ID) {
		t.Error("the expired session should be gone")
	}
	if !st.SessionExists(grace.ID) {
		t.Error("the live session must survive the sweep")
	}

	// The session that is still here is told why the call ended, and is free
	// to match again.
	last, ok := sink.lastFor(grace.ID)
	if !ok || last.Type != EventPartnerLeft {
		t.Fatalf("Grace's last event = %+v, want partner_left", last)
	}
	if payload, ok := last.Data.(PartnerLeftPayload); !ok || payload.Reason != models.ReasonSessionEnded {
		t.Errorf("partner_left payload = %+v, want reason %q", last.Data, models.ReasonSessionEnded)
	}

	view, err := st.Status(grace.ID)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if view.MatchID != "" || view.State != models.StateEnded {
		t.Errorf("Grace's state = %+v, want ended with no match", view)
	}

	// A session that is gone cannot be reported on, and the token is dead.
	if _, ok := st.Authenticate(adaToken); ok {
		t.Error("an expired session must not authenticate")
	}
	if len(st.Reports()) != 0 {
		t.Error("no reports were filed, so none should be stored")
	}
}

func TestDropSessionEndsTheMatchAndNotifiesThePartner(t *testing.T) {
	st, sink, _ := newTestStore(t)

	first, second, _ := matchTwo(t, st)

	st.DropSession(first)

	if st.SessionExists(first) {
		t.Error("a dropped session should be gone")
	}

	last, ok := sink.lastFor(second)
	if !ok || last.Type != EventPartnerLeft {
		t.Fatalf("partner event = %+v, want partner_left", last)
	}
	if payload := last.Data.(PartnerLeftPayload); payload.Reason != models.ReasonLeft {
		t.Errorf("reason = %q, want %q", payload.Reason, models.ReasonLeft)
	}

	view, _ := st.Status(second)
	if view.MatchID != "" {
		t.Error("the partner should be out of the ended match")
	}
}

func TestRelayTargetOnlyAllowsTheTwoParticipants(t *testing.T) {
	st, _, clock := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	target, err := st.RelayTarget(first, matchID)
	if err != nil {
		t.Fatalf("RelayTarget failed: %v", err)
	}
	if target != second {
		t.Errorf("relay target = %q, want the partner %q", target, second)
	}

	// A third session may not reach into somebody else's call.
	outsider := newSession(t, st, "Outsider")
	if _, err := st.RelayTarget(outsider, matchID); err != ErrNotInMatch {
		t.Errorf("outsider relay error = %v, want ErrNotInMatch", err)
	}

	// Neither may a participant use a match id it is not part of.
	if _, err := st.RelayTarget(first, "some-other-match"); err != ErrNotInMatch {
		t.Errorf("foreign match relay error = %v, want ErrNotInMatch", err)
	}

	// Once the call is over, signaling stops.
	clock.Advance(6 * time.Second)
	if _, err := st.Stop(first, matchID); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if _, err := st.RelayTarget(first, matchID); err != ErrNotInMatch {
		t.Errorf("relay after end error = %v, want ErrNotInMatch", err)
	}
}

func TestReportsAreRateLimitedAndScopedToTheMatch(t *testing.T) {
	st, _, _ := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	report, err := st.SaveReport(first, matchID, "harassment_or_hate", "was rude")
	if err != nil {
		t.Fatalf("SaveReport failed: %v", err)
	}
	if report.Note != "was rude" {
		t.Errorf("note = %q, want the text that was filed", report.Note)
	}
	if report.ReporterID != first {
		t.Error("a report should record which session filed it")
	}

	// The budget is five reports per ten minutes.
	for i := 1; i < 5; i++ {
		if _, err := st.SaveReport(first, matchID, "other", ""); err != nil {
			t.Fatalf("report %d failed: %v", i+1, err)
		}
	}
	if _, err := st.SaveReport(first, matchID, "other", ""); err != ErrReportRateLimited {
		t.Fatalf("sixth report error = %v, want ErrReportRateLimited", err)
	}

	// The partner has their own budget.
	if _, err := st.SaveReport(second, matchID, "spam_or_scam", ""); err != nil {
		t.Fatalf("partner report failed: %v", err)
	}

	// Nobody may report a match they were never in.
	outsider := newSession(t, st, "Outsider")
	if _, err := st.SaveReport(outsider, matchID, "other", ""); err != ErrNotInMatch {
		t.Errorf("outsider report error = %v, want ErrNotInMatch", err)
	}
	if _, err := st.SaveReport(first, "unknown-match", "other", ""); err != ErrNotInMatch {
		t.Errorf("unknown match report error = %v, want ErrNotInMatch", err)
	}

	if stats := st.Stats(); stats.Reports != 6 {
		t.Errorf("stored %d reports, want 6", stats.Reports)
	}
}
