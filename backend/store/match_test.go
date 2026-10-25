package store

import (
	"testing"
	"time"

	"strgback/models"
)

func TestTwoWaitingSessionsAreMatchedExactlyOnce(t *testing.T) {
	st, sink, _ := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	view, err := st.Status(first)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if view.MatchID != matchID {
		t.Fatalf("first session match = %q, want %q", view.MatchID, matchID)
	}
	if view.PartnerUsername != "Grace" {
		t.Errorf("first session partner = %q, want Grace", view.PartnerUsername)
	}
	if !view.Initiator {
		t.Error("the session that was already waiting should be the initiator")
	}

	view, err = st.Status(second)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if view.PartnerUsername != "Ada" {
		t.Errorf("second session partner = %q, want Ada", view.PartnerUsername)
	}
	if view.Initiator {
		t.Error("the session that completed the pair should not be the initiator")
	}

	if stats := st.Stats(); stats.ActiveMatches != 1 || stats.Queued != 0 {
		t.Errorf("stats = %+v, want exactly one active match and an empty queue", stats)
	}

	for _, id := range []string{first, second} {
		last, ok := sink.lastFor(id)
		if !ok || last.Type != EventMatched {
			t.Fatalf("session %s did not receive a matched event: %+v", id, last)
		}
		payload, ok := last.Data.(MatchPayload)
		if !ok {
			t.Fatalf("matched payload has unexpected type %T", last.Data)
		}
		if payload.MatchID != matchID || payload.SkipLockMs != 5000 {
			t.Errorf("matched payload = %+v, want match %s and a 5000ms lock", payload, matchID)
		}
	}
}

func TestSessionCannotQueueTwiceOrMatchItself(t *testing.T) {
	st, _, _ := newTestStore(t)

	solo := newSession(t, st, "Ada")

	if _, err := st.Join(solo); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	res, err := st.Join(solo)
	if err != nil {
		t.Fatalf("second join failed: %v", err)
	}
	if !res.AlreadyQueued {
		t.Fatal("a session already waiting must be told it is already queued")
	}
	if res.MatchID != "" {
		t.Fatal("a session must never be matched with itself")
	}

	if stats := st.Stats(); stats.ActiveMatches != 0 || stats.Queued != 1 {
		t.Errorf("stats = %+v, want no matches and one queued session", stats)
	}
}

func TestSessionCannotJoinWhileInAMatch(t *testing.T) {
	st, _, _ := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	res, err := st.Join(first)
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if !res.InMatch || res.MatchID != matchID {
		t.Fatalf("join while in a match = %+v, want the existing match %s", res, matchID)
	}
	if res.AlreadyQueued {
		t.Error("an in-call session must not be queued")
	}

	if stats := st.Stats(); stats.Queued != 0 || stats.ActiveMatches != 1 {
		t.Errorf("stats = %+v, want one match and no queue", stats)
	}

	// The same holds for the partner.
	if res, err := st.Join(second); err != nil || !res.InMatch {
		t.Fatalf("partner join = %+v, err = %v; want in-match", res, err)
	}
}

func TestCancelRemovesWaitingSession(t *testing.T) {
	st, _, _ := newTestStore(t)

	solo := newSession(t, st, "Ada")
	if _, err := st.Join(solo); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	res, err := st.Cancel(solo)
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if res.State != models.StateIdle {
		t.Errorf("cancel state = %q, want idle", res.State)
	}

	view, err := st.Status(solo)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if view.State != models.StateIdle {
		t.Errorf("status after cancel = %q, want idle", view.State)
	}

	// Cancelling again is harmless.
	if _, err := st.Cancel(solo); err != nil {
		t.Fatalf("second cancel failed: %v", err)
	}
}

