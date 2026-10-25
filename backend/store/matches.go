package store

import (
	"time"

	"strgback/models"
)

// EndResult is the outcome of a skip or stop request.
type EndResult struct {
	MatchID      string
	Ended        bool
	AlreadyEnded bool
	TooEarly     bool
	RemainingMs  int64
	PartnerID    string
}

// PartnerLeftPayload tells the remaining participant why the call ended.
type PartnerLeftPayload struct {
	MatchID    string    `json:"matchId"`
	Reason     string    `json:"reason"`
	ServerTime time.Time `json:"serverTime"`
}

// Skip ends the match on request, but only once the server side window has
// elapsed. A request inside the window is refused with the remaining time
// instead of ending the call, which is what makes the five second rule real
// rather than a disabled button.
func (s *Store) Skip(sessionID, matchID string) (EndResult, error) {
	now := s.now()

	s.mu.Lock()
	if s.byID[sessionID] == nil {
		s.mu.Unlock()
		return EndResult{}, ErrUnknownSession
	}

	m := s.matches[matchID]
	if m == nil || !m.Has(sessionID) {
		s.mu.Unlock()
		return EndResult{}, ErrNotInMatch
	}

	if !m.IsActive() {
		partner := m.PartnerOf(sessionID)
		s.mu.Unlock()
		return EndResult{MatchID: matchID, Ended: true, AlreadyEnded: true, PartnerID: partner}, nil
	}

	if remaining := remainingSkipMs(m, now, s.opts.SkipLock); remaining > 0 {
		partner := m.PartnerOf(sessionID)
		s.mu.Unlock()
		return EndResult{MatchID: matchID, TooEarly: true, RemainingMs: remaining, PartnerID: partner}, nil
	}

	events := s.endMatchLocked(m, models.ReasonSkipped, sessionID, now)
	partner := m.PartnerOf(sessionID)
	s.mu.Unlock()

	s.dispatch(events)
	return EndResult{MatchID: matchID, Ended: true, PartnerID: partner}, nil
}

// Stop ends the match immediately. It is never subject to the Skip window:
// a visitor must always be able to leave a call.
func (s *Store) Stop(sessionID, matchID string) (EndResult, error) {
	now := s.now()

	s.mu.Lock()
	if s.byID[sessionID] == nil {
		s.mu.Unlock()
		return EndResult{}, ErrUnknownSession
	}

	m := s.matches[matchID]
	if m == nil || !m.Has(sessionID) {
		s.mu.Unlock()
		return EndResult{}, ErrNotInMatch
	}

	if !m.IsActive() {
		partner := m.PartnerOf(sessionID)
		s.mu.Unlock()
		return EndResult{MatchID: matchID, Ended: true, AlreadyEnded: true, PartnerID: partner}, nil
	}

	events := s.endMatchLocked(m, models.ReasonStopped, sessionID, now)
	partner := m.PartnerOf(sessionID)
	s.mu.Unlock()

	s.dispatch(events)
	return EndResult{MatchID: matchID, Ended: true, PartnerID: partner}, nil
}

// MarkReady records that a client's peer connection is up, and tells the
// partner so the "connecting" state can clear on both sides.
func (s *Store) MarkReady(sessionID, matchID string) (bool, error) {
	now := s.now()

	s.mu.Lock()
	if s.byID[sessionID] == nil {
		s.mu.Unlock()
		return false, ErrUnknownSession
	}

	m := s.matches[matchID]
	if m == nil || !m.Has(sessionID) {
		s.mu.Unlock()
		return false, ErrNotInMatch
	}
	if !m.IsActive() {
		s.mu.Unlock()
		return false, ErrNotInMatch
	}

	m.SetReady(sessionID, true)
	if sess := s.byID[sessionID]; sess != nil {
		sess.Ready = true
	}

	bothReady := m.BothReady()
	partner := m.PartnerOf(sessionID)

	var events []pendingEvent
	if partner != "" && m.Ready(partner) {
		events = append(events, pendingEvent{
			sessionID: partner,
			ev: Event{
				Type: EventPartnerReady,
				Data: map[string]any{"matchId": m.ID, "serverTime": now},
			},
		})
	}
	s.mu.Unlock()

	s.dispatch(events)
	return bothReady, nil
}

