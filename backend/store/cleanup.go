package store

import (
	"time"

	"strgback/models"
)

// CleanupStats describes what one sweep removed. It is logged at debug level
// and returned so tests can assert on it.
type CleanupStats struct {
	SessionsExpired int
	QueueDropped    int
	MatchesEnded    int
	MatchesPurged   int
	ReportsPurged   int
}

// Empty reports whether a sweep changed nothing.
func (c CleanupStats) Empty() bool {
	return c.SessionsExpired == 0 && c.QueueDropped == 0 && c.MatchesEnded == 0 &&
		c.MatchesPurged == 0 && c.ReportsPurged == 0
}

// Cleanup is the janitor pass: it expires sessions, drops abandoned queue
// entries, ends matches whose participants vanished or that ran too long, and
// forgets finished matches and old reports. Everything it does is also done
// lazily on the hot paths; this sweep is what keeps a process that nobody is
// using from holding memory.
func (s *Store) Cleanup() CleanupStats {
	now := s.now()
	stats := CleanupStats{}
	var events []pendingEvent

	s.mu.Lock()

	// 1. Sessions past their TTL. Removing one also ends its match, and drops
	// it from the queue if it was still waiting.
	for _, sess := range s.sessionsLocked() {
		if !sess.Expired(now) {
			continue
		}
		if _, queued := s.queuedAt[sess.ID]; queued {
			stats.QueueDropped++
		}
		if m := s.activeMatchLocked(sess); m != nil {
			stats.MatchesEnded++
		}
		events = append(events, s.removeSessionLocked(sess, models.ReasonSessionEnded, now)...)
		s.counters.SessionsExpired++
		stats.SessionsExpired++
	}

	// 2. Waiting sessions that went away, waited too long, or have no socket
	// left to be matched over.
	kept := s.queue[:0]
	for _, id := range s.queue {
		queuedAt, waiting := s.queuedAt[id]
		if !waiting {
			continue
		}
		sess := s.byID[id]

		// A session with no signaling socket cannot answer an offer, so it
		// must not stay in line: it would be matched and leave the other side
		// with a partner that is not there. It gets the same grace a live
		// match gets, so a reconnect is not punished.
		unreachable := sess != nil && !sess.Connected &&
			s.idleForLocked(sess, now) > s.opts.MatchIdleTTL

		stale := sess == nil ||
			now.Sub(queuedAt) > s.opts.QueueTTL ||
			s.activeMatchLocked(sess) != nil ||
			unreachable
		if stale {
			delete(s.queuedAt, id)
			s.counters.StaleQueueDropped++
			stats.QueueDropped++
			continue
		}
		kept = append(kept, id)
	}
	s.queue = kept

	// 3. Matches with a participant that stopped answering, or that ran past
	// the maximum call length.
	for _, m := range s.matchesLocked() {
		if !m.IsActive() {
			continue
		}

		reason := ""
		switch {
		case m.Elapsed(now) > s.opts.MaxMatchDuration:
			reason = models.ReasonTooLong
		default:
			for _, sid := range []string{m.A, m.B} {
				sess := s.byID[sid]
				if sess == nil {
					reason = models.ReasonSessionEnded
					break
				}
				if !sess.Connected && s.idleForLocked(sess, now) > s.opts.MatchIdleTTL {
					reason = models.ReasonDisconnected
					break
				}
			}
		}
		if reason == "" {
			continue
		}

		events = append(events, s.endMatchLocked(m, reason, "", now)...)
		stats.MatchesEnded++
	}

	// 4. Finished matches are kept briefly so a duplicate skip/stop is
	// answered idempotently, then forgotten.
	for id, endedAt := range s.endedAt {
		if now.Sub(endedAt) > s.opts.MatchRetention {
			delete(s.endedAt, id)
			delete(s.matches, id)
			stats.MatchesPurged++
		}
	}

	// 5. Reports age out; the first release keeps them in memory only.
	if len(s.reports) > 0 {
		cutoff := now.Add(-s.opts.ReportRetention)
		keptReports := s.reports[:0]
		for _, r := range s.reports {
			if r.CreatedAt.After(cutoff) {
				keptReports = append(keptReports, r)
			} else {
				stats.ReportsPurged++
			}
		}
		s.reports = keptReports
	}

	s.mu.Unlock()

	s.dispatch(events)

	if !stats.Empty() {
		s.logger.Debug("store cleanup",
			"sessions_expired", stats.SessionsExpired,
			"queue_dropped", stats.QueueDropped,
			"matches_ended", stats.MatchesEnded,
			"matches_purged", stats.MatchesPurged,
			"reports_purged", stats.ReportsPurged,
		)
	}

	return stats
}

// idleForLocked is how long since a session last did anything at all, whether
// over HTTP or over the signaling socket.
func (s *Store) idleForLocked(sess *models.Session, now time.Time) time.Duration {
	last := sess.LastSeenAt
	if sess.LastSignalAt.After(last) {
		last = sess.LastSignalAt
	}
	return now.Sub(last)
}

func (s *Store) sessionsLocked() []*models.Session {
	out := make([]*models.Session, 0, len(s.byID))
	for _, sess := range s.byID {
		out = append(out, sess)
	}
	return out
}

func (s *Store) matchesLocked() []*models.Match {
	out := make([]*models.Match, 0, len(s.matches))
	for _, m := range s.matches {
		out = append(out, m)
	}
	return out
}

// Stats is a point in time snapshot for /readyz and for logs.
func (s *Store) Stats() Stats {
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()

	active := 0
	for _, m := range s.matches {
		if m.IsActive() {
			active++
		}
	}

	return Stats{
		Sessions:      len(s.byID),
		Queued:        len(s.queuedAt),
		ActiveMatches: active,
		Reports:       len(s.reports),
		UptimeSeconds: int64(now.Sub(s.started).Seconds()),
		Counters:      s.counters,
	}
}