func TestQueueMatchesTheEarliestEligibleWaiter(t *testing.T) {
	st, _, clock := newTestStore(t)

	// A session that queues and then expires leaves a stale entry behind.
	ghost := newSession(t, st, "Ghost")
	if _, err := st.Join(ghost); err != nil {
		t.Fatalf("ghost join failed: %v", err)
	}
	clock.Advance(3 * time.Hour)

	waiting := newSession(t, st, "Waiting")
	if _, err := st.Join(waiting); err != nil {
		t.Fatalf("waiting join failed: %v", err)
	}

	// The arrival must be paired with the earliest session that is still
	// eligible, never with the stale entry ahead of it.
	arrival := newSession(t, st, "Arrival")
	res, err := st.Join(arrival)
	if err != nil {
		t.Fatalf("arrival join failed: %v", err)
	}
	if res.PartnerUsername != "Waiting" {
		t.Fatalf("arrival matched with %q, want the earliest eligible waiter Waiting", res.PartnerUsername)
	}

	view, err := st.Status(waiting)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if view.State != models.StateConnecting || view.MatchID != res.MatchID {
		t.Errorf("waiting session = %+v, want the new match %s", view, res.MatchID)
	}

	if stats := st.Stats(); stats.Queued != 0 || stats.ActiveMatches != 1 {
		t.Errorf("stats = %+v, want one match and an empty queue", stats)
	}
}

func TestStatusReportsConnectingUntilBothSidesAreReady(t *testing.T) {
	st, sink, _ := newTestStore(t)

	first, second, matchID := matchTwo(t, st)

	view, _ := st.Status(first)
	if view.State != models.StateConnecting {
		t.Fatalf("state before readiness = %q, want connecting", view.State)
	}

	both, err := st.MarkReady(first, matchID)
	if err != nil {
		t.Fatalf("MarkReady failed: %v", err)
	}
	if both {
		t.Error("one side being ready must not mean both are")
	}

	both, err = st.MarkReady(second, matchID)
	if err != nil {
		t.Fatalf("MarkReady failed: %v", err)
	}
	if !both {
		t.Error("both sides ready should report both ready")
	}

	view, _ = st.Status(first)
	if view.State != models.StateMatched || !view.PeerReady {
		t.Errorf("state after both ready = %+v, want matched with a ready peer", view)
	}

	// The session that marked ready second notifies the one that was already
	// ready, so neither side is left waiting on a "connecting" screen.
	found := false
	for _, ev := range sink.eventsFor(first) {
		if ev.Type == EventPartnerReady {
			found = true
		}
	}
	if !found {
		t.Error("the partner should be notified when a peer becomes ready")
	}
}

func TestQueueDropsSessionsThatExpiredWhileWaiting(t *testing.T) {
	st, _, clock := newTestStore(t)

	ghost := newSession(t, st, "Ghost")
	if _, err := st.Join(ghost); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	// The waiting session goes away (TTL) before anybody else arrives.
	clock.Advance(3 * time.Hour)

	stats := st.Cleanup()
	if stats.QueueDropped != 1 {
		t.Errorf("cleanup dropped %d queue entries, want 1", stats.QueueDropped)
	}

	arrival := newSession(t, st, "Arrival")
	res, err := st.Join(arrival)
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if res.MatchID != "" || res.State != models.StateQueued {
		t.Fatalf("arrival was matched with a stale entry: %+v", res)
	}
}

func TestQueueDropsEntriesThatWaitedTooLong(t *testing.T) {
	st, _, clock := newTestStore(t)

	waiting := newSession(t, st, "Waiting")
	if _, err := st.Join(waiting); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	// Past QUEUE_TTL, but the session itself is still alive.
	clock.Advance(3 * time.Minute)

	arrival := newSession(t, st, "Arrival")
	res, err := st.Join(arrival)
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if res.MatchID != "" {
		t.Fatal("a queue entry past its TTL must not be matched")
	}
	if res.State != models.StateQueued {
		t.Fatalf("arrival state = %q, want queued", res.State)
	}

	// The stale entry was dropped lazily, so the still-live session can ask
	// again and is matched with the visitor who is now waiting.
	again, err := st.Join(waiting)
	if err != nil {
		t.Fatalf("re-join failed: %v", err)
	}
	if again.MatchID == "" || again.PartnerUsername != "Arrival" {
		t.Fatalf("re-join = %+v, want a match with Arrival", again)
	}
}
