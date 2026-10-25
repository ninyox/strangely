package models

import "time"

// State is the lifecycle value the API reports for a session. The frontend
// renders one screen per state, so these strings are part of the contract.
type State string

const (
	// StateIdle means the visitor has a session but is not in the queue and
	// not in a call.
	StateIdle State = "idle"
	// StateQueued means the session is waiting for a partner.
	StateQueued State = "queued"
	// StateConnecting means two sessions are paired but the peer has not
	// finished negotiating its peer connection yet.
	StateConnecting State = "connecting"
	// StateMatched means both sides reported their peer connection is up.
	StateMatched State = "matched"
	// StateEnded means the last match finished; the visitor is back to idle
	// unless they ask for a new match.
	StateEnded State = "ended"
)

// Reasons a match can end. They are surfaced to the remaining participant so
// the UI can say "Your partner left" rather than showing a frozen video.
const (
	ReasonSkipped      = "skipped"
	ReasonStopped      = "stopped"
	ReasonDisconnected = "disconnected"
	ReasonLeft         = "left"
	ReasonSessionEnded = "session_ended"
	ReasonTooLong      = "too_long"
)

// Session is one anonymous visit. Only the hash of the token is stored: the
// raw value lives in the browser's HttpOnly cookie and nowhere else.
type Session struct {
	ID         string
	TokenHash  string
	Username   string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time

	// MatchID is the current match, empty when not in a call.
	MatchID string

	// Ready is set when the client reports its peer connection is up.
	Ready bool

	// Connected / LastSignalAt are maintained by the signaling hub. They are
	// what the janitor uses to notice an abandoned tab.
	Connected    bool
	LastSignalAt time.Time

	// LastEnded* remembers how the previous call finished so the API can
	// report an explicit "ended" state (with a reason) instead of silently
	// snapping back to idle.
	LastEndedAt      time.Time
	LastEndedReason  string
	LastEndedMatchID string
}

// Expired reports whether the session has outlived its TTL at the given time.
func (s *Session) Expired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

// Match is a pairing of exactly two sessions. A is the session that was
// already waiting (the WebRTC offerer), B is the one that arrived and
// completed the pair.
type Match struct {
	ID          string
	A           string
	B           string
	MatchedAt   time.Time
	EndedAt     time.Time
	EndedReason string
	EndedBy     string

	ready map[string]bool
}

// NewMatch pairs two sessions. The caller is responsible for the atomicity of
// "both sessions are free".
func NewMatch(id, a, b string, matchedAt time.Time) *Match {
	return &Match{
		ID:        id,
		A:         a,
		B:         b,
		MatchedAt: matchedAt,
		ready:     map[string]bool{},
	}
}

// IsActive reports whether the match is still running.
func (m *Match) IsActive() bool { return m.EndedAt.IsZero() }

// Has reports whether the given session is one of the two participants.
func (m *Match) Has(sessionID string) bool {
	return m.A == sessionID || m.B == sessionID
}

// PartnerOf returns the other participant, or "" when the session is not part
// of this match.
func (m *Match) PartnerOf(sessionID string) string {
	switch sessionID {
	case m.A:
		return m.B
	case m.B:
		return m.A
	default:
		return ""
	}
}

// Initiator is the session that creates the WebRTC offer. Having the server
// pick one side keeps both clients from offering at the same time.
func (m *Match) Initiator() string { return m.A }

// Ready reports whether a participant finished negotiating.
func (m *Match) Ready(sessionID string) bool { return m.ready[sessionID] }

// SetReady records that a participant's peer connection is up.
func (m *Match) SetReady(sessionID string, ready bool) { m.ready[sessionID] = ready }

// BothReady reports whether both participants reported a working connection.
func (m *Match) BothReady() bool { return m.ready[m.A] && m.ready[m.B] }

// Elapsed is how long the match has been running at the given time.
func (m *Match) Elapsed(now time.Time) time.Duration {
	end := now
	if !m.EndedAt.IsZero() {
		end = m.EndedAt
	}
	return end.Sub(m.MatchedAt)
}

// Report is one abuse report against a match. It deliberately stores no IP
// address, no user agent, and no free-form identity: the point is to know that
// something was reported, not to build a profile.
type Report struct {
	ID         string
	MatchID    string
	ReporterID string
	Reason     string
	Note       string
	CreatedAt  time.Time
}

// ReportReasons is the closed set of reasons the API accepts.
var ReportReasons = []string{
	"nudity_or_sexual_content",
	"harassment_or_hate",
	"minor",
	"spam_or_scam",
	"self_harm",
	"other",
}

// ValidReportReason reports whether reason is one of the accepted values.
func ValidReportReason(reason string) bool {
	for _, allowed := range ReportReasons {
		if allowed == reason {
			return true
		}
	}
	return false
}
