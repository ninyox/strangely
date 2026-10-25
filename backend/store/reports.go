package store

import (
	"errors"
	"time"

	"strgback/models"
	"strgback/utils"
)

// ErrReportRateLimited means the session reported more matches than the budget
// allows in the current window.
var ErrReportRateLimited = errors.New("too many reports")

// SaveReport records a report against a match. Reporting is deliberately not
// tied to the Skip window: a visitor can report during the first five seconds
// of a call, and the report is accepted even after the call has ended.
func (s *Store) SaveReport(sessionID, matchID, reason, note string) (models.Report, error) {
	now := s.now()

	s.mu.Lock()
	sess := s.byID[sessionID]
	if sess == nil {
		s.mu.Unlock()
		return models.Report{}, ErrUnknownSession
	}

	// A report is only accepted for a match the reporter is (or was) part of.
	m := s.matches[matchID]
	if m == nil || !m.Has(sessionID) {
		s.mu.Unlock()
		return models.Report{}, ErrNotInMatch
	}

	cutoff := now.Add(-10 * time.Minute)
	kept := s.reportAt[sessionID][:0]
	for _, at := range s.reportAt[sessionID] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	s.reportAt[sessionID] = kept

	if limit := s.reportLimit(); limit > 0 && len(kept) >= limit {
		s.mu.Unlock()
		return models.Report{}, ErrReportRateLimited
	}

	id, err := utils.RandomToken(8)
	if err != nil {
		id = utils.ShortID(utils.HashToken(now.Format(time.RFC3339Nano)))
	}

	report := models.Report{
		ID:         id,
		MatchID:    matchID,
		ReporterID: sessionID,
		Reason:     reason,
		Note:       note,
		CreatedAt:  now,
	}

	s.reports = append(s.reports, report)
	s.reportAt[sessionID] = append(s.reportAt[sessionID], now)
	s.counters.ReportsReceived++

	// Keep the newest MaxReports so a long running process cannot grow
	// without bound; the durable copy belongs in a datastore, see README.
	if max := s.opts.MaxReports; max > 0 && len(s.reports) > max {
		s.reports = append([]models.Report(nil), s.reports[len(s.reports)-max:]...)
	}

	s.mu.Unlock()

	return report, nil
}

// Reports returns a copy of the stored reports, newest last. It exists for the
// tests and for a future moderation surface.
func (s *Store) Reports() []models.Report {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]models.Report(nil), s.reports...)
}

func (s *Store) reportLimit() int {
	return s.opts.ReportsPerTenMinutes
}
