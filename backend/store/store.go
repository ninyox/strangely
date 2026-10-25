package store

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"strgback/models"
)

var (
	// ErrUnknownSession means the session id is not (or no longer) known.
	ErrUnknownSession = errors.New("unknown session")
	// ErrNotInMatch means the caller tried to act on a match it is not part of.
	ErrNotInMatch = errors.New("session is not a participant of this match")
)

// Event names the store pushes to a connected browser over the signaling
// socket. The frontend switches on these strings.
const (
	EventMatched         = "matched"
	EventPartnerLeft     = "partner_left"
	EventPartnerReady    = "partner_ready"
	EventPeerUnreachable = "peer_unreachable"
	EventQueued          = "queued"
	EventCancelled       = "cancelled"
	EventSessionClosed   = "session_closed"
)

// Event is one server push addressed to a single session.
type Event struct {
	Type string
	Data any
}

// Sink is how the store reaches connected clients without knowing anything
// about WebSockets. The signaling hub implements it; tests use a recorder.
type Sink interface {
	Send(sessionID string, ev Event)
}

// Options are the timers the store enforces. They are configuration driven so
// tests can compress five seconds into a few milliseconds.
type Options struct {
	SessionTTL       time.Duration
	QueueTTL         time.Duration
	MatchIdleTTL     time.Duration
	MaxMatchDuration time.Duration
	SkipLock         time.Duration

	// MatchRetention keeps an ended match answerable so a duplicate
	// skip/stop arrives as "already ended" instead of "unknown match".
	MatchRetention  time.Duration
	ReportRetention time.Duration
	MaxReports      int
	// ReportsPerTenMinutes is how many reports one session may file inside a
	// ten minute window.
	ReportsPerTenMinutes int

	// Now is injectable for tests.
	Now func() time.Time
}

// DefaultOptions returns the timers the production server runs with.
func DefaultOptions() Options {
	return Options{
		SessionTTL:       2 * time.Hour,
		QueueTTL:         2 * time.Minute,
		MatchIdleTTL:     20 * time.Second,
		MaxMatchDuration: 60 * time.Minute,
		SkipLock:         5 * time.Second,
		MatchRetention:   2 * time.Minute,
		ReportRetention:  24 * time.Hour,
		MaxReports:       1000,

		ReportsPerTenMinutes: 5,
		Now:                  time.Now,
	}
}

func (o Options) withDefaults() Options {
	defaults := DefaultOptions()
	if o.SessionTTL <= 0 {
		o.SessionTTL = defaults.SessionTTL
	}
	if o.QueueTTL <= 0 {
		o.QueueTTL = defaults.QueueTTL
	}
	if o.MatchIdleTTL <= 0 {
		o.MatchIdleTTL = defaults.MatchIdleTTL
	}
	if o.MaxMatchDuration <= 0 {
		o.MaxMatchDuration = defaults.MaxMatchDuration
	}
	if o.SkipLock < 0 {
		o.SkipLock = defaults.SkipLock
	}
	if o.MatchRetention <= 0 {
		o.MatchRetention = defaults.MatchRetention
	}
	if o.ReportRetention <= 0 {
		o.ReportRetention = defaults.ReportRetention
	}
	if o.MaxReports <= 0 {
		o.MaxReports = defaults.MaxReports
	}
	if o.ReportsPerTenMinutes <= 0 {
		o.ReportsPerTenMinutes = defaults.ReportsPerTenMinutes
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}

// Counters are lifetime totals, useful on /readyz and in logs.
type Counters struct {
	SessionsCreated   uint64
	SessionsExpired   uint64
	MatchesCreated    uint64
	MatchesEnded      uint64
	StaleQueueDropped uint64
	ReportsReceived   uint64
}

// Stats is a snapshot of the in-memory state.
type Stats struct {
	Sessions      int
	Queued        int
	ActiveMatches int
	Reports       int
	UptimeSeconds int64
	Counters      Counters
}

// Store keeps sessions, the matchmaking queue and matches in process memory.
//
// It is deliberately a concrete type with small methods rather than a database
// abstraction: the first release is a single process, and the methods
// (Enqueue, Skip, Stop, Cleanup) are the seams a Redis backed implementation
// would fill later. In-memory state does not survive a restart and does not
// work across replicas; that is documented in README.md.
type Store struct {
	mu sync.Mutex

	opts   Options
	logger *slog.Logger

	byHash map[string]*models.Session
	byID   map[string]*models.Session

	queue    []string
	queuedAt map[string]time.Time

	matches  map[string]*models.Match
	endedAt  map[string]time.Time
	reports  []models.Report
	reportAt map[string][]time.Time

	sink Sink

	counters Counters
	started  time.Time

	janitorStop chan struct{}
	janitorOnce sync.Once
}

// New builds an empty store.
func New(opts Options, logger *slog.Logger) *Store {
	if logger == nil {
		logger = slog.Default()
	}
	return &Store{
		opts:        opts.withDefaults(),
		logger:      logger,
		byHash:      make(map[string]*models.Session),
		byID:        make(map[string]*models.Session),
		queue:       make([]string, 0, 64),
		queuedAt:    make(map[string]time.Time),
		matches:     make(map[string]*models.Match),
		endedAt:     make(map[string]time.Time),
		reportAt:    make(map[string][]time.Time),
		janitorStop: make(chan struct{}),
		started:     time.Now(),
	}
}

// SetSink attaches the delivery mechanism for server pushes. It is called once,
// after the signaling hub exists.
func (s *Store) SetSink(sink Sink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sink = sink
}

func (s *Store) now() time.Time { return s.opts.Now() }

// SkipLock is the configured five second window, exposed so handlers and the
// public /api/config endpoint cannot drift from what the store enforces.
func (s *Store) SkipLock() time.Duration { return s.opts.SkipLock }

type pendingEvent struct {
	sessionID string
	ev        Event
}

// dispatch delivers events after the store lock has been released, so a sink
// can never deadlock the store by calling back into it.
func (s *Store) dispatch(events []pendingEvent) {
	if len(events) == 0 {
		return
	}
	s.mu.Lock()
	sink := s.sink
	s.mu.Unlock()
	if sink == nil {
		return
	}
	for _, e := range events {
		sink.Send(e.sessionID, e.ev)
	}
}

// StartJanitor runs Cleanup on a ticker until StopJanitor is called.
func (s *Store) StartJanitor(interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.janitorStop:
				return
			case <-ticker.C:
				s.Cleanup()
			}
		}
	}()
}

// StopJanitor stops the background sweep. It is safe to call more than once.
func (s *Store) StopJanitor() {
	s.janitorOnce.Do(func() { close(s.janitorStop) })
}

// SessionView is the read-only projection of a session handed to handlers. It
// is a copy on purpose: handlers must not be able to mutate store state or read
// a field that another goroutine is writing.
type SessionView struct {
	ID       string
	Username string
}