// RelayTarget validates that a signaling message may be forwarded: the sender
// must be a participant of an active match, and the match must be the one the
// message names. It is the only way one browser can reach another, so this is
// where cross-session signaling is prevented.
func (s *Store) RelayTarget(senderID, matchID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.byID[senderID] == nil {
		return "", ErrUnknownSession
	}

	m := s.matches[matchID]
	if m == nil || !m.IsActive() || !m.Has(senderID) {
		return "", ErrNotInMatch
	}

	return m.PartnerOf(senderID), nil
}

// endMatchLocked finishes a match, clears both sessions, and builds the
// "your partner left" pushes. When bySessionID is empty nobody is singled out
// as the leaver and both participants are notified.
func (s *Store) endMatchLocked(m *models.Match, reason, bySessionID string, now time.Time) []pendingEvent {
	m.EndedAt = now
	m.EndedReason = reason
	m.EndedBy = bySessionID
	s.endedAt[m.ID] = now
	s.counters.MatchesEnded++

	events := make([]pendingEvent, 0, 2)
	for _, sid := range []string{m.A, m.B} {
		sess := s.byID[sid]
		if sess == nil {
			continue
		}
		if sess.MatchID == m.ID {
			sess.MatchID = ""
		}
		sess.Ready = false
		sess.LastEndedAt = now
		sess.LastEndedReason = reason
		sess.LastEndedMatchID = m.ID

		if bySessionID != "" && sid == bySessionID {
			continue
		}
		events = append(events, pendingEvent{
			sessionID: sid,
			ev: Event{
				Type: EventPartnerLeft,
				Data: PartnerLeftPayload{MatchID: m.ID, Reason: reason, ServerTime: now},
			},
		})
	}

	return events
}

// Leave ends a match with an explicit reason. Skip and Stop are the two gated
// special cases; the signaling hub uses this for an explicit goodbye.
func (s *Store) Leave(sessionID, matchID, reason string) (EndResult, error) {
	now := s.now()

	s.mu.Lock()
	if s.byID[sessionID] == nil {
		s.mu.Unlock()
		return EndResult{}, ErrUnknownSession
	}

	m := s.matches[matchID]
	if m == nil || !m.Has(sessionID) {
		s.mu.Unlock()
		return EndResult{}, ErrNotInMatch
	}

	if !m.IsActive() {
		partner := m.PartnerOf(sessionID)
		s.mu.Unlock()
		return EndResult{MatchID: matchID, Ended: true, AlreadyEnded: true, PartnerID: partner}, nil
	}

	events := s.endMatchLocked(m, reason, sessionID, now)
	partner := m.PartnerOf(sessionID)
	s.mu.Unlock()

	s.dispatch(events)
	return EndResult{MatchID: matchID, Ended: true, PartnerID: partner}, nil
}

// NotifyPeerUnreachable tells the remaining participant that the other socket
// dropped, so the UI can say "reconnecting" during the grace window instead of
// showing a frozen video.
func (s *Store) NotifyPeerUnreachable(sessionID, matchID string) {
	s.mu.Lock()
	m := s.matches[matchID]
	if m == nil || !m.IsActive() || !m.Has(sessionID) {
		s.mu.Unlock()
		return
	}
	partner := m.PartnerOf(sessionID)
	grace := s.opts.MatchIdleTTL
	s.mu.Unlock()

	s.dispatch([]pendingEvent{{
		sessionID: partner,
		ev: Event{
			Type: EventPeerUnreachable,
			Data: map[string]any{
				"matchId": matchID,
				"graceMs": grace.Milliseconds(),
			},
		},
	}})
}
