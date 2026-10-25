package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/config"
	"strgback/middleware"
	"strgback/store"
)

// maxBodyBytes caps a JSON request body. Nothing the API accepts is large: a
// username, a report reason, a short note.
const maxBodyBytes = 8 << 10

// API bundles everything the HTTP handlers need. It is built once in New and
// shared by every request.
type API struct {
	Store  *store.Store
	Hub    *Hub
	Logger *slog.Logger

	// Limiters are per route family: creating sessions is keyed by address,
	// joining is keyed by session, and signaling messages are keyed by
	// session with a per second budget.
	SessionLimiter *middleware.Limiter
	JoinLimiter    *middleware.Limiter
	SignalLimiter  *middleware.Limiter

	// SkipLock is copied from the store so the public config endpoint and the
	// enforcement path cannot disagree.
	SkipLock time.Duration

	notReady atomic.Bool
}

// New wires the HTTP layer, including the signaling hub, and registers the hub
// as the store's event sink.
func New(st *store.Store, logger *slog.Logger) *API {
	if logger == nil {
		logger = slog.Default()
	}

	api := &API{
		Store:          st,
		Logger:         logger,
		SessionLimiter: middleware.NewLimiter(config.App.RateLimit.SessionPerMinute, time.Minute),
		JoinLimiter:    middleware.NewLimiter(config.App.RateLimit.JoinPerMinute, time.Minute),
		SignalLimiter:  middleware.NewLimiter(config.App.RateLimit.SignalPerSecond, time.Second),
		SkipLock:       st.SkipLock(),
	}

	api.Hub = NewHub(st, api.SignalLimiter, logger)
	st.SetSink(api.Hub)

	return api
}

// MarkNotReady flips /readyz to 503. main.go calls it before shutting down so
// a load balancer stops sending new traffic.
func (a *API) MarkNotReady() { a.notReady.Store(true) }

// IsReady reports whether the process should receive traffic.
func (a *API) IsReady() bool { return !a.notReady.Load() && a.Store != nil }

// decodeJSON reads a JSON body into target. An empty body is not an error:
// every request in this API has optional fields.
func decodeJSON(c *gin.Context, target any) error {
	if c.Request.Body == nil {
		return nil
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

// matchStatePayload is the shape the frontend renders its screens from. It is
// shared by join, cancel and status so all three always agree.
type matchStatePayload struct {
	State            string    `json:"state"`
	MatchID          string    `json:"matchId,omitempty"`
	PartnerUsername  string    `json:"partnerUsername,omitempty"`
	Initiator        bool      `json:"initiator"`
	MatchedAt        time.Time `json:"matchedAt,omitempty"`
	SkipLockMs       int64     `json:"skipLockMs"`
	RemainingMs      int64     `json:"remainingMs"`
	QueuePosition    int       `json:"queuePosition,omitempty"`
	Ready            bool      `json:"ready"`
	PeerReady        bool      `json:"peerReady"`
	LastEndedReason  string    `json:"lastEndedReason,omitempty"`
	LastEndedMatchID string    `json:"lastEndedMatchId,omitempty"`
	ServerTime       time.Time `json:"serverTime"`
}

func stateFromStatus(view store.StatusView, now time.Time) matchStatePayload {
	return matchStatePayload{
		State:            string(view.State),
		MatchID:          view.MatchID,
		PartnerUsername:  view.PartnerUsername,
		Initiator:        view.Initiator,
		MatchedAt:        view.MatchedAt,
		SkipLockMs:       view.SkipLockMs,
		RemainingMs:      view.RemainingMs,
		QueuePosition:    view.QueuePosition,
		Ready:            view.Ready,
		PeerReady:        view.PeerReady,
		LastEndedReason:  view.LastEndedReason,
		LastEndedMatchID: view.LastEndedMatchID,
		ServerTime:       now,
	}
}

func stateFromJoin(res store.JoinResult, skipLock time.Duration, now time.Time) matchStatePayload {
	remaining := int64(0)
	if !res.MatchedAt.IsZero() {
		remaining = remainingSkipMs(res.MatchedAt, skipLock, now)
	}
	return matchStatePayload{
		State:           string(res.State),
		MatchID:         res.MatchID,
		PartnerUsername: res.PartnerUsername,
		Initiator:       res.Initiator,
		MatchedAt:       res.MatchedAt,
		SkipLockMs:      skipLock.Milliseconds(),
		RemainingMs:     remaining,
		QueuePosition:   res.QueuePosition,
		ServerTime:      now,
	}
}

// remainingSkipMs mirrors the store's calculation so a join response reports
// the same window a status poll would.
func remainingSkipMs(matchedAt time.Time, lock time.Duration, now time.Time) int64 {
	remaining := lock - now.Sub(matchedAt)
	if remaining < 0 {
		return 0
	}
	return remaining.Milliseconds()
}
