package store

import (
	"time"

	"strgback/models"
	"strgback/utils"
)

// Join enqueues a session, or pairs it immediately when somebody is already
// waiting. All of it happens under one lock, so two simultaneous joins can
// never both claim the same waiter.
func (s *Store) Join(sessionID string) (JoinResult, error) {
	now := s.now()

	s.mu.Lock()
	sess := s.byID[sessionID]
	if sess == nil {
		s.mu.Unlock()
		return JoinResult{}, ErrUnknownSession
	}

	if m := s.activeMatchLocked(sess); m != nil {
		res := s.joinResultForMatchLocked(sess, m)
		res.InMatch = true
		s.mu.Unlock()
		return res, nil
	}

	if _, waiting := s.queuedAt[sessionID]; waiting {
		position := s.queuePositionLocked(sessionID)
		s.mu.Unlock()
		return JoinResult{State: models.StateQueued, AlreadyQueued: true, QueuePosition: position}, nil
	}

	// Asking for a match clears the "ended" marker from the previous call.
	sess.LastEndedAt = time.Time{}
	sess.LastEndedReason = ""
	sess.LastEndedMatchID = ""

	partnerID := s.popEligibleWaiterLocked(now)
	if partnerID == "" {
		s.queue = append(s.queue, sessionID)
		s.queuedAt[sessionID] = now
		s.mu.Unlock()
		return JoinResult{State: models.StateQueued, QueuePosition: len(s.queue)}, nil
	}

	partner := s.byID[partnerID]
	m := s.createMatchLocked(partnerID, sessionID, now)

	res := JoinResult{
		State:           models.StateConnecting,
		MatchID:         m.ID,
		PartnerUsername: partner.Username,
		MatchedAt:       m.MatchedAt,
		Initiator:       m.Initiator() == sessionID,
		SkipLockMs:      s.opts.SkipLock.Milliseconds(),
	}
	events := s.matchedEventsLocked(m)
	s.mu.Unlock()

	s.dispatch(events)
	return res, nil
}

// Cancel takes a session out of the queue. Cancelling while in a call is not a
// queue operation: the caller is told so it can use stop or skip instead.
func (s *Store) Cancel(sessionID string) (JoinResult, error) {
	s.mu.Lock()
	sess := s.byID[sessionID]
	if sess == nil {
		s.mu.Unlock()
		return JoinResult{}, ErrUnknownSession
	}

	if m := s.activeMatchLocked(sess); m != nil {
		res := s.joinResultForMatchLocked(sess, m)
		res.InMatch = true
		s.mu.Unlock()
		return res, nil
	}

	s.removeFromQueueLocked(sessionID)
	s.mu.Unlock()

	s.dispatch([]pendingEvent{{
		sessionID: sessionID,
		ev:        Event{Type: EventCancelled, Data: map[string]any{}},
	}})

	return JoinResult{State: models.StateIdle}, nil
}

// joinResultForMatchLocked describes an existing match to a session that is
// already part of it.
func (s *Store) joinResultForMatchLocked(sess *models.Session, m *models.Match) JoinResult {
	res := JoinResult{
		State:      models.StateConnecting,
		MatchID:    m.ID,
		MatchedAt:  m.MatchedAt,
		Initiator:  m.Initiator() == sess.ID,
		SkipLockMs: s.opts.SkipLock.Milliseconds(),
	}
	if partner := s.byID[m.PartnerOf(sess.ID)]; partner != nil {
		res.PartnerUsername = partner.Username
	}
	if m.BothReady() {
		res.State = models.StateMatched
	}
	return res
}

// createMatchLocked pairs two sessions. The caller holds the lock and has
// already checked that both are free.
func (s *Store) createMatchLocked(waiterID, joinerID string, now time.Time) *models.Match {
	id, err := utils.RandomToken(16)
	if err != nil {
		// Randomness failing is fatal for sessions too, but a match id is
		// only an address: a time based id still keeps the flow moving.
		id = utils.ShortID(utils.HashToken(now.Format(time.RFC3339Nano)))
	}

	m := models.NewMatch(id, waiterID, joinerID, now)
	s.matches[id] = m

	for _, sid := range []string{waiterID, joinerID} {
		if sess := s.byID[sid]; sess != nil {
			sess.MatchID = id
			sess.Ready = false
		}
	}

	s.counters.MatchesCreated++
	return m
}

// matchedEventsLocked builds the "you are matched" push for both participants.
func (s *Store) matchedEventsLocked(m *models.Match) []pendingEvent {
	now := s.now()
	events := make([]pendingEvent, 0, 2)

	for _, sid := range []string{m.A, m.B} {
		partnerName := ""
		if partner := s.byID[m.PartnerOf(sid)]; partner != nil {
			partnerName = partner.Username
		}
		events = append(events, pendingEvent{
			sessionID: sid,
			ev: Event{
				Type: EventMatched,
				Data: MatchPayload{
					MatchID:         m.ID,
					PartnerUsername: partnerName,
					Initiator:       m.Initiator() == sid,
					MatchedAt:       m.MatchedAt,
					SkipLockMs:      s.opts.SkipLock.Milliseconds(),
					ServerTime:      now,
				},
			},
		})
	}

	return events
}
