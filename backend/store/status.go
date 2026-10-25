package store

import (
	"strgback/models"
)

// Status reports the current state of a session. The state strings are the
// contract the frontend renders from.
func (s *Store) Status(sessionID string) (StatusView, error) {
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()

	sess := s.byID[sessionID]
	if sess == nil {
		return StatusView{}, ErrUnknownSession
	}

	view := StatusView{
		State:            models.StateIdle,
		SkipLockMs:       s.opts.SkipLock.Milliseconds(),
		LastEndedReason:  sess.LastEndedReason,
		LastEndedMatchID: sess.LastEndedMatchID,
	}

	if m := s.activeMatchLocked(sess); m != nil {
		view.MatchID = m.ID
		view.MatchedAt = m.MatchedAt
		view.Initiator = m.Initiator() == sessionID
		view.Ready = m.Ready(sessionID)
		view.PeerReady = m.Ready(m.PartnerOf(sessionID))
		if partner := s.byID[m.PartnerOf(sessionID)]; partner != nil {
			view.PartnerUsername = partner.Username
		}
		view.RemainingMs = remainingSkipMs(m, now, s.opts.SkipLock)

		// "connecting" until both peer connections report up, then "matched".
		if m.BothReady() {
			view.State = models.StateMatched
		} else {
			view.State = models.StateConnecting
		}
		return view, nil
	}

	if queuedAt, waiting := s.queuedAt[sessionID]; waiting {
		view.State = models.StateQueued
		view.QueuedAt = queuedAt
		view.QueuePosition = s.queuePositionLocked(sessionID)
		return view, nil
	}

	if !sess.LastEndedAt.IsZero() {
		view.State = models.StateEnded
	}

	return view, nil
}

// SessionExists reports whether a session is still known to the store.
func (s *Store) SessionExists(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.byID[sessionID]
	return ok
}

// Username returns the display name of a session, or "" when unknown.
func (s *Store) Username(sessionID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess := s.byID[sessionID]; sess != nil {
		return sess.Username
	}
	return ""
}
