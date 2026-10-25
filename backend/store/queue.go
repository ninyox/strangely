package store

import (
	"time"

	"strgback/models"
)

// activeMatchLocked returns the session's running match, or nil.
func (s *Store) activeMatchLocked(sess *models.Session) *models.Match {
	if sess.MatchID == "" {
		return nil
	}
	m := s.matches[sess.MatchID]
	if m == nil || !m.IsActive() {
		return nil
	}
	return m
}

// popEligibleWaiterLocked returns the oldest waiting session that can still be
// matched, dropping entries whose session expired, gave up, or already found
// somebody. FIFO order is what makes the queue fair.
func (s *Store) popEligibleWaiterLocked(now time.Time) string {
	for len(s.queue) > 0 {
		candidate := s.queue[0]
		s.queue = s.queue[1:]

		queuedAt, waiting := s.queuedAt[candidate]
		if !waiting {
			continue
		}
		delete(s.queuedAt, candidate)

		sess := s.byID[candidate]
		if sess == nil || sess.Expired(now) || s.activeMatchLocked(sess) != nil {
			s.counters.StaleQueueDropped++
			continue
		}
		if now.Sub(queuedAt) > s.opts.QueueTTL {
			s.counters.StaleQueueDropped++
			continue
		}

		return candidate
	}
	return ""
}

// removeFromQueueLocked drops a session from the waiting line if present.
func (s *Store) removeFromQueueLocked(sessionID string) []pendingEvent {
	if _, waiting := s.queuedAt[sessionID]; !waiting {
		return nil
	}
	delete(s.queuedAt, sessionID)

	kept := s.queue[:0]
	for _, id := range s.queue {
		if id != sessionID {
			kept = append(kept, id)
		}
	}
	s.queue = kept

	return nil
}

// queuePositionLocked is the 1-based place in line, or 0 when not waiting.
func (s *Store) queuePositionLocked(sessionID string) int {
	for i, id := range s.queue {
		if id == sessionID {
			return i + 1
		}
	}
	return 0
}

// remainingSkipMs is how much of the five second window is left, never below
// zero. It is always derived from the server's match timestamp.
func remainingSkipMs(m *models.Match, now time.Time, lock time.Duration) int64 {
	remaining := lock - now.Sub(m.MatchedAt)
	if remaining < 0 {
		return 0
	}
	return remaining.Milliseconds()
}
