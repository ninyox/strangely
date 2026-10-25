package store

import (
	"time"

	"strgback/models"
	"strgback/utils"
)

// CreateSession mints a new anonymous session and returns the raw token that
// belongs in the visitor's HttpOnly cookie. Only the hash is retained.
func (s *Store) CreateSession(username string) (SessionView, string, error) {
	token, err := utils.RandomToken(32)
	if err != nil {
		return SessionView{}, "", err
	}

	hash := utils.HashToken(token)
	now := s.now()

	sess := &models.Session{
		ID:         utils.ShortID(hash),
		TokenHash:  hash,
		Username:   username,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(s.opts.SessionTTL),
	}

	s.mu.Lock()
	s.byHash[hash] = sess
	s.byID[sess.ID] = sess
	s.counters.SessionsCreated++
	s.mu.Unlock()

	return SessionView{ID: sess.ID, Username: sess.Username}, token, nil
}

// Authenticate resolves a raw cookie token to a live session and slides its
// expiry forward. An expired session is removed (and any match it was in is
// ended) before the caller is told it is unknown.
func (s *Store) Authenticate(token string) (SessionView, bool) {
	if token == "" {
		return SessionView{}, false
	}

	hash := utils.HashToken(token)
	now := s.now()

	s.mu.Lock()
	sess := s.byHash[hash]
	if sess == nil {
		s.mu.Unlock()
		return SessionView{}, false
	}

	if sess.Expired(now) {
		events := s.removeSessionLocked(sess, models.ReasonSessionEnded, now)
		s.counters.SessionsExpired++
		s.mu.Unlock()
		s.dispatch(events)
		return SessionView{}, false
	}

	sess.LastSeenAt = now
	sess.ExpiresAt = now.Add(s.opts.SessionTTL)
	view := SessionView{ID: sess.ID, Username: sess.Username}
	s.mu.Unlock()

	return view, true
}

// SetUsername stores the display name a visitor chose for this visit. The
// caller validates it; the store only records it.
func (s *Store) SetUsername(sessionID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess := s.byID[sessionID]
	if sess == nil {
		return ErrUnknownSession
	}
	sess.Username = username
	return nil
}

// MarkConnected records that a signaling socket opened or closed for a
// session. The janitor uses it to notice an abandoned tab.
func (s *Store) MarkConnected(sessionID string, connected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess := s.byID[sessionID]
	if sess == nil {
		return
	}
	sess.Connected = connected
	sess.LastSignalAt = s.now()
}

// TouchSignal records signaling activity, which keeps an otherwise quiet match
// from being treated as abandoned.
func (s *Store) TouchSignal(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess := s.byID[sessionID]; sess != nil {
		sess.LastSignalAt = s.now()
	}
}

// DropSession forgets a session immediately: it leaves the queue, any active
// match ends, and the partner is told why.
func (s *Store) DropSession(sessionID string) {
	now := s.now()

	s.mu.Lock()
	sess := s.byID[sessionID]
	if sess == nil {
		s.mu.Unlock()
		return
	}
	events := s.removeSessionLocked(sess, models.ReasonLeft, now)
	s.mu.Unlock()

	s.dispatch(events)
}

// removeSessionLocked deletes a session and every trace of it from the queue
// and from any match. Callers hold the lock and dispatch the returned events.
func (s *Store) removeSessionLocked(sess *models.Session, reason string, now time.Time) []pendingEvent {
	delete(s.byHash, sess.TokenHash)
	delete(s.byID, sess.ID)

	events := s.removeFromQueueLocked(sess.ID)

	if sess.MatchID != "" {
		if m := s.matches[sess.MatchID]; m != nil && m.IsActive() {
			events = append(events, s.endMatchLocked(m, reason, sess.ID, now)...)
		}
	}

	return events
}
